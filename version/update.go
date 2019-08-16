package version

import (
	"time"
)

// PackageInfo describes a single package version.
type PackageInfo struct {
	Version     string     `json:"version"`
	FullVersion string     `json:"full_version"`
	BuildTime   *time.Time `json:"build_time"`
	Repo        string     `json:"repo"`
}

// UpdateInstalledResult represents `pmm-update -installed` result.
type UpdateInstalledResult struct {
	Installed PackageInfo `json:"installed"`
}

// UpdateCheckResult represents `pmm-update -check` result.
type UpdateCheckResult struct {
	Installed       PackageInfo `json:"installed"`
	Latest          PackageInfo `json:"latest,omitempty"`
	UpdateAvailable bool        `json:"update_available"`
}
