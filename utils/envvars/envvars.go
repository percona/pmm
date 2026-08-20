// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package envvars validates environment variable names that pmm-admin, pmm-managed and
// pmm-agent accept from users for pass-through to exporter processes: pmm-agent resolves each
// requested name against its own environment and injects the value into the exporter it starts.
package envvars

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	// MaxNames is the largest number of environment variable names accepted in one request.
	MaxNames = 32
	// MaxNameLength is the largest accepted length of a single environment variable name.
	MaxNameLength = 256
	// ReservedPrefix marks pmm-agent's own configuration and secrets, e.g. PMM_AGENT_SERVER_PASSWORD.
	// Names in this namespace must never be resolved for pass-through to an exporter process.
	ReservedPrefix = "PMM_AGENT_"
)

// namePattern is the POSIX portable environment variable name set.
var namePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// ValidateName reports whether name is an acceptable environment variable name for pass-through:
// syntactically valid, within length bounds, and outside pmm-agent's own reserved namespace.
func ValidateName(name string) error {
	if name == "" {
		return fmt.Errorf("environment variable name cannot be empty")
	}

	if len(name) > MaxNameLength {
		return fmt.Errorf("environment variable name %q is too long (max %d characters)", name, MaxNameLength)
	}

	if !namePattern.MatchString(name) {
		return fmt.Errorf("invalid environment variable name: %s (must match [A-Za-z_][A-Za-z0-9_]*)", name)
	}

	if strings.HasPrefix(strings.ToUpper(name), ReservedPrefix) {
		return fmt.Errorf("environment variable name %q is reserved for pmm-agent's own configuration and cannot be selected", name)
	}

	return nil
}

// ValidateNames validates a full list of names, including the list's length bound.
func ValidateNames(names []string) error {
	if len(names) > MaxNames {
		return fmt.Errorf("too many environment variable names: %d (max %d)", len(names), MaxNames)
	}

	for _, name := range names {
		if err := ValidateName(name); err != nil {
			return err
		}
	}

	return nil
}
