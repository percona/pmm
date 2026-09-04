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

// standaloneRemoteWrite picks the default remote-write pair for a vmagent when PMM Server is not
// clustered.
//
// Internal VictoriaMetrics (the default): every PMM Client writes through PMM Server's
// /victoriametrics/ write endpoint, authenticated with its own PMM Server credentials. The
// server's own agent has no vmagent in this mode, because VictoriaMetrics scrapes it directly.
//
// External VictoriaMetrics (PMM_VM_URL points off-box; see the external VictoriaMetrics guide in
// the user documentation): PMM Clients and the server's own agent write straight to it,
// authenticated with the credentials carried by PMM_VM_URL, if any. The documented
// VMAGENT_remoteWrite_basicAuth_* variables override those through the passthrough.
func standaloneRemoteWrite(params victoriaMetricsParams) (remoteWrite, error) {
	if params.ExternalVM() {
		return vmRemoteWrite(params.URL())
	}

	return serverProxyRemoteWrite(), nil
}
