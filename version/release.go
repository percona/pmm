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

// Package version provides helpers for working with versions and build info.
package version

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

// https://goreleaser.com/templates/
var (
	// Component name, e.g. "pmm-managed" or "mongodb_exporter".
	// {{ .ProjectName }} for GoReleaser.
	ProjectName string

	// Component version, e.g. "2.1.2-beta1" for pmm-managed or "0.6.3" for mongodb_exporter.
	// {{ .Version }} for GoReleaser.
	Version string

	// PMM version. Empty for non-PMM builds.
	// For example, "2.1.2-beta1" for mongodb_exporter when built with PMM Client, empty otherwise.
	PMMVersion string

	// Build UNIX timestamp, e.g. "1545226908".
	// {{ .Timestamp }} for GoReleaser.
	Timestamp string

	// Full git commit hash, e.g. "6559a94ab33831deeda04193f74413b735edb1a1".
	// {{ .FullCommit }} for GoReleaser.
	FullCommit string

	// Git branch name, e.g. "master".
	Branch string
)

// ShortInfo returns project name and short version as one line.
func ShortInfo() string {
	if ProjectName == "" {
		return ""
	}

	res := ProjectName + " v" + Version
	if PMMVersion != "" && PMMVersion != Version {
		res += " (PMM v" + PMMVersion + ")"
	}
	return res
}

// Time returns parsed Timestamp.
func Time() (time.Time, error) {
	sec, err := strconv.ParseInt(Timestamp, 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(sec, 0).UTC(), nil
}

// timestampFormatted returns timestamp in format "YYYY-MM-DD HH:mm:ss (UTC)".
func timestampFormatted() string {
	timestamp := Timestamp
	if t, err := Time(); err == nil {
		timestamp = t.Format("2006-01-02 15:04:05 (UTC)")
	}
	return timestamp
}

// FullInfo returns multi-line version information.
func FullInfo() string {
	res := []string{
		"ProjectName: " + ProjectName,
		"Version: " + Version,
		"PMMVersion: " + PMMVersion,
		"Timestamp: " + timestampFormatted(),
		"FullCommit: " + FullCommit,
	}
	if Branch != "" {
		res = append(res, "Branch: "+Branch)
	}

	return strings.Join(res, "\n")
}

// FullInfoJSON returns version information in JSON format.
func FullInfoJSON() string {
	resMap := map[string]string{
		"ProjectName": ProjectName,
		"Version":     Version,
		"PMMVersion":  PMMVersion,
		"Timestamp":   timestampFormatted(),
		"FullCommit":  FullCommit,
	}
	if Branch != "" {
		resMap["Branch"] = Branch
	}

	bytes, err := json.Marshal(resMap)
	if err != nil {
		panic(err)
	}
	return string(bytes)
}
