package version

import (
	"testing"
)

func TestParse(t *testing.T) {
	for s, expected := range map[string]ParsedVersion{
		"1.2.3":                  {Major: 1, Minor: 2, Patch: 3},
		"11.22.33-beefc0fe+meta": {Major: 11, Minor: 22, Patch: 33, Rest: "-beefc0fe+meta"},
	} {
		t.Run(s, func(t *testing.T) {
			actual, err := Parse(s)
			if err != nil {
				t.Fatal(err)
			}
			if expected != actual {
				t.Fatalf("\nexpected: %+v\nactual: %+v", expected, actual)
			}
		})
	}
}
