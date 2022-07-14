// Copyright 2019 Percona LLC
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

package commands

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/admin/logger"
)

func init() {
	logrus.SetFormatter(&logger.TextFormatter{})
}

func CreateDummyCredentialsSource(data string, p string, exec bool) (string, error) {
	tmpFile, err := os.CreateTemp(os.TempDir(), "CreateDummyCredentialsSource.*"+p)
	if err != nil {
		return "", fmt.Errorf("%w", err)
	}

	defer func() {
		if tmpErr := tmpFile.Close(); tmpErr != nil {
			err = tmpErr
		}
	}()

	content := []byte(data)
	if _, err := tmpFile.Write(content); err != nil {
		return "", fmt.Errorf("%w", err)
	}

	if exec {
		if err := tmpFile.Chmod(0o111); err != nil {
			return "", fmt.Errorf("%w", err)
		}
	}

	return tmpFile.Name(), err
}

func CreateDummyCredentialsExecutable(d string) (string, error) {
	credSource, err := CreateDummyCredentialsSource(`
#!/bin/sh

echo `+d, "sh", true)
	if err != nil {
		return "", err
	}

	return credSource, nil
}

func TestCredentials(t *testing.T) {
	t.Parallel()

	data := `{"username": "testuser", "password": "testpass", "agentpassword": "testagentpass"}`
	credSource, _ := CreateDummyCredentialsSource(data, "json", false)
	credSourceX, _ := CreateDummyCredentialsExecutable(data)

	t.Cleanup(func() {
		assert.NoError(t, os.Remove(credSource))
		assert.NoError(t, os.Remove(credSourceX))
	})

	t.Run("Reading", func(t *testing.T) {
		// Test reading is OK
		t.Parallel()
		creds, _ := ReadFromSource(credSource)
		assert.Equal(t, creds.Username, "testuser")
	})

	t.Run("Executing", func(t *testing.T) {
		// Ensure that execution currently errors
		t.Parallel()
		_, err := ReadFromSource(credSourceX)
		require.Error(t, err)
	})
}

func TestParseRenderTemplate(t *testing.T) {
	t.Parallel()

	var stderr bytes.Buffer
	logrus.SetOutput(&stderr)
	defer logrus.SetOutput(os.Stderr)

	tmpl := ParseTemplate(`{{ .Missing }}`)
	data := map[string]string{
		"foo": "bar",
	}

	assert.Panics(t, func() { RenderTemplate(tmpl, data) })

	expected := strings.TrimSpace(`
Failed to render response.
template: :1:3: executing "" at <.Missing>: map has no entry for key "Missing".
Template data: map[string]string{"foo":"bar"}.
Please report this bug.
	`) + "\n"
	assert.Equal(t, expected, stderr.String())
}

func TestParseCustomLabel(t *testing.T) {
	t.Parallel()
	errWrongFormat := "wrong custom label format"
	for _, tt := range []struct {
		name     string
		input    string
		expected map[string]string
		expErr   string
	}{
		{"simple label", "foo=bar", map[string]string{"foo": "bar"}, ""},
		{"two labels", "foo=bar,bar=foo", map[string]string{"foo": "bar", "bar": "foo"}, ""},
		{"no value", "foo=", nil, errWrongFormat},
		{"no key", "=foo", nil, errWrongFormat},
		{"wrong format", "foo=bar,bar+foo", nil, errWrongFormat},
		{"empty value", "", make(map[string]string), ""},
		{"PMM-4078 hyphen", "region=us-east1, mylabel=mylab-22", map[string]string{"region": "us-east1", "mylabel": "mylab-22"}, ""},
	} {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			customLabels, err := ParseCustomLabels(tt.input)
			assert.Equal(t, tt.expected, customLabels)
			if tt.expErr != "" {
				assert.EqualError(t, err, tt.expErr)
			}
		})
	}
}

func TestReadFile(t *testing.T) {
	t.Parallel()

	t.Run("Normal", func(t *testing.T) {
		t.Parallel()

		cert, err := os.CreateTemp("", "cert")
		require.NoError(t, err)
		defer func() {
			err = cert.Close()
			assert.NoError(t, err)
			err = os.Remove(cert.Name())
			assert.NoError(t, err)
		}()
		_, err = cert.Write([]byte("cert"))
		require.NoError(t, err)

		certificate, err := ReadFile(cert.Name())
		assert.NoError(t, err)
		assert.Equal(t, "cert", certificate)
	})

	t.Run("WrongPath", func(t *testing.T) {
		t.Parallel()

		filepath := "not-existed-file"
		certificate, err := ReadFile(filepath)
		assert.EqualError(t, err, fmt.Sprintf("cannot load file in path %q: open not-existed-file: no such file or directory", filepath))
		assert.Empty(t, certificate)
	})

	t.Run("EmptyFilePath", func(t *testing.T) {
		t.Parallel()

		certificate, err := ReadFile("")
		require.NoError(t, err)
		require.Empty(t, certificate)
	})
}
