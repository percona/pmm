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

// Package tarball holds logic for pmm-client tarball specific operations.
package tarball

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/admin/pkg/client"
)

// Base represents base structure for interacting with tarball.
type Base struct {
	// InstallPath is destination where pmm-client is installed
	InstallPath string
	// User sets file ownership instead of the current user
	User string
	// Group sets group ownership instead of the current group
	Group string
	// Version install. Defaults to latest
	Version string
	// SkipChecksum skips tarball checksum validation
	SkipChecksum bool
	// IsUpgrade represents if pmm-client shall be upgraded
	IsUpgrade bool
}

// Install installs pmm-client from tarball.
func (b *Base) Install(ctx context.Context) error {
	if b.Version == "" {
		latestVersion, err := client.GetLatestVersion(ctx)
		if err != nil {
			return err
		}
		b.Version = latestVersion
	}

	link := fmt.Sprintf(
		"https://downloads.percona.com/downloads/pmm2/%s/binary/tarball/pmm2-client-%s.tar.gz",
		b.Version,
		b.Version)

	logrus.Infof("Downloading %s", link)
	tarPath, err := b.downloadTarball(ctx, link)
	if err != nil {
		return err
	}

	defer os.Remove(tarPath) //nolint:errcheck

	if !b.SkipChecksum {
		logrus.Infof("Verifying tarball %s", tarPath)
		if err := b.checksumTarball(ctx, link, tarPath); err != nil {
			return err
		}
	}

	logrus.Infof("Extracting tarball %s", tarPath)
	dir, err := os.MkdirTemp("", "pmm-client")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	if err := b.extractTarball(tarPath, dir); err != nil {
		return err
	}

	extractedPath := path.Join(dir, fmt.Sprintf("pmm2-client-%s", b.Version))

	if err := b.installTarball(ctx, extractedPath); err != nil {
		return err
	}

	return nil
}

// ErrHTTPStatusNotOk is returned when HTTP call returns other than HTTP 200 response.
var ErrHTTPStatusNotOk = fmt.Errorf("HTTPStatusNotOk")

func (b *Base) downloadTarball(ctx context.Context, link string) (string, error) {
	base := path.Base(link)
	f, err := os.CreateTemp("", base)
	if err != nil {
		return "", err
	}

	defer f.Close() //nolint:gosec,errcheck,nolintlint

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, nil)
	if err != nil {
		return "", err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}

	defer res.Body.Close() //nolint:errcheck,gosec,nolintlint
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

// ErrSumsDontMatch is returned when checksums do not match.
var ErrSumsDontMatch = fmt.Errorf("SumsDontMatch")

func (b *Base) checksumTarball(ctx context.Context, link string, path string) error {
	shaLink := link + ".sha256sum"
	logrus.Debugf("Downloading tarball sha256sum from %s", shaLink)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, shaLink, nil)
	if err != nil {
		return err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close() //nolint:gosec,errcheck,nolintlint
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: cannot download tarball's sha256sum (http %d)", ErrHTTPStatusNotOk, res.StatusCode)
	}

	sumLine := &bytes.Buffer{}
	if _, err := io.Copy(sumLine, res.Body); err != nil {
		return err
	}

	sum, _, found := strings.Cut(sumLine.String(), " ")
	if !found {
		return fmt.Errorf("%w: invalid checksum", ErrInvalidChecksum)
	}

	logrus.Infof("Downloaded checksum %s", sum)

	f, err := os.Open(path) //nolint:gosec
	if err != nil {
		return err
	}

	defer f.Close() //nolint:errcheck,gosec,nolintlint

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}

	computedSum := hex.EncodeToString(h.Sum(nil))
	logrus.Infof("Computed sum %s", computedSum)
	if computedSum != sum {
		return fmt.Errorf("%w: downloaded file verification failed", ErrSumsDontMatch)
	}

	return nil
}

func (b *Base) extractTarball(tarPath, targetDir string) error {
	readFile, err := os.Open(tarPath) //nolint:gosec
	if err != nil {
		return err
	}

	defer readFile.Close() //nolint:gosec

	reader, err := gzip.NewReader(readFile)
	if err != nil {
		return err
	}

	defer reader.Close()

	tarReader := tar.NewReader(reader)

	for {
		hdr, err := tarReader.Next()
		if err == io.EOF {
			// end of tar archive
			break
		}
		if err != nil {
			return err
		}

		hdrPath := path.Join(targetDir, hdr.Name) //nolint:gosec

		abs, err := filepath.Abs(hdrPath)
		if err != nil {
			return err
		}

		if !strings.HasPrefix(abs, targetDir) {
			return errors.Errorf("failed to extract %s file as the resolved path is outside of the destination folder", hdr.Name)
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			logrus.Infof("Creating dir:    %s", hdr.Name)

			err = os.MkdirAll(abs, os.FileMode(hdr.Mode))
			if err != nil {
				return err
			}
		case tar.TypeReg:
			logrus.Infof("Extracting file: %s", hdr.Name)

			w, err := os.OpenFile(abs, os.O_CREATE|os.O_RDWR, os.FileMode(hdr.Mode)) //nolint:gosec
			if err != nil {
				return err
			}

			_, err = io.Copy(w, tarReader) //nolint:gosec
			if err != nil {
				return err
			}

			w.Close()
		}
	}

	return nil
}

func (b *Base) installTarball(ctx context.Context, extractedPath string) error {
	logrus.Infof("Installing to %s", b.InstallPath)

	args := make([]string, 0, 2)
	args = append(args, path.Join(extractedPath, "install_tarball"))
	if b.IsUpgrade {
		args = append(args, "-u")
	}

	logrus.Infof("Running command %q", strings.Join(args, " "))
	cmd := exec.CommandContext(ctx, args[0], args[1:]...) //nolint:gosec
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if b.User != "" {
		cmd.Env = append(cmd.Env, "PMM_USER="+b.User)
	}

	if b.Group != "" {
		cmd.Env = append(cmd.Env, "PMM_GROUP="+b.Group)
	}

	if b.InstallPath != "" {
		cmd.Env = append(cmd.Env, "PMM_DIR="+b.InstallPath)
	}

	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}
