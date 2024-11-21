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

package version

// Versions list.
var (
	V3_0_0 = MustParse("3.0.0-0") //nolint:revive,stylecheck
)

// FeatureVersion represents a minimum version feature being supported.
type FeatureVersion *Parsed

// Features list.
var (
	NodeExporterNewTLSConfigVersion FeatureVersion = V3_0_0
)

// IsFeatureSupported checks if the feature is supported by the version.
func (p *Parsed) IsFeatureSupported(f FeatureVersion) bool {
	return !p.Less(f)
}
