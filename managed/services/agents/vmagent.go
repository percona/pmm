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
	"net/url"
	"os"
	"sort"
	"strings"

	"github.com/sirupsen/logrus"

	agentv1 "github.com/percona/pmm/api/agent/v1"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	"github.com/percona/pmm/managed/utils/envvars"
)

var (
	maxScrapeSizeEnv     = "PMM_PROMSCRAPE_MAX_SCRAPE_SIZE"
	maxScrapeSizeDefault = "64MiB"
)

// defaultRemoteWriteMaxDiskUsage is the on-disk queue vmagent may fill per remote-write URL
// while the endpoint is unreachable: 1 GiB.
const defaultRemoteWriteMaxDiskUsage = "1073741824"

// Environment variable names vmagent reads through -envflag.prefix=VMAGENT_.
// The password name is a variable name, not a secret.
const (
	envRemoteWriteURL      = envvars.EnvVMAgentRemoteWriteURL
	envRemoteWriteUsername = envvars.EnvVMAgentRemoteWriteUsername
	envRemoteWritePassword = envvars.EnvVMAgentRemoteWritePassword
)

// Placeholders rendered on the client from its own pmm-agent.yaml (agent/agents/supervisor).
// They are templates, not secrets.
const (
	serverProxyWriteURL = "{{.server_url}}/victoriametrics/api/v1/write"
	serverUsernameTmpl  = "{{.server_username}}"
	serverPasswordTmpl  = "{{.server_password}}" //nolint:gosec
)

// vmAgentDeployment selects which remote-write path a vmagent gets. It is the only input that
// depends on the deployment mode; each path is self-contained after the selection.
type vmAgentDeployment struct {
	// PMM Server runs in HA (clustered) mode.
	haEnabled bool
	// This is PMM Server's own built-in agent.
	isServerAgent bool
}

// credentialSource names where a remote-write credential comes from. Used in logs only.
type credentialSource string

const (
	// The client's own PMM Server credentials, rendered on the client.
	credentialPMMServer credentialSource = "pmm-server"
	// The userinfo of PMM_VM_URL.
	credentialVMURL credentialSource = "vm-url"
	// Operator-injected VMAGENT_remoteWrite_* authentication.
	credentialInjected credentialSource = "injected"
	// No credential: the endpoint needs none, or PMM has none for it.
	credentialNone credentialSource = "none"
)

// remoteWrite is the default (url, credential) pair a path picks for one vmagent.
// Operator-injected VMAGENT_* variables are layered on top of it by buildVMAgentProcess.
type remoteWrite struct {
	url      string
	username string
	password string
	source   credentialSource
}

// vmAgentConfig returns the desired configuration of a vmagent process. The deployment mode is
// consulted here and nowhere else: HA and standalone each pick their default remote-write pair
// (vmagent_ha.go, vmagent_standalone.go), and the shared builder applies the operator's
// VMAGENT_* environment on top.
func vmAgentConfig(l *logrus.Entry, scrapeCfg string, params victoriaMetricsParams, d vmAgentDeployment) *agentv1.SetStateRequest_AgentProcess {
	var rw remoteWrite
	if d.haEnabled {
		rw = haRemoteWrite(params, d.isServerAgent)
	} else {
		rw = standaloneRemoteWrite(params)
	}

	return buildVMAgentProcess(l, scrapeCfg, rw)
}

// serverProxyRemoteWrite writes through PMM Server's /victoriametrics/ write endpoint with the
// client's own PMM Server credentials. Every placeholder is rendered on the client, so the URL is
// the address the client already reaches and the credentials are the ones it already holds.
func serverProxyRemoteWrite() remoteWrite {
	return remoteWrite{
		url:      serverProxyWriteURL,
		username: serverUsernameTmpl,
		password: serverPasswordTmpl,
		source:   credentialPMMServer,
	}
}

// vmRemoteWrite writes straight to the VictoriaMetrics at vmURL. Credentials, if the URL carries
// any, move out of the URL and into the pair so that they reach vmagent through its environment
// only, never on its command line or inside the URL.
func vmRemoteWrite(vmURL *url.URL) remoteWrite {
	base, username, password := splitUserinfo(vmURL)

	source := credentialVMURL
	if username == "" && password == "" {
		source = credentialNone
	}

	return remoteWrite{
		url:      base.JoinPath("api/v1/write").String(),
		username: username,
		password: password,
		source:   source,
	}
}

// splitUserinfo returns a copy of u without its userinfo, and the username and password the
// userinfo carried, URL-decoded and empty when absent.
func splitUserinfo(u *url.URL) (*url.URL, string, string) {
	stripped := *u
	stripped.User = nil
	password, _ := u.User.Password()

	return &stripped, u.User.Username(), password
}

// injectedVMAgentEnv returns every VMAGENT_* variable set to a value in PMM Server's environment.
// This is the documented way to configure all vmagents centrally; whatever is set here is
// forwarded to every vmagent and wins over PMM's default of the same name. An empty variable is
// ignored, because vmagent cannot use an empty value; environment validation warns about it.
func injectedVMAgentEnv() map[string]string {
	injected := make(map[string]string)
	for _, env := range os.Environ() {
		if !strings.HasPrefix(env, envvars.EnvVMAgentPrefix) {
			continue
		}
		if key, value, ok := strings.Cut(env, "="); ok && value != "" {
			injected[key] = value
		}
	}

	return injected
}

// buildVMAgentProcess assembles the vmagent process from the path's default pair and the
// operator's VMAGENT_* environment. Two rules:
//  1. An injected VMAGENT_* variable always wins over PMM's default of the same name.
//  2. PMM's default credential is emitted only when the operator injected neither the endpoint nor
//     a credential of their own. An endpoint PMM did not choose never receives a credential PMM
//     derived, and an operator's credential is taken whole: half an injected basic-auth pair is
//     not completed with PMM's other half, and a bearer token or OAuth2 client is not combined
//     with PMM's pair, since vmagent refuses to start with both. Custom headers and a client TLS
//     certificate compose with basic auth and leave PMM's pair in place. Injected variables
//     travel regardless, because they belong with whatever the operator configured.
func buildVMAgentProcess(l *logrus.Entry, scrapeCfg string, rw remoteWrite) *agentv1.SetStateRequest_AgentProcess {
	interfaceToBind := envvars.GetInterfaceToBind()
	// These stay command-line flags on purpose: vmagent gives a flag priority over the
	// environment variable of the same name, so an injected VMAGENT_* cannot move the scrape
	// config, the temp dir, or the listen address away from where pmm-agent manages them.
	args := []string{
		"-envflag.enable=true",
		"-envflag.prefix=VMAGENT_",
		"-remoteWrite.tmpDataPath={{.tmp_dir}}/vmagent-temp-dir",
		"-promscrape.config={{.TextFiles.vmagentscrapecfg}}",
		"-httpListenAddr=" + interfaceToBind + ":{{.listen_port}}",
	}

	maxScrapeSize := maxScrapeSizeDefault
	if space := os.Getenv(maxScrapeSizeEnv); space != "" {
		maxScrapeSize = space
	}

	injected := injectedVMAgentEnv()
	injectedURL, urlInjected := injected[envRemoteWriteURL]
	credentialReplaced := envvars.VMAgentRemoteWriteReplacesBasicAuth(injected)

	var envs []string
	addEnvIfNotInjected := func(key, value string) {
		if _, exists := injected[key]; !exists {
			envs = append(envs, key+"="+value)
		}
	}

	addEnvIfNotInjected(envRemoteWriteURL, rw.url)
	addEnvIfNotInjected("VMAGENT_remoteWrite_tlsInsecureSkipVerify", "{{.server_insecure}}")
	addEnvIfNotInjected("VMAGENT_promscrape_maxScrapeSize", maxScrapeSize)
	addEnvIfNotInjected("VMAGENT_remoteWrite_maxDiskUsagePerURL", defaultRemoteWriteMaxDiskUsage)
	addEnvIfNotInjected("VMAGENT_loggerLevel", "INFO")

	if !urlInjected && !credentialReplaced {
		if rw.username != "" {
			envs = append(envs, envRemoteWriteUsername+"="+rw.username)
		}
		if rw.password != "" {
			envs = append(envs, envRemoteWritePassword+"="+rw.password)
		}
	}

	for key, value := range injected {
		envs = append(envs, key+"="+value)
	}

	sort.Strings(envs)
	sort.Strings(args)

	source := rw.source
	switch {
	case credentialReplaced:
		source = credentialInjected
	case urlInjected:
		source = credentialNone
	}
	remoteWriteURL := rw.url
	if urlInjected {
		// The injected URL may carry userinfo; log it without.
		remoteWriteURL = "injected"
		u, err := url.Parse(injectedURL)
		if err == nil {
			u.User = nil
			remoteWriteURL = u.String()
		}
	}
	l.WithFields(logrus.Fields{
		"remote_write_url":  remoteWriteURL,
		"credential_source": source,
	}).Debug("vmagent remote-write configured")

	return &agentv1.SetStateRequest_AgentProcess{
		Type:               inventoryv1.AgentType_AGENT_TYPE_VM_AGENT,
		TemplateLeftDelim:  "{{",
		TemplateRightDelim: "}}",
		Args:               args,
		Env:                envs,
		TextFiles: map[string]string{
			"vmagentscrapecfg": scrapeCfg,
		},
	}
}
