// Package version provides helpers for working with versions and build info.
package version

import (
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

// FullInfo returns multi-line version information.
func FullInfo() string {
	timestamp := Timestamp
	if t, err := Time(); err == nil {
		timestamp = t.Format("2006-01-02 15:04:05 (UTC)")
	}

	res := []string{
		"ProjectName: " + ProjectName,
		"Version: " + Version,
		"PMMVersion: " + PMMVersion,
		"Timestamp: " + timestamp,
		"FullCommit: " + FullCommit,
	}
	if Branch != "" {
		res = append(res, "Branch: "+Branch)
	}

	return strings.Join(res, "\n")
}
