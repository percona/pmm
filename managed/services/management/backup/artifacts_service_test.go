// Copyright (C) 2022 Percona LLC
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

package backup

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/brianvoe/gofakeit/v6"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/api/agentpb"
	backuppb "github.com/percona/pmm/api/managementpb/backup"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
	"github.com/percona/pmm/managed/utils/tests"
)

func TestListPitrTimelines(t *testing.T) {
	ctx := context.Background()
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
	agent := setup(t, db.Querier, models.MongoDBServiceType, "test-mongo-service-to-list-pitr-timeranges")

	mockedAgentService := &mockAgentService{}

	timeranges := []*agentpb.PBMPitrTimerange{
		{
			StartTimestamp: timestamppb.New(time.Now()),
			EndTimestamp:   timestamppb.New(time.Now()),
		},
	}

	mockedAgentService.On("ListPITRTimeranges", ctx, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(timeranges, nil).Once()
	artifactsService := NewArtifactsService(db, nil, mockedAgentService)
	var locationID string

	params := models.CreateBackupLocationParams{
		Name:        gofakeit.Name(),
		Description: "",
	}
	params.S3Config = &models.S3LocationConfig{
		Endpoint:     "https://awsS3.us-west-2.amazonaws.com/",
		AccessKey:    "access_key",
		SecretKey:    "secret_key",
		BucketName:   "example_bucket",
		BucketRegion: "us-east-1",
	}
	loc, err := models.CreateBackupLocation(db.Querier, params)
	require.NoError(t, err)
	require.NotEmpty(t, loc.ID)

	locationID = loc.ID

	t.Run("successfully lists PITR time ranges", func(t *testing.T) {
		artifact, err := models.CreateArtifact(db.Querier, models.CreateArtifactParams{
			Name:       "test_artifact",
			Vendor:     "test_vendor",
			LocationID: locationID,
			ServiceID:  pointer.GetString(agent.ServiceID),
			Mode:       models.PITR,
			DataModel:  models.LogicalDataModel,
			Status:     models.PendingBackupStatus,
		})
		assert.NoError(t, err)
		assert.NotEmpty(t, artifact.ID)

		response, err := artifactsService.ListPitrTimeranges(ctx, &backuppb.ListPitrTimerangesRequest{
			ArtifactId: artifact.ID,
		})
		require.NoError(t, err)
		require.NotNil(t, response)
		assert.Len(t, response.Timeranges, 1)
	})

	t.Run("fails for invalid artifact ID", func(t *testing.T) {
		unknownID := "artifact_id/" + uuid.New().String()
		response, err := artifactsService.ListPitrTimeranges(ctx, &backuppb.ListPitrTimerangesRequest{
			ArtifactId: unknownID,
		})
		tests.AssertGRPCError(t, status.New(codes.NotFound, fmt.Sprintf("Artifact with ID %q not found.", unknownID)), err)
		assert.Nil(t, response)
	})

	t.Run("fails for non-PITR artifact", func(t *testing.T) {
		artifact, err := models.CreateArtifact(db.Querier, models.CreateArtifactParams{
			Name:       "test_non_pitr_artifact",
			Vendor:     "test_vendor",
			LocationID: locationID,
			ServiceID:  "test_service",
			Mode:       models.Snapshot,
			DataModel:  models.LogicalDataModel,
			Status:     models.PendingBackupStatus,
		})
		assert.NoError(t, err)
		assert.NotEmpty(t, artifact.ID)

		response, err := artifactsService.ListPitrTimeranges(ctx, &backuppb.ListPitrTimerangesRequest{
			ArtifactId: artifact.ID,
		})
		tests.AssertGRPCError(t, status.New(codes.FailedPrecondition, "Artifact is not a PITR artifact"), err)
		assert.Nil(t, response)
	})
	mock.AssertExpectationsForObjects(t, mockedAgentService)
}
