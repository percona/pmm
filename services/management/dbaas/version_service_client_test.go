// pmm-managed
// Copyright (C) 2017 Percona LLC
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

package dbaas

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVersionServiceClient(t *testing.T) {
	c := NewVersionServiceClient(versionServiceURL)

	for _, tt := range []struct {
		params componentsParams
	}{
		{params: componentsParams{product: psmdbOperator}},
		{params: componentsParams{product: psmdbOperator, productVersion: "1.6.0"}},
		{params: componentsParams{product: psmdbOperator, productVersion: "1.7.0", dbVersion: "4.2.8-8"}},
		{params: componentsParams{product: pxcOperator}},
		{params: componentsParams{product: pxcOperator, productVersion: "1.7.0"}},
		{params: componentsParams{product: pxcOperator, productVersion: "1.7.0", dbVersion: "8.0.20-11.2"}},
	} {
		t.Run("NotEmptyMatrix", func(t *testing.T) {
			response, err := c.Matrix(context.TODO(), tt.params)
			require.NoError(t, err)
			require.NotEmpty(t, response.Versions)
			for _, v := range response.Versions {
				switch tt.params.product {
				case psmdbOperator:
					assert.NotEmpty(t, v.Matrix.Mongod)
				case pxcOperator:
					assert.NotEmpty(t, v.Matrix.Pxc)
					assert.NotEmpty(t, v.Matrix.Proxysql)
				}
				assert.NotEmpty(t, v.Matrix.Backup)
			}
		})
	}
}

type fakeLatestVersionServer struct {
	response *VersionServiceResponse
}

func (f fakeLatestVersionServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	encoder := json.NewEncoder(w)
	var response *VersionServiceResponse
	if strings.Contains(r.URL.Path, "pmm-server/") {
		segments := strings.Split(r.URL.Path, "/")
		pmmServerVersion := segments[len(segments)-1]
		for _, v := range f.response.Versions {
			if v.ProductVersion == pmmServerVersion {
				response = &VersionServiceResponse{
					Versions: []struct {
						Product        string `json:"product"`
						ProductVersion string `json:"operator"`
						Matrix         matrix `json:"matrix"`
					}{v},
				}
			}
		}
	} else if strings.HasSuffix(r.URL.Path, "pmm-server") {
		response = f.response
	} else {
		panic("path not expected")
	}
	err := encoder.Encode(response)
	if err != nil {
		log.Fatal(err)
	}
}

func newFakeVersionService(response *VersionServiceResponse, port string) (versionService, func(*testing.T)) {
	var httpServer *http.Server
	waitForListener := make(chan struct{})
	server := fakeLatestVersionServer{
		response: response,
	}
	fakeHostAndPort := "localhost:" + port
	go func() {
		httpServer = &http.Server{Addr: fakeHostAndPort, Handler: server}
		listener, err := net.Listen("tcp", fakeHostAndPort)
		if err != nil {
			log.Fatal(err)
		}
		close(waitForListener)
		_ = httpServer.Serve(listener)
	}()
	<-waitForListener

	return NewVersionServiceClient("http://" + fakeHostAndPort + "/versions/v1"), func(t *testing.T) {
		assert.NoError(t, httpServer.Shutdown(context.TODO()))
	}
}

func TestLatestVersionGetting(t *testing.T) {
	t.Parallel()
	t.Run("Invalid url", func(t *testing.T) {
		t.Parallel()
		c := NewVersionServiceClient("wrongschema://check.percona.com/versions/invalid")
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		operator, pmm, err := c.GetLatestOperatorVersion(ctx, "2.19")
		assert.Error(t, err, "err is expected")
		assert.Nil(t, operator)
		assert.Nil(t, pmm)
	})
	response := &VersionServiceResponse{
		Versions: []struct {
			Product        string `json:"product"`
			ProductVersion string `json:"operator"`
			Matrix         matrix `json:"matrix"`
		}{
			{
				ProductVersion: twoPointEighteen,
				Product:        "pmm-server",
				Matrix: matrix{
					PXCOperator: map[string]componentVersion{
						"1.8.0": {},
						"1.7.0": {},
					},
					PSMDBOperator: map[string]componentVersion{
						"1.9.0": {},
						"1.8.0": {},
						"1.7.0": {},
					},
				},
			},
		},
	}
	c, cleanup := newFakeVersionService(response, "5897")
	t.Cleanup(func() { cleanup(t) })
	t.Run("Get latest", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		pxcOperatorVersion, psmdbOperatorVersion, err := c.GetLatestOperatorVersion(ctx, twoPointEighteen)
		require.NoError(t, err, "request to fakeserver for latest version should not fail")
		assert.Equal(t, "1.8.0", pxcOperatorVersion.String())
		assert.Equal(t, "1.9.0", psmdbOperatorVersion.String())
	})
	t.Run("Get latest", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		pxcOperatorVersion, psmdbOperatorVersion, err := c.GetLatestOperatorVersion(ctx, "2.220.0")
		require.NoError(t, err, "request to fakeserver for latest version should not fail")
		assert.Nil(t, pxcOperatorVersion)
		assert.Nil(t, psmdbOperatorVersion)
	})
}
