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

// Package templatefs provides an embedded filesystem with templating capabilities.
package templatefs

import (
	"bytes"
	"embed"
	"fmt"
	"path/filepath"
	"text/template"
)

// TemplateFS wraps an embed.FS and applies Go text/template processing to file
// content when reading via ReadFile.
type TemplateFS struct {
	EmbedFS embed.FS
	Data    map[string]any
	dir     string
}

// NewTemplateFS creates a new TemplateFS with the given embedded filesystem and template data.
func NewTemplateFS(embedFS embed.FS, data map[string]any, dir string) *TemplateFS {
	return &TemplateFS{
		EmbedFS: embedFS,
		Data:    data,
		dir:     dir,
	}
}

// ReadFile reads the named file and returns its content with templating applied.
func (tfs *TemplateFS) ReadFile(name string) ([]byte, error) {
	fullName := filepath.Join(tfs.dir, name)
	content, err := tfs.EmbedFS.ReadFile(fullName)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", fullName, err)
	}

	sql := string(content)
	if tfs.Data != nil {
		tmpl, err := template.New(name).Parse(sql)
		if err == nil {
			var buf bytes.Buffer
			err = tmpl.Execute(&buf, tfs.Data)
			if err == nil {
				sql = buf.String()
			}
		}
	}

	return []byte(sql), nil
}

// Names returns the file names in the directory managed by TemplateFS.
func (tfs *TemplateFS) Names() ([]string, error) {
	dir, err := tfs.EmbedFS.ReadDir(tfs.dir)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(dir))
	for _, entry := range dir {
		names = append(names, entry.Name())
	}
	return names, nil
}
