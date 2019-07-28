package version

import (
	"time"
)

// UpdateCheckResult represents `pmm-update -check` result.
type UpdateCheckResult struct {
	InstalledVersion string    `json:"installed_version"`
	InstalledTime    time.Time `json:"installed_time"`

	LatestVersion string    `json:"latest_version"`
	LatestTime    time.Time `json:"latest_time"`
	LatestRepo    string    `json:"latest_repo"`
}
