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

package tests

import (
	// go-fuzz uses SHA1 for non-cryptographic hashing
	"crypto/sha1" //nolint:gosec
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

var corpusM sync.Mutex

// AddToFuzzCorpus adds data to go-fuzz corpus.
func AddToFuzzCorpus(t testing.TB, prefix string, data []byte) {
	corpusM.Lock()
	defer corpusM.Unlock()

	_, file, _, ok := runtime.Caller(1)
	require.True(t, ok)
	dir := filepath.Join(filepath.Dir(file), "fuzzdata", "corpus")
	err := os.MkdirAll(dir, 0o750)
	require.NoError(t, err)

	// go-fuzz uses SHA1 for non-cryptographic hashing
	file = fmt.Sprintf("%040x", sha1.Sum(data)) //nolint:gosec
	if prefix != "" {
		file = prefix + "-" + file
	}

	path := filepath.Join(dir, file)
	err = ioutil.WriteFile(path, data, 0o640) //nolint:gosec
	require.NoError(t, err)
}
