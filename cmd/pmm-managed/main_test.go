// pmm-managed
// Copyright (C) 2017 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/percona/pmm-managed/api"
	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/services/consul"
	"github.com/percona/pmm-managed/services/logs"
	"github.com/percona/pmm-managed/services/prometheus"
	"github.com/percona/pmm-managed/utils/logger"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/mysql"
)

func TestServer(t *testing.T) {
	var err error

	var wg sync.WaitGroup
	defer wg.Wait()

	// todo Fix below globals.
	*prometheusConfigF = "testdata/prometheus/prometheus.yml" // todo Fix this global.
	*dbNameF = "pmm-managed-dev"                              // todo Fix this global.
	*agentMySQLdExporterF = "mysqld_exporter"
	*agentRDSExporterF = "rds_exporter"
	*agentRDSExporterConfigF = "testdata/rds_exporter/rds_exporter.yml"

	ctx, cancel := context.WithCancel(context.Background())
	ctx, _ = logger.Set(ctx, "main") // todo runGRPCServer panics without this global being set.
	defer cancel()

	l := logs.New(ctx, logs.DefaultLogs, 1000)
	consulClient, err := consul.NewClient(*consulAddrF)
	require.NoError(t, err)
	sqlDB, err := models.OpenDB(*dbNameF, *dbUsernameF, *dbPasswordF, logrus.WithField("component", "main").Debugf)
	require.NoError(t, err)
	defer sqlDB.Close()
	db := reform.NewDB(sqlDB, mysql.Dialect, nil)
	prometheus, err := prometheus.NewService(*prometheusConfigF, *prometheusURLF, *promtoolF, consulClient)
	require.NoError(t, err)
	err = prometheus.Check(ctx)
	require.NoError(t, err)

	wg.Add(1)
	go func() {
		defer wg.Done()
		runGRPCServer(ctx, consulClient, db, l)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		runRESTServer(ctx, l)
	}()

	_, err = waitForBody(fmt.Sprintf("http://%s/v0/logs", *restAddrF))
	require.NoError(t, err)

	tests := []func(*testing.T){
		testLogs,
		testAnnotations,
	}
	t.Run("pmm-admin", func(t *testing.T) {
		for _, f := range tests {
			f := f // capture range variable
			fName := runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
			t.Run(fName, func(t *testing.T) {
				t.Parallel()
				f(t)
			})
		}
	})

}

func testLogs(t *testing.T) {
	body, err := waitForBody(fmt.Sprintf("http://%s/logs.zip", *restAddrF))
	require.NoError(t, err)

	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		t.Fatalf("NewReader: %v", err)
	}
	assert.Len(t, zr.File, len(logs.DefaultLogs))

	for i := range zr.File {
		f, err := zr.File[i].Open()
		assert.NoError(t, err)
		b, err := ioutil.ReadAll(f)
		assert.NoError(t, err)
		f.Close()
		fName := filepath.Base(zr.File[i].Name)
		assert.Contains(t, string(b), fName)
	}
}

func testAnnotations(t *testing.T) {
	a := api.AnnotationsCreateRequest{
		Tags: []string{"a", "b"},
		Text: "xyz",
	}
	b := &bytes.Buffer{}
	err := json.NewEncoder(b).Encode(a)
	require.NoError(t, err)

	r, err := http.Post(fmt.Sprintf("http://%s/v0/annotations", *restAddrF), "application/json", b)
	require.NoError(t, err)
	defer r.Body.Close()

	body, err := ioutil.ReadAll(r.Body)
	require.NoError(t, err)

	resp := api.AnnotationsCreateResponse{}
	err = json.Unmarshal(body, &resp)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, r.StatusCode)
	assert.Equal(t, "Annotation added", resp.Message, "body: %s", string(body))
}

// waitForBody is a helper function which makes http calls until http server is up
// and then returns body of the successful call.
func waitForBody(urlToGet string) (body []byte, err error) {
	tries := 60

	// Get data, but we need to wait a bit for http server.
	for i := 0; i <= tries; i++ {
		// Try to get web page.
		body, err = getBody(urlToGet)
		if err == nil {
			return body, err
		}

		// If there is a syscall.ECONNREFUSED error (web server not available) then retry.
		if urlError, ok := err.(*url.Error); ok {
			if opError, ok := urlError.Err.(*net.OpError); ok {
				if osSyscallError, ok := opError.Err.(*os.SyscallError); ok {
					if osSyscallError.Err == syscall.ECONNREFUSED {
						time.Sleep(1 * time.Second)
						continue
					}
				}
			}
		}

		// There was an error, and it wasn't syscall.ECONNREFUSED.
		return nil, err
	}

	return nil, fmt.Errorf("failed to GET %s after %d tries: %s", urlToGet, tries, err)
}

// getBody is a helper function which retrieves http body from given address.
func getBody(urlToGet string) ([]byte, error) {
	resp, err := http.Get(urlToGet)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}
