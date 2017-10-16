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

package prometheus

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/percona/pmm-managed/services/prometheus/internal"
)

// returns index and config if found, (0, nil) otherwise
func findScrapeConfigByJobName(fileData []*internal.ScrapeConfig, jobName string) (int, *ScrapeConfig) {
	for i, sc := range fileData {
		if sc.JobName == jobName {
			return i, convertInternalScrapeConfig(sc)
		}
	}
	return 0, nil
}

// ensureNotBuiltIn returns error if given job name is built-in scrape job
// (i.e. present in prometheus.yml file, but not in Consul).
func ensureNotBuiltIn(fileData []*internal.ScrapeConfig, consulData []ScrapeConfig, jobName string) error {
	for _, sc := range consulData {
		if sc.JobName == jobName {
			return nil
		}
	}

	if _, sc := findScrapeConfigByJobName(fileData, jobName); sc != nil {
		return status.Errorf(codes.FailedPrecondition, "scrape config with job name %q is built-in", jobName)
	}
	return nil
}

// configUpdater implements Prometheus configuration updating logic:
// it changes both sources while keeping them in sync.
// Input-output is done in Service.
type configUpdater struct {
	consulData []ScrapeConfig
	fileData   []*internal.ScrapeConfig
}

func (cu *configUpdater) addScrapeConfig(scrapeConfig *ScrapeConfig) error {
	cfg, err := convertScrapeConfig(scrapeConfig)
	if err != nil {
		return err
	}

	for _, sc := range cu.consulData {
		if sc.JobName == cfg.JobName {
			return status.Errorf(codes.AlreadyExists, "scrape config with job name %q already exist", cfg.JobName)
		}
	}

	if err = ensureNotBuiltIn(cu.fileData, cu.consulData, cfg.JobName); err != nil {
		return err
	}

	cu.consulData = append(cu.consulData, *scrapeConfig)
	cu.fileData = append(cu.fileData, cfg)
	return nil
}

func (cu *configUpdater) removeScrapeConfig(jobName string) error {
	consulDataI := -1
	for i, sc := range cu.consulData {
		if sc.JobName == jobName {
			consulDataI = i
			break
		}
	}
	if consulDataI < 0 {
		return status.Errorf(codes.NotFound, "scrape config with job name %q not found", jobName)
	}

	fileDataI, cfg := findScrapeConfigByJobName(cu.fileData, jobName)
	if cfg == nil {
		return status.Errorf(codes.FailedPrecondition, "scrape config with job name %q not found in configuration file", jobName)
	}

	cu.consulData = append(cu.consulData[:consulDataI], cu.consulData[consulDataI+1:]...)
	cu.fileData = append(cu.fileData[:fileDataI], cu.fileData[fileDataI+1:]...)
	return nil
}
