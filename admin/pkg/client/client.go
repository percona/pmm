// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package client provides common methods for working with pmm-client.
package client

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/admin/pkg/common"
)

// DistributionType represents type of distribution of the pmm-agent.
type DistributionType string

const (
	// Unknown represents unknown distribution type of PMM Agent or Server.
	Unknown DistributionType = "unknown"
	// Docker represents Docker installation of PMM Agent or Server.
	Docker DistributionType = "docker"
	// PackageManager represents installation of PMM Agent or Server via a package manager.
	PackageManager DistributionType = "package-manager"
	// Tarball represents installation of PMM Agent or Server via a tarball.
	Tarball DistributionType = "tarball"
)

// ErrLatestVersionNotFound is returned when we cannot determine what the latest version is.
var ErrLatestVersionNotFound = fmt.Errorf("LatestVersionNotFound")

// GetLatestVersion retrieves latest version of pmm-client available.
func GetLatestVersion(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, "https://github.com/percona/pmm/releases/latest", nil)
	if err != nil {
		return "", err
	}

	cl := &http.Client{ //nolint:exhaustruct
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Timeout: 10 * time.Second,
	}
	res, err := cl.Do(req)
	if err != nil {
		return "", err
	}

	defer res.Body.Close() //nolint:errcheck,gosec,nolintlint

	url, err := res.Location()
	if err != nil {
		logrus.Debug(err)
		return "", fmt.Errorf("%w: could not find latest version", ErrLatestVersionNotFound)
	}

	tag := path.Base(url.Path)
	latest := strings.TrimPrefix(tag, "v")

	return latest, nil
}

// DetectDistributionType detects distribution type of pmm-agent.
func DetectDistributionType(ctx context.Context, tarballInstallPath string) (DistributionType, error) {
	// Check tarball
	isTarball, err := detectTarballDistribution(tarballInstallPath)
	if err != nil {
		return Unknown, err
	}

	if isTarball {
		logrus.Debug("Found pmm-client installed via tarball")
		return Tarball, nil
	}

	// Check package manager
	isPm, err := checkPackageManager(ctx)
	if err != nil {
		return Unknown, err
	}

	if isPm {
		logrus.Debug("Found pmm-client installed via a package manager")
		return PackageManager, nil
	}

	return Unknown, nil
}

func checkPackageManager(ctx context.Context) (bool, error) {
	pm, err := common.DetectPackageManager()
	if err != nil {
		return false, err
	}

	if pm != common.UnknownPackageManager {
		pmInstallation, err := detectPackageManagerInstallation(ctx, pm)
		if err != nil {
			return false, err
		}

		if pmInstallation {
			return true, nil
		}
	}

	return false, nil
}

func detectTarballDistribution(tarballInstallPath string) (bool, error) {
	p := "/usr/local/percona/pmm"
	if tarballInstallPath != "" {
		p = tarballInstallPath
	}

	data, err := os.ReadFile(path.Join(p, "pmm-distribution"))
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	trimmedData := strings.TrimSpace(string(data))
	if trimmedData != string(Tarball) {
		return false, nil
	}

	return true, nil
}

func detectPackageManagerInstallation(ctx context.Context, pm common.PackageManager) (bool, error) {
	var cmds [][]string
	switch pm {
	case common.Dnf:
		cmds = [][]string{
			{"dnf", "list", "installed", "pmm-client"},
		}
	case common.Yum:
		cmds = [][]string{
			{"yum", "list", "installed", "pmm-client"},
		}
	case common.Apt:
		return queryDpkg(ctx)
	default:
		return false, nil
	}

	for _, cmd := range cmds {
		logrus.Infof("Running command %q", strings.Join(cmd, " "))

		cmd := exec.CommandContext(ctx, cmd[0], cmd[1:]...) //nolint:gosec
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			var exitErr *exec.ExitError
			if ok := errors.As(err, &exitErr); ok {
				if exitErr.ExitCode() == 1 {
					// This means the package has not been found
					return false, nil
				}
			}
			return false, err
		}
	}

	return true, nil
}

func queryDpkg(ctx context.Context) (bool, error) {
	cmd := exec.CommandContext(
		ctx,
		"dpkg-query",
		"--show",
		"-f=${Package}\t${db:Status-Status}\n",
		"pmm-client")

	out, err := cmd.CombinedOutput()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.Is(err, exitErr) && bytes.Contains(out, []byte("no packages found matching")) {
			return false, nil
		}
		return false, err
	}

	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		data := strings.Split(scanner.Text(), "\t")
		if data[1] == "installed" {
			return true, nil
		}
	}

	return false, nil
}
