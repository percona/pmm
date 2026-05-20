// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package base provides helpers for all commands.
package base

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/go-openapi/runtime"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/admin/agentlocal"
	"github.com/percona/pmm/admin/pkg/flags"
	agentconfig "github.com/percona/pmm/agent/config"
	inventoryClient "github.com/percona/pmm/api/inventory/v1/json/client"
	managementClient "github.com/percona/pmm/api/management/v1/json/client"
	serverClient "github.com/percona/pmm/api/server/v1/json/client"
	"github.com/percona/pmm/utils/encryption"
	"github.com/percona/pmm/utils/tlsconfig"
)

type nginxError string

func (e nginxError) Error() string {
	return "response from nginx: " + string(e)
}

func (e nginxError) GoString() string {
	return fmt.Sprintf("nginxError(%q)", string(e))
}

// check interfaces.
var (
	_ error          = nginxError("")
	_ fmt.GoStringer = nginxError("")
)

// SetupClients configures local and PMM Server API clients.
func SetupClients(ctx context.Context, globalFlags *flags.GlobalFlags) {
	//nolint:nestif
	if globalFlags.ServerURL == nil || globalFlags.ServerURL.String() == "" {
		status, err := agentlocal.GetStatus(agentlocal.DoNotRequestNetworkInfo) //nolint:contextcheck
		if err != nil {
			if err == agentlocal.ErrNotSetUp { //nolint:errorlint
				logrus.Fatalf("Failed to get PMM Server parameters from local pmm-agent: %s.\n"+
					"Please run `pmm-admin config` with --server-url flag.", err)
			}

			if err == agentlocal.ErrNotConnected { //nolint:errorlint
				logrus.Fatalf("Failed to get PMM Server parameters from local pmm-agent: %s.\n", err)
			}
			logrus.Fatalf("Failed to get PMM Server parameters from local pmm-agent: %s.\n"+
				"Please use --server-url flag to specify PMM Server URL.", err)
		}
		globalFlags.ServerURL, _ = url.Parse(status.ServerURL)
		globalFlags.SkipTLSCertificateCheck = status.ServerInsecureTLS

		mergeCredsFromAgentConfig(globalFlags, status.ConfigFilepath)
	} else {
		if globalFlags.ServerURL.Path == "" {
			globalFlags.ServerURL.Path = "/"
		}
		switch globalFlags.ServerURL.Scheme {
		case "http", "https":
			// nothing
		default:
			logrus.Fatalf("Invalid PMM Server URL %q: scheme (https:// or http://) is missing.", globalFlags.ServerURL)
		}
		if globalFlags.ServerURL.Host == "" {
			logrus.Fatalf("Invalid PMM Server URL %q: host is missing.", globalFlags.ServerURL)
		}
	}

	// use JSON APIs over HTTP/1.1
	transport := httptransport.New(globalFlags.ServerURL.Host, globalFlags.ServerURL.Path, []string{globalFlags.ServerURL.Scheme})
	if u := globalFlags.ServerURL.User; u != nil {
		user := u.Username()
		password, _ := u.Password()
		if user == "service_token" || user == "api_key" {
			transport.DefaultAuthentication = httptransport.BearerToken(password)
		} else {
			transport.DefaultAuthentication = httptransport.BasicAuth(user, password)
		}
	}
	transport.SetLogger(logrus.WithField("component", "server-transport"))
	transport.SetDebug(globalFlags.EnableDebug || globalFlags.EnableTrace)
	transport.Context = ctx

	// set error handlers for nginx responses if pmm-managed is down
	errorConsumer := runtime.ConsumerFunc(func(reader io.Reader, _ any) error {
		b, _ := io.ReadAll(reader)
		return nginxError(string(b))
	})
	transport.Consumers = map[string]runtime.Consumer{
		runtime.JSONMime:    runtime.JSONConsumer(),
		"application/zip":   runtime.ByteStreamConsumer(),
		runtime.HTMLMime:    errorConsumer,
		runtime.TextMime:    errorConsumer,
		runtime.DefaultMime: errorConsumer,
	}

	// disable HTTP/2, set TLS config
	httpTransport, ok := transport.Transport.(*http.Transport)
	if !ok {
		panic("cannot assert transport as http.Transport")
	}

	httpTransport.TLSNextProto = make(map[string]func(string, *tls.Conn) http.RoundTripper)
	if globalFlags.ServerURL.Scheme == "https" {
		httpTransport.TLSClientConfig = tlsconfig.Get()
		httpTransport.TLSClientConfig.ServerName = globalFlags.ServerURL.Hostname()
		httpTransport.TLSClientConfig.InsecureSkipVerify = globalFlags.SkipTLSCertificateCheck
	}

	inventoryClient.Default.SetTransport(transport)
	managementClient.Default.SetTransport(transport)
	serverClient.Default.SetTransport(transport)
}

// mergeCredsFromAgentConfig reads the pmm-agent configuration file and merges
// server credentials into globalFlags.ServerURL. When pmm-agent uses config file
// encryption, the /local/Status endpoint returns a redacted URL.
// This method reads the config file directly to obtain the real credentials.
func mergeCredsFromAgentConfig(globalFlags *flags.GlobalFlags, statusConfigFilepath string) {
	// Skip the file read when Status already returned full credentials (plaintext config case).
	if u := globalFlags.ServerURL.User; u != nil {
		if password, ok := u.Password(); ok && password != agentconfig.RedactedPassword {
			return
		}
	}
	configFile := globalFlags.AgentConfigFile
	if configFile == "" {
		configFile = statusConfigFilepath
	}
	if configFile == "" {
		logrus.Warn("Agent config file path is not available; cannot read server credentials. " +
			"Use --agent-config-file or --server-url flag.")
		return
	}

	var enc *encryption.Encryption
	if globalFlags.AgentConfigKeyFile != "" {
		enc = &encryption.Encryption{
			KeyFile:         globalFlags.AgentConfigKeyFile,
			KeyFilePassword: globalFlags.AgentConfigKeyPassword,
		}
	}

	cfg, err := agentconfig.LoadFromFile(configFile, enc)
	if err != nil {
		logrus.Warnf("Failed to read agent config file %s: %s. "+
			"Use --server-url flag to specify PMM Server URL with credentials.", configFile, err)
		return
	}

	serverURL := cfg.Server.URL()
	if serverURL == nil {
		logrus.Warn("Agent config file has no server address configured.")
		return
	}

	globalFlags.ServerURL.User = serverURL.User
}
