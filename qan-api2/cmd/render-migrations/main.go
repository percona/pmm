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

// Package main is used for prepared SQL migrations for Clickhouse client.
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"text/template"
)

func main() {
	var (
		sqlDir  string
		engine  string
		cluster string
	)
	flag.StringVar(&sqlDir, "sql", "migrations/sql", "Directory with .up.sql migration templates")
	flag.StringVar(&engine, "engine", "MergeTree", "Engine to use in templates")
	flag.StringVar(&cluster, "cluster", "", "Cluster clause (e.g. ON CLUSTER 'test_cluster')")
	flag.Parse()

	data := map[string]any{
		"engine":  engine,
		"cluster": cluster,
	}

	files, err := filepath.Glob(filepath.Join(sqlDir, "*.up.sql"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to list migration files: %v\n", err)
		os.Exit(1)
	}

	for _, file := range files {
		content, err := ioutil.ReadFile(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to read %s: %v\n", file, err)
			os.Exit(1)
		}
		tmpl, err := template.New(filepath.Base(file)).Parse(string(content))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to parse template %s: %v\n", file, err)
			os.Exit(1)
		}
		err = tmpl.Execute(os.Stdout, data)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to render template %s: %v\n", file, err)
			os.Exit(1)
		}
	}
}
