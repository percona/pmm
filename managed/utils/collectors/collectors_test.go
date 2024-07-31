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

package collectors

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDisableDefaultEnabledCollectors(t *testing.T) {
	type args struct {
		prefix             string
		defaultCollectors  []string
		disabledCollectors []string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "Disable single default enabled collectors",
			args: args{
				prefix:             "--no-collector.",
				defaultCollectors:  []string{"a", "b", "c", "d", "e"},
				disabledCollectors: []string{"b"},
			},
			want: []string{"--no-collector.b"},
		},
		{
			name: "Disable multiple default enabled collectors",
			args: args{
				prefix:             "--no-collector.",
				defaultCollectors:  []string{"a", "b", "c", "d", "e", "f"},
				disabledCollectors: []string{"a", "c"},
			},
			want: []string{"--no-collector.a", "--no-collector.c"},
		},
		{
			name: "Disable all default enabled collectors",
			args: args{
				prefix:             "--no-collector.",
				defaultCollectors:  []string{"a", "b", "c"},
				disabledCollectors: []string{"a", "b", "c"},
			},
			want: []string{"--no-collector.a", "--no-collector.b", "--no-collector.c"},
		},
		{
			name: "Disable non-default enabled collectors",
			args: args{
				prefix:             "--no-collector.",
				defaultCollectors:  []string{"a", "b", "c"},
				disabledCollectors: []string{"d", "e", "f"},
			},
			want: []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := DisableDefaultEnabledCollectors(tt.args.prefix, tt.args.defaultCollectors, tt.args.disabledCollectors)
			require.Equal(t, tt.want, actual, "DisableDefaultEnabledCollectors() = %v, want %v", actual, tt.want)
		})
	}
}
