// Copyright (C) 2024 Percona LLC
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

	"github.com/pkg/errors"

	"github.com/percona/pmm/update/pkg/run"
	"github.com/percona/pmm/version"
)

const (
	yumInfoCancelTimeout   = 30 * time.Second  // must be _much_ less than stopwaitsecs in supervisord config
	yumUpdateCancelTimeout = 120 * time.Second // must be less than stopwaitsecs in supervisord config
	changeLogURLPath       = "https://per.co.na/pmm/"
)

// http://man7.org/linux/man-pages/man8/yum.8.html#LIST_OPTIONS

// Installed returns current version information for a package with given name.
// It runs quickly.
func Installed(ctx context.Context, name string) (*version.UpdateInstalledResult, error) {
	cmdLine := "yum --verbose info installed " + name
	stdout, _, err := run.Run(ctx, yumInfoCancelTimeout, cmdLine, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "%#q failed", cmdLine)
	}

	info, err := parseInfo(stdout, "Name")
	if err != nil {
		return nil, err
	}
	res := version.PackageInfo{
		Version:     niceVersion(info),
		FullVersion: fullVersion(info),
		Repo:        info["From repo"],
	}
	buildTime, err := parseInfoTime(info["Buildtime"])
	if err == nil {
		res.BuildTime = &buildTime
	}
	return &version.UpdateInstalledResult{
		Installed: res,
	}, nil
}

// getReleaseTime returns date and time when repo was updated (packages published or repo got rebuilt).
func getReleaseTime(ctx context.Context, repo string) (string, error) {
	cmdLine := "yum repoinfo " + repo
	stdout, _, err := run.Run(ctx, yumInfoCancelTimeout, cmdLine, nil)
	if err != nil {
		return "", errors.Wrapf(err, "%#q failed", cmdLine)
	}

	info, err := parseInfo(stdout, "Repo-id")
	if err != nil {
		return "", err
	}

	if time, ok := info["Repo-updated"]; ok {
		return time, nil
	}

	return "", errors.New("Repo-updated field is not found in repoinfo")
}

// Check returns up-to-date versions information for a package with given name.
// It runs slowly.
func Check(ctx context.Context, name string) (*version.UpdateCheckResult, error) {
	repoPropName := "Repository" // default value for RHEL9

	installed, err := Installed(ctx, name)
	if err != nil {
		return nil, err
	}
	res := &version.UpdateCheckResult{
		Installed: installed.Installed,
	}

	cmdLine := "yum --verbose info updates " + name
	stdout, stderr, err := run.Run(ctx, yumInfoCancelTimeout, cmdLine, nil)
	if err != nil {
		if strings.Contains(strings.Join(stderr, "\n"), "Error: No matching Packages to list") {
			// no update available, return the same values
			res.Latest = res.Installed
			return res, nil
		}

		return nil, errors.Wrapf(err, "%#q failed", cmdLine)
	}

	info, err := parseInfo(stdout, "Name")
	if err != nil {
		return nil, err
	}

	v, err := getRHELVersion()
	if err == nil && v == "7" {
		// change the prop name for EL7
		repoPropName = "Repo"
	}
	repo, ok := info[repoPropName]
	if !ok {
		return nil, errors.New("Repository field is not found in yum info")
	}

	res.Latest = version.PackageInfo{
		Version:     niceVersion(info),
		FullVersion: fullVersion(info),
		Repo:        repo,
	}

	// replace Buildtime with repo release time to show time of release.
	repoUpdated, err := getReleaseTime(ctx, repo)
	if err != nil {
		return nil, err
	}
	releaseTime, err := parseInfoTime(repoUpdated)
	if err == nil {
		res.Latest.BuildTime = &releaseTime
	}

	res.LatestNewsURL = changeLogURLPath + res.Latest.Version

	res.UpdateAvailable = true
	return res, nil
}

// Update updates package with given name.
func Update(ctx context.Context, name string) error {
	cmdLine := "yum update --assumeyes " + name
	_, _, err := run.Run(ctx, yumUpdateCancelTimeout, cmdLine, nil)
	if err != nil {
		return errors.Wrapf(err, "%#q failed", cmdLine)
	}
	return nil
}
