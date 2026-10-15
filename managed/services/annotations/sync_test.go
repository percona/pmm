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
	"database/sql/driver"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
)

const (
	testAnnotationsQuery = `SELECT a.id, a.epoch, a.text, t.key, t.value
FROM annotation a
JOIN annotation_tag at ON at.annotation_id = a.id
JOIN tag t ON t.id = at.tag_id
WHERE a.org_id = 1 AND a.alert_id = 0 AND a.epoch >= $1`
	testExistingQuery = `SELECT id FROM annotation WHERE id = ANY($1)`
)

// sinceWindow matches the epoch lower bound, in milliseconds, of a pass that reads the given window.
type sinceWindow time.Duration

func (w sinceWindow) Match(v driver.Value) bool {
	ms, ok := v.(int64)
	want := time.Now().Add(-time.Duration(w)).UnixMilli()

	return ok && ms <= want && ms > want-time.Minute.Milliseconds()
}

func TestSync(t *testing.T) {
	sqlDB := testdb.Open(t, models.SetupFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	createNode := func(t *testing.T, name string) *models.Node {
		t.Helper()

		node, err := models.CreateNode(db.Querier, models.GenericNodeType, &models.CreateNodeParams{NodeName: name})
		require.NoError(t, err)

		return node
	}

	port := uint16(3306)
	createService := func(t *testing.T, name, nodeID string, customLabels map[string]string) *models.Service {
		t.Helper()

		port++
		service, err := models.AddNewService(db.Querier, models.MySQLServiceType, &models.AddDBMSServiceParams{
			ServiceName:  name,
			NodeID:       nodeID,
			CustomLabels: customLabels,
			Address:      new("127.0.0.1"),
			Port:         new(port),
		})
		require.NoError(t, err)

		return service
	}

	n1 := createNode(t, "n1")
	s1 := createService(t, "s1", n1.NodeID, nil)
	colonService := createService(t, "db:1", n1.NodeID, nil)
	bothNode := createNode(t, "both")
	bothService := createService(t, "both", n1.NodeID, nil)

	wideLabels := make(map[string]string, MaxLabels)
	for i := range MaxLabels {
		wideLabels[fmt.Sprintf("label_%02d", i)] = "value"
	}
	createService(t, "wide", n1.NodeID, wideLabels)

	settings, err := models.GetSettings(db.Querier)
	require.NoError(t, err)
	window := settings.DataRetention

	epoch := time.Now().Add(-time.Hour).UnixMilli()

	serviceEvent := func(t *testing.T, id int64, service *models.Service, text string, tags []string) Event {
		t.Helper()

		labels, err := ObjectLabels(n1, service)
		require.NoError(t, err)

		return Event{AnnotationID: id, Time: time.UnixMilli(epoch), Text: text, Tags: tags, Labels: labels}
	}

	nodeEvent := func(t *testing.T, id int64, node *models.Node, text string) Event {
		t.Helper()

		labels, err := ObjectLabels(node, nil)
		require.NoError(t, err)

		return Event{AnnotationID: id, Time: time.UnixMilli(epoch), Text: text, Labels: labels}
	}

	setup := func(t *testing.T) (*Syncer, sqlmock.Sqlmock, *mockSyncStore) {
		t.Helper()

		grafanaDB, grafanaMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		require.NoError(t, err)

		store := &mockSyncStore{}
		store.Test(t)

		t.Cleanup(func() {
			assert.NoError(t, grafanaMock.ExpectationsWereMet())
			store.AssertExpectations(t)
			_ = grafanaDB.Close()
		})

		return NewSyncer(db, grafanaDB, store), grafanaMock, store
	}

	annotationRows := func() *sqlmock.Rows {
		return sqlmock.NewRows([]string{"id", "epoch", "text", "key", "value"})
	}

	expectAnnotations := func(grafanaMock sqlmock.Sqlmock, rows *sqlmock.Rows) {
		grafanaMock.ExpectQuery(testAnnotationsQuery).WithArgs(sinceWindow(window)).WillReturnRows(rows)
	}

	captureWrites := func(store *mockSyncStore) *[][]Event {
		var calls [][]Event
		store.On("Write", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			calls = append(calls, args.Get(1).([]Event))
		})

		return &calls
	}

	t.Run("Backfills missing series", func(t *testing.T) {
		s, grafanaMock, store := setup(t)

		text := "Deploy (Service Name: s1)"
		expectAnnotations(grafanaMock, annotationRows().
			AddRow(5, epoch, text, "s1", "").
			AddRow(5, epoch, text, "deploy", ""))
		store.On("Keys", mock.Anything, window).Return(nil, nil).Once()
		store.On("Write", mock.Anything, []Event{serviceEvent(t, 5, s1, text, []string{"deploy"})}).Return(nil).Once()

		require.NoError(t, s.syncOnce(t.Context()))
		store.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
	})

	t.Run("Skips present", func(t *testing.T) {
		s, grafanaMock, store := setup(t)

		text := "Deploy (Service Name: s1)"
		expectAnnotations(grafanaMock, annotationRows().
			AddRow(5, epoch, text, "s1", "").
			AddRow(5, epoch, text, "deploy", ""))
		store.On("Keys", mock.Anything, window).Return([]Key{{AnnotationID: 5, NodeName: "n1", ServiceName: "s1"}}, nil).Once()
		grafanaMock.ExpectQuery(testExistingQuery).WithArgs(pq.Array([]int64{5})).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(5))

		require.NoError(t, s.syncOnce(t.Context()))
		store.AssertNotCalled(t, "Write", mock.Anything, mock.Anything)
		store.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
	})

	t.Run("Skips global, alert-free and unknown", func(t *testing.T) {
		// Alert annotations never reach the sync job: the query itself requires alert_id = 0.
		s, grafanaMock, store := setup(t)

		expectAnnotations(grafanaMock, annotationRows().
			AddRow(6, epoch, "Global", "pmm_annotation", "").
			AddRow(6, epoch, "Global", "s1", "").
			AddRow(7, epoch, "Unknown", "no-such-object", "").
			AddRow(7, epoch, "Unknown", "deploy", ""))
		store.On("Keys", mock.Anything, window).Return(nil, nil).Once()

		require.NoError(t, s.syncOnce(t.Context()))
		store.AssertNotCalled(t, "Write", mock.Anything, mock.Anything)
	})

	t.Run("Rejoins key:value tags", func(t *testing.T) {
		// Grafana stores the tag "db:1" as key "db" and value "1".
		s, grafanaMock, store := setup(t)

		text := "Failover (Service Name: db:1)"
		expectAnnotations(grafanaMock, annotationRows().
			AddRow(8, epoch, text, "db", "1").
			AddRow(8, epoch, text, "env", "prod"))
		store.On("Keys", mock.Anything, window).Return(nil, nil).Once()
		store.On("Write", mock.Anything, []Event{serviceEvent(t, 8, colonService, text, []string{"env:prod"})}).Return(nil).Once()

		require.NoError(t, s.syncOnce(t.Context()))
	})

	t.Run("Node and service suffix", func(t *testing.T) {
		s, grafanaMock, store := setup(t)

		nodeText := "Reboot (Node Name: both)"
		servicesText := "Slow (Service Name: s1, both)"
		bothText := "Upgrade (Service Name: both. Node Name: both)"
		expectAnnotations(grafanaMock, annotationRows().
			AddRow(10, epoch, nodeText, "both", "").
			AddRow(11, epoch, servicesText, "s1", "").
			AddRow(11, epoch, servicesText, "both", "").
			AddRow(12, epoch, bothText, "both", ""))
		store.On("Keys", mock.Anything, window).Return(nil, nil).Once()
		writes := captureWrites(store)

		require.NoError(t, s.syncOnce(t.Context()))
		require.Len(t, *writes, 1)
		assert.ElementsMatch(t, []Event{
			nodeEvent(t, 10, bothNode, nodeText),
			serviceEvent(t, 11, s1, servicesText, nil),
			serviceEvent(t, 11, bothService, servicesText, nil),
			serviceEvent(t, 12, bothService, bothText, nil),
			nodeEvent(t, 12, bothNode, bothText),
		}, (*writes)[0])
	})

	t.Run("Node and service without suffix", func(t *testing.T) {
		// Without a suffix nothing tells them apart, so the annotation shows for both, as Grafana's tag match did.
		s, grafanaMock, store := setup(t)

		text := "Imported"
		expectAnnotations(grafanaMock, annotationRows().AddRow(13, epoch, text, "both", ""))
		store.On("Keys", mock.Anything, window).Return(nil, nil).Once()
		writes := captureWrites(store)

		require.NoError(t, s.syncOnce(t.Context()))
		require.Len(t, *writes, 1)
		assert.ElementsMatch(t, []Event{
			serviceEvent(t, 13, bothService, text, nil),
			nodeEvent(t, 13, bothNode, text),
		}, (*writes)[0])
	})

	t.Run("Skips invalid events", func(t *testing.T) {
		s, grafanaMock, store := setup(t)

		expectAnnotations(grafanaMock, annotationRows().
			AddRow(20, epoch, "Too wide", "wide", "").
			AddRow(21, epoch, "Fine", "s1", ""))
		store.On("Keys", mock.Anything, window).Return(nil, nil).Once()
		store.On("Write", mock.Anything, []Event{serviceEvent(t, 21, s1, "Fine", nil)}).Return(nil).Once()

		require.NoError(t, s.syncOnce(t.Context()))
	})

	t.Run("Deletes positively absent", func(t *testing.T) {
		s, grafanaMock, store := setup(t)

		expectAnnotations(grafanaMock, annotationRows().AddRow(5, epoch, "Deploy", "s1", ""))
		store.On("Keys", mock.Anything, window).Return([]Key{
			{AnnotationID: 9, NodeName: "n1", ServiceName: "s1"},
			{AnnotationID: 5, NodeName: "n1", ServiceName: "s1"},
			{AnnotationID: 9, NodeName: "n1"},
		}, nil).Once()
		grafanaMock.ExpectQuery(testExistingQuery).WithArgs(pq.Array([]int64{5, 9})).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(5))
		store.On("Delete", mock.Anything, []int64{9}).Return(nil).Once()

		require.NoError(t, s.syncOnce(t.Context()))
		store.AssertNotCalled(t, "Write", mock.Anything, mock.Anything)
	})

	t.Run("Never deletes on read failure", func(t *testing.T) {
		s, grafanaMock, store := setup(t)

		readErr := errors.New("connection refused")
		expectAnnotations(grafanaMock, annotationRows())
		store.On("Keys", mock.Anything, window).Return([]Key{{AnnotationID: 9, NodeName: "n1", ServiceName: "s1"}}, nil).Once()
		grafanaMock.ExpectQuery(testExistingQuery).WithArgs(pq.Array([]int64{9})).WillReturnError(readErr)

		err := s.syncOnce(t.Context())
		require.ErrorIs(t, err, readErr)
		store.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
	})

	t.Run("Never deletes on a partial read", func(t *testing.T) {
		// A read that breaks off after some rows must not make the rest look deleted.
		s, grafanaMock, store := setup(t)

		readErr := errors.New("connection reset")
		expectAnnotations(grafanaMock, annotationRows())
		store.On("Keys", mock.Anything, window).Return([]Key{
			{AnnotationID: 5, NodeName: "n1", ServiceName: "s1"},
			{AnnotationID: 9, NodeName: "n1", ServiceName: "s1"},
		}, nil).Once()
		grafanaMock.ExpectQuery(testExistingQuery).WithArgs(pq.Array([]int64{5, 9})).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(5).AddRow(9).RowError(1, readErr))

		err := s.syncOnce(t.Context())
		require.ErrorIs(t, err, readErr)
		store.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
	})

	t.Run("Does nothing when reading annotations fails", func(t *testing.T) {
		s, grafanaMock, store := setup(t)

		readErr := errors.New("connection refused")
		grafanaMock.ExpectQuery(testAnnotationsQuery).WithArgs(sinceWindow(window)).WillReturnError(readErr)

		err := s.syncOnce(t.Context())
		require.ErrorIs(t, err, readErr)
		store.AssertNotCalled(t, "Keys", mock.Anything, mock.Anything)
		store.AssertNotCalled(t, "Write", mock.Anything, mock.Anything)
		store.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
	})

	t.Run("Does nothing when listing series fails", func(t *testing.T) {
		s, grafanaMock, store := setup(t)

		keysErr := errors.New("vmselect unavailable")
		expectAnnotations(grafanaMock, annotationRows().AddRow(5, epoch, "Deploy", "s1", ""))
		store.On("Keys", mock.Anything, window).Return(nil, keysErr).Once()

		err := s.syncOnce(t.Context())
		require.ErrorIs(t, err, keysErr)
		store.AssertNotCalled(t, "Write", mock.Anything, mock.Anything)
		store.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
	})

	t.Run("Writes and deletes in batches", func(t *testing.T) {
		s, grafanaMock, store := setup(t)

		const n = syncBatchSize + 1
		rows := annotationRows()
		keys := make([]Key, 0, n)
		absent := make([]int64, 0, n)
		for i := range int64(n) {
			rows.AddRow(i+1, epoch, "Batch", "s1", "")
			keys = append(keys, Key{AnnotationID: 1001 + i, NodeName: "n1", ServiceName: "s1"})
			absent = append(absent, 1001+i)
		}

		expectAnnotations(grafanaMock, rows)
		store.On("Keys", mock.Anything, window).Return(keys, nil).Once()
		writes := captureWrites(store)
		grafanaMock.ExpectQuery(testExistingQuery).WithArgs(pq.Array(absent)).WillReturnRows(sqlmock.NewRows([]string{"id"}))
		store.On("Delete", mock.Anything, absent[:syncBatchSize]).Return(nil).Once()
		store.On("Delete", mock.Anything, absent[syncBatchSize:]).Return(nil).Once()

		require.NoError(t, s.syncOnce(t.Context()))
		require.Len(t, *writes, 2)
		assert.Len(t, (*writes)[0], syncBatchSize)
		assert.Len(t, (*writes)[1], 1)
	})

	t.Run("Run syncs at start and stops with its context", func(t *testing.T) {
		s, grafanaMock, store := setup(t)

		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		expectAnnotations(grafanaMock, annotationRows())
		store.On("Keys", mock.Anything, window).Return(nil, nil).Once().Run(func(mock.Arguments) { cancel() })

		done := make(chan struct{})
		go func() {
			s.Run(ctx)
			close(done)
		}()

		select {
		case <-done:
		case <-time.After(10 * time.Second):
			t.Fatal("Run did not return after its context was canceled")
		}
	})
}
