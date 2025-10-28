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
	"bytes"
	"embed"
	"fmt"
	"path/filepath"
	"text/template"
)

// TemplateFS wraps an embed.FS and applies templating to file content during reads.
// It implements the fs.FS interface and delegates most operations to the underlying embed.FS,
// but applies Go text/template processing when reading file content via ReadFile.
type TemplateFS struct {
	// EmbedFS is the underlying embedded filesystem
	EmbedFS embed.FS
	// Data contains template data that will be used for all files
	Data map[string]any
	dir  string
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
// This is where the templating magic happens.
func (tfs *TemplateFS) ReadFile(name string) ([]byte, error) {
	// Read original content from embed.FS
	fullName := filepath.Join(tfs.dir, name)
	content, err := tfs.EmbedFS.ReadFile(fullName)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", fullName, err)
	}

	// Apply templating using the same logic as in the user's example
	upSQL := string(content)

	// Apply template if data exists
	if tfs.Data != nil {
		if tmpl, err := template.New(name).Parse(upSQL); err == nil {
			var buf bytes.Buffer
			if err := tmpl.Execute(&buf, tfs.Data); err == nil {
				upSQL = buf.String()
			}
		}
	}

	return []byte(upSQL), nil
}

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
