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

package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/api/serverpb"
	"github.com/percona/pmm/version"
)

// defaultLatestPMMImage is the default image name to use when the latest version cannot be determined.
const defaultLatestPMMImage = "perconalab/pmm-server:3-dev-latest"

// Updater is a service to check for updates and trigger the update process.
type Updater struct {
	l              *logrus.Entry
	supervisord    supervisordService
	watchtowerHost *url.URL
}

// NewUpdater creates a new Updater service.
func NewUpdater(supervisord supervisordService, watchtowerHost *url.URL) *Updater {
	return &Updater{
		l:              logrus.WithField("service", "updater"),
		supervisord:    supervisord,
		watchtowerHost: watchtowerHost,
	}
}

func (s *Updater) sendRequestToWatchtower(ctx context.Context, newImageName string) error {
	hostname, err := os.Hostname()
	if err != nil {
		return errors.Wrap(err, "failed to get hostname")
	}

	u, err := s.watchtowerHost.Parse("/v1/update")
	if err != nil {
		return errors.Wrap(err, "failed to parse URL")
	}
	q := u.Query()
	q.Set("hostname", hostname)
	q.Set("newImageName", newImageName)
	u.RawQuery = q.Encode()

	// Create a new request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return errors.Wrap(err, "failed to create request")
	}

	// Add the token to the request header
	token := os.Getenv("PMM_WATCHTOWER_TOKEN")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return errors.Wrap(err, "failed to send request")
	}
	defer resp.Body.Close() //nolint:errcheck

	// Check the response
	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("received non-OK response: %v", resp.StatusCode)
	}

	s.l.Info("Successfully triggered update")
	return nil
}

func (s *Updater) currentVersion() string {
	return version.Version
}

// StartUpdate triggers the update process.
func (s *Updater) StartUpdate(ctx context.Context, newImageName string) error {
	if newImageName == "" {
		latest, err := s.latest(ctx)
		if err != nil {
			s.l.WithError(err).Error("Failed to get latest version")
			newImageName = defaultLatestPMMImage
		} else {
			newImageName = fmt.Sprintf("%s:%s", latest.Repo, latest.Version)
		}
	}

	if err := s.sendRequestToWatchtower(ctx, newImageName); err != nil {
		s.l.WithError(err).Error("Failed to trigger update")
		return errors.Wrap(err, "failed to trigger update")
	}
	return nil
}

func (s *Updater) onlyInstalledVersionResponse() *serverpb.CheckUpdatesResponse {
	return &serverpb.CheckUpdatesResponse{
		Installed: &serverpb.VersionInfo{
			Version: s.currentVersion(),
		},
	}
}

// ForceCheckUpdates forces an update check.
func (s *Updater) ForceCheckUpdates(_ context.Context) error {
	// TODO: PMM-11261 Implement this method
	return nil
}

// TagsResponse is a response from DockerHub.
type TagsResponse struct {
	Results []struct {
		Name string `json:"name"`
	} `json:"results"`
}

// LastCheckUpdatesResult returns the result of the last update check.
func (s *Updater) LastCheckUpdatesResult(ctx context.Context) (*version.UpdateCheckResult, time.Time) {
	buildTime, err := version.Time()
	if err != nil {
		s.l.WithError(err).Error("Failed to get build time")
		return nil, time.Now()
	}
	latest, err := s.latest(ctx)
	if err != nil {
		s.l.WithError(err).Error("Failed to get latest version")
		return nil, time.Now()
	}
	return &version.UpdateCheckResult{
		Installed: version.PackageInfo{
			Version:     s.currentVersion(),
			FullVersion: version.PMMVersion,
			BuildTime:   &buildTime,
			Repo:        "local",
		},
		Latest:          *latest,
		UpdateAvailable: true,
		LatestNewsURL:   "",
	}, time.Now()
}

func (s *Updater) latest(ctx context.Context) (*version.PackageInfo, error) {
	// Read from file, if it's not exist read from ENV variable, if it's not exist get the latest tag from DockerHub.
	fileName := "/etc/pmm-server-update-version.json"
	content, err := os.ReadFile(fileName)
	if err != nil && !os.IsNotExist(err) {
		s.l.WithError(err).Error("Failed to read file")
		return nil, errors.Wrap(err, "failed to read file")
	}
	if err == nil {
		info := version.PackageInfo{}
		err = json.Unmarshal(content, &info)
		if err != nil {
			s.l.WithError(err).Error("Failed to unmarshal file")
			return nil, errors.Wrap(err, "failed to unmarshal file")
		}
		return &info, nil
	}
	if os.Getenv("PMM_SERVER_UPDATE_VERSION") != "" {
		return s.parseDockerTag(os.Getenv("PMM_SERVER_UPDATE_VERSION")), nil
	}

	// If file does not exist, and ENV variable is not set, go get the latest tag from DockerHub
	u := "https://registry.hub.docker.com/v2/repositories/percona/pmm-server/tags/"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		s.l.WithError(err).Error("Failed to create request")
		return nil, errors.Wrap(err, "failed to create request")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		s.l.WithError(err).Error("Failed to get tags from DockerHub")
		return nil, errors.Wrap(err, "failed to get tags from DockerHub")
	}
	defer resp.Body.Close() //nolint:errcheck

	var tagsResponse TagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&tagsResponse); err != nil {
		s.l.WithError(err).Error("Failed to decode response")
		return nil, errors.Wrap(err, "failed to decode response")
	}

	if len(tagsResponse.Results) != 0 {
		// Assuming the first tag is the latest
		return s.parseDockerTag(tagsResponse.Results[0].Name), nil
	}
	return nil, errors.New("no tags found")
}

func (s *Updater) parseDockerTag(tag string) *version.PackageInfo {
	splitTag := strings.Split(tag, ":")
	if len(splitTag) != 2 {
		return nil
	}
	return &version.PackageInfo{
		Version:     splitTag[1],
		FullVersion: tag,
		Repo:        splitTag[0],
	}
}
