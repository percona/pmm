package telemetry

import (
	"bytes"
	pmmv1 "github.com/percona-platform/saas/gen/telemetry/events/pmm"
	"github.com/percona/pmm/api/serverpb"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"regexp"
)

//go:generate ../../../bin/mockery -name=DistributionUtilService -case=snake -inpkg -testonly

type DistributionUtilService interface {
	getDistributionMethodAndOS(l *logrus.Entry) (serverpb.DistributionMethod, pmmv1.DistributionMethod, string)
	getLinuxDistribution(procVersion string) string
}

type distributionUtilServiceImpl struct{}

func (d distributionUtilServiceImpl) getDistributionMethodAndOS(l *logrus.Entry) (serverpb.DistributionMethod, pmmv1.DistributionMethod, string) {
	b, err := ioutil.ReadFile(distributionInfoFilePath)
	if err != nil {
		l.Debugf("Failed to read %s: %s", distributionInfoFilePath, err)
	}

	b = bytes.ToLower(bytes.TrimSpace(b))
	switch string(b) {
	case "ovf":
		return serverpb.DistributionMethod_OVF, pmmv1.DistributionMethod_OVF, "ovf"
	case "ami":
		return serverpb.DistributionMethod_AMI, pmmv1.DistributionMethod_AMI, "ami"
	case "azure":
		return serverpb.DistributionMethod_AZURE, pmmv1.DistributionMethod_AZURE, "azure"
	case "digitalocean":
		return serverpb.DistributionMethod_DO, pmmv1.DistributionMethod_DO, "digitalocean"
	case "docker", "": // /srv/pmm-distribution does not exist in PMM 2.0.
		if b, err = ioutil.ReadFile(osInfoFilePath); err != nil {
			l.Debugf("Failed to read %s: %s", osInfoFilePath, err)
		}
		return serverpb.DistributionMethod_DOCKER, pmmv1.DistributionMethod_DOCKER, d.getLinuxDistribution(string(b))
	default:
		return serverpb.DistributionMethod_DISTRIBUTION_METHOD_INVALID, pmmv1.DistributionMethod_DISTRIBUTION_METHOD_INVALID, ""
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
func (d distributionUtilServiceImpl) getLinuxDistribution(procVersion string) string {
	for _, p := range procVersionRegexps {
		match := p.re.FindStringSubmatchIndex(procVersion)
		if match != nil {
			return string(p.re.ExpandString(nil, p.t, procVersion, match))
		}
	}
	return "unknown"
}
