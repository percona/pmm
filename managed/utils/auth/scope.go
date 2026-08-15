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

package auth

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type nodeScopeKey struct{}

// WithNodeScope confines the context to a single node. It is set for callers that
// authenticated with a service token bound to that node, and never for human users.
func WithNodeScope(ctx context.Context, nodeID string) context.Context {
	if nodeID == "" {
		return ctx
	}

	return context.WithValue(ctx, nodeScopeKey{}, nodeID)
}

// NodeScope returns the node the caller is confined to. The second result is false for
// unconfined callers - users and anything authorized by Grafana role - who see everything.
func NodeScope(ctx context.Context) (string, bool) {
	nodeID, ok := ctx.Value(nodeScopeKey{}).(string)
	return nodeID, ok && nodeID != ""
}

// CheckNodeScope returns an error when the caller is confined to a node other than nodeID.
// Unconfined callers are always allowed.
func CheckNodeScope(ctx context.Context, nodeID string) error {
	scoped, ok := NodeScope(ctx)
	if !ok || scoped == nodeID {
		return nil
	}

	return status.Error(codes.PermissionDenied, "This token may only act on its own node.")
}
