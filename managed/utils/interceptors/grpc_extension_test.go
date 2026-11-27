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
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"
)

func TestCallerOriginFromRequest(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		referer  string
		method   string
		expected callerOrigin
	}{
		"API client":             {method: "/server.v1.ServerService/Version", expected: externalCallerOrigin},
		"UI":                     {referer: "https://pmm-server/", method: "/server.v1.ServerService/Version", expected: internalCallerOrigin},
		"readiness probe":        {method: "/server.v1.ServerService/Readiness", expected: internalCallerOrigin},
		"HAProxy health check":   {method: "/server.v1.ServerService/LeaderHealthCheck", expected: internalCallerOrigin},
		"pmm-agent stream":       {method: "/agent.v1.AgentService/Connect", expected: internalCallerOrigin},
		"PMM 2 pmm-agent stream": {method: "/agent.Agent/Connect", expected: internalCallerOrigin},
		"RTA collector":          {method: "/realtimeanalytics.v1.CollectorService/Collect", expected: internalCallerOrigin},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()
			if tc.referer != "" {
				ctx = metadata.NewIncomingContext(ctx, metadata.Pairs("grpcgateway-referer", tc.referer))
			}
			assert.Equal(t, tc.expected, callerOriginFromRequest(ctx, tc.method))
		})
	}
}
