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

// Package distribution provides structures and methods to determine the distribution method and OS of the PMM Server.
package distribution

import (
	"bytes"
	"os"
	"regexp"

	pmmv1 "github.com/percona/saas/gen/telemetry/events/pmm"
	"github.com/sirupsen/logrus"

	serverv1 "github.com/percona/pmm/api/server/v1"
)

// Service provides methods to determine the distribution method and OS of the PMM Server.
type Service struct {
	distributionInfoFilePath string
	osInfoFilePath           string

	l *logrus.Entry
}

// NewService creates a new Distribution Service.
func NewService(distributionFilePath, osInfoFilePath string, l *logrus.Entry) *Service {
	return &Service{
		distributionInfoFilePath: distributionFilePath,
		osInfoFilePath:           osInfoFilePath,
		l:                        l,
	}
}

// GetDistributionMethodAndOS returns the distribution method and OS of the PMM Server.
func (d Service) GetDistributionMethodAndOS() (serverv1.DistributionMethod, pmmv1.DistributionMethod, string) {
	dm := os.Getenv("PMM_DISTRIBUTION_METHOD")
	if dm == "" {
		b, err := os.ReadFile(d.distributionInfoFilePath)
		if err != nil {
			d.l.Debugf("Failed to read %s: %s", d.distributionInfoFilePath, err)
		}

		b = bytes.ToLower(bytes.TrimSpace(b))
		dm = string(b)
	}
	switch dm {
	case "ovf":
		return serverv1.DistributionMethod_DISTRIBUTION_METHOD_OVF, pmmv1.DistributionMethod_OVF, "ovf"
	case "ami":
		return serverv1.DistributionMethod_DISTRIBUTION_METHOD_AMI, pmmv1.DistributionMethod_AMI, "ami"
	case "azure":
		return serverv1.DistributionMethod_DISTRIBUTION_METHOD_AZURE, pmmv1.DistributionMethod_AZURE, "azure"
	case "digitalocean":
		return serverv1.DistributionMethod_DISTRIBUTION_METHOD_DO, pmmv1.DistributionMethod_DO, "digitalocean"
	case "docker", "": // /srv/pmm-distribution does not exist in PMM 2.0.
		b, err := os.ReadFile(d.osInfoFilePath)
		if err != nil {
			d.l.Debugf("Failed to read %s: %s", d.osInfoFilePath, err)
		}
		return serverv1.DistributionMethod_DISTRIBUTION_METHOD_DOCKER, pmmv1.DistributionMethod_DOCKER, d.getLinuxDistribution(string(b))
	default:
		return serverv1.DistributionMethod_DISTRIBUTION_METHOD_UNSPECIFIED, pmmv1.DistributionMethod_DISTRIBUTION_METHOD_INVALID, ""
	}
}

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
func (d Service) getLinuxDistribution(procVersion string) string {
	for _, p := range procVersionRegexps {
		match := p.re.FindStringSubmatchIndex(procVersion)
		if match != nil {
			return string(p.re.ExpandString(nil, p.t, procVersion, match))
		}
	}
	return "unknown"
}
