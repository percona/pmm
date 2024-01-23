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

package qan

import (
	"context"

	"google.golang.org/grpc"

	qanpb "github.com/percona/pmm/api/qanpb"
)

// qanClient is a subset of methods of qanpb.CollectorClient used by this package.
// We use it instead of real type for testing.
type qanCollectorClient interface {
	Collect(ctx context.Context, in *qanpb.CollectRequest, opts ...grpc.CallOption) (*qanpb.CollectResponse, error)
}
