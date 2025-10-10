// Copyright (C) 2023 Percona LLC
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

package templatefs

import (
	"embed"
	"io"
	iofs "io/fs"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed testdata
var testFS embed.FS

func TestNewTemplateFS(t *testing.T) {
	data := map[string]any{
		"TableName":    "users",
		"DatabaseName": "testdb",
	}
	tfs := NewTemplateFS(testFS, data)
	assert.NotNil(t, tfs)
	assert.Equal(t, testFS, tfs.EmbedFS)
	assert.Equal(t, data, tfs.Data)
}

func TestTemplateFS_Open(t *testing.T) {
	tfs := NewTemplateFS(testFS, nil)
	file, err := tfs.Open("testdata/simple.sql")
	require.NoError(t, err)
	require.NotNil(t, file)
	defer file.Close()
	content, err := io.ReadAll(file)
	require.NoError(t, err)
	assert.Contains(t, string(content), "{{.TableName}}")
	_, err = tfs.Open("nonexistent.sql")
	assert.Error(t, err)
}

func TestTemplateFS_ReadFile_WithTemplating(t *testing.T) {
	data := map[string]any{
		"TableName":    "users",
		"DatabaseName": "testdb",
	}
	tfs := NewTemplateFS(testFS, data)
	content, err := tfs.ReadFile("testdata/simple.sql")
	require.NoError(t, err)
	contentStr := string(content)
	assert.Contains(t, contentStr, "users")
	assert.Contains(t, contentStr, "testdb")
	assert.NotContains(t, contentStr, "{{.TableName}}")
	assert.NotContains(t, contentStr, "{{.DatabaseName}}")
}

func TestTemplateFS_ReadFile_WithoutTemplateData(t *testing.T) {
	tfs := NewTemplateFS(testFS, nil)
	content, err := tfs.ReadFile("testdata/simple.sql")
	require.NoError(t, err)
	contentStr := string(content)
	assert.Contains(t, contentStr, "{{.TableName}}")
	assert.Contains(t, contentStr, "{{.DatabaseName}}")
}

func TestTemplateFS_ReadFile_WithEmptyTemplateData(t *testing.T) {
	data := map[string]any{}
	tfs := NewTemplateFS(testFS, data)
	content, err := tfs.ReadFile("testdata/simple.sql")
	require.NoError(t, err)
	contentStr := string(content)
	assert.NotContains(t, contentStr, "{{.TableName}}")
	assert.NotContains(t, contentStr, "{{.DatabaseName}}")
}

func TestTemplateFS_ReadFile_InvalidTemplate(t *testing.T) {
	data := map[string]any{
		"TableName": "users",
	}
	tfs := NewTemplateFS(testFS, data)
	content, err := tfs.ReadFile("testdata/invalid.sql")
	require.NoError(t, err)
	contentStr := string(content)
	assert.Contains(t, contentStr, "{{.TableName")
}

func TestTemplateFS_ReadFile_NonexistentFile(t *testing.T) {
	tfs := NewTemplateFS(testFS, nil)
	_, err := tfs.ReadFile("nonexistent.sql")
	assert.Error(t, err)
}

func TestTemplateFS_ReadDir(t *testing.T) {
	tfs := NewTemplateFS(testFS, nil)
	entries, err := tfs.ReadDir("testdata")
	require.NoError(t, err)
	assert.NotEmpty(t, entries)
	var names []string
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	assert.Contains(t, names, "simple.sql")
}

func TestTemplateFS_ReadDir_NonexistentDir(t *testing.T) {
	tfs := NewTemplateFS(testFS, nil)
	_, err := tfs.ReadDir("nonexistent")
	assert.Error(t, err)
}

func TestTemplateFS_WithStandardLibraryFunctions(t *testing.T) {
	data := map[string]any{
		"TableName":    "products",
		"DatabaseName": "shop",
	}
	tfs := NewTemplateFS(testFS, data)
	subFS, err := iofs.Sub(tfs, "testdata")
	require.NoError(t, err)
	content, err := iofs.ReadFile(subFS, "simple.sql")
	require.NoError(t, err)
	contentStr := string(content)
	assert.Contains(t, contentStr, "products")
	assert.Contains(t, contentStr, "shop")
	matches, err := iofs.Glob(tfs, "testdata/*.sql")
	require.NoError(t, err)
	assert.NotEmpty(t, matches)
	assert.Contains(t, matches, "testdata/simple.sql")
}

func TestTemplateFS_FilenameExtraction(t *testing.T) {
	data := map[string]any{
		"TableName": "extracted",
	}
	tfs := NewTemplateFS(testFS, data)
	content, err := tfs.ReadFile("testdata/simple.sql")
	require.NoError(t, err)
	contentStr := string(content)
	assert.Contains(t, contentStr, "extracted")
}

func TestTemplateFS_ConditionalTemplating(t *testing.T) {
	data := map[string]any{
		"TableName":  "users",
		"AddIndexes": true,
		"IndexName":  "idx_users_email",
		"ColumnName": "email",
	}
	tfs := NewTemplateFS(testFS, data)
	content, err := tfs.ReadFile("testdata/conditional.sql")
	require.NoError(t, err)
	contentStr := string(content)
	assert.Contains(t, contentStr, "CREATE TABLE users")
	assert.Contains(t, contentStr, "CREATE INDEX idx_users_email")
	assert.Contains(t, contentStr, "ON users (email)")
}

func TestTemplateFS_ConditionalTemplating_False(t *testing.T) {
	data := map[string]any{
		"TableName":  "users",
		"AddIndexes": false,
	}
	tfs := NewTemplateFS(testFS, data)
	content, err := tfs.ReadFile("testdata/conditional.sql")
	require.NoError(t, err)
	contentStr := string(content)
	assert.Contains(t, contentStr, "CREATE TABLE users")
	assert.NotContains(t, contentStr, "CREATE INDEX")
}
