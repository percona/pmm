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
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services"
	"github.com/percona/pmm/managed/utils/envvars"
	"github.com/percona/pmm/version"
)

const (
	pmmInitLog             = "/srv/logs/pmm-init.log"
	updateCheckInterval    = 24 * time.Hour
	updateCheckResultFresh = updateCheckInterval + 10*time.Minute
	updateDefaultTimeout   = 30 * time.Second
	pmmEnvfilePath         = "/home/pmm/update/pmm-server.env"
	watchtowerEnvfilePath  = "/home/pmm/update/watchtower.env"
)

var fileName = "/etc/pmm-server-update-version.json"

// Updater is a service to check for updates and trigger the update process.
type Updater struct {
	l                  *logrus.Entry
	db                 *reform.DB
	watchtowerHost     *url.URL
	gRPCMessageMaxSize uint32

	performM sync.Mutex
	running  bool

	checkRW         sync.RWMutex
	lastCheckResult *version.DockerVersionInfo
	lastCheckTime   time.Time

	// releaseNotes holds a map of PMM server versions to their release notes.
	releaseNotes   map[string]string
	releaseNotesRW sync.RWMutex
}

// NewUpdater creates a new Updater service.
func NewUpdater(watchtowerHost *url.URL, gRPCMessageMaxSize uint32, db *reform.DB) *Updater {
	u := &Updater{
		l:                  logrus.WithField("service", "updater"),
		db:                 db,
		watchtowerHost:     watchtowerHost,
		gRPCMessageMaxSize: gRPCMessageMaxSize,
		releaseNotes:       make(map[string]string),
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

func (up *Updater) sendRequestToWatchtower(ctx context.Context, newImageName string, stopWatchtower bool) error {
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
	q.Set("stopWatchtower", strconv.FormatBool(stopWatchtower)) // We stop watchtower on AMI and OVF, because systemd will restart it with new image.
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

	if resp.StatusCode == http.StatusBadRequest ||
		resp.StatusCode == http.StatusPreconditionFailed {
		bytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return errors.Wrap(err, "failed to read response body")
		}
		return grpcstatus.Error(codes.FailedPrecondition, string(bytes))
	}
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
	settings, err := models.GetSettings(up.db)
	if err != nil {
		return grpcstatus.Error(codes.Internal, "failed to get PMM server settings")
	}

	if !settings.IsUpdatesEnabled() {
		up.l.Debug("Updates are disabled")
		return grpcstatus.Error(codes.FailedPrecondition, "updates are disabled")
	}
	if up.running {
		return grpcstatus.Error(codes.FailedPrecondition, "update already in progress")
	}
	up.running = true
	if newImageName == "" {
		up.running = false
		return errors.New("newImageName is empty")
	}

	err = up.checkWatchtowerHost()
	if err != nil {
		up.running = false
		up.l.WithError(err).Error("Failed to check watchtower host")
		return grpcstatus.Errorf(codes.FailedPrecondition, "failed to check watchtower host")
	}

	restartWatchtower := false
	if _, e := os.Stat(pmmEnvfilePath); e == nil {
		watchtowerImageName := strings.Replace(newImageName, "pmm-server-fb", "pmm-watchtower-fb", 1) // for FB images
		watchtowerImageName = strings.Replace(watchtowerImageName, "3-dev-latest", "dev-latest", 1)   // for dev images
		watchtowerImageName = strings.Replace(watchtowerImageName, "pmm-server", "watchtower", 1)
		err := up.updatePodmanEnvironmentVariables(watchtowerEnvfilePath, "WATCHTOWER_IMAGE", watchtowerImageName)
		if err != nil {
			up.running = false
			up.l.WithError(err).Error("Failed to update environment variables file for watchtower")
			return errors.Wrap(err, "failed to update environment variables file for watchtower")
		}
		err = up.updatePodmanEnvironmentVariables(pmmEnvfilePath, "PMM_IMAGE", newImageName)
		if err != nil {
			up.running = false
			up.l.WithError(err).Error("Failed to update environment variables file")
			return errors.Wrap(err, "failed to update environment variables file")
		}
		restartWatchtower = true
	} else if !os.IsNotExist(e) {
		up.running = false
		up.l.WithError(e).Error("Failed to check environment variables file")
		return errors.Wrap(e, "failed to check environment variables file")
	}

	if err := up.sendRequestToWatchtower(ctx, newImageName, restartWatchtower); err != nil {
		up.running = false
		up.l.WithError(err).Error("Failed to trigger update")
		return err
	}
	return nil
}

// ForceCheckUpdates forces an update check.
func (up *Updater) ForceCheckUpdates(ctx context.Context) error {
	return up.check(ctx)
}

// LastCheckUpdatesResult returns the result of the last update check.
func (up *Updater) LastCheckUpdatesResult(ctx context.Context) (*version.UpdateCheckResult, time.Time) {
	var latest version.DockerVersionInfo
	installed := up.InstalledPMMVersion()
	vi, lastCheckTime := up.checkResult(ctx)
	if vi != nil {
		latest = *vi
	}
	return &version.UpdateCheckResult{
		Installed:       installed,
		Latest:          latest,
		UpdateAvailable: latest.DockerImage != "",
		LatestNewsURL:   "https://per.co.na/pmm/" + latest.Version.String(),
	}, lastCheckTime
}

// ListUpdates returns the list of available versions between installed and latest.
func (up *Updater) ListUpdates(ctx context.Context) ([]*version.DockerVersionInfo, error) {
	all, _, err := up.latest(ctx)
	if err != nil {
		return nil, err
	}
	return all, nil
}

func (up *Updater) latest(ctx context.Context) ([]*version.DockerVersionInfo, *version.DockerVersionInfo, error) {
	settings, err := models.GetSettings(up.db)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to get PMM server settings")
	}

	if !settings.IsUpdatesEnabled() {
		return nil, nil, services.ErrPMMUpdatesDisabled
	}

	info, err := up.readFromFile()
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to read from file")
	}
	if info != nil {
		return nil, info, nil
	}
	if os.Getenv("PMM_DEV_UPDATE_DOCKER_IMAGE") != "" {
		return up.parseDockerTag(os.Getenv("PMM_DEV_UPDATE_DOCKER_IMAGE"))
	}

	// If file does not exist, and ENV variable is not set, go get the latest versions from Percona version service.
	return up.latestAvailableFromVersionService(ctx)
}

func (up *Updater) readFromFile() (*version.DockerVersionInfo, error) {
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
	Version   string    `json:"version"`
	ImageInfo imageInfo `json:"imageInfo"`
}

type imageInfo struct {
	ImagePath             string    `json:"imagePath"`
	ImageReleaseTimestamp time.Time `json:"imageReleaseTimestamp"`
}

// MetadataResponse is a response from the metadata endpoint on Percona version service.
type MetadataResponse struct {
	Versions []result `json:"versions"`
}

// ReleaseNotesResponse is a response from the release-notes endpoint on Percona version service.
type ReleaseNotesResponse struct {
	ReleaseNote string `json:"releaseNote"`
}

// latestAvailableFromVersionService queries Percona version service and returns:
// - list of versions between the installed version and the latest version (inclusive)
// - the latest available version (i.e., the latest minor version for the current major version).
// If the current version is the latest minor version, it returns the next major version as the latest.
// If the current version is the latest version, it returns the current version as the latest.
func (up *Updater) latestAvailableFromVersionService(ctx context.Context) ([]*version.DockerVersionInfo, *version.DockerVersionInfo, error) {
	versionServiceURL, err := envvars.GetPlatformAddress()
	if err != nil {
		up.l.WithError(err).Error("Failed to get version service address")
		return nil, nil, errors.Wrap(err, "failed to get version service address")
	}
	u := versionServiceURL + "/metadata/v2/pmm-server"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		up.l.WithError(err).Error("Failed to create request")
		return nil, nil, errors.Wrap(err, "failed to create request")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		up.l.WithError(err).Error("Failed to get PMM server versions")
		return nil, nil, errors.Wrap(err, "failed to get PMM server versions")
	}
	defer resp.Body.Close() //nolint:errcheck

	var metadataResponse MetadataResponse
	if err := json.NewDecoder(resp.Body).Decode(&metadataResponse); err != nil {
		up.l.WithError(err).Error("Failed to decode response")
		return nil, nil, errors.Wrap(err, "failed to decode response")
	}

	if len(metadataResponse.Versions) != 0 {
		up.l.Debugf("Found %d versions", len(metadataResponse.Versions))
		updates, next := up.next(*up.currentVersion(), metadataResponse.Versions)
		return updates, next, err
	}
	up.l.Debug("No new PMM version available")
	return nil, nil, nil
}

func (up *Updater) parseDockerTag(tag string) ([]*version.DockerVersionInfo, *version.DockerVersionInfo, error) {
	splitTag := strings.Split(tag, ":")
	if len(splitTag) != 2 {
		return nil, nil, fmt.Errorf("invalid tag: %s", tag)
	}
	parsed, err := version.Parse(splitTag[1])
	if err != nil {
		up.l.Debugf("Failed to parse version: %s", splitTag[1])
		return nil, &version.DockerVersionInfo{DockerImage: tag}, nil //nolint:nilerr
	}
	return nil, &version.DockerVersionInfo{
		Version:     *parsed,
		DockerImage: tag,
	}, nil
}

func (up *Updater) next(currentVersion version.Parsed, results []result) ([]*version.DockerVersionInfo, *version.DockerVersionInfo) {
	repo := os.Getenv("PMM_DEV_UPDATE_DOCKER_REPO")
	if repo == "" {
		repo = "percona/pmm-server"
	}
	nextMinor := &version.DockerVersionInfo{
		Version: currentVersion,
	}
	updates := version.DockerVersionsInfo{}
	var nextMajor *version.DockerVersionInfo
	for _, result := range results {
		v, err := version.Parse(result.Version)
		if err != nil {
			up.l.Debugf("Failed to parse version: %s", result.Version)
			continue
		}
		if !currentVersion.Less(v) {
			continue
		}
		releaseNotesURL := "https://per.co.na/pmm/" + v.String()
		releaseNote, err := up.getReleaseNotesText(context.Background(), *v)
		if err != nil {
			up.l.Errorf("Failed to get release notes for version: %s, %s", v.String(), err.Error())
		}

		dockerImage := result.ImageInfo.ImagePath
		if dockerImage == "" {
			dockerImage = repo + ":" + result.Version
		}
		// versions with pre-lease labels (e.g 2.40.1-rc) are not considered for the update diffs
		if v.Rest == "" && currentVersion.Less(v) {
			updates = append(updates, &version.DockerVersionInfo{
				Version:          *v,
				DockerImage:      dockerImage,
				BuildTime:        result.ImageInfo.ImageReleaseTimestamp,
				ReleaseNotesURL:  releaseNotesURL,
				ReleaseNotesText: releaseNote,
			})
		}

		if v.Major == currentVersion.Major && nextMinor.Version.Less(v) {
			nextMinor = &version.DockerVersionInfo{
				Version:          *v,
				DockerImage:      dockerImage,
				BuildTime:        result.ImageInfo.ImageReleaseTimestamp,
				ReleaseNotesURL:  releaseNotesURL,
				ReleaseNotesText: releaseNote,
			}
		}
		if v.Major > currentVersion.Major &&
			(nextMajor == nil || (nextMajor.Version.Less(v) && nextMajor.Version.Major == v.Major) || v.Major < nextMajor.Version.Major) {
			nextMajor = &version.DockerVersionInfo{
				Version:          *v,
				DockerImage:      dockerImage,
				BuildTime:        result.ImageInfo.ImageReleaseTimestamp,
				ReleaseNotesURL:  releaseNotesURL,
				ReleaseNotesText: releaseNote,
			}
		}
	}

	sort.Sort(updates)
	if nextMinor.Version == currentVersion && nextMajor != nil {
		return updates, nextMajor
	}
	return updates, nextMinor
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

	f, err := os.Open(pmmInitLog)
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
	_, latest, err := up.latest(ctx)
	if err != nil {
		if errors.Is(err, services.ErrPMMUpdatesDisabled) {
			up.l.Info("PMM updates are disabled")
			return grpcstatus.Error(codes.FailedPrecondition, "PMM updates are disabled")
		}
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

func (up *Updater) updatePodmanEnvironmentVariables(filename string, key string, imageName string) error {
	if len(strings.Split(imageName, "/")) < 3 {
		imageName = "docker.io/" + imageName
	}
	file, err := os.ReadFile(filename)
	if err != nil {
		return errors.Wrap(err, "failed to read file")
	}
	lines := strings.Split(string(file), "\n")
	for i, line := range lines {
		if strings.Contains(line, key) {
			lines[i] = fmt.Sprintf(key+"=%s", imageName)
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

// getReleaseNotesText is a placeholder for getting release notes in MarkDown format
// until we finalize the implementation on version service.
func (up *Updater) getReleaseNotesText(ctx context.Context, version version.Parsed) (string, error) {
	if version.Rest != "" {
		version.Rest = ""
	}

	up.releaseNotesRW.Lock()
	defer up.releaseNotesRW.Unlock()
	versionString := version.String()
	if releaseNotes, ok := up.releaseNotes[versionString]; ok {
		return releaseNotes, nil
	}

	versionServiceURL, err := envvars.GetPlatformAddress()
	if err != nil {
		up.l.WithError(err).Error("Failed to get version service address")
		return "", errors.Wrap(err, "failed to get version service address")
	}
	u := versionServiceURL + "/release-notes/v1/pmm/" + versionString
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		up.l.WithError(err).Error("Failed to create request")
		return "", errors.Wrap(err, "failed to create request")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		up.l.WithError(err).Errorf("Failed to get release note for version: %s", versionString)
		return "", errors.Wrapf(err, "failed to get release notes for version: %s", versionString)
	}

	if resp.StatusCode != http.StatusOK {
		up.l.Errorf("Failed to get release notes for PMM %s, got HTTP %d", version.String(), resp.StatusCode)
		return "", nil
	}
	defer resp.Body.Close() //nolint:errcheck
	var rnResponse ReleaseNotesResponse
	if err := json.NewDecoder(resp.Body).Decode(&rnResponse); err != nil {
		up.l.WithError(err).Error("Failed to decode response")
		return "", errors.Wrap(err, "failed to decode response")
	}

	up.releaseNotes[versionString] = rnResponse.ReleaseNote
	return rnResponse.ReleaseNote, nil
}
