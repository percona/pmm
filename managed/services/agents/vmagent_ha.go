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
//   - PMM Server's own agent runs inside the cluster and writes to vmauth directly. It connects to
//     its own pod's nginx at 127.0.0.1:8443 and holds no PMM Server credentials, so neither the
//     client URL (which never reaches the ingress) nor the authenticated write endpoint would
//     work for it.
//   - PMM Client agents may run outside the cluster, where vmauth is not reachable, so they write to
//     the PMM Server address they already connect to. The ingress in front of PMM Server (HAProxy
//     in the pmm-ha chart) must route /victoriametrics/api/v1/write to vmauth; PMM Server pods are
//     not on the metrics write path.
//
// A clustered PMM Server with a built-in VictoriaMetrics is not a supported topology. If
// PMM_VM_URL is left at its internal default, every agent gets the standalone pair: the server's
// own /victoriametrics/ write endpoint with the client's PMM Server credentials. That only lands
// metrics when nothing in front of PMM Server intercepts the path; HARemoteWriteWarning reports
// the shape at startup.
func haRemoteWrite(params victoriaMetricsParams, isServerAgent bool) (remoteWrite, error) {
	if !params.ExternalVM() {
		return serverProxyRemoteWrite(), nil
	}

	rw, err := vmRemoteWrite(params.URL())
	if err != nil {
		return remoteWrite{}, err
	}
	if isServerAgent {
		return rw, nil
	}

	rw.url = serverProxyWriteURL

	return rw, nil
}

// HARemoteWriteWarning returns a startup warning when the HA remote-write configuration cannot
// work as deployed, or "" when it can. The check is by shape only: an unparsable PMM_VM_URL is
// rejected by environment validation before this runs.
func HARemoteWriteWarning(params victoriaMetricsParams) string {
	if !params.ExternalVM() {
		return "HA mode with the built-in VictoriaMetrics is not supported: PMM_VM_URL must point at the cluster's VictoriaMetrics"
	}

	_, username, password, err := splitURLCredentials(params.URL())
	if err != nil || username != "" || password != "" {
		return ""
	}
	injected := injectedVMAgentEnv()
	if _, ok := injected[envRemoteWriteUsername]; ok {
		return ""
	}
	if _, ok := injected[envRemoteWritePassword]; ok {
		return ""
	}

	return "PMM_VM_URL carries no credentials and no VMAGENT_remoteWrite_basicAuth_* is set: " +
		"PMM Client metric writes will be sent without authentication and rejected by VictoriaMetrics; " +
		"store the VictoriaMetrics credentials in the PMM secret so that they reach PMM_VM_URL"
}
