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

// UpdateCurrentResult represents `pmm-update -current` result.
type UpdateCurrentResult struct {
	Installed PackageInfo `json:"installed"`
}

// UpdateCheckResult represents `pmm-update -check` result.
type UpdateCheckResult struct {
	Installed       PackageInfo `json:"installed"`
	Latest          PackageInfo `json:"latest,omitempty"`
	UpdateAvailable bool        `json:"update_available"`
}
