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

package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

// isLeader reports whether the pmm-managed running next to this process holds cluster
// leadership, by asking its leader health check endpoint.
//
// In an HA cluster every node runs its own qan-api2, all writing to the same ClickHouse,
// so data retention has to be enforced by a single node. pmm-managed already exposes that
// decision: the endpoint answers 200 on the leader and a gRPC FailedPrecondition, rendered
// as 400, everywhere else. Those two are the only answers about leadership; any other status
// is an error rather than a follower verdict. It is the same check HAProxy uses to pick a
// backend, and it needs no credentials. When high availability is disabled the endpoint
// always answers 200, so a single-container deployment keeps enforcing retention.
//
// An error means leadership could not be determined, which callers must treat as "not the
// leader": deleting data on a stale answer is worse than deleting it a few minutes later.
func isLeader(ctx context.Context, client *http.Client, url string) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, fmt.Errorf("failed to build leader check request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to reach %s: %w", url, err)
	}
	defer resp.Body.Close() //nolint:errcheck

	// Drain the body so the connection can be reused by the next check.
	_, _ = io.Copy(io.Discard, resp.Body)

	switch resp.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusBadRequest:
		// FailedPrecondition, the endpoint's way of saying this node is a follower.
		return false, nil
	default:
		// Anything else is not an answer about leadership: a renamed route, a proxy in
		// front of the port, a broken pmm-managed. Reading it as "follower" would stop
		// retention everywhere with nothing in the log, so report it as undetermined.
		return false, fmt.Errorf("unexpected status %s from %s", resp.Status, url)
	}
}

// shouldApplyRetention reports whether this node is the one that should drop old partitions.
// An empty leaderCheckURL disables the check, for deployments that run qan-api2 without a
// pmm-managed next to it.
func shouldApplyRetention(ctx context.Context, client *http.Client, leaderCheckURL string) (bool, error) {
	if leaderCheckURL == "" {
		return true, nil
	}
	return isLeader(ctx, client, leaderCheckURL)
}
