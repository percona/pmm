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
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"text/template"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	agentv1 "github.com/percona/pmm/api/agent/v1"
)

// "_" at the begginging is reserved for possible extensions.
var textFileRE = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*$`) //nolint:gochecknoglobals

// TemplateRenderer replaces creates files and replaces placeholders for files in text.
type TemplateRenderer struct {
	TextFiles          map[string]string
	TemplateLeftDelim  string
	TemplateRightDelim string
	TempDir            string
}

// RenderTemplate replaces placeholders with real values in text.
func (tr *TemplateRenderer) RenderTemplate(name, text string, templateParams map[string]interface{}) ([]byte, error) {
	t := template.New(name)
	t.Delims(tr.TemplateLeftDelim, tr.TemplateRightDelim)
	t.Option("missingkey=error")

	var buf bytes.Buffer
	if _, err := t.Parse(text); err != nil {
		return nil, errors.WithStack(err)
	}
	if err := t.Execute(&buf, templateParams); err != nil {
		return nil, errors.WithStack(err)
	}
	return buf.Bytes(), nil
}

// RenderFiles creates temporary files and returns paths to created files.
func (tr *TemplateRenderer) RenderFiles(templateParams map[string]interface{}) (map[string]interface{}, error) {
	// render files only if they are present to avoid creating temporary directory for every agent
	if len(tr.TextFiles) == 0 {
		return templateParams, nil
	}

	if err := os.RemoveAll(tr.TempDir); err != nil {
		return nil, errors.WithStack(err)
	}
	if err := os.MkdirAll(tr.TempDir, 0o700); err != nil {
		return nil, errors.WithStack(err)
	}

	textFiles := make(map[string]string, len(tr.TextFiles)) // template name => full file path
	for name, text := range tr.TextFiles {
		// avoid /, .., ., \, and other special symbols
		if !textFileRE.MatchString(name) {
			return nil, errors.Errorf("invalid text file name %q", name)
		}

		b, err := tr.RenderTemplate(name, text, templateParams)
		if err != nil {
			return nil, err
		}

		path := filepath.Join(tr.TempDir, name)
		if err = os.WriteFile(path, b, 0o600); err != nil {
			return nil, errors.WithStack(err)
		}
		textFiles[name] = path
	}
	templateParams["TextFiles"] = textFiles
	return templateParams, nil
}

// RenderDSN creates temporary files and replaces placeholders with real paths.
func RenderDSN(dsn string, files *agentv1.TextFiles, tempDir string) (string, error) {
	if files != nil {
		tr := &TemplateRenderer{
			TextFiles:          files.Files,
			TemplateLeftDelim:  files.TemplateLeftDelim,
			TemplateRightDelim: files.TemplateRightDelim,
			TempDir:            tempDir,
		}

		templateParams, err := tr.RenderFiles(make(map[string]interface{}))
		if err != nil {
			return "", err
		}

		b, err := tr.RenderTemplate("dsn", dsn, templateParams)
		if err != nil {
			return "", err
		}
		dsn = string(b)
	}
	return dsn, nil
}

// CleanupTempDir removes the temporary directory.
func CleanupTempDir(tempDir string, logger *logrus.Entry) {
	if err := os.RemoveAll(tempDir); err != nil {
		if logger != nil {
			logger.Debugf("failed to remove the temporary directory: %s", err)
		}
	}
}
