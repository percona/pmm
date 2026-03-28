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
	"os"
	"strings"

	"gopkg.in/ini.v1"
)

func main() {
	if len(os.Args) < 4 { //nolint:mnd
		fmt.Fprintf(os.Stderr, "Usage: %s <gitmodules-file> <component> <field>\n", os.Args[0])
		os.Exit(1)
	}

	gitmodulesFile := os.Args[1]
	component := os.Args[2]
	field := os.Args[3]

	cfg, err := ini.Load(gitmodulesFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load .gitmodules: %v\n", err)
		os.Exit(1)
	}

	// Find the submodule section
	sectionName := fmt.Sprintf("submodule \"%s\"", component)
	section := cfg.Section(sectionName)
	if section == nil {
		fmt.Fprintf(os.Stderr, "Component not found in .gitmodules: %s\n", component)
		fmt.Fprintln(os.Stderr, "Available submodules:")
		for _, sec := range cfg.Sections() {
			if strings.HasPrefix(sec.Name(), "submodule") {
				fmt.Fprintln(os.Stderr, sec.Name())
			}
		}
		os.Exit(1)
	}

	// Get the requested field (url, branch, or tag)
	value := section.Key(field).String()
	if value == "" {
		fmt.Fprintf(os.Stderr, "Field '%s' not found for component '%s'\n", field, component)
		os.Exit(1)
	}

	fmt.Println(value)
}
