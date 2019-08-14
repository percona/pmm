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

package supervisord

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

func TestCustomLogs(t *testing.T) {
	ctx := logger.Set(context.Background(), t.Name())
	dir, err := ioutil.TempDir("", t.Name())
	require.NoError(t, err)

	defer func() {
		err := os.RemoveAll(dir)
		require.NoError(t, err)
	}()

	logs := map[string]logInfo{
		"test1.log": {FilePath: filepath.Join(dir, "test1.log")},
	}
	for name := range logs {
		err = ioutil.WriteFile(filepath.Join(dir, name), []byte(fmt.Sprintf("%s: test\n", name)), 0600)
		require.NoError(t, err)
	}

	l := NewLogs("1.2.3")
	l.logs = logs
	buf := new(bytes.Buffer)
	err = l.Zip(ctx, buf)
	require.NoError(t, err)

	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	require.NoError(t, err)
	assert.Len(t, zr.File, len(logs)+6)

	for _, zf := range zr.File {
		if _, ok := logs[zf.Name]; !ok {
			continue
		}
		r, err := zf.Open()
		require.NoError(t, err)
		b, err := ioutil.ReadAll(r)
		require.NoError(t, err)
		require.NoError(t, r.Close())
		assert.Equal(t, fmt.Sprintf("%s: test\n", zf.Name), string(b))
	}
}

func TestDefaultLogs(t *testing.T) {
	ctx := logger.Set(context.Background(), t.Name())

	l := NewLogs("1.2.3")
	buf := new(bytes.Buffer)
	err := l.Zip(ctx, buf)
	require.NoError(t, err)

	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	require.NoError(t, err)
	assert.Len(t, zr.File, len(defaultLogs)+6)
}
