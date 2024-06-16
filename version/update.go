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

import "time"

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
	Installed       PackageInfo          `json:"installed"`
	Latest          DockerVersionInfo    `json:"latest,omitempty"`
	NewerVersions   []*DockerVersionInfo `json:"newer_versions"`
	UpdateAvailable bool                 `json:"update_available"`
}

// DockerVersionInfo describes the version of the Docker image.
type DockerVersionInfo struct {
	Version          Parsed    `json:"version"`
	DockerImage      string    `json:"docker_image"`
	BuildTime        time.Time `json:"build_time"`
	ReleaseNotesURL  string    `json:"release_notes_url"`
	ReleaseNotesText string    `json:"release_notes_text"`
}
