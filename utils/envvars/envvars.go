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
	"errors"
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
		return errors.New("environment variable name cannot be empty")
	}

	if len(name) > MaxNameLength {
		return fmt.Errorf("environment variable name '%s' is too long (max %d characters)", name, MaxNameLength)
	}

	if !namePattern.MatchString(name) {
		return fmt.Errorf("invalid environment variable name: %s (must match [A-Za-z_][A-Za-z0-9_]*)", name)
	}

	if strings.HasPrefix(strings.ToUpper(name), ReservedPrefix) {
		return fmt.Errorf("environment variable name '%s' is reserved for pmm-agent's own configuration and cannot be selected", name)
	}

	return nil
}

// NormalizeNames trims and validates each name, collapses duplicates keeping the first occurrence,
// and checks the resulting list's length bound. Both pmm-admin and the API call this rather than
// validating and deduplicating separately, so a name repeated or padded with whitespace does not
// count twice against the limit, and stored names are always deduplicated.
func NormalizeNames(names []string) ([]string, error) {
	return NormalizeNamesAllowing(names, nil)
}

// NormalizeNamesAllowing behaves like NormalizeNames, but skips ValidateName for any (already
// trimmed) name present in grandfathered. This lets a full replacement of an agent's name list
// carry forward entries that were stored before this validation existed, or under
// since-tightened rules, instead of rejecting an update that only touches other, valid names.
func NormalizeNamesAllowing(names []string, grandfathered map[string]struct{}) ([]string, error) {
	if len(names) == 0 {
		return nil, nil
	}

	result := make([]string, 0, len(names))
	seen := make(map[string]struct{}, len(names))

	for _, name := range names {
		name = strings.TrimSpace(name)

		if _, ok := grandfathered[name]; !ok {
			err := ValidateName(name)
			if err != nil {
				return nil, err
			}
		}

		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}

		result = append(result, name)
	}

	// An agent may already store more than MaxNames names: this bound did not exist before it was
	// introduced alongside the rest of this policy. Applying it unconditionally would leave such a
	// list uneditable forever, since the field is full-replace — the owner could not even remove a
	// name without first truncating to MaxNames and losing the rest. Take the stored count as the
	// effective bound instead, so an oversized list can only shrink toward MaxNames, never grow.
	limit := max(MaxNames, len(grandfathered))

	if len(result) > limit {
		return nil, fmt.Errorf("too many environment variable names: %d (max %d)", len(result), limit)
	}

	return result, nil
}
