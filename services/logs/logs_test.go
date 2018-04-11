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
)

func setup(t *testing.T) string {
	tmpDir, err := ioutil.TempDir("", t.Name())
	require.NoError(t, err)

	logFileName := "test-1.log"
	f, err := os.Create(filepath.Join(tmpDir, logFileName))
	require.NoError(t, err)
	f.WriteString(fmt.Sprintf("%s: log line %d\n", logFileName, 1))
	f.Close()

	return f.Name()
}

func teardown(t *testing.T, logFileName string) {
	err := os.RemoveAll(filepath.Dir(logFileName))
	require.NoError(t, err)
}

func TestZip(t *testing.T) {
	logFileName := setup(t)
	defer teardown(t, logFileName)

	logs := []Log{
		{logFileName, ""},
	}
	l := New(logs, 1000)

	buf := new(bytes.Buffer)
	err := l.Zip(context.Background(), buf)
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
		assert.Equal(t, fmt.Sprintf("%s: log line %d\n", fName, 1), string(b))
	}
}

func TestZipDefaultLogs(t *testing.T) {
	l := New(DefaultLogs, 1000)

	buf := new(bytes.Buffer)
	err := l.Zip(context.Background(), buf)
	require.NoError(t, err)

	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	require.NoError(t, err)
	assert.Len(t, zr.File, len(DefaultLogs))
}

func TestFiles(t *testing.T) {
	logFileName := setup(t)
	defer teardown(t, logFileName)

	logs := []Log{
		{logFileName, ""},
	}
	l := New(logs, 1000)

	files := l.Files(context.Background())
	assert.Len(t, files, len(logs))

	for i := range files {
		assert.NoError(t, files[i].Err)
		assert.Equal(t, fmt.Sprintf("%s: log line %d\n", files[i].Name, 1), string(files[i].Data))
	}
}
