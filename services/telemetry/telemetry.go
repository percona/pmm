// pmm-managed
// Copyright (C) 2017 Percona LLC
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

// Package telemetry provides Call Home functionality.
package telemetry

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm-managed/models"
)

const (
	interval   = 24 * time.Hour
	timeout    = 5 * time.Second
	defaultURL = "https://v.percona.com/"

	// environment variables that affect telemetry service
	envURL = "PERCONA_VERSION_CHECK_URL" // the same name as for the Toolkit
	envOS  = "TELEMETRY_OS"              // set by AMI and OVF, empty for Docker image
)

// Service is responsible for interactions with Percona Call Home service.
type Service struct {
	db         *reform.DB
	pmmVersion string
	l          *logrus.Entry

	os  string
	url string
}

// NewService creates a new service with given UUID and PMM version.
func NewService(db *reform.DB, pmmVersion string) *Service {
	return &Service{
		db:         db,
		pmmVersion: pmmVersion,
		l:          logrus.WithField("component", "telemetry"),
	}
}

func (s *Service) init() {
	if os := os.Getenv(envOS); os != "" {
		s.os = os
	} else {
		b, err := ioutil.ReadFile("/proc/version")
		if err != nil {
			s.l.Debugf("Failed to read /proc/version: %s", err)
		}
		s.os = getLinuxDistribution(string(b))
	}
	s.l.Debugf("Using %q as OS.", s.os)

	if u := os.Getenv(envURL); u != "" {
		s.url = u
	} else {
		s.url = defaultURL
	}
	s.l.Debugf("Using %q as the endpoint.", s.url)
}

// Run runs telemetry service, sending data every interval until context is canceled.
func (s *Service) Run(ctx context.Context) {
	s.init()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		if err := s.sendOnce(ctx); err != nil {
			s.l.Debugf("Telemetry info not send: %s.", err)
		}

		select {
		case <-ticker.C:
			// continue with next loop iteration
		case <-ctx.Done():
			return
		}
	}
}

func (s *Service) sendOnce(ctx context.Context) error {
	var settings *models.Settings
	err := s.db.InTransaction(func(tx *reform.TX) error {
		var e error
		if settings, e = models.GetSettings(tx); e != nil {
			return e
		}

		if settings.Telemetry.Disabled {
			return errors.New("disabled via settings")
		}
		if settings.Telemetry.UUID == "" {
			settings.Telemetry.UUID, e = generateUUID()
			if e != nil {
				return e
			}
			return models.SaveSettings(tx, settings)
		}
		return nil
	})
	if err != nil {
		return err
	}

	payload := s.makePayload(settings.Telemetry.UUID)
	return s.sendRequest(ctx, payload)
}

func (s *Service) makePayload(uuid string) []byte {
	var w bytes.Buffer
	fmt.Fprintf(&w, "%s;%s;%s\n", uuid, "OS", s.os)
	fmt.Fprintf(&w, "%s;%s;%s\n", uuid, "PMM", s.pmmVersion)
	return w.Bytes()
}

func (s *Service) sendRequest(ctx context.Context, data []byte) error {
	body := bytes.NewReader(data)
	req, err := http.NewRequest("POST", s.url, body)
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "plain/text")
	req.Header.Add("X-Percona-Toolkit-Tool", "pmm")

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req = req.WithContext(ctx)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != 200 {
		return fmt.Errorf("status code %d", resp.StatusCode)
	}
	return nil
}

func generateUUID() (string, error) {
	uuid, err := uuid.NewRandom()
	if err != nil {
		return "", errors.Wrap(err, "can't generate UUID")
	}

	// Old telemetry IDs have only 32 chars in the table but UUIDs + "-" = 36
	cleanUUID := strings.Replace(uuid.String(), "-", "", -1)
	return cleanUUID, nil
}

// Currently, we only detect OS (Linux distribution) version from the kernel version (/proc/version).
// For both AMI and OVF images this value is fixed by the environment variable and not autodetected –
// we know OS for them because we make those images ourselves.
// If/when we decide to support installation with "normal" Linux package managers (apt, yum, etc.),
// we could use the code that was there. See PMM-1333 and PMM-1507 in both git logs and Jira for details.

type pair struct {
	re *regexp.Regexp
	t  string
}

var procVersionRegexps = []pair{
	{regexp.MustCompile(`ubuntu\d+~(?P<version>\d+\.\d+)`), "Ubuntu ${version}"},
	{regexp.MustCompile(`ubuntu`), "Ubuntu"},
	{regexp.MustCompile(`Debian`), "Debian"},
	{regexp.MustCompile(`\.fc(?P<version>\d+)\.`), "Fedora ${version}"},
	{regexp.MustCompile(`\.centos\.`), "CentOS"},
	{regexp.MustCompile(`\-ARCH`), "Arch"},
	{regexp.MustCompile(`\-moby`), "Moby"},
	{regexp.MustCompile(`\.amzn\d+\.`), "Amazon"},
	{regexp.MustCompile(`Microsoft`), "Microsoft"},
}

// getLinuxDistribution detects Linux distribution and version from /proc/version information.
func getLinuxDistribution(procVersion string) string {
	for _, p := range procVersionRegexps {
		match := p.re.FindStringSubmatchIndex(procVersion)
		if match != nil {
			return string(p.re.ExpandString(nil, p.t, procVersion, match))
		}
	}
	return "unknown"
}
