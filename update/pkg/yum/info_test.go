// Copyright (C) 2024 Percona LLC
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

func TestParseInfoEL7(t *testing.T) {
	v, _ := getRHELVersion()
	if v == "9" {
		t.Skip("Skip running EL7 tests on EL9")
	}
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
		// yum --verbose --showduplicates info available pmm-update, abbreviated
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
		// yum --verbose --showduplicates info available pmm-update, abbreviated
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

func TestParseInfoEL9(t *testing.T) {
	v, _ := getRHELVersion()
	if v == "7" {
		t.Skip("Skip running EL9 tests on EL7")
	}
	t.Run("Installed EL9", func(t *testing.T) {
		stdout := strings.Split(`
			Starting "yum --verbose info installed pmm-managed" ...
			Loaded plugins: builddep, changelog, config-manager, copr, debug, debuginfo-install, download, generate_completion_cache, groups-manager, needs-restarting, playground, repoclosure, repodiff, repograph, repomanage, reposync, system-upgrade
			YUM version: 4.14.0
			cachedir: /var/cache/dnf
			Unknown configuration option: async = 1 in /etc/yum.repos.d/clickhouse.repo
			Unknown configuration option: async = 1 in /etc/yum.repos.d/local.repo
			Unknown configuration option: async = 1 in /etc/yum.repos.d/nginx.repo
			Unknown configuration option: async = 1 in /etc/yum.repos.d/percona-ppg-11.repo
			Unknown configuration option: async = 1 in /etc/yum.repos.d/percona-ppg-14.repo
			Unknown configuration option: async = 1 in /etc/yum.repos.d/pmm2-server.repo
			User-Agent: constructed: 'libdnf (Oracle Linux Server 9.2; server; Linux.x86_64)'
			Installed Packages
			Name         : pmm-managed
			Version      : 2.39.0
			Release      : 20.2306271313.b6d58b6.el9
			Architecture : x86_64
			Size         : 125 M
			Source       : pmm-managed-2.39.0-20.2306271313.b6d58b6.el9.src.rpm
			Repository   : @System
			From repo    : local
			Packager     : None
			Buildtime    : Tue 27 Jun 2023 01:13:03 PM UTC
			Install time : Tue 27 Jun 2023 01:31:05 PM UTC
			Installed by : System <unset>
			Summary      : Percona Monitoring and Management management daemon
			URL          : https://github.com/percona/pmm
			License      : AGPLv3
			Description  : pmm-managed manages configuration of PMM server components (VictoriaMetrics,
									: Grafana, etc.) and exposes API for that. Those APIs are used by pmm-admin tool.
									: See PMM docs for more information.
		`, "\n")
		expected := map[string]string{
			"Name":         "pmm-managed",
			"Version":      "2.39.0",
			"Release":      "20.2306271313.b6d58b6.el9",
			"Architecture": "x86_64",
			"Size":         "125 M",
			"Source":       "pmm-managed-2.39.0-20.2306271313.b6d58b6.el9.src.rpm",
			"Repository":   "@System",
			"From repo":    "local",
			"Packager":     "None",
			"Buildtime":    "Tue 27 Jun 2023 01:13:03 PM UTC",
			"Install time": "Tue 27 Jun 2023 01:31:05 PM UTC",
			"Installed by": "System <unset>",
			"Summary":      "Percona Monitoring and Management management daemon",
			"URL":          "https://github.com/percona/pmm",
			"License":      "AGPLv3",
			"Description": "pmm-managed manages configuration of PMM server components (VictoriaMetrics, " +
				"Grafana, etc.) and exposes API for that. Those APIs are used by pmm-admin tool. " +
				"See PMM docs for more information.",
		}
		actual, err := parseInfo(stdout, "Name")
		require.NoError(t, err)
		assert.Equal(t, expected, actual)
		buildtime, err := parseInfoTime(actual["Buildtime"])
		require.NoError(t, err)
		assert.Equal(t, time.Date(2023, 6, 27, 13, 13, 3, 0, time.UTC), buildtime)
		assert.Equal(t, "2.39.0-20.2306271313.b6d58b6.el9", fullVersion(actual))
		assert.Equal(t, "2.39.0", niceVersion(actual))
	})

	t.Run("Updates EL9", func(t *testing.T) {
		// yum --verbose info updates pmm-update
		stdout := strings.Split(`
			Loaded plugins: builddep, changelog, config-manager, copr, debug, debuginfo-install, download, generate_completion_cache, groups-manager, needs-restarting, playground, repoclosure, repodiff, repograph, repomanage, reposync, system-upgrade
			YUM version: 4.14.0
			cachedir: /var/cache/dnf
			Unknown configuration option: async = 1 in /etc/yum.repos.d/clickhouse.repo
			Unknown configuration option: async = 1 in /etc/yum.repos.d/local.repo
			Unknown configuration option: async = 1 in /etc/yum.repos.d/nginx.repo
			Unknown configuration option: async = 1 in /etc/yum.repos.d/percona-ppg-11.repo
			Unknown configuration option: async = 1 in /etc/yum.repos.d/percona-ppg-14.repo
			Unknown configuration option: async = 1 in /etc/yum.repos.d/pmm2-server.repo
			User-Agent: constructed: 'libdnf (Oracle Linux Server 9.2; server; Linux.x86_64)'
			repo: using cache for: ol9_developer_EPEL
			ol9_developer_EPEL: using metadata from Wed 28 Jun 2023 02:28:50 PM UTC.
			repo: using cache for: ol9_baseos_latest
			ol9_baseos_latest: using metadata from Fri 23 Jun 2023 04:59:24 AM UTC.
			repo: using cache for: ol9_appstream
			ol9_appstream: using metadata from Fri 23 Jun 2023 05:03:17 AM UTC.
			repo: using cache for: percona-release-x86_64
			percona-release-x86_64: using metadata from Mon 26 Jun 2023 01:02:27 PM UTC.
			repo: using cache for: percona-release-noarch
			percona-release-noarch: using metadata from Wed 06 Jul 2022 08:25:44 PM UTC.
			repo: using cache for: percona-testing-x86_64
			percona-testing-x86_64: using metadata from Wed 28 Jun 2023 05:27:06 PM UTC.
			repo: using cache for: percona-testing-noarch
			percona-testing-noarch: using metadata from Wed 06 Jul 2022 08:20:55 PM UTC.
			repo: using cache for: percona-ppg-11
			percona-ppg-11: using metadata from Mon 22 May 2023 08:40:15 AM UTC.
			repo: using cache for: percona-ppg-14
			percona-ppg-14: using metadata from Wed 28 Jun 2023 02:57:51 PM UTC.
			repo: using cache for: prel-release-noarch
			prel-release-noarch: using metadata from Thu 16 Sep 2021 06:35:55 AM UTC.
			repo: using cache for: pmm2-server
			pmm2-server: using metadata from Wed 28 Jun 2023 02:46:09 PM UTC.
			Last metadata expiration check: 1:42:51 ago on Wed 28 Jun 2023 07:06:43 PM UTC.
			Available Upgrades
			Name         : pmm-update
			Version      : 2.39.0
			Release      : 67.2306281336.d0d7fcb.el9
			Architecture : noarch
			Size         : 886 k
			Source       : pmm-update-2.39.0-67.2306281336.d0d7fcb.el9.src.rpm
			Repository   : pmm2-server
			Packager     : None
			Buildtime    : Wed 28 Jun 2023 01:36:03 PM UTC
			Summary      : Tool for updating packages and OS configuration for PMM Server
			URL          : https://github.com/percona/pmm
			License      : AGPLv3
			Description  : Tool for updating packages and OS configuration for PMM Server
		`, "\n")
		expected := map[string]string{
			"Name":         "pmm-update",
			"Architecture": "noarch",
			"Version":      "2.39.0",
			"Release":      "67.2306281336.d0d7fcb.el9",
			"Size":         "886 k",
			"Source":       "pmm-update-2.39.0-67.2306281336.d0d7fcb.el9.src.rpm",
			"Repository":   "pmm2-server",
			"Packager":     "None",
			"Buildtime":    "Wed 28 Jun 2023 01:36:03 PM UTC",
			"Summary":      "Tool for updating packages and OS configuration for PMM Server",
			"URL":          "https://github.com/percona/pmm",
			"License":      "AGPLv3",
			"Description":  "Tool for updating packages and OS configuration for PMM Server",
		}
		actual, err := parseInfo(stdout, "Name")
		require.NoError(t, err)
		assert.Equal(t, expected, actual)
		buildtime, err := parseInfoTime(actual["Buildtime"])
		require.NoError(t, err)
		assert.Equal(t, time.Date(2023, 6, 28, 13, 36, 3, 0, time.UTC), buildtime)
		assert.Equal(t, "2.39.0-67.2306281336.d0d7fcb.el9", fullVersion(actual))
		assert.Equal(t, "2.39.0", niceVersion(actual))
	})

	t.Run("AvailableGA EL9", func(t *testing.T) {
		// yum --verbose --showduplicates info available pmm-update (just two versions)
		stdout := strings.Split(`
			Loaded plugins: builddep, changelog, config-manager, copr, debug, debuginfo-install, download, generate_completion_cache, groups-manager, needs-restarting, playground, repoclosure, repodiff, repograph, repomanage, reposync, system-upgrade
			YUM version: 4.14.0
			cachedir: /var/cache/dnf
			Unknown configuration option: async = 1 in /etc/yum.repos.d/clickhouse.repo
			Unknown configuration option: async = 1 in /etc/yum.repos.d/local.repo
			Unknown configuration option: async = 1 in /etc/yum.repos.d/nginx.repo
			Unknown configuration option: async = 1 in /etc/yum.repos.d/percona-ppg-11.repo
			Unknown configuration option: async = 1 in /etc/yum.repos.d/percona-ppg-14.repo
			Unknown configuration option: async = 1 in /etc/yum.repos.d/pmm2-server.repo
			User-Agent: constructed: 'libdnf (Oracle Linux Server 9.2; server; Linux.x86_64)'
			repo: using cache for: ol9_developer_EPEL
			ol9_developer_EPEL: using metadata from Wed 28 Jun 2023 02:28:50 PM UTC.
			repo: using cache for: ol9_baseos_latest
			ol9_baseos_latest: using metadata from Fri 23 Jun 2023 04:59:24 AM UTC.
			repo: using cache for: ol9_appstream
			ol9_appstream: using metadata from Fri 23 Jun 2023 05:03:17 AM UTC.
			repo: using cache for: percona-release-x86_64
			percona-release-x86_64: using metadata from Mon 26 Jun 2023 01:02:27 PM UTC.
			repo: using cache for: percona-release-noarch
			percona-release-noarch: using metadata from Wed 06 Jul 2022 08:25:44 PM UTC.
			repo: using cache for: percona-testing-x86_64
			percona-testing-x86_64: using metadata from Wed 28 Jun 2023 05:27:06 PM UTC.
			repo: using cache for: percona-testing-noarch
			percona-testing-noarch: using metadata from Wed 06 Jul 2022 08:20:55 PM UTC.
			repo: using cache for: percona-ppg-11
			percona-ppg-11: using metadata from Mon 22 May 2023 08:40:15 AM UTC.
			repo: using cache for: percona-ppg-14
			percona-ppg-14: using metadata from Wed 28 Jun 2023 02:57:51 PM UTC.
			repo: using cache for: prel-release-noarch
			prel-release-noarch: using metadata from Thu 16 Sep 2021 06:35:55 AM UTC.
			repo: using cache for: pmm2-server
			pmm2-server: using metadata from Wed 28 Jun 2023 02:46:09 PM UTC.
			Last metadata expiration check: 1:18:00 ago on Wed 28 Jun 2023 07:06:43 PM UTC.
			Available Packages
			Name         : pmm-update
			Version      : 2.39.0
			Release      : 67.2306280932.70f3748.el9
			Architecture : noarch
			Size         : 887 k
			Source       : pmm-update-2.39.0-67.2306280932.70f3748.el9.src.rpm
			Repository   : pmm2-server
			Packager     : None
			Buildtime    : Wed 28 Jun 2023 09:32:21 AM UTC
			Summary      : Tool for updating packages and OS configuration for PMM Server
			URL          : https://github.com/percona/pmm
			License      : AGPLv3
			Description  : Tool for updating packages and OS configuration for PMM Server

			Name         : pmm-update
			Version      : 2.39.0
			Release      : 67.2306281012.fe8e947.el9
			Architecture : noarch
			Size         : 887 k
			Source       : pmm-update-2.39.0-67.2306281012.fe8e947.el9.src.rpm
			Repository   : pmm2-server
			Packager     : None
			Buildtime    : Wed 28 Jun 2023 10:12:02 AM UTC
			Summary      : Tool for updating packages and OS configuration for PMM Server
			URL          : https://github.com/percona/pmm
			License      : AGPLv3
			Description  : Tool for updating packages and OS configuration for PMM Server
		`, "\n")
		expected := map[string]string{
			"Name":         "pmm-update",
			"Architecture": "noarch",
			"Version":      "2.39.0",
			"Release":      "67.2306280932.70f3748.el9",
			"Size":         "887 k",
			"Source":       "pmm-update-2.39.0-67.2306280932.70f3748.el9.src.rpm",
			"Repository":   "pmm2-server",
			"Packager":     "None",
			"Buildtime":    "Wed 28 Jun 2023 09:32:21 AM UTC",
			"Summary":      "Tool for updating packages and OS configuration for PMM Server",
			"URL":          "https://github.com/percona/pmm",
			"License":      "AGPLv3",
			"Description":  "Tool for updating packages and OS configuration for PMM Server",
		}
		actual, err := parseInfo(stdout, "Name")
		assert.EqualError(t, err, "second `Name` encountered")
		assert.Equal(t, expected, actual)
		buildtime, err := parseInfoTime(actual["Buildtime"])
		require.NoError(t, err)
		assert.Equal(t, time.Date(2023, 6, 28, 9, 32, 21, 0, time.UTC), buildtime)
		assert.Equal(t, "2.39.0-67.2306280932.70f3748.el9", fullVersion(actual))
		assert.Equal(t, "2.39.0", niceVersion(actual))
	})

	t.Run("Empty EL9", func(t *testing.T) {
		// yum --verbose info updates pmm-managed
		// "Error: No matching Packages to list" goes to stderr.
		// The output below is generated when there are no updates available.
		stdout := strings.Split(`
			Loaded plugins: builddep, changelog, config-manager, copr, debug, debuginfo-install, download, generate_completion_cache, groups-manager, needs-restarting, playground, repoclosure, repodiff, repograph, repomanage, reposync, system-upgrade
			YUM version: 4.14.0
			cachedir: /var/cache/dnf
			Unknown configuration option: async = 1 in /etc/yum.repos.d/clickhouse.repo
			Unknown configuration option: async = 1 in /etc/yum.repos.d/local.repo
			Unknown configuration option: async = 1 in /etc/yum.repos.d/nginx.repo
			Unknown configuration option: async = 1 in /etc/yum.repos.d/percona-ppg-11.repo
			Unknown configuration option: async = 1 in /etc/yum.repos.d/percona-ppg-14.repo
			Unknown configuration option: async = 1 in /etc/yum.repos.d/pmm2-server.repo
			User-Agent: constructed: 'libdnf (Oracle Linux Server 9.2; server; Linux.x86_64)'
			repo: using cache for: ol9_developer_EPEL
			ol9_developer_EPEL: using metadata from Wed 28 Jun 2023 02:28:50 PM UTC.
			repo: using cache for: ol9_baseos_latest
			ol9_baseos_latest: using metadata from Fri 23 Jun 2023 04:59:24 AM UTC.
			repo: using cache for: ol9_appstream
			ol9_appstream: using metadata from Fri 23 Jun 2023 05:03:17 AM UTC.
			repo: using cache for: percona-release-x86_64
			percona-release-x86_64: using metadata from Mon 26 Jun 2023 01:02:27 PM UTC.
			repo: using cache for: percona-release-noarch
			percona-release-noarch: using metadata from Wed 06 Jul 2022 08:25:44 PM UTC.
			repo: using cache for: percona-testing-x86_64
			percona-testing-x86_64: using metadata from Wed 28 Jun 2023 05:27:06 PM UTC.
			repo: using cache for: percona-testing-noarch
			percona-testing-noarch: using metadata from Wed 06 Jul 2022 08:20:55 PM UTC.
			repo: using cache for: percona-ppg-11
			percona-ppg-11: using metadata from Mon 22 May 2023 08:40:15 AM UTC.
			repo: using cache for: percona-ppg-14
			percona-ppg-14: using metadata from Wed 28 Jun 2023 02:57:51 PM UTC.
			repo: using cache for: prel-release-noarch
			prel-release-noarch: using metadata from Thu 16 Sep 2021 06:35:55 AM UTC.
			repo: using cache for: pmm2-server
			pmm2-server: using metadata from Wed 28 Jun 2023 02:46:09 PM UTC.
			Last metadata expiration check: 0:59:54 ago on Wed 28 Jun 2023 07:06:43 PM UTC.
		`, "\n")
		actual, err := parseInfo(stdout, "Name")
		require.NoError(t, err)
		assert.Empty(t, actual)
	})

	t.Run("RepoInfo EL9", func(t *testing.T) {
		// yum repoinfo pmm2-server.
		stdout := strings.Split(`
			Last metadata expiration check: 9:26:06 ago on Wed 28 Jun 2023 09:26:18 AM UTC.
			Repo-id            : pmm2-server
			Repo-name          : PMM Server YUM repository - x86_64
			Repo-status        : enabled
			Repo-revision      : 1687873070
			Repo-updated       : Tue 27 Jun 2023 01:25:23 PM UTC
			Repo-pkgs          : 478
			Repo-available-pkgs: 478
			Repo-size          : 3.7 G
			Repo-baseurl       : https://repo.percona.com/pmm2-components/yum/experimental/9/RPMS/x86_64/
			Repo-expire        : 172,800 second(s) (last: Wed 28 Jun 2023 09:26:18 AM UTC)
			Repo-filename      : /etc/yum.repos.d/pmm2-server.repo
			Total packages: 478
		`, "\n")
		expected := map[string]string{
			"Repo-id":             "pmm2-server",
			"Repo-name":           "PMM Server YUM repository - x86_64",
			"Repo-status":         "enabled",
			"Repo-revision":       "1687873070",
			"Repo-updated":        "Tue 27 Jun 2023 01:25:23 PM UTC",
			"Repo-pkgs":           "478",
			"Repo-available-pkgs": "478",
			"Repo-size":           "3.7 G",
			"Repo-baseurl":        "https://repo.percona.com/pmm2-components/yum/experimental/9/RPMS/x86_64/",
			"Repo-expire":         "172,800 second(s) (last: Wed 28 Jun 2023 09:26:18 AM UTC)",
			"Repo-filename":       "/etc/yum.repos.d/pmm2-server.repo",
			"Total packages":      "478",
		}
		actual, err := parseInfo(stdout, "Repo-id")
		require.NoError(t, err)
		assert.Equal(t, expected, actual)
		releasetime, err := parseInfoTime(actual["Repo-updated"])
		require.NoError(t, err)
		assert.Equal(t, time.Date(2023, time.June, 27, 13, 25, 23, 0, time.UTC), releasetime)
	})
}

func TestGetRHELVersion(t *testing.T) {
	t.Run("getRHELVersion EL9", func(t *testing.T) {
		actual, err := getRHELVersion()
		if actual == "7" {
			t.Skip("Skip running EL9 tests on EL7")
		}
		expected := "9"
		require.NoError(t, err)
		assert.Equal(t, expected, actual)
	})

	t.Run("getRHELVersion EL7", func(t *testing.T) {
		actual, err := getRHELVersion()
		if actual == "9" {
			t.Skip("Skip running EL7 test on EL9")
		}
		expected := "7"
		require.NoError(t, err)
		assert.Equal(t, expected, actual)
	})
}
