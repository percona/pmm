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

package logs

import (
	"archive/zip"
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/mysql"

	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/services/consul"
	"github.com/percona/pmm-managed/utils/logger"
	"github.com/percona/pmm-managed/utils/tests"
)

// TODO add RDS service
func setup(t *testing.T) (context.Context, *consul.Client, *reform.DB, string) {
	ctx, _ := logger.Set(context.Background(), t.Name())

	consulClient, err := consul.NewClient("127.0.0.1:8500")
	require.NoError(t, err)

	db := reform.NewDB(tests.OpenTestDB(t), mysql.Dialect, reform.NewPrintfLogger(t.Logf))

	tmpDir, err := ioutil.TempDir("", t.Name())
	require.NoError(t, err)
	logsRootDir = tmpDir + "/"

	for _, name := range []string{"test-1.log", "pmm-test-agent-12345.log"} {
		err = ioutil.WriteFile(filepath.Join(tmpDir, name), []byte(fmt.Sprintf("%s: test\n", name)), 0600)
		require.NoError(t, err)
	}

	return ctx, consulClient, db, filepath.Join(tmpDir, "test-1.log")
}

func teardown(t *testing.T, db *reform.DB, logFileName string) {
	err := db.DBInterface().(*sql.DB).Close()
	assert.NoError(t, err)

	err = os.RemoveAll(filepath.Dir(logFileName))
	require.NoError(t, err)
}

func TestZip(t *testing.T) {
	ctx, consulClient, db, logFileName := setup(t)
	defer teardown(t, db, logFileName)

	logs := []Log{
		{logFileName, "", nil},
	}
	l := New("1.2.3", consulClient, db, nil, logs)

	buf := new(bytes.Buffer)
	err := l.Zip(ctx, buf)
	require.NoError(t, err)

	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	require.NoError(t, err)
	assert.Len(t, zr.File, len(logs))

	for i := range zr.File {
		f, err := zr.File[i].Open()
		assert.NoError(t, err)
		b, err := ioutil.ReadAll(f)
		assert.NoError(t, err)
		f.Close()
		fName := filepath.Base(zr.File[i].Name)
		assert.Equal(t, fmt.Sprintf("%s: test\n", fName), string(b))
	}
}

func TestZipDefaultLogs(t *testing.T) {
	ctx, consulClient, db, logFileName := setup(t)
	defer teardown(t, db, logFileName)

	l := New("1.2.3", consulClient, db, nil, nil)

	buf := new(bytes.Buffer)
	err := l.Zip(ctx, buf)
	require.NoError(t, err)

	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	require.NoError(t, err)
	assert.Len(t, zr.File, len(defaultLogs))
}

func TestFiles(t *testing.T) {
	ctx, consulClient, db, logFileName := setup(t)
	defer teardown(t, db, logFileName)

	err := db.Insert(&models.Agent{Type: "test-agent", RunsOnNodeID: 1, ListenPort: pointer.ToUint16(12345)})
	require.NoError(t, err)

	logs := []Log{
		{logFileName, "", nil},
	}
	l := New("1.2.3", consulClient, db, nil, logs)

	files := l.Files(ctx)
	assert.Len(t, files, 2)

	for i := range files {
		assert.NoError(t, files[i].Err)
		fName := filepath.Base(files[i].Name)
		assert.Equal(t, fmt.Sprintf("%s: test\n", fName), string(files[i].Data))
	}
}
