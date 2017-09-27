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

package telemetry

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type file struct {
	path    string
	content []byte
}

func newFile() os.FileInfo {
	return &file{}
}

func (f *file) Name() string {
	return f.path
}
func (f *file) Size() int64 {
	return int64(len(f.content))
}
func (f *file) IsDir() bool {
	return false
}
func (f *file) Sys() interface{} {
	return ""
}
func (f *file) ModTime() time.Time {
	return time.Now()
}
func (f *file) Mode() os.FileMode {
	return os.ModePerm
}

func TestService(t *testing.T) {
	var count int
	var lastHeader string

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		defer r.Body.Close()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintln(w, fmt.Sprintf("cannot decode body: %s", err.Error()))
			return
		}
		if len(body) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if xHeader, ok := r.Header["X-Percona-Toolkit-Tool"]; ok {
			if len(xHeader) > 0 {
				lastHeader = xHeader[0]
			}
		}
		count++
	}))
	defer ts.Close()

	uuid, err := GenerateUUID()
	require.NoError(t, err)
	service := &Service{
		UUID:       uuid,
		URL:        ts.URL,
		PMMVersion: "1.3.0",
		Interval:   1 * time.Second,
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		service.Run(ctx)
		close(done)
	}()

	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, count, 1)
	cancel()
	<-done
	assert.Equal(t, lastHeader, "pmm")

	// Test a service restart
	ctx, cancel = context.WithCancel(context.Background())
	done = make(chan struct{})
	go func() {
		service.Run(ctx)
		close(done)
	}()

	time.Sleep(1100 * time.Millisecond)
	assert.Equal(t, count, 3)
	cancel()
	<-done
}

func TestServiceIntegration(t *testing.T) {
	integrationTests := os.Getenv("INTEGRATION_TESTS")
	if integrationTests == "" {
		t.Skipf("Env var INTEGRATION_TESTS is not set. Skipping integration test")
	}

	// Using this env var for compatibility with the Toolkit
	telemetryEnvURL := os.Getenv("PERCONA_VERSION_CHECK_URL")
	if telemetryEnvURL == "" {
		t.Skipf("Env var PERCONA_VERSION_CHECK_URL is not set. Skipping integration test")
	}
	uuid, err := GenerateUUID()
	require.NoError(t, err)
	service := &Service{
		UUID:       uuid,
		URL:        telemetryEnvURL,
		PMMVersion: "1.3.0",
	}
	assert.Contains(t, service.runOnce(context.Background()), "telemetry data sent")
}

func TestCollectData(t *testing.T) {
	service := &Service{
		PMMVersion: "1.3.0",
	}

	m := service.collectData()
	assert.NotEmpty(t, m)

	assert.Contains(t, m, "OS")
	assert.Contains(t, m, "PMM")
}

func TestMakePayload(t *testing.T) {
	service := &Service{
		UUID: "ABCDEFG12345",
	}

	m := map[string]string{
		"OS":  "Kubuntu",
		"PMM": "1.2.3",
	}

	b := service.makePayload(m)
	// Don't remove \n at the end of the strings. They are needed by the API
	// so I want to ensure makePayload adds them
	assert.Contains(t, string(b), "ABCDEFG12345;OS;Kubuntu\n")
	assert.Contains(t, string(b), "ABCDEFG12345;PMM;1.2.3\n")
}

func TestGetLinuxDistribution(t *testing.T) {
	for expected, procVersion := range map[string][]string{
		// cat /proc/version
		"Ubuntu 16.04": {
			`Linux version 4.4.0-57-generic (buildd@lgw01-54) (gcc version 5.4.0 20160609 (Ubuntu 5.4.0-6ubuntu1~16.04.4) ) #78-Ubuntu SMP Fri Dec 9 23:50:32 UTC 2016`,
			`Linux version 4.4.0-96-generic (buildd@lgw01-10) (gcc version 5.4.0 20160609 (Ubuntu 5.4.0-6ubuntu1~16.04.4) ) #119-Ubuntu SMP Tue Sep 12 14:59:54 UTC 2017`,
			`Linux version 4.10.0-27-generic (buildd@lgw01-60) (gcc version 5.4.0 20160609 (Ubuntu 5.4.0-6ubuntu1~16.04.4) ) #30~16.04.2-Ubuntu SMP Thu Jun 29 16:07:46 UTC 2017`,
		},

		"Fedora 26": {
			`Linux version 4.12.13-300.fc26.x86_64 (mockbuild@bkernel01.phx2.fedoraproject.org) (gcc version 7.1.1 20170622 (Red Hat 7.1.1-3) (GCC) ) #1 SMP Thu Sep 14 16:00:38 UTC 2017`,
		},

		"CentOS": {
			`Linux version 3.10.0-327.22.2.el7.x86_64 (builder@kbuilder.dev.centos.org) (gcc version 4.8.3 20140911 (Red Hat 4.8.3-9) (GCC) ) #1 SMP Thu Jun 23 17:05:11 UTC 2016`,
			`Linux version 3.10.0-327.18.2.el7.x86_64 (builder@kbuilder.dev.centos.org) (gcc version 4.8.3 20140911 (Red Hat 4.8.3-9) (GCC) ) #1 SMP Thu May 12 11:03:55 UTC 2016`,
			`Linux version 3.10.0-327.28.3.el7.x86_64 (builder@kbuilder.dev.centos.org) (gcc version 4.8.3 20140911 (Red Hat 4.8.3-9) (GCC) ) #1 SMP Thu Aug 18 19:05:49 UTC 2016`,
			`Linux version 3.10.0-327.36.3.el7.x86_64 (builder@kbuilder.dev.centos.org) (gcc version 4.8.5 20150623 (Red Hat 4.8.5-4) (GCC) ) #1 SMP Mon Oct 24 16:09:20 UTC 2016`,
			`Linux version 3.10.0-514.10.2.el7.x86_64 (builder@kbuilder.dev.centos.org) (gcc version 4.8.5 20150623 (Red Hat 4.8.5-11) (GCC) ) #1 SMP Fri Mar 3 00:04:05 UTC 2017`,
		},

		"Arch": {
			`Linux version 4.9.43-1-ARCH (builduser@leming) (gcc version 7.1.1 20170630 (GCC) ) #1 SMP Fri Aug 18 01:10:29 UTC 2017`,
		},

		"Amazon": {
			`Linux version 4.9.38-16.35.amzn1.x86_64 (mockbuild@gobi-build-60006) (gcc version 4.8.3 20140911 (Red Hat 4.8.3-9) (GCC) ) #1 SMP Sat Aug 5 01:39:35 UTC 2017`,
		},

		"Microsoft": {
			`Linux version 4.4.0-43-Microsoft (Microsoft@Microsoft.com) (gcc version 5.4.0 (GCC) ) #1-Microsoft Wed Dec 31 14:42:53 PST 2014`,
		},

		"unknown": {
			``,
		},
	} {
		for _, v := range procVersion {
			actual := getLinuxDistribution(v)
			assert.Equal(t, expected, actual)
		}
	}
}
