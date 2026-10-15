// Copyright (C) 2023 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package annotations

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/lib/pq"
	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
)

const (
	syncInterval = 5 * time.Minute
	// Most events per Write call, and most annotation IDs per Delete call.
	syncBatchSize = 500
	// Tag of an annotation that is not scoped to a node or service.
	globalTag = "pmm_annotation"

	annotationsQuery = `SELECT a.id, a.epoch, a.text, t.key, t.value
FROM annotation a
JOIN annotation_tag at ON at.annotation_id = a.id
JOIN tag t ON t.id = at.tag_id
WHERE a.org_id = 1 AND a.alert_id = 0 AND a.epoch >= $1`
	existingQuery = `SELECT id FROM annotation WHERE id = ANY($1)`
)

// Syncer writes the missing series of scoped Grafana annotations, and deletes the series of deleted annotations.
type Syncer struct {
	db        *reform.DB
	grafanaDB *sql.DB
	store     syncStore
	l         *logrus.Entry
}

// grafanaAnnotation is an annotation read from Grafana's database.
type grafanaAnnotation struct {
	id    int64
	epoch int64
	text  string
	tags  []string
}

// inventory holds the nodes and services that annotation tags are resolved against.
type inventory struct {
	nodesByName    map[string]*models.Node
	nodesByID      map[string]*models.Node
	servicesByName map[string]*models.Service
}

// NewSyncer returns a Syncer that reads Grafana's database and writes to the store.
func NewSyncer(db *reform.DB, grafanaDB *sql.DB, store syncStore) *Syncer {
	return &Syncer{
		db:        db,
		grafanaDB: grafanaDB,
		store:     store,
		l:         logrus.WithField("component", "annotations/sync"),
	}
}

// Run syncs once, then every five minutes until ctx is done.
func (s *Syncer) Run(ctx context.Context) {
	ticker := time.NewTicker(syncInterval)
	defer ticker.Stop()

	for {
		// A hung connection must not stall the job past the next run.
		passCtx, cancel := context.WithTimeout(ctx, syncInterval)
		err := s.syncOnce(passCtx)
		cancel()
		if err != nil && ctx.Err() == nil {
			s.l.WithError(err).Error("Annotation sync failed, retrying at the next run.")
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// syncOnce writes the series VictoriaMetrics misses, then deletes the series of annotations Grafana no longer has.
func (s *Syncer) syncOnce(ctx context.Context) error {
	settings, err := models.GetSettings(s.db.Querier)
	if err != nil {
		return fmt.Errorf("failed to get settings: %w", err)
	}
	window := settings.DataRetention

	// Listed before Grafana is read, so that the series of an annotation removed in between are not written back.
	keys, err := s.store.Keys(ctx, window)
	if err != nil {
		return fmt.Errorf("failed to list annotation series: %w", err)
	}

	events, err := s.grafanaEvents(ctx, window)
	if err != nil {
		return err
	}

	writeErr := s.writeMissing(ctx, events, keys)
	deleteErr := s.deleteAbsent(ctx, keys)

	return errors.Join(writeErr, deleteErr)
}

// grafanaEvents returns the events of the scoped Grafana annotations within the window.
func (s *Syncer) grafanaEvents(ctx context.Context, window time.Duration) ([]Event, error) {
	inv, err := s.loadInventory()
	if err != nil {
		return nil, err
	}

	annotations, err := s.grafanaAnnotations(ctx, time.Now().Add(-window))
	if err != nil {
		return nil, err
	}

	events := make([]Event, 0, len(annotations))
	for _, a := range annotations {
		events = append(events, s.annotationEvents(a, inv)...)
	}

	return events, nil
}

// loadInventory loads the current nodes and services.
func (s *Syncer) loadInventory() (*inventory, error) {
	nodes, err := models.FindNodes(s.db.Querier, models.NodeFilters{})
	if err != nil {
		return nil, fmt.Errorf("failed to find nodes: %w", err)
	}

	services, err := models.FindServices(s.db.Querier, models.ServiceFilters{})
	if err != nil {
		return nil, fmt.Errorf("failed to find services: %w", err)
	}

	inv := &inventory{
		nodesByName:    make(map[string]*models.Node, len(nodes)),
		nodesByID:      make(map[string]*models.Node, len(nodes)),
		servicesByName: make(map[string]*models.Service, len(services)),
	}
	for _, node := range nodes {
		inv.nodesByName[node.NodeName] = node
		inv.nodesByID[node.NodeID] = node
	}
	for _, service := range services {
		inv.servicesByName[service.ServiceName] = service
	}

	return inv, nil
}

// grafanaAnnotations reads the tagged non-alert annotations of the main organization since the given time.
func (s *Syncer) grafanaAnnotations(ctx context.Context, since time.Time) ([]*grafanaAnnotation, error) {
	rows, err := s.grafanaDB.QueryContext(ctx, annotationsQuery, since.UnixMilli())
	if err != nil {
		return nil, fmt.Errorf("failed to read Grafana annotations: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	var res []*grafanaAnnotation
	byID := make(map[int64]*grafanaAnnotation)
	for rows.Next() {
		var a grafanaAnnotation
		var key, value string
		err = rows.Scan(&a.id, &a.epoch, &a.text, &key, &value)
		if err != nil {
			return nil, fmt.Errorf("failed to read Grafana annotations: %w", err)
		}

		// Grafana stores the tag "key:value" as a key and a value.
		tag := key
		if value != "" {
			tag += ":" + value
		}

		existing, ok := byID[a.id]
		if !ok {
			existing = &a
			byID[a.id] = existing
			res = append(res, existing)
		}
		existing.tags = append(existing.tags, tag)
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("failed to read Grafana annotations: %w", err)
	}

	return res, nil
}

// annotationEvents returns an event for each node and service the annotation's tags name; other tags are user tags.
func (s *Syncer) annotationEvents(a *grafanaAnnotation, inv *inventory) []Event {
	if slices.Contains(a.tags, globalTag) {
		return nil
	}

	l := s.l.WithField("annotation_id", a.id)

	var events []Event
	var userTags []string
	for _, tag := range a.tags {
		node := inv.nodesByName[tag]
		service := inv.servicesByName[tag]
		asNode, asService := node != nil, service != nil
		if asNode && asService {
			suffixNode, suffixService := suffixScope(a.text, tag)
			// Without a suffix naming it, the annotation shows for both, as Grafana's tag match did.
			if suffixNode || suffixService {
				asNode, asService = suffixNode, suffixService
			}
		}

		if !asNode && !asService {
			userTags = append(userTags, tag)
			continue
		}

		if asService {
			serviceNode := inv.nodesByID[service.NodeID]
			if serviceNode == nil {
				l.WithField("service_name", service.ServiceName).Warn("Skipping a service whose node is not found.")
			} else {
				events = appendEvent(l, events, serviceNode, service)
			}
		}

		if asNode {
			events = appendEvent(l, events, node, nil)
		}
	}

	for i := range events {
		events[i].AnnotationID = a.id
		events[i].Time = time.UnixMilli(a.epoch)
		events[i].Text = a.text
		events[i].Tags = userTags
	}

	return events
}

// appendEvent appends an event with the labels of the node, or of the service on that node.
func appendEvent(l *logrus.Entry, events []Event, node *models.Node, service *models.Service) []Event {
	labels, err := ObjectLabels(node, service)
	if err != nil {
		l.WithError(err).Warn("Skipping an object whose labels cannot be read.")
		return events
	}

	return append(events, Event{Labels: labels})
}

// suffixScope reports whether the text suffix AddAnnotation appends names name as the node, and as one of the services.
func suffixScope(text, name string) (bool, bool) {
	asNode := strings.HasSuffix(text, "Node Name: "+name+")")

	i := strings.LastIndex(text, "Service Name: ")
	if i < 0 {
		return asNode, false
	}

	list := strings.TrimSuffix(text[i+len("Service Name: "):], ")")
	list, _, _ = strings.Cut(list, ". Node Name: ")

	return asNode, slices.Contains(strings.Split(list, ", "), name)
}

// writeMissing writes the events whose series VictoriaMetrics does not have.
func (s *Syncer) writeMissing(ctx context.Context, events []Event, keys []Key) error {
	present := make(map[Key]struct{}, len(keys))
	for _, k := range keys {
		present[k] = struct{}{}
	}

	var missing []Event
	for _, e := range events {
		key := Key{AnnotationID: e.AnnotationID, NodeName: e.Labels["node_name"], ServiceName: e.Labels["service_name"]}
		if _, ok := present[key]; ok {
			continue
		}

		err := e.Validate()
		if err != nil {
			s.l.WithField("annotation_id", e.AnnotationID).WithError(err).Warn("Skipping an annotation series.")
			continue
		}

		missing = append(missing, e)
	}

	for batch := range slices.Chunk(missing, syncBatchSize) {
		err := s.store.Write(ctx, batch)
		if err != nil {
			return fmt.Errorf("failed to write annotation series: %w", err)
		}
	}

	if len(missing) != 0 {
		s.l.WithField("count", len(missing)).Info("Wrote missing annotation series.")
	}

	return nil
}

// deleteAbsent deletes the series of the annotations Grafana's database positively no longer has.
func (s *Syncer) deleteAbsent(ctx context.Context, keys []Key) error {
	ids := make([]int64, 0, len(keys))
	for _, k := range keys {
		ids = append(ids, k.AnnotationID)
	}
	slices.Sort(ids)
	ids = slices.Compact(ids)
	if len(ids) == 0 {
		return nil
	}

	existing, err := s.existingAnnotations(ctx, ids)
	if err != nil {
		return err
	}

	absent := make([]int64, 0, len(ids))
	for _, id := range ids {
		if _, ok := existing[id]; !ok {
			absent = append(absent, id)
		}
	}

	for batch := range slices.Chunk(absent, syncBatchSize) {
		err = s.store.Delete(ctx, batch)
		if err != nil {
			return fmt.Errorf("failed to delete annotation series: %w", err)
		}
	}

	if len(absent) != 0 {
		s.l.WithField("count", len(absent)).Info("Deleted the series of deleted annotations.")
	}

	return nil
}

// existingAnnotations returns which of the given annotation IDs Grafana's database has.
func (s *Syncer) existingAnnotations(ctx context.Context, ids []int64) (map[int64]struct{}, error) {
	rows, err := s.grafanaDB.QueryContext(ctx, existingQuery, pq.Array(ids))
	if err != nil {
		return nil, fmt.Errorf("failed to look up Grafana annotations: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	existing := make(map[int64]struct{}, len(ids))
	for rows.Next() {
		var id int64
		err = rows.Scan(&id)
		if err != nil {
			return nil, fmt.Errorf("failed to look up Grafana annotations: %w", err)
		}
		existing[id] = struct{}{}
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("failed to look up Grafana annotations: %w", err)
	}

	return existing, nil
}
