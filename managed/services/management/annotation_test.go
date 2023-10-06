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
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/api/managementpb"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
	"github.com/percona/pmm/managed/utils/tests"
	"github.com/percona/pmm/utils/logger"
)

func TestAnnotations(t *testing.T) {
	setup := func(t *testing.T) (ctx context.Context, db *reform.DB, teardown func(t *testing.T)) {
		t.Helper()

		ctx = logger.Set(context.Background(), t.Name())
		uuid.SetRand(&tests.IDReader{})

		sqlDB := testdb.Open(t, models.SetupFixtures, nil)
		db = reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

		teardown = func(t *testing.T) {
			t.Helper()
			uuid.SetRand(nil)

			require.NoError(t, sqlDB.Close())
		}

		return
	}
	authorization := "admin:admin"
	authorizationHeaders := []string{authorization}

	t.Run("Non-existing service", func(t *testing.T) {
		ctx, db, teardown := setup(t)
		defer teardown(t)
		grafanaClient := &mockGrafanaClient{}
		s := NewAnnotationService(db, grafanaClient)

		_, err := s.AddAnnotation(ctx, authorizationHeaders, &managementpb.AddAnnotationRequest{
			Text:         "Some text",
			ServiceNames: []string{"no-service"},
		})
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Service with name "no-service" not found.`), err)

		grafanaClient.AssertExpectations(t)
	})

	t.Run("Non-existing node", func(t *testing.T) {
		ctx, db, teardown := setup(t)
		defer teardown(t)
		grafanaClient := &mockGrafanaClient{}
		s := NewAnnotationService(db, grafanaClient)

		_, err := s.AddAnnotation(ctx, authorizationHeaders, &managementpb.AddAnnotationRequest{
			Text:     "Some text",
			NodeName: "no-node",
		})
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Node with name "no-node" not found.`), err)

		grafanaClient.AssertExpectations(t)
	})

	t.Run("Existing service", func(t *testing.T) {
		ctx, db, teardown := setup(t)
		defer teardown(t)
		grafanaClient := &mockGrafanaClient{}
		s := NewAnnotationService(db, grafanaClient)

		_, err := models.AddNewService(db.Querier, models.MySQLServiceType, &models.AddDBMSServiceParams{
			ServiceName: "service-test",
			NodeID:      models.PMMServerNodeID,
			Address:     pointer.ToString("127.0.0.1"),
			Port:        pointer.ToUint16(3306),
		})
		require.NoError(t, err)

		expectedTags := []string{"service-test"}
		expectedText := "Some text (Service Name: service-test)"
		grafanaClient.On("CreateAnnotation", ctx, expectedTags, mock.Anything, expectedText, authorization).Return("", nil)
		_, err = s.AddAnnotation(ctx, authorizationHeaders, &managementpb.AddAnnotationRequest{
			Text:         "Some text",
			ServiceNames: []string{"service-test"},
		})
		require.NoError(t, err)

		grafanaClient.AssertExpectations(t)
	})

	t.Run("Existing node", func(t *testing.T) {
		ctx, db, teardown := setup(t)
		defer teardown(t)
		grafanaClient := &mockGrafanaClient{}
		s := NewAnnotationService(db, grafanaClient)

		_, err := models.CreateNode(db.Querier, models.GenericNodeType, &models.CreateNodeParams{
			NodeName: "node-test",
		})
		require.NoError(t, err)

		expectedTags := []string{"node-test"}
		expectedText := "Some text (Node Name: node-test)"
		grafanaClient.On("CreateAnnotation", ctx, expectedTags, mock.Anything, expectedText, authorization).Return("", nil)
		_, err = s.AddAnnotation(ctx, authorizationHeaders, &managementpb.AddAnnotationRequest{
			Text:     "Some text",
			NodeName: "node-test",
		})
		require.NoError(t, err)

		grafanaClient.AssertExpectations(t)
	})

	t.Run("Non-existing service and non-existing node", func(t *testing.T) {
		ctx, db, teardown := setup(t)
		defer teardown(t)
		grafanaClient := &mockGrafanaClient{}
		s := NewAnnotationService(db, grafanaClient)

		_, err := s.AddAnnotation(ctx, authorizationHeaders, &managementpb.AddAnnotationRequest{
			Text:         "Some text",
			NodeName:     "no-node",
			ServiceNames: []string{"no-service"},
		})
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Service with name "no-service" not found.`), err)

		grafanaClient.AssertExpectations(t)
	})

	t.Run("Empty service and empty node", func(t *testing.T) {
		ctx, db, teardown := setup(t)
		defer teardown(t)
		grafanaClient := &mockGrafanaClient{}
		s := NewAnnotationService(db, grafanaClient)

		expectedTags := []string{"pmm_annotation"}
		expectedText := "Some text"
		grafanaClient.On("CreateAnnotation", ctx, expectedTags, mock.Anything, expectedText, authorization).Return("", nil)
		_, err := s.AddAnnotation(ctx, authorizationHeaders, &managementpb.AddAnnotationRequest{
			Text: "Some text",
		})
		require.NoError(t, err)

		grafanaClient.AssertExpectations(t)
	})

	t.Run("Existing service and non-existing node", func(t *testing.T) {
		ctx, db, teardown := setup(t)
		defer teardown(t)
		grafanaClient := &mockGrafanaClient{}
		s := NewAnnotationService(db, grafanaClient)

		_, err := models.AddNewService(db.Querier, models.MySQLServiceType, &models.AddDBMSServiceParams{
			ServiceName: "service-test",
			NodeID:      models.PMMServerNodeID,
			Address:     pointer.ToString("127.0.0.1"),
			Port:        pointer.ToUint16(3306),
		})
		require.NoError(t, err)

		_, err = s.AddAnnotation(ctx, authorizationHeaders, &managementpb.AddAnnotationRequest{
			Text:         "Some text",
			ServiceNames: []string{"service-test"},
			NodeName:     "node-test",
		})
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Node with name "node-test" not found.`), err)

		grafanaClient.AssertExpectations(t)
	})

	t.Run("Existing service and existing node", func(t *testing.T) {
		ctx, db, teardown := setup(t)
		defer teardown(t)
		grafanaClient := &mockGrafanaClient{}
		s := NewAnnotationService(db, grafanaClient)

		_, err := models.CreateNode(db.Querier, models.GenericNodeType, &models.CreateNodeParams{
			NodeName: "node-test",
		})
		require.NoError(t, err)

		_, err = models.AddNewService(db.Querier, models.MySQLServiceType, &models.AddDBMSServiceParams{
			ServiceName: "service-test",
			NodeID:      models.PMMServerNodeID,
			Address:     pointer.ToString("127.0.0.1"),
			Port:        pointer.ToUint16(3306),
		})
		require.NoError(t, err)

		expectedTags := []string{"service-test", "node-test"}
		expectedText := "Some text (Service Name: service-test. Node Name: node-test)"
		grafanaClient.On("CreateAnnotation", ctx, expectedTags, mock.Anything, expectedText, authorization).Return("", nil)
		_, err = s.AddAnnotation(ctx, authorizationHeaders, &managementpb.AddAnnotationRequest{
			Text:         "Some text",
			ServiceNames: []string{"service-test"},
			NodeName:     "node-test",
		})
		require.NoError(t, err)

		grafanaClient.AssertExpectations(t)
	})

	t.Run("More services", func(t *testing.T) {
		ctx, db, teardown := setup(t)
		defer teardown(t)
		grafanaClient := &mockGrafanaClient{}
		s := NewAnnotationService(db, grafanaClient)

		_, err := models.AddNewService(db.Querier, models.MySQLServiceType, &models.AddDBMSServiceParams{
			ServiceName: "service-test",
			NodeID:      models.PMMServerNodeID,
			Address:     pointer.ToString("127.0.0.1"),
			Port:        pointer.ToUint16(3306),
		})
		require.NoError(t, err)

		_, err = models.AddNewService(db.Querier, models.MySQLServiceType, &models.AddDBMSServiceParams{
			ServiceName: "service-test2",
			NodeID:      models.PMMServerNodeID,
			Address:     pointer.ToString("127.0.0.1"),
			Port:        pointer.ToUint16(3307),
		})
		require.NoError(t, err)

		expectedTags := []string{"service-test", "service-test2"}
		expectedText := "Some text (Service Name: service-test, service-test2)"
		grafanaClient.On("CreateAnnotation", ctx, expectedTags, mock.Anything, expectedText, authorization).Return("", nil)
		_, err = s.AddAnnotation(ctx, authorizationHeaders, &managementpb.AddAnnotationRequest{
			Text:         "Some text",
			ServiceNames: []string{"service-test", "service-test2"},
		})
		require.NoError(t, err)

		grafanaClient.AssertExpectations(t)
	})

	t.Run("More services, but one non-existing", func(t *testing.T) {
		ctx, db, teardown := setup(t)
		defer teardown(t)
		grafanaClient := &mockGrafanaClient{}
		s := NewAnnotationService(db, grafanaClient)

		_, err := models.AddNewService(db.Querier, models.MySQLServiceType, &models.AddDBMSServiceParams{
			ServiceName: "service-test",
			NodeID:      models.PMMServerNodeID,
			Address:     pointer.ToString("127.0.0.1"),
			Port:        pointer.ToUint16(3306),
		})
		require.NoError(t, err)

		_, err = s.AddAnnotation(ctx, authorizationHeaders, &managementpb.AddAnnotationRequest{
			Text:         "Some text",
			ServiceNames: []string{"service-test", "no-service"},
		})
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Service with name "no-service" not found.`), err)

		grafanaClient.AssertExpectations(t)
	})
}
