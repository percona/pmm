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
	"errors"
	"fmt"
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
	envRemoteWriteURL      = "VMAGENT_remoteWrite_url"
	envRemoteWriteUsername = "VMAGENT_remoteWrite_basicAuth_username"
	envRemoteWritePassword = "VMAGENT_remoteWrite_basicAuth_password" //nolint:gosec
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
	// Operator-injected VMAGENT_remoteWrite_basicAuth_*.
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
func vmAgentConfig(l *logrus.Entry, scrapeCfg string, params victoriaMetricsParams, d vmAgentDeployment) (*agentv1.SetStateRequest_AgentProcess, error) {
	var (
		rw  remoteWrite
		err error
	)
	if d.haEnabled {
		rw, err = haRemoteWrite(params, d.isServerAgent)
	} else {
		rw, err = standaloneRemoteWrite(params)
	}
	if err != nil {
		return nil, err
	}

	return buildVMAgentProcess(l, scrapeCfg, rw), nil
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
func vmRemoteWrite(vmURL string) (remoteWrite, error) {
	base, username, password, err := parseURLCredentials(vmURL)
	if err != nil {
		return remoteWrite{}, err
	}

	source := credentialVMURL
	if username == "" && password == "" {
		source = credentialNone
	}

	return remoteWrite{
		url:      base.JoinPath("api/v1/write").String(),
		username: username,
		password: password,
		source:   source,
	}, nil
}

// parseURLCredentials parses a URL and moves its userinfo, if any, out of the returned URL and
// into the returned username and password (URL-decoded). The error never contains the URL,
// because the URL may carry a password.
func parseURLCredentials(urlStr string) (*url.URL, string, string, error) {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		var urlErr *url.Error
		if errors.As(err, &urlErr) {
			err = urlErr.Err
		}
		return nil, "", "", fmt.Errorf("cannot parse the VictoriaMetrics URL: %w", err)
	}
	if parsedURL.User == nil {
		return parsedURL, "", "", nil
	}

	username := parsedURL.User.Username()
	password, _ := parsedURL.User.Password()
	parsedURL.User = nil

	return parsedURL, username, password, nil
}

// splitURLCredentials is parseURLCredentials for callers that keep the URL as a string. A URL
// without credentials is returned unchanged.
func splitURLCredentials(urlStr string) (string, string, string, error) {
	parsedURL, username, password, err := parseURLCredentials(urlStr)
	if err != nil {
		return "", "", "", err
	}
	if username == "" && password == "" {
		return urlStr, "", "", nil
	}

	return parsedURL.String(), username, password, nil
}

// injectedVMAgentEnv returns every VMAGENT_* variable set in PMM Server's environment.
// This is the documented way to configure all vmagents centrally; whatever is set here is
// forwarded to every vmagent and wins over PMM's default of the same name.
func injectedVMAgentEnv() map[string]string {
	injected := make(map[string]string)
	for _, env := range os.Environ() {
		if !strings.HasPrefix(env, envvars.EnvVMAgentPrefix) {
			continue
		}
		if key, value, ok := strings.Cut(env, "="); ok {
			injected[key] = value
		}
	}

	return injected
}

// buildVMAgentProcess assembles the vmagent process from the path's default pair and the
// operator's VMAGENT_* environment. Two rules:
//  1. An injected VMAGENT_* variable always wins over PMM's default of the same name.
//  2. PMM's default credential belongs to PMM's default URL. When the operator injects
//     VMAGENT_remoteWrite_url, no default credential is emitted: an endpoint PMM did not choose
//     never receives a credential PMM derived. Injected credentials travel regardless, because they
//     belong with whatever the operator configured.
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
	_, usernameInjected := injected[envRemoteWriteUsername]
	_, passwordInjected := injected[envRemoteWritePassword]

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

	if !urlInjected {
		if rw.username != "" {
			addEnvIfNotInjected(envRemoteWriteUsername, rw.username)
		}
		if rw.password != "" {
			addEnvIfNotInjected(envRemoteWritePassword, rw.password)
		}
	}

	for key, value := range injected {
		envs = append(envs, key+"="+value)
	}

	sort.Strings(envs)
	sort.Strings(args)

	source := rw.source
	switch {
	case usernameInjected || passwordInjected:
		source = credentialInjected
	case urlInjected:
		source = credentialNone
	}
	remoteWriteURL := rw.url
	if urlInjected {
		// The injected URL may carry userinfo; log it without.
		remoteWriteURL = "injected"
		stripped, _, _, err := splitURLCredentials(injectedURL)
		if err == nil {
			remoteWriteURL = stripped
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
