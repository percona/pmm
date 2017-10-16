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
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/percona/pmm-managed/services/prometheus/internal"
)

// ensureNotBuiltIn returns error if given job name is built-in scrape job
// (i.e. present in prometheus.yml file, but not in Consul).
func ensureNotBuiltIn(consulData []ScrapeConfig, fileData []*internal.ScrapeConfig, jobName string) error {
	for _, sc := range consulData {
		if sc.JobName == jobName {
			return nil
		}
	}

	for _, sc := range fileData {
		if sc.JobName == jobName {
			return status.Errorf(codes.FailedPrecondition, "scrape config with job name %q is built-in", jobName)
		}
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

	if err = ensureNotBuiltIn(cu.consulData, cu.fileData, cfg.JobName); err != nil {
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

	fileDataI := -1
	for i, sc := range cu.fileData {
		if sc.JobName == jobName {
			fileDataI = i
			break
		}
	}
	if fileDataI < 0 {
		return status.Errorf(codes.FailedPrecondition, "scrape config with job name %q not found in configuration file", jobName)
	}

	cu.consulData = append(cu.consulData[:consulDataI], cu.consulData[consulDataI+1:]...)
	cu.fileData = append(cu.fileData[:fileDataI], cu.fileData[fileDataI+1:]...)
	return nil
}

func (cu *configUpdater) addStaticTargets(jobName string, targets []string) error {
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

	var staticConfig StaticConfig
	switch len(cu.consulData[consulDataI].StaticConfigs) {
	case 0:
		// nothing
	case 1:
		staticConfig = cu.consulData[consulDataI].StaticConfigs[0]
	default:
		msg := fmt.Sprintf(
			"scrape config with job name %q has %d static configs, that is not supported yet",
			jobName, len(cu.consulData[consulDataI].StaticConfigs),
		)
		return status.Error(codes.Unimplemented, msg)
	}
	for _, add := range targets {
		var found bool
		for _, t := range staticConfig.Targets {
			if t == add {
				found = true
				break
			}
		}
		if found {
			continue
		}
		staticConfig.Targets = append(staticConfig.Targets, add)
	}

	scrapeConfig := cu.consulData[consulDataI]
	scrapeConfig.StaticConfigs = []StaticConfig{staticConfig}
	cfg, err := convertScrapeConfig(&scrapeConfig)
	if err != nil {
		return err
	}

	fileDataI := -1
	for i, sc := range cu.fileData {
		if sc.JobName == jobName {
			fileDataI = i
			break
		}
	}
	if fileDataI < 0 {
		return status.Errorf(codes.FailedPrecondition, "scrape config with job name %q not found in configuration file", jobName)
	}

	cu.consulData[consulDataI] = scrapeConfig
	cu.fileData[fileDataI] = cfg
	return nil
}

func (cu *configUpdater) removeStaticTargets(jobName string, targets []string) error {
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

	var staticConfig StaticConfig
	switch len(cu.consulData[consulDataI].StaticConfigs) {
	case 0:
		// nothing
	case 1:
		staticConfig = cu.consulData[consulDataI].StaticConfigs[0]
	default:
		msg := fmt.Sprintf(
			"scrape config with job name %q has %d static configs, that is not supported yet",
			jobName, len(cu.consulData[consulDataI].StaticConfigs),
		)
		return status.Error(codes.Unimplemented, msg)
	}
	for _, remove := range targets {
		for i, t := range staticConfig.Targets {
			if t == remove {
				staticConfig.Targets = append(staticConfig.Targets[:i], staticConfig.Targets[i+1:]...)
				break
			}
		}
	}

	scrapeConfig := cu.consulData[consulDataI]
	scrapeConfig.StaticConfigs = []StaticConfig{staticConfig}
	cfg, err := convertScrapeConfig(&scrapeConfig)
	if err != nil {
		return err
	}

	fileDataI := -1
	for i, sc := range cu.fileData {
		if sc.JobName == jobName {
			fileDataI = i
			break
		}
	}
	if fileDataI < 0 {
		return status.Errorf(codes.FailedPrecondition, "scrape config with job name %q not found in configuration file", jobName)
	}

	cu.consulData[consulDataI] = scrapeConfig
	cu.fileData[fileDataI] = cfg
	return nil
}
