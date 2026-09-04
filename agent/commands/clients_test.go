// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package commands

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	httptransport "github.com/go-openapi/runtime/client"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The subtests share the package level API clients, so they cannot run in parallel.
func TestServerKnowsAgent(t *testing.T) {
	const agentID = "5a2b8a4b-2b9d-4a5f-9a11-2b6a3f6f9a11"

	for _, tc := range []struct {
		name       string
		statusCode int
		hangs      bool
		known      bool
		unknowable bool
	}{
		{
			name:       "PMM Server knows the Agent",
			statusCode: http.StatusOK,
			known:      true,
		},
		{
			name:       "PMM Server does not know the Agent",
			statusCode: http.StatusNotFound,
		},
		{
			name:       "PMM Server does not accept the credentials",
			statusCode: http.StatusUnauthorized,
		},
		{
			name:       "PMM Server forbids the request",
			statusCode: http.StatusForbidden,
		},
		{
			name:       "PMM Server cannot answer",
			statusCode: http.StatusServiceUnavailable,
			unknowable: true,
		},
		{
			name:       "PMM Server hangs",
			hangs:      true,
			unknowable: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.hangs {
				// The request has to give up long before the default 30 seconds.
				defaultTimeout := httptransport.DefaultTimeout
				httptransport.DefaultTimeout = 100 * time.Millisecond
				t.Cleanup(func() { httptransport.DefaultTimeout = defaultTimeout })
			}

			server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				if tc.hangs {
					<-req.Context().Done()
					return
				}
				rw.Header().Set("Content-Type", "application/json")
				rw.WriteHeader(tc.statusCode)
				_, _ = rw.Write([]byte(`{"message": "` + tc.name + `"}`))
			}))
			t.Cleanup(server.Close)

			u, err := url.Parse(server.URL)
			require.NoError(t, err)
			setServerTransport(u, true, logrus.WithField("test", t.Name()))

			known, err := serverKnowsAgent(agentID)
			if tc.unknowable {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tc.known, known)
		})
	}
}
