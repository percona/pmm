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

package templates

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/agent/utils/secrets"
	agentv1 "github.com/percona/pmm/api/agent/v1"
	"github.com/percona/pmm/utils/vault"
)

const (
	certificateFilePlaceholder    = "certificateFilePlaceholder"
	certificateKeyFilePlaceholder = "certificateKeyFilePlaceholder"
	caFilePlaceholder             = "caFilePlaceholder"
)

func TestRenderDSN(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(os.TempDir(), fmt.Sprintf("pg_action_%05d", rand.Int63n(99999)))
	err := os.MkdirAll(dir, 0o750)
	require.NoError(t, err)

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
	require.NoError(t, err)
	assert.Equal(t, wantDSN, outDSN)

	assert.True(t, fileExist(filepath.Join(dir, caFilePlaceholder)))
	assert.True(t, fileExist(filepath.Join(dir, certificateFilePlaceholder)))
	assert.True(t, fileExist(filepath.Join(dir, certificateKeyFilePlaceholder)))

	assert.True(t, fileContentMatch(filepath.Join(dir, caFilePlaceholder), files.Files[caFilePlaceholder]))
	assert.True(t, fileContentMatch(filepath.Join(dir, certificateFilePlaceholder), files.Files[certificateFilePlaceholder]))
	assert.True(t, fileContentMatch(filepath.Join(dir, certificateKeyFilePlaceholder), files.Files[certificateKeyFilePlaceholder]))

	// Cleanup
	err = os.RemoveAll(dir)
	require.NoError(t, err)
}

func fileExist(file string) bool {
	_, err := os.Stat(file)
	return err == nil
}

func fileContentMatch(file, content string) bool {
	fileContent, err := os.ReadFile(filepath.Clean(file))
	if err != nil {
		return false
	}

	return content == string(fileContent)
}

type mapGetter map[string]string

func (m mapGetter) Get(_ context.Context, path, key string) (string, error) {
	v, ok := m[path+"#"+key]
	if !ok {
		return "", errors.New("no such secret")
	}
	return v, nil
}

// Not parallel: it configures the process-wide Vault client.
func TestRenderVaultTokens(t *testing.T) {
	// The password holds the very delimiters the template uses, which pmm-managed could not avoid
	// without knowing it: it must still come out intact, because it is inserted after rendering.
	secrets.Configure(mapGetter{"kv/mysql#password": "pa{{ss}}"})
	t.Cleanup(func() { secrets.Configure(nil) })

	token := vault.Token("kv/mysql", "password", vault.EncodingRaw)

	t.Run("Files", func(t *testing.T) {
		tr := &TemplateRenderer{
			TextFiles:          map[string]string{"myCnf": "[client]\nuser=pmm\npassword=" + token + "\n"},
			TemplateLeftDelim:  "{{",
			TemplateRightDelim: "}}",
			TempDir:            t.TempDir(),
		}

		params, err := tr.RenderFiles(map[string]any{})
		require.NoError(t, err)

		arg, err := tr.RenderTemplate("args", "--config.my-cnf={{ .TextFiles.myCnf }}", params)
		require.NoError(t, err)
		path := filepath.Join(tr.TempDir, "myCnf")
		assert.Equal(t, "--config.my-cnf="+path, string(arg))

		b, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Equal(t, "[client]\nuser=pmm\npassword=pa{{ss}}\n", string(b))
		assert.Equal(t, []string{"pa{{ss}}"}, tr.SecretValues())
	})

	t.Run("DSNWithoutFiles", func(t *testing.T) {
		dsn, err := RenderDSN("pmm:"+token+"@tcp(127.0.0.1:3306)/", nil, t.TempDir())
		require.NoError(t, err)
		assert.Equal(t, "pmm:pa{{ss}}@tcp(127.0.0.1:3306)/", dsn)
	})

	t.Run("DSNWithFiles", func(t *testing.T) {
		files := &agentv1.TextFiles{
			Files:              map[string]string{"caFilePlaceholder": "ca"},
			TemplateLeftDelim:  "{{",
			TemplateRightDelim: "}}",
		}
		dir := t.TempDir()
		dsn, err := RenderDSN("pmm:"+token+"@tcp(127.0.0.1:3306)/?ca={{ .TextFiles.caFilePlaceholder }}", files, dir)
		require.NoError(t, err)
		assert.Equal(t, "pmm:pa{{ss}}@tcp(127.0.0.1:3306)/?ca="+filepath.Join(dir, "caFilePlaceholder"), dsn)
	})

	t.Run("MissingSecret", func(t *testing.T) {
		_, err := RenderDSN(vault.Token("kv/mysql", "nope", vault.EncodingRaw), nil, t.TempDir())
		require.ErrorIs(t, err, secrets.ErrUnavailable)
	})
}
