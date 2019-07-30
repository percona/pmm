package version

import (
	"time"
)

// UpdateCheckResult represents `pmm-update -check` result.
type UpdateCheckResult struct {
	InstalledRPMVersion string `json:"installed_rpm_version"`

	UpdateAvailable bool `json:"update_available"`

	LatestRPMVersion string     `json:"latest_rpm_version"`
	LatestTime       *time.Time `json:"latest_time"`
	LatestRepo       string     `json:"latest_repo"`
}
