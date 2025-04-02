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
		fmt.Sprintf("ProjectName: %s", ProjectName),        //nolint:errcheck
		fmt.Sprintf("Version: %s", Version),                //nolint:errcheck
		fmt.Sprintf("PMMVersion: %s", PMMVersion),          //nolint:errcheck
		fmt.Sprintf("Timestamp: %s", timestampFormatted()), //nolint:errcheck
		fmt.Sprintf("FullCommit: %s", FullCommit),          //nolint:errcheck
		fmt.Sprintf("Branch: %s", Branch),                  //nolint:errcheck
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
