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

package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
)

// OmSettingScope is where a setting value lives: global, or per cluster.
type OmSettingScope string

// Setting scopes.
const (
	OmSettingGlobal  = OmSettingScope("global")
	OmSettingCluster = OmSettingScope("cluster")
)

// OmSettingSource says where an effective value came from.
type OmSettingSource string

// Effective-value sources, in resolution order.
const (
	OmSettingSourceCluster = OmSettingSource("cluster")
	OmSettingSourceGlobal  = OmSettingSource("global")
	OmSettingSourceDefault = OmSettingSource("default")
)

// OmSettingDef declares one Operations policy key. Validate checks a candidate value; Secret keys
// are stored encrypted and never returned by value. A later epic registers its own keys.
type OmSettingDef struct {
	Key         string
	Default     json.RawMessage
	Scopes      []OmSettingScope
	Secret      bool
	Description string
	Validate    func(json.RawMessage) error
}

// AllowsScope reports whether a value for this key may be set at the given scope.
func (d OmSettingDef) AllowsScope(scope OmSettingScope) bool {
	return slices.Contains(d.Scopes, scope)
}

// OmSettingsRegistry is the set of known Operations policy keys. This epic ships it empty; the
// features register theirs. It is not safe for concurrent registration.
type OmSettingsRegistry struct {
	defs map[string]OmSettingDef
}

// NewOmSettingsRegistry returns an empty registry.
func NewOmSettingsRegistry() *OmSettingsRegistry {
	return &OmSettingsRegistry{defs: make(map[string]OmSettingDef)}
}

// Register adds a key definition, rejecting a blank key, a duplicate, an empty scope set, or a
// default that does not validate.
func (r *OmSettingsRegistry) Register(def OmSettingDef) error {
	if def.Key == "" {
		return errors.New("settings registry: key must not be empty")
	}
	_, exists := r.defs[def.Key]
	if exists {
		return fmt.Errorf("settings registry: key '%s' is already registered", def.Key)
	}
	if len(def.Scopes) == 0 {
		return fmt.Errorf("settings registry: key '%s' must allow at least one scope", def.Key)
	}
	if def.Validate != nil && def.Default != nil {
		err := def.Validate(def.Default)
		if err != nil {
			return fmt.Errorf("settings registry: key '%s' default is invalid: %w", def.Key, err)
		}
	}
	r.defs[def.Key] = def
	return nil
}

// Def returns a key's definition.
func (r *OmSettingsRegistry) Def(key string) (OmSettingDef, bool) {
	def, ok := r.defs[key]
	return def, ok
}

// Keys returns the registered keys, sorted.
func (r *OmSettingsRegistry) Keys() []string {
	keys := make([]string, 0, len(r.defs))
	for k := range r.defs {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

// validateValue checks a key is known, the scope is allowed, and the value passes the key's
// validator.
func (r *OmSettingsRegistry) validateValue(key string, scope OmSettingScope, value json.RawMessage) (OmSettingDef, error) {
	def, ok := r.defs[key]
	if !ok {
		return OmSettingDef{}, fmt.Errorf("settings: unknown key '%s'", key)
	}
	if !def.AllowsScope(scope) {
		return OmSettingDef{}, fmt.Errorf("settings: key '%s' cannot be set at scope %s", key, scope)
	}
	if def.Validate != nil {
		err := def.Validate(value)
		if err != nil {
			return OmSettingDef{}, fmt.Errorf("settings: value for '%s' is invalid: %w", key, err)
		}
	}
	return def, nil
}
