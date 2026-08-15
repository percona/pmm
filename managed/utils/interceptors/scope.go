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

package interceptors

import (
	"context"

	"google.golang.org/grpc"

	"github.com/percona/pmm/managed/utils/auth"
)

// nodeScopeResolver resolves a caller to the node its service token was issued for.
type nodeScopeResolver interface {
	BoundNodeID(ctx context.Context) string
}

// UnaryNodeScopeInterceptor resolves the caller's node scope once per request and puts it in
// the context, so handlers can confine a node's own token to that node. Callers that are not
// bound to a node - users, unbound tokens - are left unconfined and see everything as before.
func UnaryNodeScopeInterceptor(r nodeScopeResolver) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		return handler(auth.WithNodeScope(ctx, r.BoundNodeID(ctx)), req)
	}
}
