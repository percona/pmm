// pmm-update
// Copyright (C) 2019 Percona LLC
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

package yum

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//nolint:lll
func TestParseChangeLog(t *testing.T) {
	stdout := strings.Split(`
		Loaded plugins: changelog, fastestmirror, ovl
		Resolving Dependencies
		--> Running transaction check
		---> Package pmm-update.noarch 0:2.0.0-17.rc4.1909170601.6de91ea.el7 will be updated
		---> Package pmm-update.noarch 0:2.0.0-18.1909180919.aa709c6.el7 will be an update
		--> Finished Dependency Resolution

		Changes in packages about to be updated:

		ChangeLog for: pmm-update-2.0.0-18.1909180919.aa709c6.el7.noarch
		* Wed Sep 18 12:00:00 2019 Alexey Palazhchenko <alexey.palazhchenko@percona.com> - 2.0.0-18
		- https://per.co.na/pmm/2.0.0


		Dependencies Resolved

		=========================================================================================================================================================================================================
		Package                                     Arch                                    Version                                                          Repository                                    Size
		=========================================================================================================================================================================================================
		Updating:
		pmm-update                                  noarch                                  2.0.0-18.1909180919.aa709c6.el7                                  pmm2-server                                  857 k

		Transaction Summary
		=========================================================================================================================================================================================================
		Upgrade  1 Package

		Total download size: 857 k
		Exiting on user command
		Your transaction was saved, rerun it with:
		yum load-transaction /tmp/yum_save_tx.2019-09-19.12-57.gVfAds.yumtx
	`, "\n")
	cl, err := parseChangeLog(stdout)
	require.NoError(t, err)
	assert.Equal(t, &changeLog{url: "https://per.co.na/pmm/2.0.0"}, cl)
}
