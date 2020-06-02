// pmm-managed
// Copyright (C) 2017 Percona LLC
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

	"github.com/percona/pmm/api/managementpb"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/percona/pmm-managed/services"
	"github.com/percona/pmm-managed/utils/tests"
)

func TestStartSecurityChecks(t *testing.T) {
	t.Run("internal error", func(t *testing.T) {
		var checksService mockChecksService
		checksService.On("StartChecks", mock.Anything).Return(errors.New("random error"))

		s := NewChecksAPIService(&checksService)

		resp, err := s.StartSecurityChecks(context.Background(), &managementpb.StartSecurityChecksRequest{})
		tests.AssertGRPCError(t, status.New(codes.Internal, "Failed to start security checks."), err)
		assert.Nil(t, resp)
	})

	t.Run("STT disabled error", func(t *testing.T) {
		var checksService mockChecksService
		checksService.On("StartChecks", mock.Anything).Return(services.ErrSTTDisabled)

		s := NewChecksAPIService(&checksService)

		resp, err := s.StartSecurityChecks(context.Background(), &managementpb.StartSecurityChecksRequest{})
		tests.AssertGRPCError(t, status.New(codes.FailedPrecondition, "STT is disabled."), err)
		assert.Nil(t, resp)
	})
}
