// pmm-update
// Copyright (C) 2019 Percona LLC
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

package yum

import (
	"context"
	"strings"
	"time"

	"github.com/percona/pmm/version"
	"github.com/pkg/errors"

	"github.com/percona/pmm-update/pkg/run"
)

const yumCancelTimeout = 30 * time.Second

// CheckVersions returns up-to-date versions information for a package with given name.
func CheckVersions(ctx context.Context, name string) (*version.UpdateCheckResult, error) {
	// http://man7.org/linux/man-pages/man8/yum.8.html#LIST_OPTIONS

	var res version.UpdateCheckResult

	cmdLine := "yum --verbose info installed " + name
	stdout, _, err := run.Run(ctx, yumCancelTimeout, cmdLine)
	if err != nil {
		return nil, errors.Wrapf(err, "%#q failed", cmdLine)
	}

	info, err := parseInfo(stdout)
	if err != nil {
		return nil, err
	}
	res.InstalledRPMVersion = fullVersion(info)
	res.InstalledRPMNiceVersion = niceVersion(info)
	installedTime, err := parseInfoTime(info["Buildtime"])
	if err == nil {
		res.InstalledTime = &installedTime
	}

	cmdLine = "yum --verbose info updates " + name
	stdout, stderr, err := run.Run(ctx, yumCancelTimeout, cmdLine)
	if err != nil {
		if strings.Contains(strings.Join(stderr, "\n"), "Error: No matching Packages to list") {
			// no update available, return the same values
			res.LatestRPMVersion = res.InstalledRPMVersion
			res.LatestRPMNiceVersion = res.InstalledRPMNiceVersion
			res.LatestRepo = info["From repo"]
			res.LatestTime = res.InstalledTime
			return &res, nil
		}

		return nil, errors.Wrapf(err, "%#q failed", cmdLine)
	}

	info, err = parseInfo(stdout)
	if err != nil {
		return nil, err
	}
	res.UpdateAvailable = true
	res.LatestRPMVersion = fullVersion(info)
	res.LatestRPMNiceVersion = niceVersion(info)
	res.LatestRepo = info["Repo"]
	latestTime, err := parseInfoTime(info["Buildtime"])
	if err == nil {
		res.LatestTime = &latestTime
	}

	return &res, nil
}

// UpdatePackage updates package with given name.
func UpdatePackage(ctx context.Context, name string) error {
	cmdLine := "yum update " + name
	_, _, err := run.Run(ctx, yumCancelTimeout, cmdLine)
	if err != nil {
		return errors.Wrapf(err, "%#q failed", cmdLine)
	}
	return nil
}
