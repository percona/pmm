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
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/api/serverpb"
	"github.com/percona/pmm/version"
)

// defaultLatestPMMImage is the default image name to use when the latest version cannot be determined.
const defaultLatestPMMImage = "perconalab/pmm-server:3-dev-latest"
const pmmUpdatePerformLog = "/srv/logs/pmm-update-perform-init.log"

// Updater is a service to check for updates and trigger the update process.
type Updater struct {
	l                  *logrus.Entry
	watchtowerHost     *url.URL
	gRPCMessageMaxSize uint32

	performM sync.Mutex
	running  bool

	checkRW         sync.RWMutex
	lastCheckResult *version.PackageInfo
	lastCheckTime   time.Time
}

// NewUpdater creates a new Updater service.
func NewUpdater(watchtowerHost *url.URL, gRPCMessageMaxSize uint32) *Updater {
	return &Updater{
		l:                  logrus.WithField("service", "updater"),
		watchtowerHost:     watchtowerHost,
		gRPCMessageMaxSize: gRPCMessageMaxSize,
	}
}

// run runs check for updates loop until ctx is canceled.
func (up *Updater) run(ctx context.Context) {
	up.l.Info("Starting...")
	ticker := time.NewTicker(updateCheckInterval)
	defer ticker.Stop()

	for {
		_ = up.check(ctx)

		select {
		case <-ticker.C:
			// continue with next loop iteration
		case <-ctx.Done():
			up.l.Info("Done.")
			return
		}
	}
}

func (up *Updater) sendRequestToWatchtower(ctx context.Context, newImageName string) error {
	hostname, err := os.Hostname()
	if err != nil {
		return errors.Wrap(err, "failed to get hostname")
	}

	u, err := up.watchtowerHost.Parse("/v1/update")
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

	up.l.Info("Successfully triggered update")
	return nil
}

func (up *Updater) currentVersion() string {
	return version.Version
}

// StartUpdate triggers the update process.
func (up *Updater) StartUpdate(ctx context.Context, newImageName string) error {
	up.performM.Lock()
	defer up.performM.Unlock()
	if up.running {
		return errors.New("update already in progress")
	}
	up.running = true
	up.performM.Unlock()
	if newImageName == "" {
		return errors.New("newImageName is empty")
	}

	if err := up.sendRequestToWatchtower(ctx, newImageName); err != nil {
		up.l.WithError(err).Error("Failed to trigger update")
		return errors.Wrap(err, "failed to trigger update")
	}
	return nil
}

func (up *Updater) onlyInstalledVersionResponse() *serverpb.CheckUpdatesResponse {
	return &serverpb.CheckUpdatesResponse{
		Installed: &serverpb.VersionInfo{
			Version: up.currentVersion(),
		},
	}
}

// ForceCheckUpdates forces an update check.
func (up *Updater) ForceCheckUpdates(_ context.Context) error {
	// TODO: PMM-11261 Implement this method
	return nil
}

type result struct {
	Name string `json:"name"`
}

type versionInfo struct {
	Version     version.Parsed `json:"version"`
	DockerImage string         `json:"docker_image"`
}

// TagsResponse is a response from DockerHub.
type TagsResponse struct {
	Results []result `json:"results"`
}

// LastCheckUpdatesResult returns the result of the last update check.
func (up *Updater) LastCheckUpdatesResult(ctx context.Context) (*version.UpdateCheckResult, time.Time) {
	buildTime, err := version.Time()
	if err != nil {
		up.l.WithError(err).Error("Failed to get build time")
		return nil, time.Now()
	}
	latest, lastCheckTime := up.checkResult(ctx)
	return &version.UpdateCheckResult{
		Installed: version.PackageInfo{
			Version:     up.currentVersion(),
			FullVersion: version.PMMVersion,
			BuildTime:   &buildTime,
			Repo:        "local",
		},
		Latest:          *latest,
		UpdateAvailable: true,
		LatestNewsURL:   "",
	}, lastCheckTime
}

func (up *Updater) latest(ctx context.Context) (*version.PackageInfo, error) {
	fileName := "/etc/pmm-server-update-version.json"
	content, err := os.ReadFile(fileName)
	switch {
	case err == nil:
		info := version.PackageInfo{}
		err = json.Unmarshal(content, &info)
		if err != nil {
			up.l.WithError(err).Error("Failed to unmarshal file")
			return nil, errors.Wrap(err, "failed to unmarshal file")
		}
		return &info, nil
	case err != nil && !os.IsNotExist(err):
		up.l.WithError(err).Error("Failed to read file")
		return nil, errors.Wrap(err, "failed to read file")
	case os.Getenv("PMM_SERVER_UPDATE_VERSION") != "":
		return up.parseDockerTag(os.Getenv("PMM_SERVER_UPDATE_VERSION")), nil
	default: // os.IsNotExist(err)
		// File does not exist, get the latest tag from DockerHub
		u := "https://registry.hub.docker.com/v2/repositories/percona/pmm-server/tags/"
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			up.l.WithError(err).Error("Failed to create request")
			return nil, errors.Wrap(err, "failed to create request")
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			up.l.WithError(err).Error("Failed to get tags from DockerHub")
			return nil, errors.Wrap(err, "failed to get tags from DockerHub")
		}
		defer resp.Body.Close() //nolint:errcheck

		var tagsResponse TagsResponse
		if err := json.NewDecoder(resp.Body).Decode(&tagsResponse); err != nil {
			up.l.WithError(err).Error("Failed to decode response")
			return nil, errors.Wrap(err, "failed to decode response")
		}

		if len(tagsResponse.Results) != 0 {
			currentVersion, err := version.Parse(up.currentVersion())
			if err != nil {
				up.l.WithError(err).Error("Failed to parse current version")
				return nil, errors.Wrap(err, "failed to parse current version")
			}

			update, err := up.next(*currentVersion, tagsResponse.Results)
			if err != nil {
				up.l.WithError(err).Error("Failed to get latest minor version")
				return nil, errors.Wrap(err, "failed to get latest minor version")
			}
			return up.parseDockerTag(update.DockerImage), nil
		}
		return nil, errors.New("no tags found")
	}
}

func (up *Updater) parseDockerTag(tag string) *version.PackageInfo {
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

func (up *Updater) next(currentVersion version.Parsed, results []result) (*versionInfo, error) {
	latest := versionInfo{
		Version: currentVersion,
	}
	for _, result := range results {
		splitTag := strings.Split(result.Name, ":")
		if len(splitTag) != 2 {
			continue
		}
		v, err := version.Parse(splitTag[1])
		if err != nil {
			up.l.Debugf("Failed to parse version: %s", splitTag[1])
			continue
		}
		if v.Major == currentVersion.Major && v.Minor > currentVersion.Minor {
			latest = versionInfo{
				Version:     *v,
				DockerImage: result.Name,
			}
		} else if v.Major > currentVersion.Major && v.Major < latest.Version.Major {
			latest = versionInfo{
				Version:     *v,
				DockerImage: result.Name,
			}
		}
	}
	return &latest, nil
}

func (up *Updater) InstalledPMMVersion() version.PackageInfo {
	t, _ := version.Time()
	return version.PackageInfo{
		Version:     up.currentVersion(),
		FullVersion: version.PMMVersion,
		BuildTime:   &t,
		Repo:        "local",
	}
}

func (up *Updater) IsRunning() bool {
	up.performM.Lock()
	defer up.performM.Unlock()
	return up.running
}

func (up *Updater) UpdateLog(offset uint32) ([]string, uint32, error) {
	up.performM.Lock()
	defer up.performM.Unlock()

	f, err := os.Open(pmmUpdatePerformLog)
	if err != nil {
		return nil, 0, errors.WithStack(err)
	}
	defer f.Close() //nolint:errcheck,gosec,nolintlint

	if _, err = f.Seek(int64(offset), io.SeekStart); err != nil {
		return nil, 0, errors.WithStack(err)
	}

	lines := make([]string, 0, 10)
	reader := bufio.NewReader(f)
	newOffset := offset
	for {
		line, err := reader.ReadString('\n')
		if err == nil {
			newOffset += uint32(len(line))
			if newOffset-offset > up.gRPCMessageMaxSize {
				return lines, newOffset - uint32(len(line)), nil
			}
			lines = append(lines, strings.TrimSuffix(line, "\n"))
			continue
		}
		if err == io.EOF {
			err = nil
		}
		return lines, newOffset, errors.WithStack(err)
	}
}

// checkResult returns last `pmm-update -check` result and check time.
// It may force re-check if last result is empty or too old.
func (up *Updater) checkResult(ctx context.Context) (*version.PackageInfo, time.Time) {
	up.checkRW.RLock()
	defer up.checkRW.RUnlock()

	if time.Since(up.lastCheckTime) > updateCheckResultFresh {
		up.checkRW.RUnlock()
		_ = up.check(ctx)
		up.checkRW.RLock()
	}

	return up.lastCheckResult, up.lastCheckTime
}

// check calls `pmm-update -check` and fills lastInstalledPackageInfo/lastCheckResult/lastCheckTime on success.
func (up *Updater) check(ctx context.Context) error {
	up.checkRW.Lock()
	defer up.checkRW.Unlock()

	latest, err := up.latest(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to get latest version")
	}
	up.lastCheckResult = latest
	up.lastCheckTime = time.Now()
	return nil
}
