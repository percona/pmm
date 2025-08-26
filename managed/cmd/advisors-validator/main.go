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

package main

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/alecthomas/kong"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/percona/saas/pkg/alert"
	"github.com/percona/saas/pkg/check"
)

type advisorsCommand struct {
	AdvisorsDir string `name:"advisors.dir" help:"Advisors directory" default:"data/advisors"`
	ChecksDir   string `name:"checks.dir" help:"Checks directory" default:"data/checks"`
}

type templatesCommand struct {
	Dir string `name:"templates.dir" help:"alerting templates directory" default:"data/templates"`
}

func (a *advisorsCommand) Run() error {
	return validateAdvisorsAndChecks(a.AdvisorsDir, a.ChecksDir)
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

func validateAdvisorsAndChecks(advisorsDir, checksDir string) error {
	advisors, err := loadAndValidateAdvisors(advisorsDir)
	if err != nil {
		return err
	}
	checks, err := loadAndValidateChecks(checksDir)

	for _, c := range checks {
		a, ok := advisors[c.Advisor]
		if !ok {
			log.Fatalf("check '%s' refers unknown advisor '%s'", c.Name, c.Advisor)
		}

		a.Checks = append(a.Checks, c)
	}
	return nil
}

func loadAndValidateAdvisors(dir string) (map[string]*check.Advisor, error) {
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
		log.Printf("no advisor files found in %s", dir)
		return nil, nil
	}

	res := make(map[string]*check.Advisor, len(matches))
	for _, file := range matches {
		log.Printf("Loading advisor file: %s", file)
		_, fileName := filepath.Split(file)

		var validationErrors []error
		b, err := os.ReadFile(file) //nolint:gosec
		if err != nil {
			validationErrors = append(validationErrors, fmt.Errorf("failed to read check file %s: %+v", fileName, err))
		}
		body := strings.TrimSpace(string(b))
		if !strings.HasPrefix(body, "---") {
			validationErrors = append(validationErrors, fmt.Errorf("file %s should start with '---' separator", fileName))
		}

		if len(validationErrors) != 0 {
			return nil, errors.Join(validationErrors...)
		}
		advisors, err := check.ParseAdvisors(strings.NewReader(body), &check.ParseParams{
			DisallowUnknownFields: true,
			DisallowInvalidChecks: true,
		})
		if err != nil {
			validationErrors = append(validationErrors, fmt.Errorf("failed to parse advisors file %s: %+v", fileName, err))
		}

		if len(advisors) != 1 {
			validationErrors = append(validationErrors, fmt.Errorf("expected exactly one advisor in %s", fileName))
		}
		a := advisors[0]

		if a.Name != strings.TrimSuffix(strings.TrimSuffix(fileName, ".example"), ".yml") {
			validationErrors = append(validationErrors, fmt.Errorf("advisor name does not match file name %s", file))
		}

		if _, ok := res[a.Name]; ok {
			validationErrors = append(validationErrors, fmt.Errorf("advisor name collision detected for: %s", a.Name))
		}

		res[a.Name] = &a
		if len(validationErrors) != 0 {
			return nil, errors.Join(validationErrors...)
		}
	}

	return res, nil
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
		log.Printf("no check files found in %s", dir)
		return nil, nil
	}

	var validationErrors []error
	res := make(map[string]check.Check, len(matches))
	for _, file := range matches {
		log.Print(file)
		_, fileName := filepath.Split(file)

		b, err := os.ReadFile(file) //nolint:gosec
		if err != nil {
			validationErrors = append(validationErrors, fmt.Errorf("failed to read check file %s: %+v", fileName, err))
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
			validationErrors = append(validationErrors, fmt.Errorf("failed to parse checks file %s: %+v", fileName, err))
		}

		if len(checks) != 1 {
			validationErrors = append(validationErrors, fmt.Errorf("expected exactly one check in %s", fileName))
			log.Fatalf("expected exactly one check in %s.", file)
		}
		c := checks[0]

		if c.Name != strings.TrimSuffix(strings.TrimSuffix(fileName, ".example"), ".yml") {
			validationErrors = append(validationErrors, fmt.Errorf("check name does not match file name %s", file))
			log.Fatalf("check name does not match file name %s.", file)
		}

		if _, ok := res[c.Name]; ok {
			validationErrors = append(validationErrors, fmt.Errorf("check name collision detected for: %s", c.Name))
			log.Fatalf("check name collision detected for: %s", c.Name)
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
			validationErrors = append(validationErrors, fmt.Errorf("failed to read template file %s: %+v", file, err))
		}
		templates, err := alert.Parse(bytes.NewReader(b), &alert.ParseParams{
			DisallowUnknownFields:    true,
			DisallowInvalidTemplates: true,
		})
		if err != nil {
			validationErrors = append(validationErrors, fmt.Errorf("failed to parse templates file %s: %+v", file, err))
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

	templates, err := tableTemplates(res, markdown)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Print(templates) //nolint:forbidigo
	return nil
}

type flavor int

const (
	markdown flavor = iota
)

// tableTemplates returns a Confluence markup or Markdown table with templates.
func tableTemplates(templates map[string]alert.Template, flavor flavor) (string, error) {
	// (ab)use tabwriter to generate Confluence markup or Markdown tableChecks
	var buf bytes.Buffer
	w := tabwriter.NewWriter(&buf, 0, 0, 1, ' ', tabwriter.Debug)
	_, _ = fmt.Fprintf(w, "\tName\tTiers\tDescription\t\n")

	// that's the only thing that should be skipped for "Confluence wiki" markup to work
	if flavor == markdown {
		_, _ = fmt.Fprintf(w, "\t----\t-----\t-----------\t\n")
	}

	for _, template := range templates {
		tiers := make([]string, len(template.Tiers))
		for i, t := range template.Tiers {
			tiers[i] = string(t)
		}

		_, _ = fmt.Fprintf(w, "\t%s\t%s\t%s\t\n", template.Name, strings.Join(tiers, ", "), template.Summary)
	}

	if err := w.Flush(); err != nil {
		return "", err
	}
	return buf.String(), nil
}
