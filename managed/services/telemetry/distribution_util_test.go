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

package telemetry

import (
	"fmt"
	"io/fs"
	"os"
	"testing"

	"github.com/google/uuid"
	pmmv1 "github.com/percona-platform/saas/gen/telemetry/events/pmm"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/percona/pmm/api/serverpb"
)

func Test_distributionUtilServiceImpl_getDistributionMethodAndOS(t *testing.T) {
	const (
		tmpDistributionFilePathPrefix = "/tmp/distribution"
		tmpOsInfoFilePathPrefix       = "/tmp/version"

		ami          = "ami"
		ovf          = "ovf"
		azure        = "azure"
		digitalocean = "digitalocean"
		docker       = "docker"

		DistributionMethod_DISTRIBUTION_METHOD_INVALID = 0
		DistributionMethod_DOCKER                      = 1
		DistributionMethod_OVF                         = 2
		DistributionMethod_AMI                         = 3
		DistributionMethod_AZURE                       = 4
		DistributionMethod_DO                          = 5

		PmmV1DistributionMethod_DISTRIBUTION_METHOD_INVALID = 0
		PmmV1DistributionMethod_DOCKER                      = 1
		PmmV1DistributionMethod_OVF                         = 2
		PmmV1DistributionMethod_AMI                         = 3
		PmmV1DistributionMethod_AZURE                       = 4
		PmmV1DistributionMethod_DO                          = 5
	)

	logger := logrus.StandardLogger()
	logger.SetLevel(logrus.DebugLevel)
	logEntry := logrus.NewEntry(logger)

	tests := []struct {
		name             string
		distributionName string
		dokcerOsVersion  string
		want             serverpb.DistributionMethod
		want1            pmmv1.DistributionMethod
		want2            string
	}{
		{
			name:             "should return AMI distribution method",
			distributionName: ami,
			want:             DistributionMethod_AMI,
			want1:            PmmV1DistributionMethod_AMI,
			want2:            "ami",
		},
		{
			name:             "should return OVF distribution method",
			distributionName: ovf,
			want:             DistributionMethod_OVF,
			want1:            PmmV1DistributionMethod_OVF,
			want2:            "ovf",
		},
		{
			name:             "should return Azure distribution method",
			distributionName: azure,
			want:             DistributionMethod_AZURE,
			want1:            PmmV1DistributionMethod_AZURE,
			want2:            "azure",
		},
		{
			name:             "should return Digital Ocean distribution method",
			distributionName: digitalocean,
			want:             DistributionMethod_DO,
			want1:            PmmV1DistributionMethod_DO,
			want2:            "digitalocean",
		},
		{
			name:             "should return Docker distribution method",
			distributionName: docker,
			dokcerOsVersion:  "ubuntu",
			want:             DistributionMethod_DOCKER,
			want1:            PmmV1DistributionMethod_DOCKER,
			want2:            "Ubuntu",
		},
		{
			name:             "should return Docker distribution method",
			distributionName: "",
			dokcerOsVersion:  "ubuntu",
			want:             DistributionMethod_DOCKER,
			want1:            PmmV1DistributionMethod_DOCKER,
			want2:            "Ubuntu",
		},
		{
			name:             "should return Invalid distribution method",
			distributionName: "invalid",
			want:             DistributionMethod_DISTRIBUTION_METHOD_INVALID,
			want1:            PmmV1DistributionMethod_DISTRIBUTION_METHOD_INVALID,
			want2:            "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// make sure that each test will run messages in different files
			suffixFileName := uuid.New().String()
			tmpDistributionFilePath := fmt.Sprintf("%s%s", tmpDistributionFilePathPrefix, suffixFileName)
			tmpOsInfoFilePath := fmt.Sprintf("%s%s", tmpOsInfoFilePathPrefix, suffixFileName)

			err := writeToFile(tmpDistributionFilePath, tt.distributionName)
			assert.NoError(t, err)
			if tt.dokcerOsVersion != "" {
				err := writeToFile(tmpOsInfoFilePath, tt.dokcerOsVersion)
				assert.NoError(t, err)
			}

			d := NewDistributionUtilServiceImpl(tmpDistributionFilePath, tmpOsInfoFilePath, logEntry)
			got, got1, got2 := d.getDistributionMethodAndOS()
			assert.Equalf(t, tt.want, got, "getDistributionMethodAndOS() serverpb.DistributionMethod")
			assert.Equalf(t, tt.want1, got1, "getDistributionMethodAndOS() pmmv1.DistributionMethod")
			assert.Equalf(t, tt.want2, got2, "getDistributionMethodAndOS() name")
		})
	}
}

func writeToFile(tmpDistributionFile string, s string) error {
	err := os.WriteFile(tmpDistributionFile, []byte(s), fs.ModePerm)
	if err != nil {
		return err
	}
	return nil
}

func Test_distributionUtilServiceImpl_getLinuxDistribution(t *testing.T) {
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

			d := distributionUtilServiceImpl{
				distributionInfoFilePath: tmpDistributionFile,
				osInfoFilePath:           tmpOsInfoFilePath,
				l:                        logEntry,
			}
			assert.Equalf(t, tt.want, d.getLinuxDistribution(tt.args.procVersion), "getLinuxDistribution(%v)", tt.args.procVersion)
		})
	}
}
