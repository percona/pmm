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

package agentv1

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// incomingStream is a server stream whose context has the metadata that a client sent.
type incomingStream struct {
	grpc.ServerStream

	ctx context.Context //nolint:containedctx
}

func (s incomingStream) Context() context.Context { return s.ctx }

func TestAgentConnectMetadata(t *testing.T) {
	t.Parallel()

	for name, sent := range map[string]*AgentConnectMetadata{
		"with a sealing key": {ID: "pmm-agent-1", Version: "3.10.0", MetricsPort: 7777, SealingKey: "BASE64KEY"},
		"of an older agent":  {ID: "pmm-agent-1", Version: "3.9.0", MetricsPort: 7777},
	} {
		ctx := AddAgentConnectMetadata(t.Context(), sent)
		md, ok := metadata.FromOutgoingContext(ctx)
		require.True(t, ok, name)

		received, err := ReceiveAgentConnectMetadata(incomingStream{ctx: metadata.NewIncomingContext(t.Context(), md)})
		require.NoError(t, err, name)
		assert.Equal(t, sent, received, name)
	}
}
