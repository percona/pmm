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

package common

import "fmt"

// DistributionType represents type of distribution of the pmm-agent.
type DistributionType int

const (
	// Docker represents Docker installation of PMM Agent or Server.
	Docker DistributionType = iota
	// PackageManager represents installation of PMM Agent or Server via a package manager.
	PackageManager
	// Tarball represents installation of PMM Agent or Server via a tarball.
	Tarball
)

// DetectDistributionType detects distribution type of pmm-agent.
func DetectDistributionType() DistributionType {
	return PackageManager
}

// OSPackageManager represents a specific package manager used by the system.
type OSPackageManager int

const (
	// UnknownPackageManager represents unknown package manager.
	UnknownPackageManager OSPackageManager = iota
	// Dnf represents "dnf" package manager.
	Dnf
	// Yum represents "yum" package manager.
	Yum
	// Apt represents "apt" package manager.
	Apt
)

// ErrCouldNotDetectPackageManager is returned when package manager cannot be detected.
var ErrCouldNotDetectPackageManager = fmt.Errorf("CouldNotDetectPackageManager")

// DetectPackageManager detects the package manager available in the current system.
func DetectPackageManager() (OSPackageManager, error) {
	cmd, err := LookupCommand("dnf")
	if err != nil {
		return UnknownPackageManager, err
	}

	if cmd != "" {
		return Dnf, nil
	}

	cmd, err = LookupCommand("yum")
	if err != nil {
		return UnknownPackageManager, err
	}

	if cmd != "" {
		return Yum, nil
	}

	cmd, err = LookupCommand("apt")
	if err != nil {
		return UnknownPackageManager, err
	}

	if cmd != "" {
		return Apt, nil
	}

	return UnknownPackageManager, fmt.Errorf("%w: Could not detect package manager", ErrCouldNotDetectPackageManager)
}
