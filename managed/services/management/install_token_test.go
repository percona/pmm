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
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	managementv1 "github.com/percona/pmm/api/management/v1"
)

func TestManagementService_CreateNodeInstallToken(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	exp := time.Now().Add(24 * time.Hour)

	t.Run("ok", func(t *testing.T) {
		t.Parallel()
		gc := &mockGrafanaClient{}
		gc.On("CreateNodeInstallToken", mock.Anything, mock.MatchedBy(func(s string) bool { return s != "" }), defaultInstallTokenTTLSeconds).
			Return(int64(42), "tok", exp, nil).Once()

		s := &ManagementService{grafanaClient: gc}
		res, err := s.CreateNodeInstallToken(ctx, &managementv1.CreateNodeInstallTokenRequest{
			TtlSeconds: 0,
			Technology: "mysql",
		})
		require.NoError(t, err)
		require.Equal(t, "tok", res.Token)
		require.Equal(t, int64(42), res.ServiceAccountId)
		require.NotNil(t, res.ExpiresAt)
	})

	t.Run("invalid technology", func(t *testing.T) {
		t.Parallel()
		s := &ManagementService{grafanaClient: &mockGrafanaClient{}}
		_, err := s.CreateNodeInstallToken(ctx, &managementv1.CreateNodeInstallTokenRequest{Technology: "oracle"})
		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
	})

	t.Run("ttl clamp max", func(t *testing.T) {
		t.Parallel()
		gc := &mockGrafanaClient{}
		gc.On("CreateNodeInstallToken", mock.Anything, mock.Anything, int64(maxInstallTokenTTLSeconds)).
			Return(int64(1), "t", exp, nil).Once()
		s := &ManagementService{grafanaClient: gc}
		_, err := s.CreateNodeInstallToken(ctx, &managementv1.CreateNodeInstallTokenRequest{
			TtlSeconds: 999999999,
			Technology: "postgresql",
		})
		require.NoError(t, err)
	})

	t.Run("ttl clamp min", func(t *testing.T) {
		t.Parallel()
		gc := &mockGrafanaClient{}
		gc.On("CreateNodeInstallToken", mock.Anything, mock.Anything, int64(minInstallTokenTTLSeconds)).
			Return(int64(1), "t", exp, nil).Once()
		s := &ManagementService{grafanaClient: gc}
		_, err := s.CreateNodeInstallToken(ctx, &managementv1.CreateNodeInstallTokenRequest{
			TtlSeconds: 30,
			Technology: "mongodb",
		})
		require.NoError(t, err)
	})
}
