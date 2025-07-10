// Copyright (C) 2023 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

// Package models contains generated Reform records and helpers.
//
// Common order of helpers:
//   - unexported validators (checkXXX);
//   - FindAllXXX;
//   - FindXXXByID;
//   - other finder (e.g. FindNodesForAgent);
//   - CreateXXX;
//   - ChangeXXX;
//   - RemoveXXX.
package models

import (
	"database/sql/driver"
	"encoding/json"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Now returns current time with database precision.
var Now = func() time.Time {
	return time.Now().Truncate(time.Microsecond).UTC()
}

// RemoveMode defines how Remove functions deal with dependent objects.
type RemoveMode int

const (
	// RemoveRestrict returns error if there dependent objects.
	RemoveRestrict RemoveMode = iota
	// RemoveCascade removes dependend objects recursively.
	RemoveCascade
)

// getExtraDSNParams deserializes custom DSN parameters from JSON-encoded byte slice.
func getExtraDSNParams(b []byte) (map[string]string, error) {
	if len(b) == 0 {
		return nil, nil //nolint:nilnil
	}
	m := make(map[string]string)
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, errors.Wrap(err, "failed to decode extra DSN parameters")
	}
	return m, nil
}

// MergeLabels merges unified labels of Node, Service, and Agent (each can be nil).
func MergeLabels(node *Node, service *Service, agent *Agent) (map[string]string, error) {
	res := make(map[string]string, 16)

	if node != nil {
		labels, err := node.UnifiedLabels()
		if err != nil {
			return nil, err
		}
		for name, value := range labels {
			res[name] = value
		}
	}

	if service != nil {
		labels, err := service.UnifiedLabels()
		if err != nil {
			return nil, err
		}
		for name, value := range labels {
			res[name] = value
		}
	}

	if agent != nil {
		labels, err := agent.UnifiedLabels()
		if err != nil {
			return nil, err
		}
		for name, value := range labels {
			res[name] = value
		}
	}

	return res, nil
}

// deduplicateStrings deduplicates elements in string slice.
func deduplicateStrings(strings []string) []string {
	set := make(map[string]struct{})
	for _, p := range strings {
		set[p] = struct{}{}
	}

	slice := make([]string, 0, len(set))
	for s := range set {
		slice = append(slice, s)
	}
	sort.Strings(slice)

	return slice
}

var labelNameRE = regexp.MustCompile("^[a-zA-Z_][a-zA-Z0-9_]*$")

// prepareLabels checks that label names are valid, and trims or removes empty values.
func prepareLabels(m map[string]string, removeEmptyValues bool) error {
	for name, value := range m {
		if !labelNameRE.MatchString(name) {
			return status.Errorf(codes.InvalidArgument, "Invalid label name %q.", name)
		}
		if strings.HasPrefix(name, "__") {
			return status.Errorf(codes.InvalidArgument, "Invalid label name %q.", name)
		}

		value = strings.TrimSpace(value)
		if value == "" {
			if removeEmptyValues {
				delete(m, name)
			} else {
				m[name] = value
			}
		}
	}

	return nil
}

// getLabels deserializes model's Prometheus labels.
func getLabels(b []byte) (map[string]string, error) {
	if len(b) == 0 {
		return nil, nil //nolint:nilnil
	}
	m := make(map[string]string)
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, errors.Wrap(err, "failed to decode custom labels")
	}
	return m, nil
}

// getLabels serializes model's Prometheus labels.
func setLabels(m map[string]string, res *[]byte) error {
	if err := prepareLabels(m, false); err != nil {
		return err
	}

	if len(m) == 0 {
		*res = nil
		return nil
	}

	b, err := json.Marshal(m)
	if err != nil {
		return errors.Wrap(err, "failed to encode custom labels")
	}
	*res = b
	return nil
}

// jsonValue implements database/sql/driver.Valuer interface for v that should be a value.
func jsonValue(v interface{}) (driver.Value, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal JSON column")
	}
	return b, nil
}

// jsonScan implements database/sql.Scanner interface for v that should be a pointer.
func jsonScan(v, src interface{}) error {
	var b []byte
	switch v := src.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	case nil:
		return nil
	default:
		return errors.Errorf("expected []byte or string, got %T (%q)", src, src)
	}

	if err := json.Unmarshal(b, v); err != nil {
		return errors.Wrap(err, "failed to unmarshal JSON column")
	}
	return nil
}
