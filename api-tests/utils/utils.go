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

//nolint:revive
package utils

import (
	"context"
	"fmt"
	"time"

	serverClient "github.com/percona/pmm/api/server/v1/json/client"
	server "github.com/percona/pmm/api/server/v1/json/client/server_service"
	userClient "github.com/percona/pmm/api/user/v1/json/client"
	"github.com/percona/pmm/api/user/v1/json/client/user_service"
)

// WaitServerReady checks if the server is ready by calling the readiness endpoint and fetching user details.
func WaitServerReady(ctx context.Context) error {
	return retryWithBackoff(ctx, 10, func() error {
		_, err := serverClient.Default.ServerService.Readiness(&server.ReadinessParams{
			Context: ctx,
		})
		if err != nil {
			return fmt.Errorf("failed to pass the server readiness probe: %w", err)
		}

		_, err = userClient.Default.UserService.GetUser(&user_service.GetUserParams{
			Context: ctx,
		})
		if err != nil {
			return fmt.Errorf("failed to get user details: %w", err)
		}
		return nil
	})
}

// retryWithBackoff retries fn with capped exponential backoff until it succeeds,
// attempts are exhausted, or ctx is done.
func retryWithBackoff(ctx context.Context, attempts int, fn func() error) error {
	var lastErr error
	for i := range attempts {
		lastErr = fn()
		if lastErr == nil {
			return nil
		}
		if i == attempts-1 {
			break
		}
		select {
		case <-time.After(backoff(i)):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return fmt.Errorf("retries exhausted: %w", lastErr)
}

func backoff(attempt int) time.Duration {
	d := time.Duration(1<<attempt) * time.Second
	return min(d, 5*time.Second)
}
