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

// Package telemetry provides telemetry functionality.
package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/alecthomas/kingpin/v2"
	"github.com/percona/saas/pkg/alert"
	"github.com/percona/saas/pkg/check"
)

func main() {
	app := kingpin.New("advisors-validator", "Validates Advisors and Percona alerting template files, creates descriptions in markdown format")
	markdownF := app.Flag("markdown", "Output a list of checks in Markdown format").Default("false").Bool()

	advisors := app.Command("advisors", "Validates advisors")
	advisorsDirF := advisors.Flag("advisors.dir", "Advisors directory").Default("data/advisors").String()
	checksDirF := advisors.Flag("checks.dir", "Checks directory").Default("data/checks").String()

	templates := app.Command("templates", "Validates templates")
	templatesDirF := templates.Flag("templates.dir", "IA templates directory").Default("data/templates").String()

	switch kingpin.MustParse(app.Parse(os.Args[1:])) {
	case advisors.FullCommand():
		validateAdvisorsAndChecks(*advisorsDirF, *checksDirF, *markdownF)
	case templates.FullCommand():
		validateTemplates(*templatesDirF, *markdownF)
	}
}

func validateAdvisorsAndChecks(advisorsDir, checksDir string, markdownT bool) { //nolint:unparam,revive
	advisors := loadAndValidateAdvisors(advisorsDir)
	checks := loadAndValidateChecks(checksDir)

	for _, c := range checks {
		a, ok := advisors[c.Advisor]
		if !ok {
			log.Fatalf("check '%s' refers unknown advisor '%s'", c.Name, c.Advisor)
		}

		a.Checks = append(a.Checks, c)
	}

	// TODO: @artemgavrilov Fix table functions
	// // Always call this function for extra validation side-effects.
	// //nolint:godox
	// // TODO Move that validation to some better place.
	//
	// if markdownT {
	// 	res, err := tableChecks(checks, markdown)
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}
	// 	fmt.Print(res) //nolint:forbidigo
	// }
}

func loadAndValidateAdvisors(dir string) map[string]*check.Advisor {
	matches, err := filepath.Glob(filepath.Join(dir, "*")) // not "*.yml" to cover examples
	if err != nil {
		log.Fatalf("failed to find advisor files %+v", err)
	}

	res := make(map[string]*check.Advisor, len(matches))
	for _, file := range matches {
		log.Printf("Loading advisor file: %s", file)
		_, fileName := filepath.Split(file)

		b, err := os.ReadFile(file) //nolint:gosec
		if err != nil {
			log.Fatalf("failed to read check file %+v", err)
		}
		body := strings.TrimSpace(string(b))
		if !strings.HasPrefix(body, "---") {
			log.Fatalf("file %s should start with '---' separator", fileName)
		}

		advisors, err := check.ParseAdvisors(strings.NewReader(body), &check.ParseParams{
			DisallowUnknownFields: true,
			DisallowInvalidChecks: true,
		})
		if err != nil {
			log.Fatalf("failed to parse advisors file %+v", err)
		}

		if len(advisors) != 1 {
			log.Fatalf("expected exactly one advisor in %s.", file)
		}
		a := advisors[0]

		if a.Name != strings.TrimSuffix(strings.TrimSuffix(fileName, ".example"), ".yml") {
			log.Fatalf("advisor name does not match file name %s.", file)
		}

		if len(a.Tiers) == 0 {
			log.Fatalf("advisor tiers missing: %s", file)
		}

		if _, ok := res[a.Name]; ok {
			log.Fatalf("advisor name collision detected for: %s", a.Name)
		}

		res[a.Name] = &a
	}

	return res
}

func loadAndValidateChecks(dir string) map[string]check.Check {
	matches, err := filepath.Glob(filepath.Join(dir, "*")) // not "*.yml" to cover examples
	if err != nil {
		log.Fatalf("failed to find check files %+v", err)
	}

	res := make(map[string]check.Check, len(matches))
	for _, file := range matches {
		log.Print(file)
		_, fileName := filepath.Split(file)

		b, err := os.ReadFile(file) //nolint:gosec
		if err != nil {
			log.Fatalf("failed to read check file %+v", err)
		}
		body := strings.TrimSpace(string(b))
		if !strings.HasPrefix(body, "---") {
			log.Fatalf("file %s should start with '---' separator", fileName)
		}

		checks, err := check.ParseChecks(strings.NewReader(body), &check.ParseParams{
			DisallowUnknownFields: true,
			DisallowInvalidChecks: true,
		})
		if err != nil {
			log.Fatalf("failed to parse checks file %+v", err)
		}

		if len(checks) != 1 {
			log.Fatalf("expected exactly one check in %s.", file)
		}
		c := checks[0]

		if c.Name != strings.TrimSuffix(strings.TrimSuffix(fileName, ".example"), ".yml") {
			log.Fatalf("check name does not match file name %s.", file)
		}

		if _, ok := res[c.Name]; ok {
			log.Fatalf("check name collision detected for: %s", c.Name)
		}

		res[c.Name] = c
	}

	return res
}

func validateTemplates(dir string, markdownT bool) { //nolint:cyclop
	matches, err := filepath.Glob(filepath.Join(dir, "*.yml"))
	if err != nil {
		log.Fatalf("failed to find tempatles files %+v", err)
	}

	res := make(map[string]alert.Template, len(matches))
	for _, file := range matches {
		log.Print(file)

		b, err := os.ReadFile(file) //nolint:gosec
		if err != nil {
			log.Fatalf("failed to read template file %+v", err)
		}
		templates, err := alert.Parse(bytes.NewReader(b), &alert.ParseParams{
			DisallowUnknownFields:    true,
			DisallowInvalidTemplates: true,
		})
		if err != nil {
			log.Fatalf("failed to parse templates file %+v", err)
		}

		if len(templates) != 1 {
			log.Fatalf("expected exactly one template in %s.", file)
		}
		r := templates[0]

		_, fileName := filepath.Split(file)
		if r.Name != strings.TrimSuffix(fileName, ".yml") {
			log.Fatalf("template name does not match file name %s.", file)
		}

		if len(r.Tiers) == 0 {
			log.Fatalf("template tiers missing: %s", file)
		}

		if _, ok := res[r.Name]; ok {
			log.Fatalf("template name collision detected for: %s", r.Name)
		}

		res[r.Name] = r
	}

	// Always call this function for extra validation side-effects.
	//nolint:godox
	if markdownT {
		res, err := tableTemplates(res, markdown)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Print(res) //nolint:forbidigo
	}
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

// // tableChecks returns a Markdown table with checks.
// func tableChecks(checks []check.Check, flavor flavor) (string, error) {
// 	// (ab)use tabwriter to generate Confluence markup or Markdown table
// 	var buf bytes.Buffer
// 	w := tabwriter.NewWriter(&buf, 0, 0, 1, ' ', tabwriter.Debug)
// 	fmt.Fprintf(w, "\tName\tTiers\tDescription\t\n")
//
// 	// that's the only thing that should be skipped for "Confluence wiki" markup to work
// 	if flavor == markdown {
// 		fmt.Fprintf(w, "\t----\t-----\t-----------\t\n")
// 	}
//
// 	predeclared := map[string]starlark.GoFunc{
// 		"format_version_num": nil,
// 		"parse_version":      nil,
// 	}
//
// 	var err error
// 	for _, c := range checks {
// 		err = starlark.CheckGlobals(&c, predeclared) //nolint:gosec,scopelint
// 		if err != nil {
// 			return "", err
// 		}
//
// 		tiers := make([]string, len(c.Tiers))
// 		for i, t := range c.Tiers {
// 			tiers[i] = string(t)
// 		}
// 		fmt.Fprintf(w, "\t%s\t%s\t%s\t\n", c.Name, strings.Join(tiers, ", "), c.Description)
// 	}
//
// 	if err := w.Flush(); err != nil {
// 		return "", err
// 	}
// 	return buf.String(), nil
// }
