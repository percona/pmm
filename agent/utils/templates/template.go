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
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sync"
	"text/template"

	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/agent/utils/secrets"
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

	m            sync.Mutex
	secretValues []string
}

// RenderTemplate replaces placeholders with real values in text, and then the Vault tokens in the result
// with the secrets they stand for. Resolving after rendering keeps the secrets out of the template, where
// they could clash with delimiters that pmm-managed chose without knowing them.
func (tr *TemplateRenderer) RenderTemplate(name, text string, templateParams map[string]any) ([]byte, error) {
	t := template.New(name)
	t.Delims(tr.TemplateLeftDelim, tr.TemplateRightDelim)
	t.Option("missingkey=error")

	var buf bytes.Buffer
	_, err := t.Parse(text)
	if err != nil {
		return nil, err
	}
	err = t.Execute(&buf, templateParams)
	if err != nil {
		return nil, err
	}

	res, values, err := secrets.Resolve(context.Background(), buf.String())
	if err != nil {
		return nil, err
	}

	tr.m.Lock()
	tr.secretValues = append(tr.secretValues, values...)
	tr.m.Unlock()

	return []byte(res), nil
}

// SecretValues returns the Vault secrets inserted by RenderTemplate so far, for redacting them from logs.
func (tr *TemplateRenderer) SecretValues() []string {
	tr.m.Lock()
	defer tr.m.Unlock()

	return slices.Clone(tr.secretValues)
}

// RenderFiles creates temporary files and returns paths to created files.
func (tr *TemplateRenderer) RenderFiles(templateParams map[string]any) (map[string]any, error) {
	// render files only if they are present to avoid creating temporary directory for every agent
	if len(tr.TextFiles) == 0 {
		return templateParams, nil
	}

	err := os.RemoveAll(tr.TempDir)
	if err != nil {
		return nil, err
	}
	err = os.MkdirAll(tr.TempDir, 0o700) //nolint:mnd
	if err != nil {
		return nil, err
	}

	textFiles := make(map[string]string, len(tr.TextFiles)) // template name => full file path

	for name := range tr.TextFiles {
		// avoid /, .., ., \, and other special symbols
		if !textFileRE.MatchString(name) {
			return nil, fmt.Errorf("invalid text file name %q", name)
		}

		path := filepath.Join(tr.TempDir, name)
		textFiles[name] = path
	}
	templateParams["TextFiles"] = textFiles

	for name, text := range tr.TextFiles {
		b, err := tr.RenderTemplate(name, text, templateParams)
		if err != nil {
			return nil, err
		}

		path := filepath.Join(tr.TempDir, name)
		err = os.WriteFile(path, b, 0o600) //nolint:mnd
		if err != nil {
			return nil, err
		}
	}
	return templateParams, nil
}

// RenderDSN creates temporary files, replaces placeholders with real paths, and Vault tokens with secrets.
func RenderDSN(dsn string, files *agentv1.TextFiles, tempDir string) (string, error) {
	if files == nil {
		return secrets.ResolveString(context.Background(), dsn)
	}

	tr := &TemplateRenderer{
		TextFiles:          files.Files,
		TemplateLeftDelim:  files.TemplateLeftDelim,
		TemplateRightDelim: files.TemplateRightDelim,
		TempDir:            tempDir,
	}

	templateParams, err := tr.RenderFiles(make(map[string]any))
	if err != nil {
		return "", err
	}

	b, err := tr.RenderTemplate("dsn", dsn, templateParams)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// CleanupTempDir removes the temporary directory.
func CleanupTempDir(tempDir string, logger *logrus.Entry) {
	err := os.RemoveAll(tempDir)
	if err != nil {
		if logger != nil {
			logger.Debugf("failed to remove the temporary directory: %s", err)
		}
	}
}
