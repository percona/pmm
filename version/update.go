package version

import (
	"time"
)

// UpdateCheckResult represents `pmm-update -check` result.
type UpdateCheckResult struct {
	InstalledRPMVersion     string     `json:"installed_rpm_version"`
	InstalledRPMNiceVersion string     `json:"installed_rpm_nice_version"`
	InstalledTime           *time.Time `json:"installed_time"`

	UpdateAvailable bool `json:"update_available"`

	LatestRPMVersion     string     `json:"latest_rpm_version"`
	LatestRPMNiceVersion string     `json:"latest_rpm_nice_version"`
	LatestRepo           string     `json:"latest_repo"`
	LatestTime           *time.Time `json:"latest_time"`
}
