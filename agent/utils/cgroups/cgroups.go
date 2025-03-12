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

// Package cgroups provides utilities for working with cgroups.
package cgroups

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"

	"github.com/sirupsen/logrus"
)

// CgroupAccess represents the status of cgroup filesystem access
type CgroupAccess struct {
	Available   bool   // Whether cgroup filesystem is available
	Version     string // Cgroup version (v1, v2, or unknown)
	WriteAccess bool   // Whether we have write access
}

// CheckAccess performs a comprehensive check of cgroup access
func CheckAccess(l *logrus.Entry) CgroupAccess {
	result := CgroupAccess{
		Available:   false,
		Version:     "unknown",
		WriteAccess: false,
	}

	cgroupPath := "/sys/fs/cgroup"
	if _, err := os.Stat(cgroupPath); err != nil {
		l.Warn("cgroup filesystem not available: ", err.Error())
		return result
	}
	result.Available = true

	// Detect cgroup version
	if _, err := os.Stat(filepath.Join(cgroupPath, "cgroup.controllers")); err == nil {
		result.Version = "v2"
	} else if _, err := os.Stat(filepath.Join(cgroupPath, "memory")); err == nil {
		result.Version = "v1"
	}

	// Check write permissions
	testPath := filepath.Join(cgroupPath, "test_write")
	err := os.Mkdir(testPath, 0o755)
	if err == nil {
		result.WriteAccess = true
		os.Remove(testPath) //nolint:errcheck
	} else if os.IsPermission(err) || errors.Is(err, syscall.EROFS) {
		result.WriteAccess = false
		l.Warn("cgroup filesystem is read-only:", err)
	}

	return result
}
