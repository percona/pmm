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
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"google.golang.org/grpc/codes"
)

// The leader check answers with an empty body or a small gRPC error document, so anything
// larger is something else on the port and is not worth reading into memory.
const maxLeaderCheckBody = 4 * 1024

// isLeader reports whether the pmm-managed running next to this process holds cluster
// leadership, by asking its leader health check endpoint.
//
// In an HA cluster every node runs its own qan-api2, all writing to the same ClickHouse,
// so data retention has to be enforced by a single node. pmm-managed already exposes that
// decision: the endpoint answers 200 on the leader and a gRPC FailedPrecondition, rendered
// as 400 carrying gRPC code 9, everywhere else. A 200, or a 400 that says FailedPrecondition,
// are the only answers about leadership; anything else is an error rather than a verdict. It is
// the same check HAProxy uses to pick a backend, and it needs no credentials. When high
// availability is disabled the endpoint always answers 200, so a single-container deployment
// keeps enforcing retention.
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

	// Read the body so it can be decoded below, and so the connection can be reused by the
	// next check.
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxLeaderCheckBody))
	if err != nil {
		return false, fmt.Errorf("failed to read the answer from %s: %w", url, err)
	}

	switch resp.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusBadRequest:
		// 400 alone does not mean "follower": the gateway renders InvalidArgument,
		// FailedPrecondition and OutOfRange alike, and anything sitting on the port can
		// answer 400 too. Require the code the endpoint actually sends, so a follower
		// verdict is confirmed rather than inferred.
		return false, followerVerdict(body, url)
	default:
		// Anything else is not an answer about leadership: a renamed route, a proxy in
		// front of the port, a broken pmm-managed. Reading it as "follower" would stop
		// retention everywhere with nothing in the log, so report it as undetermined.
		return false, fmt.Errorf("unexpected status %s from %s", resp.Status, url)
	}
}

// followerVerdict confirms that a 400 came from the leader check saying this node is a
// follower, rather than from something else on the port.
//
// LeaderHealthCheck returns FailedPrecondition and nothing else, see the handler in
// managed/services/server/server.go. If that ever changes, every node reads as undetermined
// and stops deleting, which is the safe direction to fail.
func followerVerdict(body []byte, url string) error {
	var answer struct {
		Code int32 `json:"code"`
	}
	err := json.Unmarshal(body, &answer)
	if err != nil {
		return fmt.Errorf("failed to decode the 400 answer from %s: %w", url, err)
	}

	got := codes.Code(answer.Code) //nolint:gosec
	if got != codes.FailedPrecondition {
		return fmt.Errorf("unexpected gRPC code %s in the 400 answer from %s", got, url)
	}

	return nil
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
