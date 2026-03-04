package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <folder>")
		return
	}
	root := os.Args[1]
	results := make(map[string][]string)
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".log") {
			file, err := os.Open(path)
			if err != nil {
				fmt.Printf("Failed to open %s: %v\n", path, err)
				return nil
			}
			defer file.Close()
			scanner := bufio.NewScanner(file)
			lineNum := 1
			for scanner.Scan() {
				line := scanner.Text()
				lowerLine := strings.ToLower(line)
				if strings.Contains(lowerLine, "error") || strings.Contains(lowerLine, "fatal") {
					results[path] = append(results[path], fmt.Sprintf("%d: %s", lineNum, line))
				}
				lineNum++
			}
			if err := scanner.Err(); err != nil {
				fmt.Printf("Error reading %s: %v\n", path, err)
			}
		}
		return nil
	})
	if err != nil {
		fmt.Printf("Walk error: %v\n", err)
	}

	if len(results) == 0 {
		fmt.Println("No errors found.")
		return
	}

	for file, lines := range results {
		fmt.Printf("\n=== %s ===\n", file)
		for _, l := range lines {
			fmt.Printf("  %s\n", l)
		}
	}
}
