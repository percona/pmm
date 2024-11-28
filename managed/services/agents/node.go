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
	"sort"

	"github.com/AlekSi/pointer"
	agentv1 "github.com/percona/pmm/api/agent/v1"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/collectors"
	"github.com/percona/pmm/version"
)

// The node exporter prior 2.28 use exporter_shared and gets basic auth config from env.
// Starting with pmm 2.28, the exporter uses Prometheus Web Toolkit and needs a config file
// with the basic auth users.
var (
	v2_28_00 = version.MustParse("2.28.0-0")
)

func nodeExporterConfig(node *models.Node, exporter *models.Agent, agentVersion *version.Parsed) (*agentv1.SetStateRequest_AgentProcess, error) {
	listenAddress := getExporterListenAddress(node, exporter)
	if exporter.ExporterOptions == nil {
		exporter.ExporterOptions = &models.ExporterOptions{}
	}
	tdp := models.TemplateDelimsPair(pointer.GetString(exporter.ExporterOptions.MetricsPath))
	args := []string{
		"--collector.textfile.directory.lr=" + pathsBase(agentVersion, tdp.Left, tdp.Right) + "/collectors/textfile-collector/low-resolution",
		"--collector.textfile.directory.mr=" + pathsBase(agentVersion, tdp.Left, tdp.Right) + "/collectors/textfile-collector/medium-resolution",
		"--collector.textfile.directory.hr=" + pathsBase(agentVersion, tdp.Left, tdp.Right) + "/collectors/textfile-collector/high-resolution",

		"--web.disable-exporter-metrics", // we enable them as a part of HR metrics

		"--web.listen-address=" + listenAddress + ":" + tdp.Left + " .listen_port " + tdp.Right,
	}

	// do not tweak collectors on macOS as many (but not) of them are Linux-specific
	if node.Distro != "darwin" {
		args = append(args,
			// LR
			"--collector.bonding",
			"--collector.entropy",
			"--collector.uname",
			"--collector.textfile.lr",

			// MR
			"--collector.textfile.mr",
			"--collector.hwmon",

			// HR
			"--collector.buddyinfo",
			"--collector.cpu",
			"--collector.diskstats",
			"--collector.filefd",
			"--collector.filesystem",
			"--collector.loadavg",
			"--collector.meminfo",
			"--collector.meminfo_numa",
			"--collector.netdev",
			"--collector.netstat",
			"--collector.processes",
			"--collector.stat",
			"--collector.time",
			"--collector.vmstat",
			"--collector.textfile.hr",
			"--collector.standard.go",
			"--collector.standard.process",

			// disabled
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
				"|unevictable|mlock|mapped|bounce|page_table_pages|kernel_stack)|drop_slab|slabs_scanned|pgd?e?activate"+
				"|pgpg(in|out)|pswp(in|out)|pgm?a?j?fault)$",
		)
	}

	args = collectors.FilterOutCollectors("--collector.", args, exporter.ExporterOptions.DisabledCollectors)

	if exporter.ExporterOptions.MetricsPath != nil {
		args = append(args, "--web.telemetry-path="+*exporter.ExporterOptions.MetricsPath)
	}

	args = withLogLevel(args, exporter.LogLevel, agentVersion, false)

	sort.Strings(args)

	params := &agentv1.SetStateRequest_AgentProcess{
		Type:               inventoryv1.AgentType_AGENT_TYPE_NODE_EXPORTER,
		TemplateLeftDelim:  tdp.Left,
		TemplateRightDelim: tdp.Right,
		Args:               args,
	}

	if err := ensureAuthParams(exporter, params, agentVersion, v2_28_00, agentVersion.IsFeatureSupported(version.NodeExporterNewTLSConfigVersion)); err != nil {
		return nil, err
	}

	return params, nil
}
