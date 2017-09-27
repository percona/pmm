// pmm-managed
// Copyright (C) 2017 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

// Package telemetry provides Call Home functionality.
package telemetry

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"time"

	guuid "github.com/google/uuid"
	"github.com/pkg/errors"
)

const (
	timeout = 5 * time.Second
)

// Service is responsible for interactions with Percona Call Home service.
type Service struct {
	UUID       string
	URL        string
	PMMVersion string
	OS         string
	Interval   time.Duration
}

// Run runs telemetry service, sending data every Config.Interval until context is canceled.
func (s *Service) Run(ctx context.Context) {
	ticker := time.NewTicker(s.Interval)
	defer ticker.Stop()

	for {
		s.runOnce(ctx)

		select {
		case <-ticker.C:
			// continue with next loop iteration
		case <-ctx.Done():
			return
		}
	}
}

func (s *Service) runOnce(ctx context.Context) bool {
	data := s.collectData()
	payload := s.makePayload(data)
	err := s.sendRequest(ctx, payload)
	return err == nil
}

func (s *Service) collectData() map[string]string {
	if s.OS == "" {
		b, _ := ioutil.ReadFile("/proc/version")
		s.OS = getLinuxDistribution(string(b))
	}
	return map[string]string{
		"PMM": s.PMMVersion,
		"OS":  s.OS,
	}
}

func (s *Service) makePayload(data map[string]string) []byte {
	var w bytes.Buffer

	for key, value := range data {
		w.WriteString(fmt.Sprintf("%s;%s;%s\n", s.UUID, key, value))
	}

	return w.Bytes()
}

func (s *Service) sendRequest(ctx context.Context, data []byte) error {
	body := bytes.NewReader(data)
	req, err := http.NewRequest("POST", s.URL, body)
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
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("status code %d", resp.StatusCode)
	}
	return nil
}

// GenerateUUID generates new UUID version 4 (random).
func GenerateUUID() (string, error) {
	uuid, err := guuid.NewRandom()
	if err != nil {
		return "", errors.Wrap(err, "can't generate UUID")
	}

	// Old telemetry IDs have only 32 chars in the table but UUIDs + "-" = 36
	cleanUUID := strings.Replace(uuid.String(), "-", "", -1)
	return cleanUUID, nil
}

var procVersionRegexps = map[*regexp.Regexp]string{
	regexp.MustCompile(`ubuntu\d+~(?P<version>\d+\.\d+)`): "Ubuntu ${version}",
	regexp.MustCompile(`\.fc(?P<version>\d+)\.`):          "Fedora ${version}",
	regexp.MustCompile(`dev\.centos\.org`):                "CentOS",
	regexp.MustCompile(`builduser@leming`):                "Arch",
	regexp.MustCompile(`\.amzn\d+\.`):                     "Amazon",
	regexp.MustCompile(`Microsoft`):                       "Microsoft",
}

// getLinuxDistribution detects Linux distribution and version from /proc/version information.
func getLinuxDistribution(procVersion string) string {
	for re, t := range procVersionRegexps {
		match := re.FindStringSubmatchIndex(procVersion)
		if match != nil {
			return string(re.ExpandString(nil, t, procVersion, match))
		}
	}
	return "unknown"
}
