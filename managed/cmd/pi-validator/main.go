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

// Package main provides an entrypoint for the pi-validator tool.
package main

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/alecthomas/kong"

	"github.com/percona/pmm/managed/pi/alert"
	"github.com/percona/pmm/managed/pi/check"
)

type advisorsCommand struct {
	ChecksDir string `name:"checks.dir" help:"Checks directory" default:"data/checks"`
}

type templatesCommand struct {
	Dir string `name:"templates.dir" help:"alerting templates directory" default:"data/templates"`
}

func (a *advisorsCommand) Run() error {
	_, err := loadAndValidateChecks(a.ChecksDir)
	return err
}

func (t *templatesCommand) Run() error {
	return validateTemplates(t.Dir)
}

func main() {
	type CLI struct {
		Advisors  advisorsCommand  `cmd:"" help:"Validate advisors and checks"`
		Templates templatesCommand `cmd:"" help:"Validate alerting templates"`
	}
	cli := CLI{}
	kongCtx := kong.Parse(&cli)
	err := kongCtx.Run()
	kongCtx.FatalIfErrorf(err)
}

func loadAndValidateChecks(dir string) (map[string]check.Check, error) {
	patterns := []string{
		filepath.Join(dir, "*.yml"),
		filepath.Join(dir, "*.yml.example"),
	}

	var matches []string

	for _, pattern := range patterns {
		files, err := filepath.Glob(pattern)
		if err != nil {
			log.Printf("failed to find advisor files matching '%s': %+v", pattern, err)
		}
		matches = append(matches, files...)
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("no check files found in %s", dir)
	}

	var validationErrors []error
	res := make(map[string]check.Check, len(matches))
	for _, file := range matches {
		log.Print(file)
		_, fileName := filepath.Split(file)

		b, err := os.ReadFile(file) //nolint:gosec
		if err != nil {
			validationErrors = append(validationErrors, fmt.Errorf("failed to read check file %s: %w", fileName, err))
		}
		body := strings.TrimSpace(string(b))
		if !strings.HasPrefix(body, "---") {
			validationErrors = append(validationErrors, fmt.Errorf("file %s should start with '---' separator", fileName))
		}

		checks, err := check.ParseChecks(strings.NewReader(body), &check.ParseParams{
			DisallowUnknownFields: true,
			DisallowInvalidChecks: true,
		})
		if err != nil {
			validationErrors = append(validationErrors, fmt.Errorf("failed to parse checks file %s: %w", fileName, err))
		}

		if len(checks) != 1 {
			validationErrors = append(validationErrors, fmt.Errorf("expected exactly one check in %s", fileName))
			return nil, errors.Join(validationErrors...)
		}
		c := checks[0]

		if c.Name != strings.TrimSuffix(strings.TrimSuffix(fileName, ".example"), ".yml") {
			validationErrors = append(validationErrors, fmt.Errorf("check name does not match file name %s", file))
		}

		if _, ok := res[c.Name]; ok {
			validationErrors = append(validationErrors, fmt.Errorf("check name collision detected for: %s", c.Name))
		}

		res[c.Name] = c

		if len(validationErrors) != 0 {
			return nil, errors.Join(validationErrors...)
		}
	}

	return res, nil
}

func validateTemplates(dir string) error {
	patterns := []string{
		filepath.Join(dir, "*.yml"),
		filepath.Join(dir, "*.yml.example"),
	}

	var matches []string

	for _, pattern := range patterns {
		files, err := filepath.Glob(pattern)
		if err != nil {
			log.Printf("failed to find advisor files matching '%s': %+v", pattern, err)
		}
		matches = append(matches, files...)
	}
	if len(matches) == 0 {
		log.Printf("no template files found in %s", dir)
		return nil
	}
	matches, err := filepath.Glob(filepath.Join(dir, "*.yml"))
	if err != nil {
		log.Printf("failed to find tempatles files %+v", err)
		return nil
	}

	var validationErrors []error
	res := make(map[string]alert.Template, len(matches))
	for _, file := range matches {
		log.Print(file)

		b, err := os.ReadFile(file) //nolint:gosec
		if err != nil {
			validationErrors = append(validationErrors, fmt.Errorf("failed to read template file %s: %w", file, err))
		}
		templates, err := alert.Parse(bytes.NewReader(b), &alert.ParseParams{
			DisallowUnknownFields:    true,
			DisallowInvalidTemplates: true,
		})
		if err != nil {
			validationErrors = append(validationErrors, fmt.Errorf("failed to parse templates file %s: %w", file, err))
		}

		if len(templates) != 1 {
			validationErrors = append(validationErrors, fmt.Errorf("expected exactly one template in %s", file))
		}
		r := templates[0]

		_, fileName := filepath.Split(file)
		if r.Name != strings.TrimSuffix(fileName, ".yml") {
			validationErrors = append(validationErrors, fmt.Errorf("template name does not match file name %s", file))
		}

		if _, ok := res[r.Name]; ok {
			validationErrors = append(validationErrors, fmt.Errorf("template name collision detected for: %s", r.Name))
		}

		res[r.Name] = r

		if len(validationErrors) != 0 {
			return errors.Join(validationErrors...)
		}
	}

	templates, err := tableTemplates(res)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Print(templates) //nolint:forbidigo
	return nil
}

// tableTemplates returns a Confluence markup or Markdown table with templates.
func tableTemplates(_ map[string]alert.Template) (string, error) {
	// (ab)use tabwriter to generate Confluence markup or Markdown tableChecks
	var buf bytes.Buffer
	w := tabwriter.NewWriter(&buf, 0, 0, 1, ' ', tabwriter.Debug)
	_, _ = fmt.Fprintf(w, "\tName\tTiers\tDescription\t\n")

	_, _ = fmt.Fprintf(w, "\t----\t-----\t-----------\t\n")
	err := w.Flush()
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}
