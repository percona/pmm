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
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/admin/commands"
)

// InstallCommand is used by Kong for CLI flags and commands.
type InstallCommand struct {
	InstallPath  string `default:"/usr/local/percona/pmm2" help:"Path where PMM Server shall be installed"`
	User         string `help:"Set files ownership instead of current user"`
	Group        string `help:"Set group ownership instead of current group"`
	SkipChecksum bool   `help:"Skip checksum validation of the downloaded files"`
}

type installResult struct{}

// Result is a command run result.
func (res *installResult) Result() {}

// String stringifies command result.
func (res *installResult) String() string {
	return "works"
}

var ErrSumDontMatch = fmt.Errorf("SumsDontMatch")

// RunCmd runs install command.
func (c *InstallCommand) RunCmd() (commands.Result, error) {
	latestVersion, err := c.getLatestVersion()
	if err != nil {
		return nil, err
	}

	link := fmt.Sprintf(
		"https://downloads.percona.com/downloads/pmm2/%s/binary/tarball/pmm2-client-%s.tar.gz",
		latestVersion,
		latestVersion,
	)

	logrus.Infof("Downloading %s", link)
	tarPath, err := c.downloadTarball(link)
	if err != nil {
		return nil, err
	}

	defer os.Remove(tarPath)

	if !c.SkipChecksum {
		logrus.Info("Verifying tarball %s", tarPath)
		ok, err := c.checksumTarball(link, tarPath)
		if err != nil {
			return nil, err
		}

		if !ok {
			return nil, fmt.Errorf("%w: downloaded file verification failed", ErrSumDontMatch)
		}
	}

	logrus.Infof("Extracting tarball %s", tarPath)
	if err := c.extractTarball(tarPath, latestVersion); err != nil {
		return nil, err
	}

	dstPath := path.Join(os.TempDir(), fmt.Sprintf("pmm2-client-%s", latestVersion))
	defer os.RemoveAll(dstPath)

	if err := c.installTarball(dstPath); err != nil {
		return nil, err
	}

	return &installResult{}, nil
}

var ErrLatestVersionNotFound = fmt.Errorf("LatestVersionNotFound")

func (c *InstallCommand) getLatestVersion() (string, error) {
	req, err := http.NewRequest(http.MethodHead, "https://github.com/percona/pmm/releases/latest", nil)
	if err != nil {
		return "", err
	}

	cl := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	res, err := cl.Do(req)
	if err != nil {
		return "", err
	}

	defer res.Body.Close()

	url, err := res.Location()
	if err != nil {
		logrus.Debug(err)
		return "", fmt.Errorf("%w: could not find latest version", ErrLatestVersionNotFound)
	}

	tag := path.Base(url.Path)
	latest := strings.TrimPrefix(tag, "v")

	return latest, nil
}

var ErrHTTPStatusNotOk = fmt.Errorf("HTTPStatusNotOk")

func (c *InstallCommand) downloadTarball(link string) (string, error) {
	base := path.Base(link)
	f, err := os.CreateTemp("", base)
	if err != nil {
		return "", err
	}

	defer f.Close()

	res, err := http.Get(link)
	if err != nil {
		return "", err
	}

	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%w: cannot download latest tarball (http %d)", ErrHTTPStatusNotOk, res.StatusCode)
	}

	if _, err = io.Copy(f, res.Body); err != nil {
		return "", err
	}

	return f.Name(), nil
}

var ErrInvalidChecksum = fmt.Errorf("InvalidChecksum")

func (c *InstallCommand) checksumTarball(link string, path string) (bool, error) {
	res, err := http.Get(link + ".sha256sum")
	if err != nil {
		return false, err
	}

	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return false, fmt.Errorf("%w: cannot download tarball's sha256sum (http %d)", ErrHTTPStatusNotOk, res.StatusCode)
	}

	sumLine := bytes.NewBuffer([]byte{})
	io.Copy(sumLine, res.Body)

	sum, _, found := strings.Cut(sumLine.String(), " ")
	if !found {
		return false, fmt.Errorf("%w: invalid checksum", ErrInvalidChecksum)
	}

	logrus.Infof("Downloaded checksum %s", sum)

	f, err := os.Open(path)
	if err != nil {
		return false, err
	}

	defer f.Close()

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

func (c *InstallCommand) extractTarball(tarPath, latestVersion string) error {
	if err := os.RemoveAll(path.Join(os.TempDir(), fmt.Sprintf("pmm2-client-%s", latestVersion))); err != nil {
		return err
	}

	cmd := exec.Command("tar", "-C", os.TempDir(), "-zxf", tarPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

func (c *InstallCommand) installTarball(extractedPath string) error {
	logrus.Infof("Installing to %s", c.InstallPath)

	cmd := exec.Command(path.Join(extractedPath, "install_tarball"))
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
