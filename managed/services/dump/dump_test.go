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

package dump

import (
	"testing"
)

func Test_getDumpFilePath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		id        string
		encrypted bool
		want      string
	}{
		{
			name:      "Usual dump",
			id:        "123456789",
			encrypted: false,
			want:      dumpsDir + "/123456789.tar.gz",
		},
		{
			name:      "Encrypted dump",
			id:        "123456789",
			encrypted: true,
			want:      dumpsDir + "/123456789.tar.gz.enc",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := getDumpFilePath(tt.id, tt.encrypted); got != tt.want {
				t.Errorf("getDumpFilePath() = %v, want %v", got, tt.want)
			}
		})
	}
}
