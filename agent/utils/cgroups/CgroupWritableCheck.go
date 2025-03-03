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
	"os"
	"path/filepath"
)

// IsCgroupsWritable checks if the cgroup filesystem is writable.
func IsCgroupsWritable() bool {
	cgroupDir := "/sys/fs/cgroup"
	testFile := filepath.Join(cgroupDir, "test_write")

	// Attempt to create and remove a test file to check for write access
	file, err := os.Create(testFile)
	if err != nil {
		return false // Not writable
	}
	defer func() {
		file.Close()
		os.Remove(testFile) // Clean up the test file
	}()

	return true // Writable
}
