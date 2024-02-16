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

type Updater struct {
	l                  *logrus.Entry
	supervisord        supervisordService
	watchtowerHostname *url.URL
}

func NewUpdater(supervisord supervisordService, watchtowerHostname *url.URL) *Updater {
	return &Updater{
		l:                  logrus.WithField("service", "updater"),
		supervisord:        supervisord,
		watchtowerHostname: watchtowerHostname,
	}
}

func (s *Updater) sendRequestToWatchtower(ctx context.Context, newImageName string) error {
	hostname, err := os.Hostname()
	if err != nil {
		return errors.Wrap(err, "failed to get hostname")
	}

	u, err := s.watchtowerHostname.Parse("/v1/update")
	if err != nil {
		return errors.Wrap(err, "failed to parse URL")
	}
	q := u.Query()
	q.Set("hostname", hostname)
	q.Set("newImageName", newImageName)
	u.RawQuery = q.Encode()

	// Create a new request
	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
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
	defer resp.Body.Close()

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

func (s *Updater) StartUpdate(ctx context.Context, newImageName string) error {
	if newImageName == "" {
		latest, err := s.latest()
		if err != nil {
			s.l.WithError(err).Error("Failed to get latest version")
			newImageName = "perconalab/pmm-server:3-dev-latest"
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

func (s *Updater) ForceCheckUpdates(ctx context.Context) error {
	return nil
}

type TagsResponse struct {
	Results []struct {
		Name string `json:"name"`
	} `json:"results"`
}

func (s *Updater) LastCheckUpdatesResult(ctx context.Context) (*version.UpdateCheckResult, time.Time) {
	buildTime, err := version.Time()
	if err != nil {
		s.l.WithError(err).Error("Failed to get build time")
		return nil, time.Now()
	}
	latest, err := s.latest()
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

func (s *Updater) latest() (*version.PackageInfo, error) {
	fileName := "/etc/pmm-server-update-version.json"
	content, err := os.ReadFile(fileName)
	switch {
	case err == nil:
		info := version.PackageInfo{}
		err = json.Unmarshal(content, &info)
		if err != nil {
			s.l.WithError(err).Error("Failed to unmarshal file")
			return nil, errors.Wrap(err, "failed to unmarshal file")
		}
		return &info, nil
	case err != nil && !os.IsNotExist(err):
		s.l.WithError(err).Error("Failed to read file")
		return nil, errors.Wrap(err, "failed to read file")
	case os.Getenv("PMM_SERVER_UPDATE_VERSION") != "":
		return s.parseDockerTag(os.Getenv("PMM_SERVER_UPDATE_VERSION")), nil
	default: // os.IsNotExist(err)
		// File does not exist, get the latest tag from DockerHub
		resp, err := http.Get("https://registry.hub.docker.com/v2/repositories/percona/pmm-server/tags/")
		if err != nil {
			s.l.WithError(err).Error("Failed to get tags from DockerHub")
			return nil, errors.Wrap(err, "failed to get tags from DockerHub")
		}
		defer resp.Body.Close()

		var tagsResponse TagsResponse
		if err := json.NewDecoder(resp.Body).Decode(&tagsResponse); err != nil {
			s.l.WithError(err).Error("Failed to decode response")
			return nil, errors.Wrap(err, "failed to decode response")
		}

		if len(tagsResponse.Results) > 0 {
			// Assuming the first tag is the latest
			return s.parseDockerTag(tagsResponse.Results[0].Name), nil
		}
		return nil, errors.New("no tags found")
	}
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
