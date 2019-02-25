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
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm-managed/utils/logger"
)

// TODO add RDS service
func setup(t *testing.T) (context.Context, string) {
	ctx := logger.Set(context.Background(), t.Name())

	tmpDir, err := ioutil.TempDir("", t.Name())
	require.NoError(t, err)
	logsRootDir = tmpDir + "/"

	for _, name := range []string{"test-1.log", "pmm-test-agent-12345.log"} {
		err = ioutil.WriteFile(filepath.Join(tmpDir, name), []byte(fmt.Sprintf("%s: test\n", name)), 0600)
		require.NoError(t, err)
	}

	return ctx, filepath.Join(tmpDir, "test-1.log")
}

func teardown(t *testing.T, logFileName string) {
	err := os.RemoveAll(filepath.Dir(logFileName))
	require.NoError(t, err)
}

func TestZip(t *testing.T) {
	ctx, logFileName := setup(t)
	defer teardown(t, logFileName)

	logs := []Log{
		{logFileName, "", nil},
	}
	l := New("1.2.3", logs)

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
	ctx, logFileName := setup(t)
	defer teardown(t, logFileName)

	l := New("1.2.3", nil)
	buf := new(bytes.Buffer)
	err := l.Zip(ctx, buf)
	require.NoError(t, err)

	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	require.NoError(t, err)
	assert.Len(t, zr.File, len(defaultLogs))
}
