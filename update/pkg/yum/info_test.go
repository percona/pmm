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
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseInfo(t *testing.T) {
	t.Run("Installed", func(t *testing.T) {
		stdout := strings.Split(`
			Loading "fastestmirror" plugin
			Loading "ovl" plugin
			Config time: 0.005
			rpmdb time: 0.000
			ovl: Copying up (0) files from OverlayFS lower layer
			Yum version: 3.4.3
			Installed Packages
			Name        : pmm-managed
			Arch        : x86_64
			Version     : 2.0.0
			Release     : 9.beta5.1907301101.74f8a67.el7
			Size        : 18 M
			Repo        : installed
			From repo   : local
			Committer   : Mykola Marzhan <mykola.marzhan@percona.com>
			Committime  : Thu Sep 21 12:00:00 2017
			Buildtime   : Tue Jul 30 11:02:19 2019
			Install time: Tue Jul 30 18:43:02 2019
			Installed by: 500
			Changed by  : System <unset>
			Summary     : Percona Monitoring and Management management daemon
			URL         : https://github.com/percona/pmm-managed
			License     : AGPLv3
			Description : pmm-managed manages configuration of PMM server components (Prometheus,
						: Grafana, etc.) and exposes API for that.  Those APIs are used by pmm-admin tool.
						: See the PMM docs for more information.
		`, "\n")
		expected := map[string]string{
			"Name":         "pmm-managed",
			"Arch":         "x86_64",
			"Version":      "2.0.0",
			"Release":      "9.beta5.1907301101.74f8a67.el7",
			"Size":         "18 M",
			"Repo":         "installed",
			"From repo":    "local",
			"Committer":    "Mykola Marzhan <mykola.marzhan@percona.com>",
			"Committime":   "Thu Sep 21 12:00:00 2017",
			"Buildtime":    "Tue Jul 30 11:02:19 2019",
			"Install time": "Tue Jul 30 18:43:02 2019",
			"Installed by": "500",
			"Changed by":   "System <unset>",
			"Summary":      "Percona Monitoring and Management management daemon",
			"URL":          "https://github.com/percona/pmm-managed",
			"License":      "AGPLv3",
			"Description": "pmm-managed manages configuration of PMM server components (Prometheus, " +
				"Grafana, etc.) and exposes API for that.  Those APIs are used by pmm-admin tool. " +
				"See the PMM docs for more information.",
		}
		actual, err := parseInfo(stdout, "Name")
		require.NoError(t, err)
		assert.Equal(t, expected, actual)
		buildtime, err := parseInfoTime(actual["Buildtime"])
		require.NoError(t, err)
		assert.Equal(t, time.Date(2019, 7, 30, 11, 2, 19, 0, time.UTC), buildtime)
		assert.Equal(t, "2.0.0-9.beta5.1907301101.74f8a67.el7", fullVersion(actual))
		assert.Equal(t, "2.0.0-9.beta5", niceVersion(actual))
	})

	t.Run("Updates", func(t *testing.T) {
		stdout := strings.Split(`
			Loading "fastestmirror" plugin
			Loading "ovl" plugin
			Config time: 0.017
			rpmdb time: 0.000
			ovl: Copying up (14) files from OverlayFS lower layer
			Yum version: 3.4.3
			Building updates object
			Setting up Package Sacks
			Determining fastest mirrors
			* base: mirror.reconn.ru
			* epel: mirror.yandex.ru
			* extras: mirror.reconn.ru
			* updates: mirror.reconn.ru
			pkgsack time: 14.667
			up:Obs Init time: 0.235
			up:simple updates time: 0.004
			up:obs time: 0.003
			up:condense time: 0.000
			updates time: 15.139
			Updated Packages
			Name        : pmm-update
			Arch        : noarch
			Version     : 2.0.0
			Release     : 9.beta5.1907301223.90149dd.el7
			Size        : 1.5 M
			Repo        : pmm2-laboratory
			Committer   : Mykola Marzhan <mykola.marzhan@percona.com>
			Committime  : Fri Jun 30 12:00:00 2017
			Buildtime   : Tue Jul 30 12:23:10 2019
			Summary     : Tool for updating packages and OS configuration for PMM Server
			URL         : https://github.com/percona/pmm/update
			License     : AGPLv3
			Description : Tool for updating packages and OS configuration for PMM Server
		`, "\n")
		expected := map[string]string{
			"Name":        "pmm-update",
			"Arch":        "noarch",
			"Version":     "2.0.0",
			"Release":     "9.beta5.1907301223.90149dd.el7",
			"Size":        "1.5 M",
			"Repo":        "pmm2-laboratory",
			"Committer":   "Mykola Marzhan <mykola.marzhan@percona.com>",
			"Committime":  "Fri Jun 30 12:00:00 2017",
			"Buildtime":   "Tue Jul 30 12:23:10 2019",
			"Summary":     "Tool for updating packages and OS configuration for PMM Server",
			"URL":         "https://github.com/percona/pmm/update",
			"License":     "AGPLv3",
			"Description": "Tool for updating packages and OS configuration for PMM Server",
		}
		actual, err := parseInfo(stdout, "Name")
		require.NoError(t, err)
		assert.Equal(t, expected, actual)
		buildtime, err := parseInfoTime(actual["Buildtime"])
		require.NoError(t, err)
		assert.Equal(t, time.Date(2019, 7, 30, 12, 23, 10, 0, time.UTC), buildtime)
		assert.Equal(t, "2.0.0-9.beta5.1907301223.90149dd.el7", fullVersion(actual))
		assert.Equal(t, "2.0.0-9.beta5", niceVersion(actual))
	})

	t.Run("Available", func(t *testing.T) {
		// yum --verbose --showduplicates info available pmm-update, abbrivated
		stdout := strings.Split(`
			Loading "fastestmirror" plugin
			Loading "ovl" plugin
			Config time: 0.007
			rpmdb time: 0.000
			ovl: Copying up (0) files from OverlayFS lower layer
			Yum version: 3.4.3
			Setting up Package Sacks
			Loading mirror speeds from cached hostfile
			* base: mirror.logol.ru
			* epel: fedora-mirror01.rbc.ru
			* extras: mirror.logol.ru
			* updates: mirror.logol.ru
			pkgsack time: 0.027
			Available Packages
			Name        : pmm-update
			Arch        : noarch
			Version     : PMM
			Release     : 7.4358.1907161009.7685dba.el7
			Size        : 20 k
			Repo        : pmm2-laboratory
			Committer   : Mykola Marzhan <mykola.marzhan@percona.com>
			Committime  : Fri Jun 30 12:00:00 2017
			Buildtime   : Tue Jul 16 10:09:01 2019
			Summary     : Tool for updating packages and OS configuration for PMM Server
			URL         : https://github.com/percona/pmm/update
			License     : AGPLv3
			Description : Tool for updating packages and OS configuration for PMM Server

			Name        : pmm-update
			Arch        : noarch
			Version     : 2.0.0
			Release     : 1.1903221448.2e245f9.el7
			Size        : 20 k
			Repo        : pmm2-laboratory
			Committer   : Mykola Marzhan <mykola.marzhan@percona.com>
			Committime  : Fri Jun 30 12:00:00 2017
			Buildtime   : Fri Mar 22 14:48:42 2019
			Summary     : Tool for updating packages and OS configuration for PMM Server
			URL         : https://github.com/percona/pmm/update
			License     : AGPLv3
			Description : Tool for updating packages and OS configuration for PMM Server

			…
		`, "\n")
		expected := map[string]string{
			"Name":        "pmm-update",
			"Arch":        "noarch",
			"Version":     "PMM",
			"Release":     "7.4358.1907161009.7685dba.el7",
			"Size":        "20 k",
			"Repo":        "pmm2-laboratory",
			"Committer":   "Mykola Marzhan <mykola.marzhan@percona.com>",
			"Committime":  "Fri Jun 30 12:00:00 2017",
			"Buildtime":   "Tue Jul 16 10:09:01 2019",
			"Summary":     "Tool for updating packages and OS configuration for PMM Server",
			"URL":         "https://github.com/percona/pmm/update",
			"License":     "AGPLv3",
			"Description": "Tool for updating packages and OS configuration for PMM Server",
		}
		actual, err := parseInfo(stdout, "Name")
		assert.EqualError(t, err, "second `Name` encountered")
		assert.Equal(t, expected, actual)
		buildtime, err := parseInfoTime(actual["Buildtime"])
		require.NoError(t, err)
		assert.Equal(t, time.Date(2019, 7, 16, 10, 9, 1, 0, time.UTC), buildtime)
		assert.Equal(t, "PMM-7.4358.1907161009.7685dba.el7", fullVersion(actual))
		assert.Equal(t, "PMM-7.4358", niceVersion(actual)) // yes, that one is broken
	})

	t.Run("AvailableGA", func(t *testing.T) {
		// yum --verbose --showduplicates info available pmm-update, abbrivated
		stdout := strings.Split(`
			Available Packages
			Name        : pmm-update
			Arch        : noarch
			Version     : 2.0.0
			Release     : 18.1909180550.6de91ea.el7
			Size        : 857 k
			Repo        : pmm2-server
			Committer   : Alexey Palazhchenko <alexey.palazhchenko@percona.com>
			Committime  : Wed Sep 18 12:00:00 2019
			Buildtime   : Wed Sep 18 05:51:01 2019
			Summary     : Tool for updating packages and OS configuration for PMM Server
			URL         : https://github.com/percona/pmm/update
			License     : AGPLv3
			Description : Tool for updating packages and OS configuration for PMM Server

			…
		`, "\n")
		expected := map[string]string{
			"Name":        "pmm-update",
			"Arch":        "noarch",
			"Version":     "2.0.0",
			"Release":     "18.1909180550.6de91ea.el7",
			"Size":        "857 k",
			"Repo":        "pmm2-server",
			"Committer":   "Alexey Palazhchenko <alexey.palazhchenko@percona.com>",
			"Committime":  "Wed Sep 18 12:00:00 2019",
			"Buildtime":   "Wed Sep 18 05:51:01 2019",
			"Summary":     "Tool for updating packages and OS configuration for PMM Server",
			"URL":         "https://github.com/percona/pmm/update",
			"License":     "AGPLv3",
			"Description": "Tool for updating packages and OS configuration for PMM Server",
		}
		actual, err := parseInfo(stdout, "Name")
		require.NoError(t, err)
		assert.Equal(t, expected, actual)
		buildtime, err := parseInfoTime(actual["Buildtime"])
		require.NoError(t, err)
		assert.Equal(t, time.Date(2019, 9, 18, 5, 51, 1, 0, time.UTC), buildtime)
		assert.Equal(t, "2.0.0-18.1909180550.6de91ea.el7", fullVersion(actual))
		assert.Equal(t, "2.0.0", niceVersion(actual))
	})

	t.Run("Empty", func(t *testing.T) {
		// "Error: No matching Packages to list" goes to stderr.
		stdout := strings.Split(`
			Loading "fastestmirror" plugin
			Loading "ovl" plugin
			Config time: 0.007
			rpmdb time: 0.000
			ovl: Copying up (0) files from OverlayFS lower layer
			Yum version: 3.4.3
			Building updates object
			Setting up Package Sacks
			Loading mirror speeds from cached hostfile
			* base: mirror.logol.ru
			* epel: fedora-mirror01.rbc.ru
			* extras: mirror.logol.ru
			* updates: mirror.logol.ru
			pkgsack time: 0.030
			up:Obs Init time: 0.217
			up:simple updates time: 0.008
			up:obs time: 0.004
			up:condense time: 0.000
			updates time: 0.469
		`, "\n")
		actual, err := parseInfo(stdout, "Name")
		require.NoError(t, err)
		assert.Empty(t, actual)
	})

	t.Run("RepoInfo", func(t *testing.T) {
		// yum repoinfo pmm2-server.
		stdout := strings.Split(`
			Loaded plugins: changelog, fastestmirror, ovl
			Loading mirror speeds from cached hostfile
			* base: centos.schlundtech.de
			* epel: mirror.netcologne.de
			* extras: centos.mirror.iphh.net
			* updates: mirror.netcologne.de
			Repo-id      : pmm2-server
			Repo-name    : PMM Server YUM repository - x86_64
			Repo-status  : enabled
			Repo-revision: 1622561436
			Repo-updated : Tue Jun  1 15:30:45 2021
			Repo-pkgs    : 237
			Repo-size    : 2.4 G
			Repo-baseurl : https://repo.percona.com/pmm2-components/yum/release/7/RPMS/x86_64/
			Repo-expire  : 21600 second(s) (last: Thu Jun 10 16:08:12 2021)
			Filter     : read-only:present
			Repo-filename: /etc/yum.repos.d/pmm2-server.repo
			
			repolist: 237
			…
		`, "\n")
		expected := map[string]string{
			"Repo-id":       "pmm2-server",
			"Repo-name":     "PMM Server YUM repository - x86_64",
			"Repo-status":   "enabled",
			"Repo-revision": "1622561436",
			"Repo-updated":  "Tue Jun  1 15:30:45 2021",
			"Repo-pkgs":     "237",
			"Repo-size":     "2.4 G",
			"Repo-baseurl":  "https://repo.percona.com/pmm2-components/yum/release/7/RPMS/x86_64/",
			"Repo-expire":   "21600 second(s) (last: Thu Jun 10 16:08:12 2021)",
			"Filter":        "read-only:present",
			"Repo-filename": "/etc/yum.repos.d/pmm2-server.repo",
			"repolist":      "237",
		}
		actual, err := parseInfo(stdout, "Repo-id")
		require.NoError(t, err)
		assert.Equal(t, expected, actual)
		releasetime, err := parseInfoTime(actual["Repo-updated"])
		require.NoError(t, err)
		assert.Equal(t, time.Date(2021, 6, 1, 15, 30, 45, 0, time.UTC), releasetime)
	})
}
