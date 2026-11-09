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

package validators

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidatePassword(t *testing.T) {
	t.Parallel()

	type args struct {
		password string
		minLen   int
	}
	tests := []struct {
		name    string
		args    args
		wantErr error
	}{
		{
			name: "valid password",
			args: args{
				password: "Valid1!P@$$w0rd",
				minLen:   8,
			},
			wantErr: nil,
		},
		{
			name: "too short",
			args: args{
				password: "V1!",
				minLen:   8,
			},
			wantErr: ErrInvalidPasswordLen(8),
		},
		{
			name: "missing letter",
			args: args{
				password: "12345678!",
				minLen:   8,
			},
			wantErr: ErrInvalidPasswordLetter,
		},
		{
			name: "missing digit",
			args: args{
				password: "Password!",
				minLen:   8,
			},
			wantErr: ErrInvalidPasswordDigit,
		},
		{
			name: "missing special character",
			args: args{
				password: "Password1",
				minLen:   8,
			},
			wantErr: ErrInvalidPasswordSpecial,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.wantErr == nil {
				assert.NoError(t, ValidatePassword(tt.args.password, tt.args.minLen), "ValidatePassword(%v, %v)", tt.args.password, tt.args.minLen)
			} else {
				assert.Equal(t, ValidatePassword(tt.args.password, tt.args.minLen).Error(), tt.wantErr.Error(), "ValidatePassword(%v, %v)", tt.args.password, tt.args.minLen)
			}
		})
	}
}
