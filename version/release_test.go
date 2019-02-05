package version

import (
	"strings"
	"testing"
)

func TestInfoManaged(t *testing.T) {
	ProjectName = "pmm-managed"
	Version = "2.1.2"
	PMMVersion = "2.1.2"
	Timestamp = "1545226908"
	FullCommit = "6559a94ab33831deeda04193f74413b735edb1a1"
	Branch = "master"

	expected := "pmm-managed v2.1.2"
	actual := ShortInfo()
	if expected != actual {
		t.Errorf("expected: %q\nactual: %q", expected, actual)
	}

	expected = strings.Join([]string{
		"ProjectName: pmm-managed",
		"Version: 2.1.2",
		"PMMVersion: 2.1.2",
		"Timestamp: 2018-12-19 13:41:48 (UTC)",
		"FullCommit: 6559a94ab33831deeda04193f74413b735edb1a1",
		"Branch: master",
	}, "\n")
	actual = FullInfo()
	if expected != actual {
		t.Errorf("expected: %q\nactual: %q", expected, actual)
	}
}

func TestInfoexporter(t *testing.T) {
	ProjectName = "external_exporter"
	Version = "0.8.5"
	PMMVersion = "2.1.2"
	Timestamp = "1545226908"
	FullCommit = "6559a94ab33831deeda04193f74413b735edb1a1"
	Branch = "master"

	expected := "external_exporter v0.8.5 (PMM v2.1.2)"
	actual := ShortInfo()
	if expected != actual {
		t.Errorf("expected: %q\nactual: %q", expected, actual)
	}

	expected = strings.Join([]string{
		"ProjectName: external_exporter",
		"Version: 0.8.5",
		"PMMVersion: 2.1.2",
		"Timestamp: 2018-12-19 13:41:48 (UTC)",
		"FullCommit: 6559a94ab33831deeda04193f74413b735edb1a1",
		"Branch: master",
	}, "\n")
	actual = FullInfo()
	if expected != actual {
		t.Errorf("expected: %q\nactual: %q", expected, actual)
	}
}
