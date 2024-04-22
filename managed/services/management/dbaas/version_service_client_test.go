// Copyright (C) 2024 Percona LLC
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

	goversion "github.com/hashicorp/go-version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVersionServiceClient(t *testing.T) {
	c := NewVersionServiceClient(versionServiceURL)

	for _, tt := range []struct {
		params componentsParams
	}{
		{params: componentsParams{product: psmdbOperator}},
		{params: componentsParams{product: psmdbOperator, productVersion: onePointSix}},
		{params: componentsParams{product: psmdbOperator, productVersion: onePointSeven, dbVersion: "4.2.8-8"}},
		{params: componentsParams{product: pxcOperator}},
		{params: componentsParams{product: pxcOperator, productVersion: onePointSeven}},
		{params: componentsParams{product: pxcOperator, productVersion: onePointSeven, dbVersion: "8.0.20-11.2"}},
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
	response   *VersionServiceResponse
	components []string
}

func (f fakeLatestVersionServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	encoder := json.NewEncoder(w)
	var response *VersionServiceResponse
	var certainVersionRequested bool
	var component string
	for _, c := range f.components {
		if strings.Contains(r.URL.Path, c) {
			component = c
			certainVersionRequested = strings.Contains(r.URL.Path, component+"/")
			break
		}
	}

	switch {
	case certainVersionRequested:
		segments := strings.Split(r.URL.Path, "/")
		version := segments[len(segments)-2]
		var dbVersion string
		// handle product/version/applyversion
		if _, err := goversion.NewVersion(version); err == nil {
			dbVersion = segments[len(segments)-1]
		} else {
			version = segments[len(segments)-1]
		}
		for _, v := range f.response.Versions {
			if v.ProductVersion == version && v.Product == component {
				if dbVersion != "" {
					var database map[string]componentVersion
					switch component {
					case pxcOperator:
						database = v.Matrix.Pxc
					case psmdbOperator:
						database = v.Matrix.Mongod
					default:
						panic(component + " not supported")
					}
					if _, ok := database[dbVersion]; !ok {
						response = nil
						break
					}
				}
				response = &VersionServiceResponse{
					Versions: []Version{v},
				}
				break
			}
		}
	case component != "":
		response = &VersionServiceResponse{}
		for _, v := range f.response.Versions {
			if v.Product == component {
				response.Versions = append(response.Versions, v)
			}
		}
	default:
		panic("path " + r.URL.Path + " not expected")
	}

	err := encoder.Encode(response)
	if err != nil {
		log.Fatal(err)
	}
}

// newFakeVersionService creates new fake version service on given port.
// It returns values based on given response but only for specified components.
func newFakeVersionService(response *VersionServiceResponse, port string, components ...string) (versionService, func(*testing.T)) {
	if len(components) == 0 {
		panic("failed to create fake version service, at least one component has to be given, none received")
	}
	var httpServer *http.Server
	waitForListener := make(chan struct{})
	server := fakeLatestVersionServer{
		response:   response,
		components: components,
	}
	fakeHostAndPort := "localhost:" + port
	go func() {
		httpServer = &http.Server{Addr: fakeHostAndPort, Handler: server} //nolint:gosec
		listener, err := net.Listen("tcp", fakeHostAndPort)
		if err != nil {
			log.Fatal(err)
		}
		close(waitForListener)
		_ = httpServer.Serve(listener)
	}()
	<-waitForListener

	return NewVersionServiceClient("http://" + fakeHostAndPort + "/versions/v1"), func(t *testing.T) {
		t.Helper()
		assert.NoError(t, httpServer.Shutdown(context.TODO()))
	}
}

func TestOperatorVersionGetting(t *testing.T) {
	t.Parallel()
	t.Run("Invalid url", func(t *testing.T) {
		t.Parallel()
		c := NewVersionServiceClient("wrongschema://check.percona.com/versions/invalid")
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		pxcOperatorVersion, psmdbOperatorVersion, err := c.LatestOperatorVersion(ctx, "2.19")
		assert.Error(t, err, "err is expected")
		assert.Nil(t, pxcOperatorVersion)
		assert.Nil(t, psmdbOperatorVersion)
	})
	response := &VersionServiceResponse{
		Versions: []Version{
			{
				ProductVersion: onePointSix,
				Product:        pxcOperator,
			},
			{
				ProductVersion: onePointSeven,
				Product:        pxcOperator,
			},
			{
				ProductVersion: onePointEight,
				Product:        pxcOperator,
			},

			{
				ProductVersion: onePointSix,
				Product:        psmdbOperator,
			},
			{
				ProductVersion: onePointSeven,
				Product:        psmdbOperator,
			},
			{
				ProductVersion: onePointEight,
				Product:        psmdbOperator,
			},

			{
				ProductVersion: twoPointEighteen,
				Product:        "pmm-server",
				Matrix: matrix{
					PXCOperator: map[string]componentVersion{
						onePointEight: {Status: "recommended"},
						onePointSeven: {},
					},
					PSMDBOperator: map[string]componentVersion{
						onePointTen:   {Status: "available"},
						onePointNine:  {Status: "recommended"},
						onePointEight: {},
						onePointSeven: {},
					},
				},
			},
		},
	}
	c, cleanup := newFakeVersionService(response, "5897", "pmm-server", psmdbOperator, pxcOperator)
	t.Cleanup(func() { cleanup(t) })
	t.Run("Get latest", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		pxcOperatorVersion, psmdbOperatorVersion, err := c.LatestOperatorVersion(ctx, twoPointEighteen)
		require.NoError(t, err, "request to fakeserver for latest version should not fail")
		assert.Equal(t, onePointEight, pxcOperatorVersion.String())
		assert.Equal(t, onePointNine, psmdbOperatorVersion.String())
	})
	t.Run("Get latest, PMM version unknown by version service", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		pxcOperatorVersion, psmdbOperatorVersion, err := c.LatestOperatorVersion(ctx, "2.2200.0")
		require.NoError(t, err, "request to fakeserver for latest version should not fail")
		assert.Nil(t, pxcOperatorVersion)
		assert.Nil(t, psmdbOperatorVersion)
	})
	t.Run("Get next, update not available", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		pxcOperatorVersion, err := c.NextOperatorVersion(ctx, pxcOperator, onePointEight)
		require.NoError(t, err, "request to fakeserver for latest version should not fail")
		psmdbOperatorVersion, err := c.NextOperatorVersion(ctx, psmdbOperator, onePointEight)
		require.NoError(t, err, "request to fakeserver for latest version should not fail")
		assert.Nil(t, pxcOperatorVersion)
		assert.Nil(t, psmdbOperatorVersion)
	})
	t.Run("Get next, update available", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		pxcOperatorVersion, err := c.NextOperatorVersion(ctx, pxcOperator, onePointSix)
		require.NoError(t, err, "request to fakeserver for latest version should not fail")
		psmdbOperatorVersion, err := c.NextOperatorVersion(ctx, psmdbOperator, onePointSeven)
		require.NoError(t, err, "request to fakeserver for latest version should not fail")
		assert.Equal(t, onePointSeven, pxcOperatorVersion.String())
		assert.Equal(t, onePointEight, psmdbOperatorVersion.String())
	})
}

const (
	pxcImage   = "percona/percona-xtradb-cluster"
	psmdbImage = "percona/percona-server-mongodb"
)

func TestGetNextDatabaseVersion(t *testing.T) { //nolint:tparallel
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
	response := &VersionServiceResponse{
		Versions: []Version{
			{
				ProductVersion: "1.8.0",
				Product:        pxcOperator,
				Matrix: matrix{
					Pxc: map[string]componentVersion{
						"5.7.26-31.37":   {ImagePath: pxcImage + ":5.7.26-31.37"},
						"5.7.29-31.43":   {ImagePath: pxcImage + ":5.7.29-31.43"},
						"5.7.31-31.45.2": {ImagePath: pxcImage + ":5.7.31-31.45.2"},
					},
				},
			},
			{
				ProductVersion: "1.8.0",
				Product:        psmdbOperator,
				Matrix: matrix{
					Mongod: map[string]componentVersion{
						"3.6.19-7.0":  {ImagePath: psmdbImage + ":3.6.19-7.0"},
						"3.6.18-5.0":  {ImagePath: psmdbImage + ":3.6.18-5.0"},
						"3.6.23-13.0": {ImagePath: psmdbImage + ":3.6.23-13.0"},
					},
				},
			},
		},
	}
	c, cleanup := newFakeVersionService(response, "5899", pxcOperator, psmdbOperator)
	t.Cleanup(func() { cleanup(t); cancel() })
	t.Run("Update available", func(t *testing.T) {
		t.Parallel()
		nextVersionImage, err := c.GetNextDatabaseImage(ctx, pxcOperator, "1.8.0", "5.7.26-31.37")
		require.NoError(t, err)
		assert.Equal(t, pxcImage+":5.7.29-31.43", nextVersionImage)

		nextVersionImage, err = c.GetNextDatabaseImage(ctx, psmdbOperator, "1.8.0", "3.6.19-7.0")
		require.NoError(t, err)
		assert.Equal(t, psmdbImage+":3.6.23-13.0", nextVersionImage)

		// older than supported version of database
		nextVersionImage, err = c.GetNextDatabaseImage(ctx, pxcOperator, "1.8.0", "5.0.0")
		require.NoError(t, err)
		assert.Equal(t, pxcImage+":5.7.26-31.37", nextVersionImage)
	})
	t.Run("Update not available", func(t *testing.T) {
		t.Parallel()
		nextVersionImage, err := c.GetNextDatabaseImage(ctx, pxcOperator, "1.8.0", "5.7.31-31.45.2")
		require.NoError(t, err)
		assert.Equal(t, "", nextVersionImage)

		nextVersionImage, err = c.GetNextDatabaseImage(ctx, psmdbOperator, "1.8.0", "3.6.23-13.0")
		require.NoError(t, err)
		assert.Equal(t, "", nextVersionImage)

		// more up to date than is supported
		nextVersionImage, err = c.GetNextDatabaseImage(ctx, psmdbOperator, "1.8.0", "4.0.0")
		require.NoError(t, err)
		assert.Equal(t, "", nextVersionImage)
	})
}
