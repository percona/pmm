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
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"

	"github.com/percona/pmm/managed/utils/distribution"
	"github.com/percona/pmm/version"
)

// defaultLatestPMMImage is the default image name to use when the latest version cannot be determined.
const (
	defaultLatestPMMImage  = "perconalab/pmm-server:3-dev-latest"
	pmmUpdatePerformLog    = "/srv/logs/pmm-update-perform-init.log"
	updateCheckInterval    = 24 * time.Hour
	updateCheckResultFresh = updateCheckInterval + 10*time.Minute
	updateDefaultTimeout   = 30 * time.Second
	envfilePath            = "/home/pmm/update/pmm-server.env"
)

var fileName = "/etc/pmm-server-update-version.json"

// Updater is a service to check for updates and trigger the update process.
type Updater struct {
	l                  *logrus.Entry
	watchtowerHost     *url.URL
	dus                *distribution.Service
	gRPCMessageMaxSize uint32

	performM sync.Mutex
	running  bool

	checkRW         sync.RWMutex
	lastCheckResult *version.DockerVersionInfo
	lastCheckTime   time.Time
}

// NewUpdater creates a new Updater service.
func NewUpdater(watchtowerHost *url.URL, dus *distribution.Service, gRPCMessageMaxSize uint32) *Updater {
	u := &Updater{
		l:                  logrus.WithField("service", "updater"),
		watchtowerHost:     watchtowerHost,
		dus:                dus,
		gRPCMessageMaxSize: gRPCMessageMaxSize,
	}
	return u
}

// Run runs check for updates loop until ctx is canceled.
func (up *Updater) Run(ctx context.Context) {
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

func (up *Updater) currentVersion() *version.Parsed {
	return version.MustParse(version.Version)
}

// StartUpdate triggers the update process.
func (up *Updater) StartUpdate(ctx context.Context, newImageName string) error {
	up.performM.Lock()
	defer up.performM.Unlock()
	if up.running {
		return errors.New("update already in progress")
	}
	up.running = true
	if newImageName == "" {
		return errors.New("newImageName is empty")
	}

	err := up.checkWatchtowerHost()
	if err != nil {
		up.l.WithError(err).Error("Failed to check watchtower host")
		return grpcstatus.Errorf(codes.FailedPrecondition, "failed to check watchtower host")
	}

	fileInfo, err := os.Stat(envfilePath)
	if os.IsExist(err) && fileInfo != nil {
		err := up.updateEnvironmentVariables(envfilePath, newImageName)
		if err != nil {
			up.running = false
			up.l.WithError(err).Error("Failed to update environment variables file")
			return errors.Wrap(err, "failed to update environment variables file")
		}
	} else if err != nil {
		up.running = false
		up.l.WithError(err).Error("Failed to check file")
		return errors.Wrap(err, "failed to check file")
	}

	if err := up.sendRequestToWatchtower(ctx, newImageName); err != nil {
		up.l.WithError(err).Error("Failed to trigger update")
		return errors.Wrap(err, "failed to trigger update")
	}
	return nil
}

// ForceCheckUpdates forces an update check.
func (up *Updater) ForceCheckUpdates(ctx context.Context) error {
	return up.check(ctx)
}

// LastCheckUpdatesResult returns the result of the last update check.
func (up *Updater) LastCheckUpdatesResult(ctx context.Context) (*version.UpdateCheckResult, time.Time) {
	installed := up.InstalledPMMVersion()
	latest, lastCheckTime := up.checkResult(ctx)
	return &version.UpdateCheckResult{
		Installed:       installed,
		Latest:          *latest,
		UpdateAvailable: latest.DockerImage != "",
		LatestNewsURL:   "https://per.co.na/pmm/" + latest.Version.String(),
	}, lastCheckTime
}

func (up *Updater) latest(ctx context.Context) (*version.DockerVersionInfo, error) {
	info, err := up.readFromFile()
	if err != nil {
		return nil, errors.Wrap(err, "failed to read from file")
	}
	if info != nil {
		return info, nil
	}
	if os.Getenv("PMM_DEV_UPDATE_DOCKER_IMAGE") != "" {
		return up.parseDockerTag(os.Getenv("PMM_DEV_UPDATE_DOCKER_IMAGE"))
	}

	// If file does not exist, and ENV variable is not set, go get the latest tag from DockerHub
	return up.latestAvailableFromDockerHub(ctx)
}

func (up *Updater) readFromFile() (*version.DockerVersionInfo, error) {
	// Read from file, if it's not exist read from ENV variable, if it's not exist get the latest tag from DockerHub.
	content, err := os.ReadFile(fileName)
	if err != nil && !os.IsNotExist(err) {
		up.l.WithError(err).Error("Failed to read file")
		return nil, errors.Wrap(err, "failed to read file")
	}
	if os.IsNotExist(err) {
		return nil, nil //nolint:nilnil
	}
	info := version.DockerVersionInfo{}
	err = json.Unmarshal(content, &info)
	if err != nil {
		up.l.WithError(err).Error("Failed to unmarshal file")
		return nil, errors.Wrap(err, "failed to unmarshal file")
	}
	return &info, nil
}

type result struct {
	Name          string    `json:"name"`
	TagLastPushed time.Time `json:"tag_last_pushed"`
}

// TagsResponse is a response from DockerHub.
type TagsResponse struct {
	Results []result `json:"results"`
}

// latestAvailableFromDockerHub returns the latest available version from DockerHub.
// It returns the latest minor version for the current major version.
// If the current version is the latest minor version, it returns the next major version.
// If the current version is the latest version, it returns the current version.
func (up *Updater) latestAvailableFromDockerHub(ctx context.Context) (*version.DockerVersionInfo, error) {
	repo := os.Getenv("PMM_DEV_UPDATE_DOCKER_REPO")
	if repo == "" {
		repo = "percona/pmm-server"
	}
	u := "https://registry.hub.docker.com/v2/repositories/" + repo + "/tags/?page_size=100"
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
		up.l.Infof("Found %d tags", len(tagsResponse.Results))
		next := up.next(*up.currentVersion(), tagsResponse.Results)
		if next.DockerImage != "" {
			next.DockerImage = repo + ":" + next.DockerImage
		}
		return next, err
	}
	return nil, errors.New("no tags found")
}

func (up *Updater) parseDockerTag(tag string) (*version.DockerVersionInfo, error) {
	splitTag := strings.Split(tag, ":")
	if len(splitTag) != 2 {
		return nil, fmt.Errorf("invalid tag: %s", tag)
	}
	parsed, err := version.Parse(splitTag[1])
	if err != nil {
		up.l.Debugf("Failed to parse version: %s", splitTag[1])
		return &version.DockerVersionInfo{DockerImage: tag}, nil //nolint:nilerr
	}
	return &version.DockerVersionInfo{
		Version:     *parsed,
		DockerImage: tag,
	}, nil
}

func (up *Updater) next(currentVersion version.Parsed, results []result) *version.DockerVersionInfo {
	nextMinor := &version.DockerVersionInfo{
		Version: currentVersion,
	}
	var nextMajor *version.DockerVersionInfo
	for _, result := range results {
		v, err := version.Parse(result.Name)
		if err != nil {
			up.l.Debugf("Failed to parse version: %s", result.Name)
			continue
		}
		if !currentVersion.Less(v) {
			continue
		}
		if v.Major == currentVersion.Major && nextMinor.Version.Less(v) { // next major
			nextMinor = &version.DockerVersionInfo{
				Version:     *v,
				DockerImage: result.Name,
				BuildTime:   result.TagLastPushed,
			}
		}
		if v.Major > currentVersion.Major &&
			(nextMajor == nil || (nextMajor.Version.Less(v) && nextMajor.Version.Major == v.Major) || v.Major < nextMajor.Version.Major) {
			nextMajor = &version.DockerVersionInfo{
				Version:     *v,
				DockerImage: result.Name,
				BuildTime:   result.TagLastPushed,
			}
		}
	}
	if nextMinor.Version == currentVersion && nextMajor != nil {
		return nextMajor
	}
	return nextMinor
}

// InstalledPMMVersion returns the currently installed PMM version.
func (up *Updater) InstalledPMMVersion() version.PackageInfo {
	t, _ := version.Time()
	return version.PackageInfo{
		Version:     version.Version,
		FullVersion: version.PMMVersion,
		BuildTime:   &t,
	}
}

// IsRunning returns true if the update process is running.
func (up *Updater) IsRunning() bool {
	up.performM.Lock()
	defer up.performM.Unlock()
	return up.running
}

// UpdateLog returns the log of the update process.
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

// checkResult returns the result of the last update check.
// It may force re-check if last result is empty or too old.
func (up *Updater) checkResult(ctx context.Context) (*version.DockerVersionInfo, time.Time) {
	up.checkRW.RLock()
	defer up.checkRW.RUnlock()

	if time.Since(up.lastCheckTime) > updateCheckResultFresh {
		up.checkRW.RUnlock()
		_ = up.check(ctx)
		up.checkRW.RLock()
	}

	return up.lastCheckResult, up.lastCheckTime
}

// check performs update check.
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

func (up *Updater) checkWatchtowerHost() error {
	// Check if watchtower host is available
	if up.watchtowerHost == nil {
		return errors.New("watchtower host is not set")
	}
	if !isHostAvailable(up.watchtowerHost.Hostname(), up.watchtowerHost.Port(), updateDefaultTimeout) {
		return errors.New("watchtower host is not available")
	}
	return nil
}

func (up *Updater) updateEnvironmentVariables(filename string, name string) error {
	if len(strings.Split(name, "/")) < 3 {
		name = "docker.io/" + name
	}
	file, err := os.ReadFile(filename)
	if err != nil {
		return errors.Wrap(err, "failed to read file")
	}
	lines := strings.Split(string(file), "\n")
	for i, line := range lines {
		if strings.Contains(line, "PMM_IMAGE") {
			lines[i] = fmt.Sprintf("PMM_IMAGE=%s", name)
		}
	}
	err = os.WriteFile(filename, []byte(strings.Join(lines, "\n")), 0o644)
	if err != nil {
		return errors.Wrap(err, "failed to write file")
	}
	return nil
}

func isHostAvailable(host string, port string, timeout time.Duration) bool {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), timeout)
	if err != nil {
		return false
	}
	if conn != nil {
		defer conn.Close() //nolint:errcheck
		return true
	}
	return false
}
