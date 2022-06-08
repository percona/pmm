package version

import (
	"fmt"
	"strings"
	"testing"
)

func setupDataForManaged() {
	ProjectName = "pmm-managed"
	Version = "2.1.2"
	PMMVersion = "2.1.2"
	Timestamp = "1545226908"
	FullCommit = "6559a94ab33831deeda04193f74413b735edb1a1"
	Branch = "master"
}

func setupDataForExporter() {
	ProjectName = "external_exporter"
	Version = "0.8.5"
	PMMVersion = "2.1.2"
	Timestamp = "1545226909"
	FullCommit = "6559a94ab33831deeda04193f74413b735edb1a2"
	Branch = "master"
}

func TestShortInfoManaged(t *testing.T) {
	setupDataForManaged()

	expected := fmt.Sprintf("%s v%s", ProjectName, PMMVersion)
	actual := ShortInfo()
	if expected != actual {
		t.Errorf("expected: %q\nactual: %q", expected, actual)
	}
}

func TestFullInfoPlainManaged(t *testing.T) {
	setupDataForManaged()

	expected := strings.Join([]string{
		fmt.Sprintf("ProjectName: %s", ProjectName),
		fmt.Sprintf("Version: %s", Version),
		fmt.Sprintf("PMMVersion: %s", PMMVersion),
		fmt.Sprintf("Timestamp: %s", timestampFormatted()),
		fmt.Sprintf("FullCommit: %s", FullCommit),
		fmt.Sprintf("Branch: %s", Branch),
	}, "\n")
	actual := FullInfo()
	if expected != actual {
		t.Errorf("expected: %q\nactual: %q", expected, actual)
	}
}

func TestFullInfoJsonManaged(t *testing.T) {
	setupDataForManaged()

	expected := "{" + strings.Join([]string{
		fmt.Sprintf(`"Branch":"%s"`, Branch),
		fmt.Sprintf(`"FullCommit":"%s"`, FullCommit),
		fmt.Sprintf(`"PMMVersion":"%s"`, PMMVersion),
		fmt.Sprintf(`"ProjectName":"%s"`, ProjectName),
		fmt.Sprintf(`"Timestamp":"%s"`, timestampFormatted()),
		fmt.Sprintf(`"Version":"%s"`, Version),
	}, ",") + "}"

	actual := FullInfoJSON()
	if actual != expected {
		t.Errorf("\nexpected: %q\nactual:   %q", expected, actual)
	}
}

func TestShortInfoExporter(t *testing.T) {
	setupDataForExporter()

	expected := fmt.Sprintf("external_exporter v%s (PMM v%s)", Version, PMMVersion)
	actual := ShortInfo()
	if expected != actual {
		t.Errorf("expected: %q\nactual: %q", expected, actual)
	}
}

func TestFullInfoPlainExporter(t *testing.T) {
	setupDataForExporter()

	expected := strings.Join([]string{
		fmt.Sprintf("ProjectName: %s", ProjectName),
		fmt.Sprintf("Version: %s", Version),
		fmt.Sprintf("PMMVersion: %s", PMMVersion),
		fmt.Sprintf("Timestamp: %s", timestampFormatted()),
		fmt.Sprintf("FullCommit: %s", FullCommit),
		fmt.Sprintf("Branch: %s", Branch),
	}, "\n")

	actual := FullInfo()
	if expected != actual {
		t.Errorf("expected: %q\nactual: %q", expected, actual)
	}
}

func TestFullInfoJsonExporter(t *testing.T) {
	setupDataForExporter()

	expected := "{" + strings.Join([]string{
		fmt.Sprintf(`"Branch":"%s"`, Branch),
		fmt.Sprintf(`"FullCommit":"%s"`, FullCommit),
		fmt.Sprintf(`"PMMVersion":"%s"`, PMMVersion),
		fmt.Sprintf(`"ProjectName":"%s"`, ProjectName),
		fmt.Sprintf(`"Timestamp":"%s"`, timestampFormatted()),
		fmt.Sprintf(`"Version":"%s"`, Version),
	}, ",") + "}"

	actual := FullInfoJSON()
	if actual != expected {
		t.Errorf("\nexpected: %q\nactual:   %q", expected, actual)
	}
}
