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

// Package templatefs provides a filesystem interface with templating support.
package templatefs

import (
	"bytes"
	"embed"
	iofs "io/fs"
	"path/filepath"
	"text/template"
)

// TemplateFS provides a filesystem interface with templating support.
// It wraps an embed.FS and applies Go text/template processing when reading file content via ReadFile.
type TemplateFS struct {
	// EmbedFS is the underlying embedded filesystem.
	EmbedFS embed.FS
	// Data contains template data that will be used for all files.
	Data map[string]any
}

// NewTemplateFS creates a new TemplateFS with the given embedded filesystem and template data.
func NewTemplateFS(embedFS embed.FS, data map[string]any) *TemplateFS {
	return &TemplateFS{
		EmbedFS: embedFS,
		Data:    data,
	}
}

// Open opens the named file for reading and returns the original iofs.File from embed.FS.
// No templating is applied here - use ReadFile for templated content.
func (tfs *TemplateFS) Open(name string) (iofs.File, error) {
	return tfs.EmbedFS.Open(name)
}

// ReadDir reads the named directory and returns a list of directory entries.
// This delegates directly to the underlying embed.FS.
func (tfs *TemplateFS) ReadDir(name string) ([]iofs.DirEntry, error) {
	return tfs.EmbedFS.ReadDir(name)
}

// ReadFile reads the named file and returns its content with templating applied.
// This is where the templating magic happens.
func (tfs *TemplateFS) ReadFile(name string) ([]byte, error) {
	// Read original content from embed.FS
	content, err := tfs.EmbedFS.ReadFile(name)
	if err != nil {
		return nil, err
	}

	// Apply templating using the same logic as in the user's example
	upSQL := string(content)

	// Extract just the filename from the path for template name
	filename := filepath.Base(name)

	// Apply template if data exists
	if tfs.Data != nil {
		if tmpl, err := template.New(filename).Parse(upSQL); err == nil {
			var buf bytes.Buffer
			if err := tmpl.Execute(&buf, tfs.Data); err == nil {
				upSQL = buf.String()
			}
		}
	}

	return []byte(upSQL), nil
}
