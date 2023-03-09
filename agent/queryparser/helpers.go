package queryparser

import (
	"bufio"
	"fmt"
	"regexp"
	"strings"
)

func parseMySQLComments(q string) (map[string]bool, error) {
	// sqlparser.ExtractMysqlComment(q) doesnt work properly
	// input: SELECT * FROM people /*! bla bla */ WHERE name = 'john'
	// output: ECT * FROM people /*! bla bla */ WHERE name = 'joh
	r, err := regexp.Compile("(?s)\\/\\*(.*?) \\*\\/")
	if err != nil {
		return nil, err
	}

	// comments using comment as a key to avoid duplicates
	comments := make(map[string]bool)
	for _, v := range r.FindAllStringSubmatch(q, -1) {
		if len(v) < 2 || len(v[1]) < 2 {
			continue
		}

		// replace all mutations of multiline comment
		// /*! and /*+
		replacer := strings.NewReplacer("!", "", "+", "")
		comments[replacer.Replace(v[1])[1:]] = true
	}

	hashComments, err := parseMySQLSinglelineComments(q, "#")
	if err != nil {
		return nil, err
	}
	for c := range hashComments {
		comments[c] = true
	}

	dashComments, err := parseMySQLSinglelineComments(q, "--")
	if err != nil {
		return nil, err
	}
	for c := range dashComments {
		comments[c] = true
	}

	return comments, nil
}

func parseMySQLSinglelineComments(q, startChar string) (map[string]bool, error) {
	r, err := regexp.Compile(fmt.Sprintf("%s.*", startChar))
	if err != nil {
		return nil, err
	}

	// comments using comment as a key to avoid duplicates
	comments := make(map[string]bool)
	lines, err := stringToLines(q)
	if err != nil {
		return nil, err
	}
	for _, l := range lines {
		for _, v := range r.FindStringSubmatch(l) {
			if len(v) < 2 {
				continue
			}

			comments[v[(len(startChar)+1):]] = true
		}
	}

	return comments, nil
}

func stringToLines(s string) (lines []string, err error) {
	scanner := bufio.NewScanner(strings.NewReader(s))
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	err = scanner.Err()

	return
}
