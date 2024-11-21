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

package agents

import (
	"testing"

	"github.com/stretchr/testify/require"

	agentv1 "github.com/percona/pmm/api/agent/v1"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/version"
)

func TestAuthWebConfig(t *testing.T) {
	t.Parallel()

	t.Run("v2.26.1", func(t *testing.T) {
		t.Parallel()

		node := &models.Node{}
		exporter := &models.Agent{
			AgentID:   "agent-id",
			AgentType: models.NodeExporterType,
		}
		agentVersion := version.MustParse("2.26.1")

		actual, err := nodeExporterConfig(node, exporter, agentVersion)
		require.NoError(t, err, "Unable to build node exporter config")

		expected := &agentv1.SetStateRequest_AgentProcess{
			Env: []string{
				"HTTP_AUTH=pmm:agent-id",
			},
			TextFiles: map[string]string(nil),
		}

		require.Equal(t, expected.Env, actual.Env)
		require.Equal(t, expected.TextFiles, actual.TextFiles)
	})

	t.Run("v2.28.0", func(t *testing.T) {
		t.Parallel()

		node := &models.Node{}
		exporter := &models.Agent{
			AgentID:   "agent-id",
			AgentType: models.NodeExporterType,
		}
		agentVersion := version.MustParse("2.28.0")

		actual, err := nodeExporterConfig(node, exporter, agentVersion)
		require.NoError(t, err, "Unable to build node exporter config")

		expected := &agentv1.SetStateRequest_AgentProcess{
			Env: []string(nil),
			TextFiles: map[string]string{
				"webConfigPlaceholder": "basic_auth_users:\n    pmm: agent-id\n",
			},
		}

		require.Equal(t, expected.Env, actual.Env)
		require.Equal(t, expected.TextFiles, actual.TextFiles)
		require.Contains(t, actual.Args, "--web.config={{ .TextFiles.webConfigPlaceholder }}")
	})

	t.Run("v3.0.0", func(t *testing.T) {
		t.Parallel()

		node := &models.Node{}
		exporter := &models.Agent{
			AgentID:   "agent-id",
			AgentType: models.NodeExporterType,
		}
		agentVersion := version.MustParse("3.0.0")

		actual, err := nodeExporterConfig(node, exporter, agentVersion)
		require.NoError(t, err, "Unable to build node exporter config")

		expected := &agentv1.SetStateRequest_AgentProcess{
			Env: []string(nil),
			TextFiles: map[string]string{
				"webConfigPlaceholder": "basic_auth_users:\n    pmm: agent-id\n",
			},
		}

		require.Equal(t, expected.Env, actual.Env)
		require.Equal(t, expected.TextFiles, actual.TextFiles)
		require.Contains(t, actual.Args, "--web.config.file={{ .TextFiles.webConfigPlaceholder }}")
	})
}

func TestNodeExporterConfig(t *testing.T) {
	t.Parallel()

	t.Run("Linux", func(t *testing.T) {
		t.Parallel()

		node := &models.Node{
			Address: "1.2.3.4",
		}
		exporter := &models.Agent{
			AgentID:   "agent-id",
			AgentType: models.NodeExporterType,
		}
		agentVersion := version.MustParse("2.15.1")

		actual, err := nodeExporterConfig(node, exporter, agentVersion)
		require.NoError(t, err, "Unable to build node exporter config")

		expected := &agentv1.SetStateRequest_AgentProcess{
			Type:               inventoryv1.AgentType_AGENT_TYPE_NODE_EXPORTER,
			TemplateLeftDelim:  "{{",
			TemplateRightDelim: "}}",
			Args: []string{
				"--collector.bonding",
				"--collector.buddyinfo",
				"--collector.cpu",
				"--collector.diskstats",
				"--collector.entropy",
				"--collector.filefd",
				"--collector.filesystem",
				"--collector.hwmon",
				"--collector.loadavg",
				"--collector.meminfo",
				"--collector.meminfo_numa",
				"--collector.netdev",
				"--collector.netstat",
				"--collector.netstat.fields=^(.*_(InErrors|InErrs|InCsumErrors)" +
					"|Tcp_(ActiveOpens|PassiveOpens|RetransSegs|CurrEstab|AttemptFails|OutSegs|InSegs|EstabResets|OutRsts|OutSegs)|Tcp_Rto(Algorithm|Min|Max)" +
					"|Udp_(RcvbufErrors|SndbufErrors)|Udp(6?|Lite6?)_(InDatagrams|OutDatagrams|RcvbufErrors|SndbufErrors|NoPorts)" +
					"|Icmp6?_(OutEchoReps|OutEchos|InEchos|InEchoReps|InAddrMaskReps|InAddrMasks|OutAddrMaskReps|OutAddrMasks|InTimestampReps|InTimestamps" +
					"|OutTimestampReps|OutTimestamps|OutErrors|InDestUnreachs|OutDestUnreachs|InTimeExcds|InRedirects|OutRedirects|InMsgs|OutMsgs)" +
					"|IcmpMsg_(InType3|OutType3)|Ip(6|Ext)_(InOctets|OutOctets)|Ip_Forwarding|TcpExt_(Listen.*|Syncookies.*|TCPTimeouts))$",
				"--collector.processes",
				"--collector.standard.go",
				"--collector.standard.process",
				"--collector.stat",
				"--collector.textfile.directory.hr=" + pathsBase(agentVersion, "{{", "}}") + "/collectors/textfile-collector/high-resolution",
				"--collector.textfile.directory.lr=" + pathsBase(agentVersion, "{{", "}}") + "/collectors/textfile-collector/low-resolution",
				"--collector.textfile.directory.mr=" + pathsBase(agentVersion, "{{", "}}") + "/collectors/textfile-collector/medium-resolution",
				"--collector.textfile.hr",
				"--collector.textfile.lr",
				"--collector.textfile.mr",
				"--collector.time",
				"--collector.uname",
				"--collector.vmstat",
				"--collector.vmstat.fields=^(pg(steal_(kswapd|direct)|refill|alloc)_(movable|normal|dma3?2?)" +
					"|nr_(dirty.*|slab.*|vmscan.*|isolated.*|free.*|shmem.*|i?n?active.*|anon_transparent_.*|writeback.*|unstable" +
					"|unevictable|mlock|mapped|bounce|page_table_pages|kernel_stack)|drop_slab|slabs_scanned|pgd?e?activate" +
					"|pgpg(in|out)|pswp(in|out)|pgm?a?j?fault)$",
				"--no-collector.arp",
				"--no-collector.bcache",
				"--no-collector.conntrack",
				"--no-collector.drbd",
				"--no-collector.edac",
				"--no-collector.infiniband",
				"--no-collector.interrupts",
				"--no-collector.ipvs",
				"--no-collector.ksmd",
				"--no-collector.logind",
				"--no-collector.mdadm",
				"--no-collector.mountstats",
				"--no-collector.netclass",
				"--no-collector.nfs",
				"--no-collector.nfsd",
				"--no-collector.ntp",
				"--no-collector.qdisc",
				"--no-collector.runit",
				"--no-collector.sockstat",
				"--no-collector.supervisord",
				"--no-collector.systemd",
				"--no-collector.tcpstat",
				"--no-collector.timex",
				"--no-collector.wifi",
				"--no-collector.xfs",
				"--no-collector.zfs",
				"--web.disable-exporter-metrics",
				"--web.listen-address=0.0.0.0:{{ .listen_port }}",
			},
			Env: []string{
				"HTTP_AUTH=pmm:agent-id",
			},
		}
		requireNoDuplicateFlags(t, actual.Args)
		require.Equal(t, expected.Args, actual.Args)
		require.Equal(t, expected.Env, actual.Env)
		require.Equal(t, expected, actual)
	})

	t.Run("LinuxDisabledCollectors", func(t *testing.T) {
		t.Parallel()
		node := &models.Node{}
		exporter := &models.Agent{
			AgentID:   "agent-id",
			AgentType: models.NodeExporterType,
			ExporterOptions: &models.ExporterOptions{
				DisabledCollectors: []string{"cpu", "netstat", "netstat.fields", "vmstat", "meminfo"},
			},
		}
		agentVersion := version.MustParse("2.15.1")

		actual, err := nodeExporterConfig(node, exporter, agentVersion)
		require.NoError(t, err, "Unable to build node exporter config")

		expected := &agentv1.SetStateRequest_AgentProcess{
			Type:               inventoryv1.AgentType_AGENT_TYPE_NODE_EXPORTER,
			TemplateLeftDelim:  "{{",
			TemplateRightDelim: "}}",
			Args: []string{
				"--collector.bonding",
				"--collector.buddyinfo",
				"--collector.diskstats",
				"--collector.entropy",
				"--collector.filefd",
				"--collector.filesystem",
				"--collector.hwmon",
				"--collector.loadavg",
				"--collector.meminfo_numa",
				"--collector.netdev",
				"--collector.processes",
				"--collector.standard.go",
				"--collector.standard.process",
				"--collector.stat",
				"--collector.textfile.directory.hr=" + pathsBase(agentVersion, "{{", "}}") + "/collectors/textfile-collector/high-resolution",
				"--collector.textfile.directory.lr=" + pathsBase(agentVersion, "{{", "}}") + "/collectors/textfile-collector/low-resolution",
				"--collector.textfile.directory.mr=" + pathsBase(agentVersion, "{{", "}}") + "/collectors/textfile-collector/medium-resolution",
				"--collector.textfile.hr",
				"--collector.textfile.lr",
				"--collector.textfile.mr",
				"--collector.time",
				"--collector.uname",
				"--collector.vmstat.fields=^(pg(steal_(kswapd|direct)|refill|alloc)_(movable|normal|dma3?2?)" +
					"|nr_(dirty.*|slab.*|vmscan.*|isolated.*|free.*|shmem.*|i?n?active.*|anon_transparent_.*|writeback.*|unstable" +
					"|unevictable|mlock|mapped|bounce|page_table_pages|kernel_stack)|drop_slab|slabs_scanned|pgd?e?activate" +
					"|pgpg(in|out)|pswp(in|out)|pgm?a?j?fault)$",
				"--no-collector.arp",
				"--no-collector.bcache",
				"--no-collector.conntrack",
				"--no-collector.drbd",
				"--no-collector.edac",
				"--no-collector.infiniband",
				"--no-collector.interrupts",
				"--no-collector.ipvs",
				"--no-collector.ksmd",
				"--no-collector.logind",
				"--no-collector.mdadm",
				"--no-collector.mountstats",
				"--no-collector.netclass",
				"--no-collector.nfs",
				"--no-collector.nfsd",
				"--no-collector.ntp",
				"--no-collector.qdisc",
				"--no-collector.runit",
				"--no-collector.sockstat",
				"--no-collector.supervisord",
				"--no-collector.systemd",
				"--no-collector.tcpstat",
				"--no-collector.timex",
				"--no-collector.wifi",
				"--no-collector.xfs",
				"--no-collector.zfs",
				"--web.disable-exporter-metrics",
				"--web.listen-address=0.0.0.0:{{ .listen_port }}",
			},
			Env: []string{
				"HTTP_AUTH=pmm:agent-id",
			},
		}
		requireNoDuplicateFlags(t, actual.Args)
		require.Equal(t, expected.Args, actual.Args)
		require.Equal(t, expected.Env, actual.Env)
		require.Equal(t, expected, actual)
	})

	t.Run("MacOS", func(t *testing.T) {
		t.Parallel()
		node := &models.Node{
			Distro: "darwin",
		}
		exporter := &models.Agent{
			AgentID:   "agent-id",
			AgentType: models.NodeExporterType,
		}
		agentVersion := version.MustParse("2.15.1")

		actual, err := nodeExporterConfig(node, exporter, agentVersion)
		require.NoError(t, err, "Unable to build node exporter config")

		expected := &agentv1.SetStateRequest_AgentProcess{
			Type:               inventoryv1.AgentType_AGENT_TYPE_NODE_EXPORTER,
			TemplateLeftDelim:  "{{",
			TemplateRightDelim: "}}",
			Args: []string{
				"--collector.textfile.directory.hr=" + pathsBase(agentVersion, "{{", "}}") + "/collectors/textfile-collector/high-resolution",
				"--collector.textfile.directory.lr=" + pathsBase(agentVersion, "{{", "}}") + "/collectors/textfile-collector/low-resolution",
				"--collector.textfile.directory.mr=" + pathsBase(agentVersion, "{{", "}}") + "/collectors/textfile-collector/medium-resolution",
				"--web.disable-exporter-metrics",
				"--web.listen-address=0.0.0.0:{{ .listen_port }}",
			},
			Env: []string{
				"HTTP_AUTH=pmm:agent-id",
			},
		}
		requireNoDuplicateFlags(t, actual.Args)
		require.Equal(t, expected.Args, actual.Args)
		require.Equal(t, expected.Env, actual.Env)
		require.Equal(t, expected, actual)
	})
}
