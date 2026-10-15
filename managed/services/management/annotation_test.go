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
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	managementv1 "github.com/percona/pmm/api/management/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services"
	"github.com/percona/pmm/managed/services/annotations"
	"github.com/percona/pmm/managed/utils/testdb"
	"github.com/percona/pmm/managed/utils/tests"
	"github.com/percona/pmm/utils/logger"
)

func TestAnnotations(t *testing.T) {
	authorization := "admin:admin"

	setup := func(t *testing.T) (context.Context, *ManagementService, *reform.DB, *mockGrafanaClient, *mockAnnotationStore, func(t *testing.T)) {
		t.Helper()

		ctx := metadata.NewIncomingContext(t.Context(), metadata.New(map[string]string{"authorization": authorization}))
		ctx = logger.Set(ctx, t.Name())
		uuid.SetRand(&tests.IDReader{})

		sqlDB := testdb.Open(t, models.SetupFixtures, nil)
		db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

		ar := &mockAgentsRegistry{}
		ar.Test(t)
		state := &mockAgentsStateUpdater{}
		state.Test(t)
		cc := &mockConnectionChecker{}
		cc.Test(t)
		sib := &mockServiceInfoBroker{}
		sib.Test(t)
		vmdb := &mockPrometheusService{}
		vmdb.Test(t)
		vc := &mockVersionCache{}
		vc.Test(t)
		grafanaClient := &mockGrafanaClient{}
		grafanaClient.Test(t)

		vmClient := &mockVictoriaMetricsClient{}
		vmClient.Test(t)
		store := &mockAnnotationStore{}
		store.Test(t)

		s := NewManagementService(db, ar, state, cc, sib, vmdb, vc, grafanaClient, vmClient, store, nil, false)

		teardown := func(t *testing.T) {
			t.Helper()
			uuid.SetRand(nil)

			require.NoError(t, sqlDB.Close())

			ar.AssertExpectations(t)
			state.AssertExpectations(t)
			cc.AssertExpectations(t)
			sib.AssertExpectations(t)
			vmdb.AssertExpectations(t)
			vc.AssertExpectations(t)
			grafanaClient.AssertExpectations(t)
			vmClient.AssertExpectations(t)
			store.AssertExpectations(t)
		}

		return ctx, s, db, grafanaClient, store, teardown
	}

	t.Run("Non-existing service", func(t *testing.T) {
		ctx, ss, _, _, _, teardown := setup(t)
		t.Cleanup(func() { teardown(t) })

		_, err := ss.AddAnnotation(ctx, &managementv1.AddAnnotationRequest{
			Text:         "Some text",
			ServiceNames: []string{"no-service"},
		})
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Service with name "no-service" not found.`), err)
	})

	t.Run("Non-existing node", func(t *testing.T) {
		ctx, s, _, _, _, teardown := setup(t)
		t.Cleanup(func() { teardown(t) })

		_, err := s.AddAnnotation(ctx, &managementv1.AddAnnotationRequest{
			Text:     "Some text",
			NodeName: "no-node",
		})
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Node with name "no-node" not found.`), err)
	})

	t.Run("Existing service", func(t *testing.T) {
		ctx, s, db, grafanaClient, store, teardown := setup(t)
		t.Cleanup(func() { teardown(t) })

		_, err := models.AddNewService(db.Querier, models.MySQLServiceType, &models.AddDBMSServiceParams{
			ServiceName: "service-test",
			NodeID:      models.PMMServerNodeID,
			Address:     new("127.0.0.1"),
			Port:        new(uint16(3306)),
		})
		require.NoError(t, err)

		expectedTags := []string{"service-test"}
		expectedText := "Some text (Service Name: service-test)"
		grafanaClient.On("CreateAnnotation", ctx, expectedTags, mock.Anything, expectedText, authorization).Return(int64(17), nil)
		store.On("Write", ctx, mock.Anything).Return(nil)
		_, err = s.AddAnnotation(ctx, &managementv1.AddAnnotationRequest{
			Text:         "Some text",
			ServiceNames: []string{"service-test"},
		})
		require.NoError(t, err)
	})

	t.Run("Existing node", func(t *testing.T) {
		ctx, s, db, grafanaClient, store, teardown := setup(t)
		t.Cleanup(func() { teardown(t) })

		_, err := models.CreateNode(db.Querier, models.GenericNodeType, &models.CreateNodeParams{
			NodeName: "node-test",
		})
		require.NoError(t, err)

		expectedTags := []string{"node-test"}
		expectedText := "Some text (Node Name: node-test)"
		grafanaClient.On("CreateAnnotation", ctx, expectedTags, mock.Anything, expectedText, authorization).Return(int64(17), nil)
		store.On("Write", ctx, mock.Anything).Return(nil)
		_, err = s.AddAnnotation(ctx, &managementv1.AddAnnotationRequest{
			Text:     "Some text",
			NodeName: "node-test",
		})
		require.NoError(t, err)
	})

	t.Run("Non-existing service and non-existing node", func(t *testing.T) {
		ctx, s, _, _, _, teardown := setup(t)
		t.Cleanup(func() { teardown(t) })

		_, err := s.AddAnnotation(ctx, &managementv1.AddAnnotationRequest{
			Text:         "Some text",
			NodeName:     "no-node",
			ServiceNames: []string{"no-service"},
		})
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Service with name "no-service" not found.`), err)
	})

	t.Run("Empty service and empty node", func(t *testing.T) {
		ctx, s, _, grafanaClient, _, teardown := setup(t)
		t.Cleanup(func() { teardown(t) })

		expectedTags := []string{"pmm_annotation"}
		expectedText := "Some text"
		grafanaClient.On("CreateAnnotation", ctx, expectedTags, mock.Anything, expectedText, authorization).Return(int64(17), nil)
		resp, err := s.AddAnnotation(ctx, &managementv1.AddAnnotationRequest{
			Text: "Some text",
		})
		require.NoError(t, err)
		require.NotNil(t, resp)

		grafanaClient.AssertExpectations(t)
	})

	t.Run("Existing service and non-existing node", func(t *testing.T) {
		ctx, s, db, _, _, teardown := setup(t)
		t.Cleanup(func() { teardown(t) })

		_, err := models.AddNewService(db.Querier, models.MySQLServiceType, &models.AddDBMSServiceParams{
			ServiceName: "service-test",
			NodeID:      models.PMMServerNodeID,
			Address:     new("127.0.0.1"),
			Port:        new(uint16(3306)),
		})
		require.NoError(t, err)

		_, err = s.AddAnnotation(ctx, &managementv1.AddAnnotationRequest{
			Text:         "Some text",
			ServiceNames: []string{"service-test"},
			NodeName:     "node-test",
		})
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Node with name "node-test" not found.`), err)
	})

	t.Run("Existing service and existing node", func(t *testing.T) {
		ctx, s, db, grafanaClient, store, teardown := setup(t)
		t.Cleanup(func() { teardown(t) })

		_, err := models.CreateNode(db.Querier, models.GenericNodeType, &models.CreateNodeParams{
			NodeName: "node-test",
		})
		require.NoError(t, err)

		_, err = models.AddNewService(db.Querier, models.MySQLServiceType, &models.AddDBMSServiceParams{
			ServiceName: "service-test",
			NodeID:      models.PMMServerNodeID,
			Address:     new("127.0.0.1"),
			Port:        new(uint16(3306)),
		})
		require.NoError(t, err)

		expectedTags := []string{"service-test", "node-test"}
		expectedText := "Some text (Service Name: service-test. Node Name: node-test)"
		grafanaClient.On("CreateAnnotation", ctx, expectedTags, mock.Anything, expectedText, authorization).Return(int64(17), nil)
		store.On("Write", ctx, mock.Anything).Return(nil)
		_, err = s.AddAnnotation(ctx, &managementv1.AddAnnotationRequest{
			Text:         "Some text",
			ServiceNames: []string{"service-test"},
			NodeName:     "node-test",
		})
		require.NoError(t, err)
	})

	t.Run("More services", func(t *testing.T) {
		ctx, s, db, grafanaClient, store, teardown := setup(t)
		t.Cleanup(func() { teardown(t) })

		_, err := models.AddNewService(db.Querier, models.MySQLServiceType, &models.AddDBMSServiceParams{
			ServiceName: "service-test",
			NodeID:      models.PMMServerNodeID,
			Address:     new("127.0.0.1"),
			Port:        new(uint16(3306)),
		})
		require.NoError(t, err)

		_, err = models.AddNewService(db.Querier, models.MySQLServiceType, &models.AddDBMSServiceParams{
			ServiceName: "service-test2",
			NodeID:      models.PMMServerNodeID,
			Address:     new("127.0.0.1"),
			Port:        new(uint16(3307)),
		})
		require.NoError(t, err)

		expectedTags := []string{"service-test", "service-test2"}
		expectedText := "Some text (Service Name: service-test, service-test2)"
		grafanaClient.On("CreateAnnotation", ctx, expectedTags, mock.Anything, expectedText, authorization).Return(int64(17), nil)
		store.On("Write", ctx, mock.Anything).Return(nil)
		_, err = s.AddAnnotation(ctx, &managementv1.AddAnnotationRequest{
			Text:         "Some text",
			ServiceNames: []string{"service-test", "service-test2"},
		})
		require.NoError(t, err)
	})

	t.Run("More services, but one non-existing", func(t *testing.T) {
		ctx, s, db, _, _, teardown := setup(t)
		t.Cleanup(func() { teardown(t) })

		_, err := models.AddNewService(db.Querier, models.MySQLServiceType, &models.AddDBMSServiceParams{
			ServiceName: "service-test",
			NodeID:      models.PMMServerNodeID,
			Address:     new("127.0.0.1"),
			Port:        new(uint16(3306)),
		})
		require.NoError(t, err)

		_, err = s.AddAnnotation(ctx, &managementv1.AddAnnotationRequest{
			Text:         "Some text",
			ServiceNames: []string{"service-test", "no-service"},
		})
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Service with name "no-service" not found.`), err)
	})

	// addServiceOnNode adds a node and a MySQL service on it.
	addServiceOnNode := func(t *testing.T, db *reform.DB, nodeName, serviceName string, labels map[string]string) {
		t.Helper()

		node, err := models.CreateNode(db.Querier, models.GenericNodeType, &models.CreateNodeParams{
			NodeName: nodeName,
		})
		require.NoError(t, err)

		_, err = models.AddNewService(db.Querier, models.MySQLServiceType, &models.AddDBMSServiceParams{
			ServiceName:  serviceName,
			NodeID:       node.NodeID,
			Address:      new("127.0.0.1"),
			Port:         new(uint16(3306)),
			CustomLabels: labels,
		})
		require.NoError(t, err)
	}

	t.Run("Service annotation writes series", func(t *testing.T) {
		ctx, s, db, grafanaClient, store, teardown := setup(t)
		t.Cleanup(func() { teardown(t) })

		addServiceOnNode(t, db, "n1", "s1", nil)

		var from time.Time
		expectedText := "Some text (Service Name: s1)"
		grafanaClient.On("CreateAnnotation", ctx, []string{"deploy", "s1"}, mock.Anything, expectedText, authorization).
			Run(func(args mock.Arguments) { from = args.Get(2).(time.Time) }).
			Return(int64(17), nil)
		var events []annotations.Event
		store.On("Write", ctx, mock.Anything).
			Run(func(args mock.Arguments) { events = args.Get(1).([]annotations.Event) }).
			Return(nil)

		resp, err := s.AddAnnotation(ctx, &managementv1.AddAnnotationRequest{
			Text:         "Some text",
			Tags:         []string{"deploy"},
			ServiceNames: []string{"s1"},
		})
		require.NoError(t, err)
		assert.Equal(t, int64(17), resp.AnnotationId)

		require.Len(t, events, 1)
		assert.Equal(t, int64(17), events[0].AnnotationID)
		assert.Equal(t, expectedText, events[0].Text)
		assert.Equal(t, []string{"deploy"}, events[0].Tags)
		assert.Equal(t, "s1", events[0].Labels["service_name"])
		assert.Equal(t, "n1", events[0].Labels["node_name"])
		assert.True(t, from.Equal(events[0].Time), "event time %s differs from Grafana annotation time %s", events[0].Time, from)
	})

	t.Run("Node and services on other nodes", func(t *testing.T) {
		ctx, s, db, grafanaClient, store, teardown := setup(t)
		t.Cleanup(func() { teardown(t) })

		_, err := models.CreateNode(db.Querier, models.GenericNodeType, &models.CreateNodeParams{
			NodeName: "n1",
		})
		require.NoError(t, err)
		addServiceOnNode(t, db, "n2", "s2", nil)

		expectedText := "Some text (Service Name: s2. Node Name: n1)"
		grafanaClient.On("CreateAnnotation", ctx, []string{"s2", "n1"}, mock.Anything, expectedText, authorization).Return(int64(17), nil)
		var events []annotations.Event
		store.On("Write", ctx, mock.Anything).
			Run(func(args mock.Arguments) { events = args.Get(1).([]annotations.Event) }).
			Return(nil)

		resp, err := s.AddAnnotation(ctx, &managementv1.AddAnnotationRequest{
			Text:         "Some text",
			NodeName:     "n1",
			ServiceNames: []string{"s2"},
		})
		require.NoError(t, err)
		assert.Equal(t, int64(17), resp.AnnotationId)

		require.Len(t, events, 2)
		byService := make(map[string]annotations.Event, len(events))
		for _, e := range events {
			assert.Equal(t, int64(17), e.AnnotationID)
			assert.Equal(t, expectedText, e.Text)
			assert.Empty(t, e.Tags)
			byService[e.Labels["service_name"]] = e
		}

		nodeEvent, ok := byService[""]
		require.True(t, ok, "no event for the node")
		assert.Equal(t, "n1", nodeEvent.Labels["node_name"])
		assert.NotContains(t, nodeEvent.Labels, "service_name")

		serviceEvent, ok := byService["s2"]
		require.True(t, ok, "no event for the service")
		assert.Equal(t, "n2", serviceEvent.Labels["node_name"])
	})

	t.Run("Global annotation writes nothing", func(t *testing.T) {
		ctx, s, _, grafanaClient, store, teardown := setup(t)
		t.Cleanup(func() { teardown(t) })

		grafanaClient.On("CreateAnnotation", ctx, []string{"pmm_annotation", "deploy"}, mock.Anything, "Some text", authorization).Return(int64(17), nil)

		resp, err := s.AddAnnotation(ctx, &managementv1.AddAnnotationRequest{
			Text: "Some text",
			Tags: []string{"deploy"},
		})
		require.NoError(t, err)
		assert.Equal(t, int64(17), resp.AnnotationId)

		store.AssertNotCalled(t, "Write", mock.Anything, mock.Anything)
	})

	t.Run("Store failure", func(t *testing.T) {
		ctx, s, db, grafanaClient, store, teardown := setup(t)
		t.Cleanup(func() { teardown(t) })

		addServiceOnNode(t, db, "n1", "s1", nil)

		grafanaClient.On("CreateAnnotation", ctx, []string{"s1"}, mock.Anything, "Some text (Service Name: s1)", authorization).Return(int64(17), nil)
		store.On("Write", ctx, mock.Anything).Return(errors.New("victoriametrics is down"))

		_, err := s.AddAnnotation(ctx, &managementv1.AddAnnotationRequest{
			Text:         "Some text",
			ServiceNames: []string{"s1"},
		})
		tests.AssertGRPCErrorRE(t, codes.Unavailable, `created.*not shown on dashboards yet.*sync job.*victoriametrics is down`, err)

		grafanaClient.AssertNumberOfCalls(t, "CreateAnnotation", 1)
	})

	t.Run("Too many labels", func(t *testing.T) {
		ctx, s, db, grafanaClient, store, teardown := setup(t)
		t.Cleanup(func() { teardown(t) })

		labels := make(map[string]string)
		for i := range 35 {
			labels[fmt.Sprintf("label_%02d", i)] = "value"
		}
		addServiceOnNode(t, db, "n1", "s1", labels)

		_, err := s.AddAnnotation(ctx, &managementv1.AddAnnotationRequest{
			Text:         "Some text",
			ServiceNames: []string{"s1"},
		})
		tests.AssertGRPCErrorRE(t, codes.InvalidArgument, `s1.*too many labels`, err)

		grafanaClient.AssertNotCalled(t, "CreateAnnotation", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		store.AssertNotCalled(t, "Write", mock.Anything, mock.Anything)
	})

	t.Run("Label value too long", func(t *testing.T) {
		ctx, s, db, grafanaClient, store, teardown := setup(t)
		t.Cleanup(func() { teardown(t) })

		addServiceOnNode(t, db, "n1", "s1", nil)

		// 41 tags of 100 bytes join to 4,140 bytes.
		tags := make([]string, 41)
		for i := range tags {
			tags[i] = fmt.Sprintf("%0100d", i)
		}

		_, err := s.AddAnnotation(ctx, &managementv1.AddAnnotationRequest{
			Text:         "Some text",
			Tags:         tags,
			ServiceNames: []string{"s1"},
		})
		tests.AssertGRPCErrorRE(t, codes.InvalidArgument, `^Cannot annotate service 's1': label value too long: pmm_annotation_tags is 4140 bytes`, err)

		grafanaClient.AssertNotCalled(t, "CreateAnnotation", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		store.AssertNotCalled(t, "Write", mock.Anything, mock.Anything)
	})

	t.Run("Remove", func(t *testing.T) {
		ctx, s, _, grafanaClient, store, teardown := setup(t)
		t.Cleanup(func() { teardown(t) })

		deleted := grafanaClient.On("DeleteAnnotation", ctx, int64(17), authorization).Return(nil).Once()
		store.On("Delete", ctx, []int64{17}).Return(nil).Once().NotBefore(deleted)

		resp, err := s.RemoveAnnotation(ctx, &managementv1.RemoveAnnotationRequest{AnnotationId: 17})
		require.NoError(t, err)
		assert.NotNil(t, resp)
	})

	t.Run("Remove unknown annotation", func(t *testing.T) {
		ctx, s, _, grafanaClient, store, teardown := setup(t)
		t.Cleanup(func() { teardown(t) })

		notFound := grafanaClient.On("DeleteAnnotation", ctx, int64(17), authorization).Return(services.ErrAnnotationNotFound).Once()
		store.On("Delete", ctx, []int64{17}).Return(nil).Once().NotBefore(notFound)

		_, err := s.RemoveAnnotation(ctx, &managementv1.RemoveAnnotationRequest{AnnotationId: 17})
		tests.AssertGRPCError(t, status.New(codes.NotFound, "Annotation with ID 17 not found."), err)
	})

	t.Run("Remove unknown annotation store failure", func(t *testing.T) {
		ctx, s, _, grafanaClient, store, teardown := setup(t)
		t.Cleanup(func() { teardown(t) })

		l, hook := logrustest.NewNullLogger()
		s.l = logrus.NewEntry(l)

		notFound := grafanaClient.On("DeleteAnnotation", ctx, int64(17), authorization).Return(services.ErrAnnotationNotFound).Once()
		store.On("Delete", ctx, []int64{17}).Return(errors.New("victoriametrics is down")).Once().NotBefore(notFound)

		_, err := s.RemoveAnnotation(ctx, &managementv1.RemoveAnnotationRequest{AnnotationId: 17})
		tests.AssertGRPCError(t, status.New(codes.NotFound, "Annotation with ID 17 not found."), err)

		require.Len(t, hook.AllEntries(), 1)
		entry := hook.LastEntry()
		assert.Equal(t, logrus.WarnLevel, entry.Level)
		assert.Equal(t, int64(17), entry.Data["annotation_id"])
		assert.ErrorContains(t, entry.Data[logrus.ErrorKey].(error), "victoriametrics is down")
	})

	t.Run("Remove Grafana failure", func(t *testing.T) {
		ctx, s, _, grafanaClient, store, teardown := setup(t)
		t.Cleanup(func() { teardown(t) })

		grafanaClient.On("DeleteAnnotation", ctx, int64(17), authorization).Return(errors.New("grafana is down")).Once()

		_, err := s.RemoveAnnotation(ctx, &managementv1.RemoveAnnotationRequest{AnnotationId: 17})
		require.ErrorContains(t, err, "grafana is down")

		store.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
	})

	t.Run("Remove store failure", func(t *testing.T) {
		ctx, s, _, grafanaClient, store, teardown := setup(t)
		t.Cleanup(func() { teardown(t) })

		deleted := grafanaClient.On("DeleteAnnotation", ctx, int64(17), authorization).Return(nil).Once()
		store.On("Delete", ctx, []int64{17}).Return(errors.New("victoriametrics is down")).Once().NotBefore(deleted)

		_, err := s.RemoveAnnotation(ctx, &managementv1.RemoveAnnotationRequest{AnnotationId: 17})
		tests.AssertGRPCErrorRE(t, codes.Unavailable, `deleted.*may still show on dashboards.*sync job.*victoriametrics is down`, err)
	})

	t.Run("Remove without authorization", func(t *testing.T) {
		_, s, _, grafanaClient, store, teardown := setup(t)
		t.Cleanup(func() { teardown(t) })

		ctx := metadata.NewIncomingContext(t.Context(), metadata.New(nil))

		_, err := s.RemoveAnnotation(ctx, &managementv1.RemoveAnnotationRequest{AnnotationId: 17})
		tests.AssertGRPCError(t, status.New(codes.Unauthenticated, "Authorization error."), err)

		grafanaClient.AssertNotCalled(t, "DeleteAnnotation", mock.Anything, mock.Anything, mock.Anything)
		store.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
	})
}
