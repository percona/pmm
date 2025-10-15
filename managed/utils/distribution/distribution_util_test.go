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

package distribution

import (
	"os"
	"testing"

	pmmv1 "github.com/percona/saas/gen/telemetry/events/pmm"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	serverv1 "github.com/percona/pmm/api/server/v1"
)

func Test_distributionUtilServiceImpl_getDistributionMethodAndOS(t *testing.T) {
	t.Parallel()
	const (
		ami          = "ami"
		ovf          = "ovf"
		azure        = "azure"
		digitalocean = "digitalocean"
		docker       = "docker"

		DistributionMethodDistributionMethodInvalid = 0
		DistributionMethodDOCKER                    = 1
		DistributionMethodOVF                       = 2
		DistributionMethodAMI                       = 3
		DistributionMethodAZURE                     = 4
		DistributionMethodDO                        = 5

		PmmV1DistributionMethodDistributionMethodInvalid = 0
		PmmV1DistributionMethodDOCKER                    = 1
		PmmV1DistributionMethodOVF                       = 2
		PmmV1DistributionMethodAMI                       = 3
		PmmV1DistributionMethodAZURE                     = 4
		PmmV1DistributionMethodDO                        = 5
	)

	logger := logrus.StandardLogger()
	logger.SetLevel(logrus.DebugLevel)
	logEntry := logrus.NewEntry(logger)

	tests := []struct {
		name             string
		distributionName string
		dockerVersion    string
		want             serverv1.DistributionMethod
		want1            pmmv1.DistributionMethod
		want2            string
	}{
		{
			name:             "should return AMI distribution method",
			distributionName: ami,
			want:             DistributionMethodAMI,
			want1:            PmmV1DistributionMethodAMI,
			want2:            "ami",
		},
		{
			name:             "should return OVF distribution method",
			distributionName: ovf,
			want:             DistributionMethodOVF,
			want1:            PmmV1DistributionMethodOVF,
			want2:            "ovf",
		},
		{
			name:             "should return Azure distribution method",
			distributionName: azure,
			want:             DistributionMethodAZURE,
			want1:            PmmV1DistributionMethodAZURE,
			want2:            "azure",
		},
		{
			name:             "should return Digital Ocean distribution method",
			distributionName: digitalocean,
			want:             DistributionMethodDO,
			want1:            PmmV1DistributionMethodDO,
			want2:            "digitalocean",
		},
		{
			name:             "should return Docker distribution method",
			distributionName: docker,
			dockerVersion:    "ubuntu",
			want:             DistributionMethodDOCKER,
			want1:            PmmV1DistributionMethodDOCKER,
			want2:            "Ubuntu",
		},
		{
			name:             "should return Docker distribution method",
			distributionName: "",
			dockerVersion:    "ubuntu",
			want:             DistributionMethodDOCKER,
			want1:            PmmV1DistributionMethodDOCKER,
			want2:            "Ubuntu",
		},
		{
			name:             "should return Invalid distribution method",
			distributionName: "invalid",
			want:             DistributionMethodDistributionMethodInvalid,
			want1:            PmmV1DistributionMethodDistributionMethodInvalid,
			want2:            "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f, err := writeToTmpFile(t, "", tt.distributionName)

			tmpDistributionFilePath := f.Name()
			tmpOsInfoFilePath := ""

			require.NoError(t, err)
			if tt.dockerVersion != "" {
				f2, err := writeToTmpFile(t, "", tt.dockerVersion)
				assert.NoError(t, err)

				tmpOsInfoFilePath = f2.Name()
			}

			d := NewService(tmpDistributionFilePath, tmpOsInfoFilePath, logEntry)
			got, got1, got2 := d.GetDistributionMethodAndOS()
			assert.Equalf(t, tt.want, got, "GetDistributionMethodAndOS() serverv1.DistributionMethod")
			assert.Equalf(t, tt.want1, got1, "GetDistributionMethodAndOS() pmmv1.DistributionMethod")
			assert.Equalf(t, tt.want2, got2, "GetDistributionMethodAndOS() name")
		})
	}
}

func writeToTmpFile(t *testing.T, tmpDistributionFile string, s string) (*os.File, error) {
	t.Helper()
	f, err := os.CreateTemp(tmpDistributionFile, "1")
	if err != nil {
		return nil, err
	}
	_, err = f.WriteString(s)
	if err != nil {
		return nil, err
	}

	t.Cleanup(func() {
		err := os.Remove(f.Name())
		assert.NoError(t, err)
	})
	return f, nil
}

func Test_distributionUtilServiceImpl_getLinuxDistribution(t *testing.T) {
	t.Parallel()
	const (
		tmpDistributionFile = "/tmp/distribution"
		tmpOsInfoFilePath   = "/tmp/version"
	)
	logger := logrus.StandardLogger()
	logger.SetLevel(logrus.DebugLevel)
	logEntry := logrus.NewEntry(logger)

	type args struct {
		procVersion string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "should return Ubuntu",
			args: args{procVersion: "ubuntu"},
			want: "Ubuntu",
		},
		{
			name: "should return Ubuntu 22.04",
			args: args{procVersion: "ubuntu1~22.04"},
			want: "Ubuntu 22.04",
		},
		{
			name: "should return Debian",
			args: args{procVersion: "Debian"},
			want: "Debian",
		},
		{
			name: "should return Fedora",
			args: args{procVersion: ".fc20."},
			want: "Fedora 20",
		},
		{
			name: "should return CentOS",
			args: args{procVersion: ".centos."},
			want: "CentOS",
		},
		{
			name: "should return Arch",
			args: args{procVersion: "-ARCH"},
			want: "Arch",
		},
		{
			name: "should return Moby",
			args: args{procVersion: "-moby"},
			want: "Moby",
		},
		{
			name: "should return Amazon",
			args: args{procVersion: ".amzn10."},
			want: "Amazon",
		},
		{
			name: "should return Microsoft",
			args: args{procVersion: "Microsoft"},
			want: "Microsoft",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			d := Service{
				distributionInfoFilePath: tmpDistributionFile,
				osInfoFilePath:           tmpOsInfoFilePath,
				l:                        logEntry,
			}
			assert.Equalf(t, tt.want, d.getLinuxDistribution(tt.args.procVersion), "getLinuxDistribution(%v)", tt.args.procVersion)
		})
	}
}
