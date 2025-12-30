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
		fmt.Fprintf(os.Stderr, "Component '%s' not found in .gitmodules\n", component)
		fmt.Fprintln(os.Stderr, "Available submodules:")
		for _, sec := range cfg.Sections() {
			if strings.HasPrefix(sec.Name(), "submodule") {
				fmt.Fprintf(os.Stderr, "  %s\n", sec.Name())
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

	fmt.Print(value)
}
