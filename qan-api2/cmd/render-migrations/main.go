package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
	"io/ioutil"
)

func main() {
	var (
		sqlDir   string
		engine   string
		cluster  string
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
		fmt.Println()
	}
}
