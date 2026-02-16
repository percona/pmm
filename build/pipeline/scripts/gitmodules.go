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

// A utility to extract submodule information from a .gitmodules file.
package main

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"gopkg.in/ini.v1"
)

func main() {
	if len(os.Args) < 4 { //nolint:mnd
		slog.Error("Usage: <gitmodules-file> <component> <field>", "program", os.Args[0])
		os.Exit(1)
	}

	gitmodulesFile := os.Args[1]
	component := os.Args[2]
	field := os.Args[3]

	cfg, err := ini.Load(gitmodulesFile)
	if err != nil {
		slog.Error("Failed to load .gitmodules", "error", err)
		os.Exit(1)
	}

	// Find the submodule section
	sectionName := fmt.Sprintf("submodule \"%s\"", component)
	section := cfg.Section(sectionName)
	if section == nil {
		slog.Error("Component not found in .gitmodules", "component", component)
		slog.Info("Available submodules:")

		for _, sec := range cfg.Sections() {
			if strings.HasPrefix(sec.Name(), "submodule") {
				slog.Info(sec.Name())
			}
		}
		os.Exit(1)
	}

	// Get the requested field (url, branch, or tag)
	value := section.Key(field).String()
	if value == "" {
		slog.Error("Field not found for component", "field", field, "component", component)
		os.Exit(1)
	}

	slog.Info(value)
}
