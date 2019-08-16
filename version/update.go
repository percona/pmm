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

// UpdateResult represents `pmm-update -current` and `pmm-update -check` result.
type UpdateResult struct {
	Installed       PackageInfo  `json:"installed"`
	Latest          *PackageInfo `json:"latest,omitempty"` // absent for -current, present for -check
	UpdateAvailable bool         `json:"update_available"`
}
