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

package filereader

import (
	"io"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func cleanup(t *testing.T, files []string) {
	t.Helper()
	for _, f := range files {
		assert.NoError(t, os.Remove(f))
	}
}

func TestContinuousFileReader(t *testing.T) {
	t.Parallel()
	t.Run("Normal", func(t *testing.T) {
		t.Parallel()

		var files []string
		defer func() {
			cleanup(t, files)
		}()

		f, err := os.CreateTemp("", "pmm-agent-test-reader-normal")
		require.NoError(t, err)
		files = append(files, f.Name())

		_, err = f.WriteString("0\n")
		require.NoError(t, err)

		r, err := NewContinuousFileReader(f.Name(), &testLogger{t})
		require.NoError(t, err)
		r.sleep = 50 * time.Millisecond
		assert.Equal(t, &ReaderMetrics{InputSize: 2, InputPos: 2}, r.Metrics())

		done := make(chan struct{})
		defer func() {
			_ = r.Close()
			<-done
		}()

		// use separate goroutine to give race detector more chances to spot problems
		lines := make(chan string, 10)
		go func() {
			for {
				line, e := r.NextLine()
				if e != nil {
					assert.Equal(t, io.EOF, e, "%s", e)
					close(lines)
					close(done)
					return
				}
				lines <- line
			}
		}()

		// test partial line read
		_, err = f.WriteString("1\n2\n3")
		require.NoError(t, err)
		assert.Equal(t, "1\n", <-lines)
		assert.Equal(t, "2\n", <-lines)
		assert.Empty(t, lines, "%d", len(lines))
		assert.Equal(t, &ReaderMetrics{InputSize: 7, InputPos: 7}, r.Metrics())

		// test rotation with rename
		newName := f.Name() + "-1"
		require.NoError(t, os.Rename(f.Name(), newName))
		files = append(files, newName)
		f, err = os.Create(f.Name())
		require.NoError(t, err)
		_, err = f.WriteString("\n4\n5")
		require.NoError(t, err)
		assert.Equal(t, "3\n", <-lines)
		assert.Equal(t, "4\n", <-lines)
		assert.Empty(t, lines, "%d", len(lines))
		assert.Equal(t, &ReaderMetrics{InputSize: 4, InputPos: 4}, r.Metrics())

		// test rotation with truncate
		f, err = os.Create(f.Name())
		require.NoError(t, err)
		_, err = f.WriteString("\n6\n") // new file should be smaller
		require.NoError(t, err)
		assert.Equal(t, "5\n", <-lines)
		assert.Equal(t, "6\n", <-lines)
		assert.Empty(t, lines, "%d", len(lines))
		assert.Equal(t, &ReaderMetrics{InputSize: 3, InputPos: 3}, r.Metrics())

		// test close
		_, err = f.WriteString("7\n8")
		require.NoError(t, err)
		assert.NoError(t, r.Close())
		l, ok := <-lines
		assert.False(t, ok, "line = %q", l)
		assert.Nil(t, r.Metrics())
	})

	t.Run("Symlink", func(t *testing.T) {
		t.Parallel()

		var files []string
		defer func() {
			cleanup(t, files)
		}()

		f, err := os.CreateTemp("", "pmm-agent-test-reader-symlink-file1")
		require.NoError(t, err)
		files = append(files, f.Name())

		symlink, err := os.CreateTemp("", "pmm-agent-test-reader-symlink")
		require.NoError(t, err)
		require.NoError(t, symlink.Close())
		symlinkName := symlink.Name()
		files = append(files, symlinkName)
		require.NoError(t, os.Remove(symlinkName))
		require.NoError(t, os.Symlink(f.Name(), symlinkName))

		_, err = f.WriteString("0\n")
		require.NoError(t, err)

		r, err := NewContinuousFileReader(symlinkName, &testLogger{t})
		require.NoError(t, err)
		r.sleep = 50 * time.Millisecond
		assert.Equal(t, &ReaderMetrics{InputSize: 2, InputPos: 2}, r.Metrics())

		done := make(chan struct{})
		defer func() {
			_ = r.Close()
			<-done
		}()

		// use separate goroutine to give race detector more chances to spot problems
		lines := make(chan string, 10)
		go func() {
			for {
				line, e := r.NextLine()
				if e != nil {
					assert.Equal(t, io.EOF, e, "%s", e)
					close(lines)
					close(done)
					return
				}
				lines <- line
			}
		}()

		// test partial line read
		_, err = f.WriteString("1\n2\n3")
		require.NoError(t, err)
		assert.Equal(t, "1\n", <-lines)
		assert.Equal(t, "2\n", <-lines)
		assert.Empty(t, lines, "%d", len(lines))
		assert.Equal(t, &ReaderMetrics{InputSize: 7, InputPos: 7}, r.Metrics())

		// test rotation with rename
		newName := f.Name() + "-1"
		require.NoError(t, os.Rename(f.Name(), newName))
		files = append(files, newName)
		f, err = os.Create(f.Name())
		require.NoError(t, err)
		_, err = f.WriteString("\n4\n5")
		require.NoError(t, err)
		assert.Equal(t, "3\n", <-lines)
		assert.Equal(t, "4\n", <-lines)
		assert.Empty(t, lines, "%d", len(lines))
		assert.Equal(t, &ReaderMetrics{InputSize: 4, InputPos: 4}, r.Metrics())

		// test rotation with truncate
		f, err = os.Create(f.Name())
		require.NoError(t, err)
		_, err = f.WriteString("\n6\n") // new file should be smaller
		require.NoError(t, err)
		assert.Equal(t, "5\n", <-lines)
		assert.Equal(t, "6\n", <-lines)
		assert.Empty(t, lines, "%d", len(lines))
		assert.Equal(t, &ReaderMetrics{InputSize: 3, InputPos: 3}, r.Metrics())

		// test symlink change
		f, err = os.CreateTemp("", "pmm-agent-test-reader-symlink-file2")
		require.NoError(t, err)
		files = append(files, f.Name())
		require.NoError(t, os.Remove(symlinkName))
		require.NoError(t, os.Symlink(f.Name(), symlinkName))
		_, err = f.WriteString("7\n8\n9")
		require.NoError(t, err)
		assert.Equal(t, "7\n", <-lines)
		assert.Equal(t, "8\n", <-lines)
		assert.Empty(t, lines, "%d", len(lines))
		assert.Equal(t, &ReaderMetrics{InputSize: 5, InputPos: 5}, r.Metrics())

		// test close
		_, err = f.WriteString("\n10\n")
		require.NoError(t, err)
		assert.NoError(t, r.Close())
		l, ok := <-lines
		assert.False(t, ok, "line = %q", l)
		assert.Nil(t, r.Metrics())
	})
}
