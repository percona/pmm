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

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsed(t *testing.T) {
	t.Run("PMM", func(t *testing.T) {
		data := []struct {
			s string
			p *Parsed
		}{
			{
				s: "2.0.0-beta4",
				p: &Parsed{Major: 2, Minor: 0, Patch: 0, Rest: "-beta4", Num: 20000},
			}, {
				s: "2.0.0-beta4-2-gff76039-dirty",
				p: &Parsed{Major: 2, Minor: 0, Patch: 0, Rest: "-beta4-2-gff76039-dirty", Num: 20000},
			}, {
				s: "2.0.0",
				p: &Parsed{Major: 2, Minor: 0, Patch: 0, Num: 20000},
			}, {
				s: "2.1.2",
				p: &Parsed{Major: 2, Minor: 1, Patch: 2, Num: 20102},
			}, {
				s: "2.1.3-0",
				p: &Parsed{Major: 2, Minor: 1, Patch: 3, Rest: "-0", Num: 20103},
			}, {
				s: "2.1.3-HEAD-abcd12",
				p: &Parsed{Major: 2, Minor: 1, Patch: 3, Rest: "-HEAD-abcd12", Num: 20103},
			}, {
				s: "2.1.3",
				p: &Parsed{Major: 2, Minor: 1, Patch: 3, Num: 20103},
			}, {
				s: "3.0.0",
				p: &Parsed{Major: 3, Minor: 0, Patch: 0, Num: 30000},
			}, {
				s: "4.0.0-12",
				p: &Parsed{Major: 4, Minor: 0, Patch: 0, Rest: "-12", Num: 40000, NumRest: 12},
			},
		}
		for i, expected := range data {
			t.Run(expected.s, func(t *testing.T) {
				actual, err := Parse(expected.s)
				require.NoError(t, err)
				assert.Equal(t, *expected.p, *actual)
				assert.Equal(t, expected.s, actual.String())

				for j := 0; j < i; j++ {
					assert.True(t, data[j].p.Less(actual), "%s is expected to be less than %s", data[j].p, actual)
				}
				for j := i + 1; j < len(data); j++ {
					assert.False(t, data[j].p.Less(actual), "%s is expected to be not less than %s", data[j].p, actual)
				}
			})
		}
	})

	t.Run("MySQL", func(t *testing.T) {
		data := []struct {
			s string
			p *Parsed
		}{
			{
				s: "5.6.47-87.0-log",
				p: &Parsed{Major: 5, Minor: 6, Patch: 47, Rest: "-87.0-log", Num: 50647, NumRest: 87},
			}, {
				s: "5.6.48-log",
				p: &Parsed{Major: 5, Minor: 6, Patch: 48, Rest: "-log", Num: 50648},
			}, {
				s: "5.7.29-32-log",
				p: &Parsed{Major: 5, Minor: 7, Patch: 29, Rest: "-32-log", Num: 50729, NumRest: 32},
			}, {
				s: "5.7.30-log",
				p: &Parsed{Major: 5, Minor: 7, Patch: 30, Rest: "-log", Num: 50730},
			}, {
				s: "8.0.19-10",
				p: &Parsed{Major: 8, Minor: 0, Patch: 19, Rest: "-10", Num: 80019, NumRest: 10},
			}, {
				s: "8.0.20",
				p: &Parsed{Major: 8, Minor: 0, Patch: 20, Rest: "", Num: 80020},
			},
		}
		for i, expected := range data {
			t.Run(expected.s, func(t *testing.T) {
				actual, err := Parse(expected.s)
				require.NoError(t, err)
				assert.Equal(t, *expected.p, *actual)
				assert.Equal(t, expected.s, actual.String())

				for j := 0; j < i; j++ {
					assert.True(t, data[j].p.Less(actual), "%s is expected to be less than %s", data[j].p, actual)
				}
				for j := i + 1; j < len(data); j++ {
					assert.False(t, data[j].p.Less(actual), "%s is expected to be not less than %s", data[j].p, actual)
				}
			})
		}
	})
}
