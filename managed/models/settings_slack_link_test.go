// Copyright (C) 2023 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package models_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/percona/pmm/managed/models"
)

func TestNormalizePMMPublicAddressOrigin(t *testing.T) {
	assert.Empty(t, models.NormalizePMMPublicAddressOrigin(""))
	assert.Equal(t, "https://192.168.2.33", models.NormalizePMMPublicAddressOrigin("192.168.2.33"))
	assert.Equal(t, "https://pmm.example.com:8443", models.NormalizePMMPublicAddressOrigin("https://pmm.example.com:8443"))
}

func TestGetEffectiveSlackLinkBaseURL(t *testing.T) {
	s := &models.Settings{}
	s.PMMPublicAddress = "https://pmmsrv"
	assert.Equal(t, "https://pmmsrv", s.GetEffectiveSlackLinkBaseURL())
}
