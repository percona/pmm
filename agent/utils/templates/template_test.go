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

// Package templates contains all logic related to template rendering.
package templates

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	agentv1 "github.com/percona/pmm/api/agent/v1"
)

const (
	certificateFilePlaceholder    = "certificateFilePlaceholder"
	certificateKeyFilePlaceholder = "certificateKeyFilePlaceholder"
	caFilePlaceholder             = "caFilePlaceholder"
)

func TestRenderDSN(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(os.TempDir(), fmt.Sprintf("pg_action_%05d", rand.Int63n(99999))) //nolint:gosec
	err := os.MkdirAll(dir, 0o750)
	assert.NoError(t, err)

	inDSN := "postgres://username:s3cur3%20p%40$$w0r4.@1.2.3.4:12345/database" +
		"?connect_timeout=1&ssl_ca_file={{.TextFiles.caFilePlaceholder}}&" +
		"ssl_cert_file={{.TextFiles.certificateFilePlaceholder}}&" +
		"ssl_key_file={{.TextFiles.certificateKeyFilePlaceholder}}&" +
		"sslmode=verify-full"
	wantDSN := "postgres://username:s3cur3%20p%40$$w0r4.@1.2.3.4:12345/database" +
		"?connect_timeout=1&ssl_ca_file=" + dir + "/caFilePlaceholder&" +
		"ssl_cert_file=" + dir + "/certificateFilePlaceholder&" +
		"ssl_key_file=" + dir + "/certificateKeyFilePlaceholder&" +
		"sslmode=verify-full"

	files := &agentv1.TextFiles{
		Files: map[string]string{
			certificateFilePlaceholder:    "== this is a mock cer-file content ABCDEF000000 ==",
			certificateKeyFilePlaceholder: "== this is a mock key-file content ABCDEF000000 ==",
			caFilePlaceholder:             "== this is a mock ca-file content ABCDEF000000 ==",
		},
		TemplateLeftDelim:  "{{",
		TemplateRightDelim: "}}",
	}

	outDSN, err := RenderDSN(inDSN, files, dir)
	assert.NoError(t, err)
	assert.Equal(t, wantDSN, outDSN)

	assert.True(t, fileExist(filepath.Join(dir, caFilePlaceholder)))
	assert.True(t, fileExist(filepath.Join(dir, certificateFilePlaceholder)))
	assert.True(t, fileExist(filepath.Join(dir, certificateKeyFilePlaceholder)))

	assert.True(t, fileContentMatch(filepath.Join(dir, caFilePlaceholder), files.Files[caFilePlaceholder]))
	assert.True(t, fileContentMatch(filepath.Join(dir, certificateFilePlaceholder), files.Files[certificateFilePlaceholder]))
	assert.True(t, fileContentMatch(filepath.Join(dir, certificateKeyFilePlaceholder), files.Files[certificateKeyFilePlaceholder]))

	// Cleanup
	err = os.RemoveAll(dir)
	assert.NoError(t, err)
}

func fileExist(file string) bool {
	if _, err := os.Stat(file); err == nil {
		return true
	}
	return false
}

func fileContentMatch(file, content string) bool {
	fileContent, err := os.ReadFile(filepath.Clean(file))
	if err != nil {
		return false
	}

	return content == string(fileContent)
}
