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

// haRemoteWrite picks the default remote-write pair for a vmagent when PMM Server runs in HA mode.
//
// In HA, VictoriaMetrics is a cluster service behind vmauth. PMM_VM_URL points at vmauth and
// carries its credentials: the pmm-ha chart composes the URL from the same secret keys vmauth is
// configured with. Every HA write therefore authenticates with that credential, and only the URL
// differs by agent:
//
//   - PMM Server's own agent runs inside the cluster and writes to vmauth directly. It also has no
//     PMM Server credentials of its own, so it could not use the authenticated write endpoint.
//   - PMM Client agents may run outside the cluster, where vmauth is not reachable, so they write to
//     the PMM Server address they already connect to. The chart's HAProxy routes
//     /victoriametrics/api/v1/write to vmauth (PMM-14678); PMM Server pods are not on the metrics
//     write path.
//
// A clustered PMM Server with a built-in VictoriaMetrics is not a supported topology, but if
// PMM_VM_URL is left at its internal default the only write route is the server's own endpoint,
// so clients get the same pair as in standalone mode.
func haRemoteWrite(params victoriaMetricsParams, isServerAgent bool) remoteWrite {
	if !params.ExternalVM() {
		return serverProxyRemoteWrite()
	}

	rw := vmRemoteWrite(params.URL())
	if isServerAgent {
		return rw
	}

	rw.url = serverProxyWriteURL

	return rw
}
