package version

import (
	"testing"
)

//

func TestGitDescribeInfo(t *testing.T) {
	data := []struct {
		s    string
		info *GitDescribeInfo
	}{
		{
			s:    "v2.0.0-beta4",
			info: &GitDescribeInfo{Major: 2, Minor: 0, Patch: 0, Rest: "-beta4"},
		},
		{
			s:    "v2.0.0-beta4-2-gff76039-dirty",
			info: &GitDescribeInfo{Major: 2, Minor: 0, Patch: 0, Rest: "-beta4-2-gff76039-dirty"},
		},
		{
			s:    "v2.0.0",
			info: &GitDescribeInfo{Major: 2, Minor: 0, Patch: 0},
		},
		{
			s:    "v2.1.2",
			info: &GitDescribeInfo{Major: 2, Minor: 1, Patch: 2},
		},
	}
	for i, expected := range data {
		t.Run(expected.s, func(t *testing.T) {
			actual, err := ParseGitDescribeInfo(expected.s)
			if err != nil {
				t.Fatal(err)
			}
			if *expected.info != *actual {
				t.Errorf("\nexpected: %+v\nactual: %+v", expected.info, actual)
			}
			if expected.s != actual.String() {
				t.Errorf("\nexpected: %q\nactual: %q", expected.s, actual.String())
			}

			for j := 0; j < i; j++ {
				if !data[j].info.Less(actual) {
					t.Errorf("%s is expected to be less than %s", data[j].info, actual)
				}
			}
			for j := i + 1; j < len(data); j++ {
				if data[j].info.Less(actual) {
					t.Errorf("%s is expected to be not less than %s", data[j].info, actual)
				}
			}
		})
	}
}

func TestRPMInfo(t *testing.T) {
	data := []struct {
		s    string
		info *RPMInfo
	}{
		{
			s:    "2.0.0-7.beta4.1907150908.7685dba.el7",
			info: &RPMInfo{Major: 2, Minor: 0, Patch: 0, Release: 7, Rest: ".beta4.1907150908.7685dba.el7"},
		},
		{
			s:    "2.0.0-7.beta5.1907221317.5aa025b.el7",
			info: &RPMInfo{Major: 2, Minor: 0, Patch: 0, Release: 7, Rest: ".beta5.1907221317.5aa025b.el7"},
		},
		{
			s:    "2.0.0-8.beta4.1907150908.7685dba.el7.noarch",
			info: &RPMInfo{Major: 2, Minor: 0, Patch: 0, Release: 8, Rest: ".beta4.1907150908.7685dba.el7.noarch"},
		},
	}
	for i, expected := range data {
		t.Run(expected.s, func(t *testing.T) {
			actual, err := ParseRPMInfo(expected.s)
			if err != nil {
				t.Fatal(err)
			}
			if *expected.info != *actual {
				t.Errorf("\nexpected: %+v\nactual: %+v", expected.info, actual)
			}
			if expected.s != actual.String() {
				t.Errorf("\nexpected: %q\nactual: %q", expected.s, actual.String())
			}

			for j := 0; j < i; j++ {
				if !data[j].info.Less(actual) {
					t.Errorf("%s is expected to be less than %s", data[j].info, actual)
				}
			}
			for j := i + 1; j < len(data); j++ {
				if data[j].info.Less(actual) {
					t.Errorf("%s is expected to be not less than %s", data[j].info, actual)
				}
			}
		})
	}
}
