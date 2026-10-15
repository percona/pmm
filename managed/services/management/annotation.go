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

package management

import (
	"context"
	"errors"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	managementv1 "github.com/percona/pmm/api/management/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services"
	"github.com/percona/pmm/managed/services/annotations"
)

// AddAnnotation creates an annotation in grafana, and writes the series of a node or service annotation to VictoriaMetrics.
func (s *ManagementService) AddAnnotation(ctx context.Context, req *managementv1.AddAnnotationRequest) (*managementv1.AddAnnotationResponse, error) {
	authorization, err := callerAuthorization(ctx)
	if err != nil {
		return nil, err
	}

	events, err := s.annotationEvents(req.ServiceNames, req.NodeName)
	if err != nil {
		return nil, err
	}

	tags := req.Tags
	if len(req.ServiceNames) == 0 && req.NodeName == "" {
		tags = append([]string{"pmm_annotation"}, tags...)
	}
	var postfix []string
	if len(req.ServiceNames) != 0 {
		tags = append(tags, req.ServiceNames...)
		postfix = append(postfix, "Service Name: "+strings.Join(req.ServiceNames, ", "))
	}

	if req.NodeName != "" {
		tags = append(tags, req.NodeName)
		postfix = append(postfix, "Node Name: "+req.NodeName)
	}

	for _, tag := range tags {
		if len(tag) > 100 { //nolint:mnd
			msg := "tag length cannot exceed 100 characters, tag: " + tag
			return nil, status.Error(codes.InvalidArgument, msg)
		}
	}

	if len(postfix) != 0 {
		req.Text += " (" + strings.Join(postfix, ". ") + ")"
	}

	now := time.Now()
	for i := range events {
		events[i].Time = now
		events[i].Text = req.Text
		events[i].Tags = req.Tags

		err = events[i].Validate()
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "Cannot annotate %s: %s.", eventObject(events[i]), err)
		}
	}

	id, err := s.grafanaClient.CreateAnnotation(ctx, tags, now, req.Text, authorization)
	if err != nil {
		return nil, err
	}

	if len(events) != 0 {
		for i := range events {
			events[i].AnnotationID = id
		}

		err = s.annotationStore.Write(ctx, events)
		if err != nil {
			return nil, status.Errorf(codes.Unavailable,
				"The annotation was created (ID %d), but it is not shown on dashboards yet: the sync job will add it. Cause: %s", id, err)
		}
	}

	return &managementv1.AddAnnotationResponse{AnnotationId: id}, nil
}

// RemoveAnnotation deletes an annotation from grafana, and the series that show it on dashboards from VictoriaMetrics.
func (s *ManagementService) RemoveAnnotation(ctx context.Context, req *managementv1.RemoveAnnotationRequest) (*managementv1.RemoveAnnotationResponse, error) {
	authorization, err := callerAuthorization(ctx)
	if err != nil {
		return nil, err
	}

	err = s.grafanaClient.DeleteAnnotation(ctx, req.AnnotationId, authorization)
	if errors.Is(err, services.ErrAnnotationNotFound) {
		// An annotation deleted through Grafana's API keeps its series until the sync job runs; deleting is idempotent.
		storeErr := s.annotationStore.Delete(ctx, []int64{req.AnnotationId})
		if storeErr != nil {
			s.l.WithField("annotation_id", req.AnnotationId).WithError(storeErr).Warn("Failed to delete the series of an annotation Grafana does not have.")
		}

		return nil, status.Errorf(codes.NotFound, "Annotation with ID %d not found.", req.AnnotationId)
	}
	if err != nil {
		return nil, err
	}

	err = s.annotationStore.Delete(ctx, []int64{req.AnnotationId})
	if err != nil {
		return nil, status.Errorf(codes.Unavailable,
			"The annotation was deleted, but it may still show on dashboards until the sync job removes its series. Cause: %s", err)
	}

	return &managementv1.RemoveAnnotationResponse{}, nil
}

// callerAuthorization returns the Authorization header of the caller's request.
func callerAuthorization(ctx context.Context) (string, error) {
	headers, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", errors.New("cannot get headers from metadata")
	}

	authorizationHeaders := headers.Get("Authorization")
	if len(authorizationHeaders) == 0 {
		return "", status.Error(codes.Unauthenticated, "Authorization error.")
	}

	return authorizationHeaders[0], nil
}

// annotationEvents returns an event for each service and for the node, in this order, with the labels of the object
// it is scoped to; a service event carries the labels of the node the service runs on.
func (s *ManagementService) annotationEvents(serviceNames []string, nodeName string) ([]annotations.Event, error) {
	var events []annotations.Event
	for _, sn := range serviceNames {
		service, err := models.FindServiceByName(s.db.Querier, sn)
		if err != nil {
			return nil, err
		}

		node, err := models.FindNodeByID(s.db.Querier, service.NodeID)
		if err != nil {
			return nil, err
		}

		labels, err := annotations.ObjectLabels(node, service)
		if err != nil {
			return nil, err
		}
		events = append(events, annotations.Event{Labels: labels})
	}

	if nodeName != "" {
		node, err := models.FindNodeByName(s.db.Querier, nodeName)
		if err != nil {
			return nil, err
		}

		labels, err := annotations.ObjectLabels(node, nil)
		if err != nil {
			return nil, err
		}
		events = append(events, annotations.Event{Labels: labels})
	}

	return events, nil
}

// eventObject describes the node or service the event is scoped to.
func eventObject(e annotations.Event) string {
	if name := e.Labels["service_name"]; name != "" {
		return "service '" + name + "'"
	}

	return "node '" + e.Labels["node_name"] + "'"
}
