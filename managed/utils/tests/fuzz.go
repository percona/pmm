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

package tests

import (
	"crypto/sha1" //nolint:gosec
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

var corpusM sync.Mutex

// AddToFuzzCorpus adds data to go-fuzz corpus.
func AddToFuzzCorpus(tb testing.TB, prefix string, data []byte) {
	tb.Helper()
	corpusM.Lock()
	defer corpusM.Unlock()

	_, file, _, ok := runtime.Caller(1)
	require.True(tb, ok)
	dir := filepath.Join(filepath.Dir(file), "fuzzdata", "corpus")
	err := os.MkdirAll(dir, 0o750)
	require.NoError(tb, err)

	// go-fuzz uses SHA1 for non-cryptographic hashing
	file = fmt.Sprintf("%040x", sha1.Sum(data)) //nolint:gosec
	if prefix != "" {
		file = prefix + "-" + file
	}

	path := filepath.Join(dir, file)
	err = os.WriteFile(path, data, 0o640) //nolint:gosec
	require.NoError(tb, err)
}
