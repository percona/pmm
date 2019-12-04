// pmm-managed
// Copyright (C) 2017 Percona LLC
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

	"github.com/percona/pmm/api/agentpb"
	"github.com/percona/pmm/api/inventorypb"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm-managed/models"
)

func TestNodeExporterConfig(t *testing.T) {
	t.Run("Linux", func(t *testing.T) {
		node := &models.Node{}
		exporter := &models.Agent{
			AgentID: "agent-id",
		}
		actual := nodeExporterConfig(node, exporter)
		expected := &agentpb.SetStateRequest_AgentProcess{
			Type:               inventorypb.AgentType_NODE_EXPORTER,
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
				"--collector.textfile.directory.hr=/usr/local/percona/pmm2/collectors/textfile-collector/high-resolution",
				"--collector.textfile.directory.lr=/usr/local/percona/pmm2/collectors/textfile-collector/low-resolution",
				"--collector.textfile.directory.mr=/usr/local/percona/pmm2/collectors/textfile-collector/medium-resolution",
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
				"--web.listen-address=:{{ .listen_port }}",
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
		node := &models.Node{
			Distro: "darwin",
		}
		exporter := &models.Agent{
			AgentID: "agent-id",
		}
		actual := nodeExporterConfig(node, exporter)
		expected := &agentpb.SetStateRequest_AgentProcess{
			Type:               inventorypb.AgentType_NODE_EXPORTER,
			TemplateLeftDelim:  "{{",
			TemplateRightDelim: "}}",
			Args: []string{
				"--collector.textfile.directory.hr=/usr/local/percona/pmm2/collectors/textfile-collector/high-resolution",
				"--collector.textfile.directory.lr=/usr/local/percona/pmm2/collectors/textfile-collector/low-resolution",
				"--collector.textfile.directory.mr=/usr/local/percona/pmm2/collectors/textfile-collector/medium-resolution",
				"--web.disable-exporter-metrics",
				"--web.listen-address=:{{ .listen_port }}",
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
