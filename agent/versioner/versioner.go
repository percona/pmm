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

// Package versioner contains Versioner component that is responsible for version retrieving of different software.
package versioner

import (
	"context"
	"os/exec"
	"regexp"
	"time"

	"github.com/pkg/errors"
)

const (
	versionCheckTimeout = 5 * time.Second
	mysqldBin           = "mysqld"
	xtrabackupBin       = "xtrabackup"
	xbcloudBin          = "xbcloud"
	qpressBin           = "qpress"
	mongodbBin          = "mongod"
	pbmBin              = "pbm"
)

var (
	mysqldVersionRegexp     = regexp.MustCompile("^.*Ver ([!-~]*).*")
	xtrabackupVersionRegexp = regexp.MustCompile("xtrabackup version ([!-~]*).*")
	xbcloudVersionRegexp    = regexp.MustCompile("^xbcloud[ ][ ]Ver ([!-~]*).*")
	qpressVersionRegexp     = regexp.MustCompile("^qpress[ ]([!-~]*).*")
	mongodbVersionRegexp    = regexp.MustCompile("^db version v([!-~]*).*")
	pbmVersionRegexp        = regexp.MustCompile("^Version:[ ]*([!-~]*).*")

	// ErrNotFound is used for indicating that binary is not found.
	ErrNotFound = errors.New("not found")
)

// CombinedOutputer is used for creating an interface for CommandContext() function.
type CombinedOutputer interface {
	CombinedOutput() ([]byte, error)
}

// ExecFunctions is an interface for the LookPath() and CommandContext() functions.
type ExecFunctions interface {
	LookPath(file string) (string, error)
	CommandContext(ctx context.Context, name string, arg ...string) CombinedOutputer
}

// RealExecFunctions is a real implementation for the LookPath() and CommandContext() functions.
type RealExecFunctions struct{}

// LookPath calls Go's implementation of the LookPath() function.
func (RealExecFunctions) LookPath(file string) (string, error) {
	return exec.LookPath(file)
}

// CommandContext calls Go's implementation of the CommandContext() function.
func (RealExecFunctions) CommandContext(ctx context.Context, name string, arg ...string) CombinedOutputer {
	return exec.CommandContext(ctx, name, arg...)
}

// Versioner implements version retrieving functions for different software.
type Versioner struct {
	ef ExecFunctions
}

// New creates an instance of Versioner.
func New(ef ExecFunctions) *Versioner {
	return &Versioner{
		ef: ef,
	}
}

func (v *Versioner) binaryVersion(
	binaryName string,
	expectedExitCode int,
	versionRegexp *regexp.Regexp,
	arg ...string,
) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), versionCheckTimeout)
	defer cancel()

	if _, err := v.ef.LookPath(binaryName); err != nil {
		if errors.Is(err.(*exec.Error).Err, exec.ErrNotFound) { //nolint:forcetypeassert,errorlint
			return "", ErrNotFound
		}

		return "", errors.Wrapf(err, "lookpath: %s", binaryName)
	}

	versionBytes, err := v.ef.CommandContext(ctx, binaryName, arg...).CombinedOutput()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok { //nolint:errorlint
			if exitError.ExitCode() != expectedExitCode {
				return "", errors.WithStack(err)
			}
		} else {
			return "", errors.WithStack(err)
		}
	}

	matches := versionRegexp.FindStringSubmatch(string(versionBytes))
	if len(matches) != 2 {
		return "", errors.Errorf("cannot match version from output %q", string(versionBytes))
	}

	return matches[1], nil
}

// MySQLdVersion retrieves mysqld binary version.
func (v *Versioner) MySQLdVersion() (string, error) {
	return v.binaryVersion(mysqldBin, 0, mysqldVersionRegexp, "--version")
}

// XtrabackupVersion retrieves xtrabackup binary version.
func (v *Versioner) XtrabackupVersion() (string, error) {
	return v.binaryVersion(xtrabackupBin, 0, xtrabackupVersionRegexp, "--version")
}

// XbcloudVersion retrieves xbcloud binary version.
func (v *Versioner) XbcloudVersion() (string, error) {
	return v.binaryVersion(xbcloudBin, 0, xbcloudVersionRegexp, "--version")
}

// QpressVersion retrieves qpress binary version.
func (v *Versioner) QpressVersion() (string, error) {
	return v.binaryVersion(qpressBin, 255, qpressVersionRegexp)
}

// MongoDBVersion retrieves mongodb binary version.
func (v *Versioner) MongoDBVersion() (string, error) {
	return v.binaryVersion(mongodbBin, 0, mongodbVersionRegexp, "--version")
}

// PBMVersion retrieves pbm binary version.
func (v *Versioner) PBMVersion() (string, error) {
	return v.binaryVersion(pbmBin, 0, pbmVersionRegexp, "version")
}

// BinaryVersion retrieves agent binary version.
func (v *Versioner) BinaryVersion(
	binaryName string,
	expectedExitCode int,
	versionRegexp *regexp.Regexp,
	arg ...string,
) (string, error) {
	return v.binaryVersion(binaryName, expectedExitCode, versionRegexp, arg...)
}
