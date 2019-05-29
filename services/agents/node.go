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
	"sort"

	"github.com/AlekSi/pointer"
	"github.com/percona/pmm/api/agentpb"

	"github.com/percona/pmm-managed/models"
)

func nodeExporterConfig(node *models.Node, exporter *models.Agent) *agentpb.SetStateRequest_AgentProcess {
	tdp := templateDelimsPair(
		pointer.GetString(exporter.MetricsURL),
	)

	args := []string{
		// TODO
		// "--collector.ntp",
		// "--collector.runit",
		// "--collector.supervisord",
		// "--collector.tcpstat",

		// TODO
		// "--collector.textfile",
		// "--collector.textfile.directory",

		"--web.listen-address=:" + tdp.left + " .listen_port " + tdp.right,
	}

	// do not enable Linux-specific collectors on macOS, that's useful for our development
	if node.Distro != "darwin" {
		args = append(args,
			// enable disabled by default
			"--collector.buddyinfo",
			"--collector.drbd",
			"--collector.interrupts",
			"--collector.ksmd",
			"--collector.meminfo_numa",
			"--collector.mountstats",
			"--collector.processes",
			"--collector.qdisc",
			"--collector.wifi",

			// add more netstat fields
			"--collector.netstat.fields=^(.*_(InErrors|InErrs|InCsumErrors)"+
				"|Tcp_(ActiveOpens|PassiveOpens|RetransSegs|CurrEstab|AttemptFails|OutSegs|InSegs|EstabResets|OutRsts|OutSegs)|Tcp_Rto(Algorithm|Min|Max)"+
				"|Udp_(RcvbufErrors|SndbufErrors)|Udp(6?|Lite6?)_(InDatagrams|OutDatagrams|RcvbufErrors|SndbufErrors|NoPorts)"+
				"|Icmp6?_(OutEchoReps|OutEchos|InEchos|InEchoReps|InAddrMaskReps|InAddrMasks|OutAddrMaskReps|OutAddrMasks|InTimestampReps|InTimestamps"+
				"|OutTimestampReps|OutTimestamps|OutErrors|InDestUnreachs|OutDestUnreachs|InTimeExcds|InRedirects|OutRedirects|InMsgs|OutMsgs)"+
				"|IcmpMsg_(InType3|OutType3)|Ip(6|Ext)_(InOctets|OutOctets)|Ip_Forwarding|TcpExt_(Listen.*|Syncookies.*|TCPTimeouts))$",

			// add more vmstat fileds
			"--collector.vmstat.fields=^(pg(steal_(kswapd|direct)|refill|alloc)_(movable|normal|dma3?2?)"+
				"|nr_(dirty.*|slab.*|vmscan.*|isolated.*|free.*|shmem.*|i?n?active.*|anon_transparent_.*|writeback.*|unstable"+
				"|unevictable|mlock|mapped|bounce|page_table_pages|kernel_stack)|drop_slab|slabs_scanned|pgd?e?activate)"+
				"|pgpg(in|out)|pswp(in|out)|pgm?a?j?fault)$",

			// Disabled for now due to https://jira.percona.com/browse/PMM-3843
			// "--collector.logind",
			// "--collector.systemd",
		)
	}

	if pointer.GetString(exporter.MetricsURL) != "" {
		args = append(args, "--web.telemetry-path="+*exporter.MetricsURL)
	}

	sort.Strings(args)

	return &agentpb.SetStateRequest_AgentProcess{
		Type:               agentpb.Type_NODE_EXPORTER,
		TemplateLeftDelim:  tdp.left,
		TemplateRightDelim: tdp.right,
		Args:               args,
	}
}
