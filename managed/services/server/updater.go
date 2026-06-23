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
	"errors"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

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
	updateCheckInterval    = 12 * time.Hour
	updateCheckResultFresh = updateCheckInterval + 10*time.Minute
)

var fileName = "/etc/pmm-server-update-version.json"

// Updater is a service to check for available PMM Server updates.
type Updater struct {
	l  *logrus.Entry
	db *reform.DB

	checkRW         sync.RWMutex
	lastCheckResult *version.DockerVersionInfo
	lastCheckTime   time.Time

	// releaseNotes holds a map of PMM server versions to their release notes.
	releaseNotes   map[string]string
	releaseNotesRW sync.RWMutex
}

// NewUpdater creates a new Updater service.
func NewUpdater(db *reform.DB) *Updater {
	u := &Updater{
		l:            logrus.WithField("service", "updater"),
		db:           db,
		releaseNotes: make(map[string]string),
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

func (up *Updater) currentVersion() *version.Parsed {
	return version.MustParse(version.Version)
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
		return nil, nil, fmt.Errorf("failed to get PMM server settings: %w", err)
	}

	if !settings.IsUpdatesEnabled() {
		return nil, nil, services.ErrPMMUpdatesDisabled
	}

	info, err := up.readFromFile()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read from file: %w", err)
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
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	if os.IsNotExist(err) {
		return nil, nil //nolint:nilnil
	}
	info := version.DockerVersionInfo{}
	err = json.Unmarshal(content, &info)
	if err != nil {
		up.l.WithError(err).Error("Failed to unmarshal file")
		return nil, fmt.Errorf("failed to unmarshal file: %w", err)
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
		return nil, nil, fmt.Errorf("failed to get version service address: %w", err)
	}
	u := versionServiceURL + "/metadata/v2/pmm-server"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		up.l.WithError(err).Error("Failed to create request")
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		up.l.WithError(err).Error("Failed to get PMM server versions")
		return nil, nil, fmt.Errorf("failed to get PMM server versions: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	var metadataResponse MetadataResponse
	err = json.NewDecoder(resp.Body).Decode(&metadataResponse)
	if err != nil {
		up.l.WithError(err).Error("Failed to decode response")
		return nil, nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(metadataResponse.Versions) != 0 {
		up.l.Debugf("Found %d versions", len(metadataResponse.Versions))
		updates, next := up.next(ctx, *up.currentVersion(), metadataResponse.Versions)
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

func (up *Updater) next(ctx context.Context, currentVersion version.Parsed, results []result) ([]*version.DockerVersionInfo, *version.DockerVersionInfo) {
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
		releaseNote, err := up.getReleaseNotesText(ctx, *v)
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
		return fmt.Errorf("failed to get latest version: %w", err)
	}
	up.lastCheckResult = latest
	up.lastCheckTime = time.Now()
	return nil
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
		return "", fmt.Errorf("failed to get version service address: %w", err)
	}
	u := versionServiceURL + "/release-notes/v1/pmm/" + versionString
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		up.l.WithError(err).Error("Failed to create request")
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		up.l.WithError(err).Errorf("Failed to get release note for version: %s", versionString)
		return "", fmt.Errorf("failed to get release notes for version: %s: %w", versionString, err)
	}

	if resp.StatusCode != http.StatusOK {
		up.l.Errorf("Failed to get release notes for PMM %s, got HTTP %d", version.String(), resp.StatusCode)
		return "", nil
	}
	defer resp.Body.Close() //nolint:errcheck
	var rnResponse ReleaseNotesResponse
	err = json.NewDecoder(resp.Body).Decode(&rnResponse)
	if err != nil {
		up.l.WithError(err).Error("Failed to decode response")
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	up.releaseNotes[versionString] = rnResponse.ReleaseNote
	return rnResponse.ReleaseNote, nil
}
