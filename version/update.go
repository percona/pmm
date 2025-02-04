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

// UpdateCheckResult describes the latest update check result.
type UpdateCheckResult struct {
	Installed       PackageInfo       `json:"installed"`
	Latest          DockerVersionInfo `json:"latest,omitempty"`
	UpdateAvailable bool              `json:"update_available"`
	LatestNewsURL   string            `json:"latest_news_url"`
}

// DockerVersionInfo describes the version of the Docker image.
type DockerVersionInfo struct {
	Version          Parsed    `json:"version"`
	DockerImage      string    `json:"docker_image"`
	BuildTime        time.Time `json:"build_time"`
	ReleaseNotesURL  string    `json:"release_notes_url"`
	ReleaseNotesText string    `json:"release_notes_text"`
}

// DockerVersionsInfo is a wrapper around a DockerVersionInfo array to implement sorting.
type DockerVersionsInfo []*DockerVersionInfo

func (d DockerVersionsInfo) Len() int { return len(d) }

func (d DockerVersionsInfo) Less(i, j int) bool {
	return d[i].Version.Less(&d[j].Version)
}

func (d DockerVersionsInfo) Swap(i, j int) {
	d[i], d[j] = d[j], d[i]
}
