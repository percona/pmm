package version

import (
	"fmt"
	"testing"
)

func TestParse(t *testing.T) {
	for s, expected := range map[string]Info{
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

func TestLess(t *testing.T) {
	type testdata struct {
		left  Info
		right Info
		less  bool
	}
	for _, td := range []testdata{
		{
			left:  Info{Major: 1, Minor: 2, Patch: 3},
			right: Info{Major: 1, Minor: 2, Patch: 4},
			less:  true,
		},
		{
			left:  Info{Major: 2, Minor: 0, Patch: 0},
			right: Info{Major: 2, Minor: 0, Patch: 0, Rest: "-dev"},
			less:  true,
		},
	} {
		t.Run(fmt.Sprintf("%s < %s", td.left.String(), td.right.String()), func(t *testing.T) {
			if td.left.Less(&td.right) != td.less {
				t.Fatalf("%s < %s != %t", td.left, td.right, td.less)
			}
		})
	}
}
