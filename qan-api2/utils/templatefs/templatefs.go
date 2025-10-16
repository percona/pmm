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
	"io"
	iofs "io/fs"
	"path/filepath"
	"text/template"
	"time"
)

// TemplateFS wraps an embed.FS and applies templating to file content during reads.
// It implements the fs.FS interface and delegates most operations to the underlying embed.FS,
// but applies Go text/template processing when reading file content via ReadFile.
type TemplateFS struct {
	// EmbedFS is the underlying embedded filesystem
	EmbedFS embed.FS
	// Data contains template data that will be used for all files
	Data map[string]any
}

// NewTemplateFS creates a new TemplateFS with the given embedded filesystem and template data.
func NewTemplateFS(embedFS embed.FS, data map[string]any) *TemplateFS {
	return &TemplateFS{
		EmbedFS: embedFS,
		Data:    data,
	}
}

// Open opens the named file for reading and returns a file with templated content.
func (tfs *TemplateFS) Open(name string) (iofs.File, error) {
	// Render the file content using the template logic
	content, err := tfs.ReadFile(name)
	if err != nil {
		return nil, err
	}
	// Return a file-like object from the rendered content
	return &templateFile{
		name:    name,
		content: content,
		offset:  0,
	}, nil
}

// templateFile implements iofs.File for a byte slice (rendered template content)
type templateFile struct {
	name    string
	content []byte
	offset  int64
}

func (f *templateFile) Stat() (iofs.FileInfo, error) {
	return &templateFileInfo{name: f.name, size: int64(len(f.content))}, nil
}

func (f *templateFile) Read(p []byte) (int, error) {
	if f.offset >= int64(len(f.content)) {
		return 0, io.EOF
	}
	n := copy(p, f.content[f.offset:])
	f.offset += int64(n)
	return n, nil
}

func (f *templateFile) Close() error { return nil }

// templateFileInfo implements iofs.FileInfo for templateFile
type templateFileInfo struct {
	name string
	size int64
}

func (fi *templateFileInfo) Name() string        { return fi.name }
func (fi *templateFileInfo) Size() int64         { return fi.size }
func (fi *templateFileInfo) Mode() iofs.FileMode { return 0o444 }
func (fi *templateFileInfo) ModTime() time.Time  { return time.Time{} }
func (fi *templateFileInfo) IsDir() bool         { return false }
func (fi *templateFileInfo) Sys() interface{}    { return nil }

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
