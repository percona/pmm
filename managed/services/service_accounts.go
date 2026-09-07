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

package services

import (
	"context"
	"time"
)

// serviceAccountCleanupTimeout bounds the Grafana calls of RemoveNodeServiceAccount. The Node is gone
// by then, so the cleanup neither waits on an unresponsive Grafana, nor stops with a client which gave
// up on the request.
const serviceAccountCleanupTimeout = 10 * time.Second

// ServiceAccountRemover deletes the Grafana service account of a Node.
type ServiceAccountRemover interface {
	DeleteServiceAccount(ctx context.Context, nodeName string, force bool) (string, error)
}

// RemoveNodeServiceAccount deletes the Grafana service account named after a removed Node, so that the
// token its pmm-agent authenticates with does not outlive the Node. It runs on a context of its own,
// derived from ctx to keep the auth headers Grafana needs: the Node is already gone, so a client which
// gave up on the request must not leave the account behind, and an unresponsive Grafana must not hold
// the caller. The returned warning is what Grafana reports when it keeps an account with custom tokens.
func RemoveNodeServiceAccount(ctx context.Context, c ServiceAccountRemover, nodeName string, force bool) (string, error) {
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), serviceAccountCleanupTimeout)
	defer cancel()

	return c.DeleteServiceAccount(cleanupCtx, nodeName, force)
}
