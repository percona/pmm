// Copyright 2019 Percona LLC
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

package client

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/admin/cli/flags"
	"github.com/percona/pmm/admin/commands"
)

// InstallCommand is used by Kong for CLI flags and commands.
type InstallCommand struct {
	InstallPath  string `default:"/usr/local/percona/pmm2" help:"Path where PMM Server shall be installed"`
	User         string `help:"Set file ownership instead of the current user"`
	Group        string `help:"Set group ownership instead of the current group"`
	Version      string `name:"use-version" help:"PMM Server version to install (default: latest)"`
	SkipChecksum bool   `help:"Skip checksum validation of the downloaded files"`
}

type installResult struct{}

// Result is a command run result.
func (res *installResult) Result() {}

// String stringifies command result.
func (res *installResult) String() string {
	return "ok"
}

// ErrSumsDontMatch is returned when checksums do not match.
var ErrSumsDontMatch = fmt.Errorf("SumsDontMatch")

// RunCmdWithContext runs install command.
func (c *InstallCommand) RunCmdWithContext(ctx context.Context, _ *flags.GlobalFlags) (commands.Result, error) {
	if c.Version == "" {
		latestVersion, err := c.getLatestVersion(ctx)
		if err != nil {
			return nil, err
		}
		c.Version = latestVersion
	}

	link := fmt.Sprintf(
		"https://downloads.percona.com/downloads/pmm2/%s/binary/tarball/pmm2-client-%s.tar.gz",
		c.Version,
		c.Version)

	logrus.Infof("Downloading %s", link)
	tarPath, err := c.downloadTarball(ctx, link)
	if err != nil {
		return nil, err
	}

	defer os.Remove(tarPath) //nolint:errcheck

	if !c.SkipChecksum {
		logrus.Infof("Verifying tarball %s", tarPath)
		ok, err := c.checksumTarball(ctx, link, tarPath)
		if err != nil {
			return nil, err
		}

		if !ok {
			return nil, fmt.Errorf("%w: downloaded file verification failed", ErrSumsDontMatch)
		}
	}

	logrus.Infof("Extracting tarball %s", tarPath)
	if err := c.extractTarball(tarPath); err != nil {
		return nil, err
	}

	extractedPath := path.Join(os.TempDir(), fmt.Sprintf("pmm2-client-%s", c.Version))
	defer os.RemoveAll(extractedPath) //nolint:errcheck

	if err := c.installTarball(extractedPath); err != nil {
		return nil, err
	}

	return &installResult{}, nil
}

// ErrLatestVersionNotFound is returned when we cannot determine what the latest version is.
var ErrLatestVersionNotFound = fmt.Errorf("LatestVersionNotFound")

func (c *InstallCommand) getLatestVersion(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, "https://github.com/percona/pmm/releases/latest", nil)
	if err != nil {
		return "", err
	}

	cl := &http.Client{ //nolint:exhaustruct
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Timeout: 10 * time.Second,
	}
	res, err := cl.Do(req)
	if err != nil {
		return "", err
	}

	defer res.Body.Close() //nolint:errcheck

	url, err := res.Location()
	if err != nil {
		logrus.Debug(err)
		return "", fmt.Errorf("%w: could not find latest version", ErrLatestVersionNotFound)
	}

	tag := path.Base(url.Path)
	latest := strings.TrimPrefix(tag, "v")

	return latest, nil
}

// ErrHTTPStatusNotOk is returned when HTTP call returns other than HTTP 200 response.
var ErrHTTPStatusNotOk = fmt.Errorf("HTTPStatusNotOk")

func (c *InstallCommand) downloadTarball(ctx context.Context, link string) (string, error) {
	base := path.Base(link)
	f, err := os.CreateTemp("", base)
	if err != nil {
		return "", err
	}

	defer f.Close() //nolint:gosec,errcheck

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, nil)
	if err != nil {
		return "", err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}

	defer res.Body.Close() //nolint:errcheck
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%w: cannot download installation tarball (http %d)", ErrHTTPStatusNotOk, res.StatusCode)
	}

	if _, err = io.Copy(f, res.Body); err != nil {
		return "", err
	}

	return f.Name(), nil
}

// ErrInvalidChecksum is returned when checksum cannot be extracted from sha256sum file.
var ErrInvalidChecksum = fmt.Errorf("InvalidChecksum")

func (c *InstallCommand) checksumTarball(ctx context.Context, link string, path string) (bool, error) {
	shaLink := link + ".sha256sum"
	logrus.Debugf("Downloading tarball sha256sum from %s", shaLink)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, shaLink, nil)
	if err != nil {
		return false, err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err
	}

	defer res.Body.Close() //nolint:errcheck
	if res.StatusCode != http.StatusOK {
		return false, fmt.Errorf("%w: cannot download tarball's sha256sum (http %d)", ErrHTTPStatusNotOk, res.StatusCode)
	}

	sumLine := &bytes.Buffer{}
	if _, err := io.Copy(sumLine, res.Body); err != nil {
		return false, err
	}

	sum, _, found := strings.Cut(sumLine.String(), " ")
	if !found {
		return false, fmt.Errorf("%w: invalid checksum", ErrInvalidChecksum)
	}

	logrus.Infof("Downloaded checksum %s", sum)

	f, err := os.Open(path) //nolint:gosec
	if err != nil {
		return false, err
	}

	defer f.Close() //nolint:errcheck,gosec

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return false, err
	}

	computedSum := hex.EncodeToString(h.Sum(nil))
	logrus.Infof("Computed sum %s", computedSum)
	if computedSum != sum {
		return false, nil
	}

	return true, nil
}

func (c *InstallCommand) extractTarball(tarPath string) error {
	if err := os.RemoveAll(path.Join(os.TempDir(), fmt.Sprintf("pmm2-client-%s", c.Version))); err != nil {
		return err
	}

	cmd := exec.Command("tar", "-C", os.TempDir(), "-zxvf", tarPath) //nolint:gosec
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

func (c *InstallCommand) installTarball(extractedPath string) error {
	logrus.Infof("Installing to %s", c.InstallPath)

	cmd := exec.Command(path.Join(extractedPath, "install_tarball")) //nolint:gosec
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if c.User != "" {
		cmd.Env = append(cmd.Env, "PMM_USER="+c.User)
	}

	if c.Group != "" {
		cmd.Env = append(cmd.Env, "PMM_GROUP="+c.Group)
	}

	if c.InstallPath != "" {
		cmd.Env = append(cmd.Env, "PMM_DIR="+c.InstallPath)
	}

	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}
