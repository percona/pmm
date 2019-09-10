// pmm-agent
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

// +build ignore

// check-license checks that ASL license header in all files matches header in this file.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
)

func getHeader() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("runtime.Caller(0) failed")
	}
	f, err := os.Open(file)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	var header string
	s := bufio.NewScanner(f)
	for s.Scan() {
		if s.Text() == "" {
			break
		}
		header += s.Text() + "\n"
	}
	header += "\n"
	if err := s.Err(); err != nil {
		log.Fatal(err)
	}
	return header
}

var generatedHeader = regexp.MustCompile(`^// Code generated .* DO NOT EDIT\.`)

func checkHeader(path string, header string) bool {
	f, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	actual := make([]byte, len(header))
	_, err = io.ReadFull(f, actual)
	if err == io.ErrUnexpectedEOF {
		err = nil // some files are shorter than license header
	}
	if err != nil {
		log.Printf("%s - %s", path, err)
		return false
	}

	if generatedHeader.Match(actual) {
		return true
	}

	if header != string(actual) {
		log.Print(path)
		return false
	}
	return true
}

func main() {
	log.SetFlags(0)
	flag.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), "Usage: go run .github/check-license.go")
		flag.CommandLine.PrintDefaults()
	}
	flag.Parse()

	header := getHeader()

	ok := true
	filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			switch info.Name() {
			case ".git", "vendor":
				return filepath.SkipDir
			default:
				return nil
			}
		}

		if filepath.Ext(info.Name()) == ".go" {
			if !checkHeader(path, header) {
				ok = false
			}
		}
		return nil
	})

	if ok {
		os.Exit(0)
	}
	log.Print("Please update license header in those files.")
	os.Exit(1)
}
