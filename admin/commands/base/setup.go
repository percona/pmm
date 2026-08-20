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
	"errors"
	"net/url"

	"github.com/go-openapi/runtime"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/admin/agentlocal"
	"github.com/percona/pmm/admin/pkg/flags"
	inventoryClient "github.com/percona/pmm/api/inventory/v1/json/client"
	managementClient "github.com/percona/pmm/api/management/v1/json/client"
	serverClient "github.com/percona/pmm/api/server/v1/json/client"
	"github.com/percona/pmm/utils/apitransport"
	"github.com/percona/pmm/utils/servererror"
)

// normalizeServerURL fills in the parts of a PMM Server URL which have a default and reports
// the ones which do not. Both the URL passed with --server-url and the one reported by the
// local pmm-agent go through it, so that an unusable URL is diagnosed here instead of turning
// into an opaque dial failure later.
func normalizeServerURL(u *url.URL) error {
	switch u.Scheme {
	case "http", "https":
	default:
		return errors.New("scheme (https:// or http://) is missing")
	}

	if u.Host == "" {
		return errors.New("host is missing")
	}

	// go-openapi requires a base path.
	if u.Path == "" {
		u.Path = "/"
	}

	return nil
}

// applyAgentServerParams fills in the PMM Server connection parameters reported by the local
// pmm-agent. An explicitly passed --server-insecure-tls is preserved: the flag is opt-in only,
// so a user asking to skip validation must not have that request dropped just because the
// local pmm-agent is configured to validate certificates.
func applyAgentServerParams(globalFlags *flags.GlobalFlags, status *agentlocal.Status) error {
	u, err := url.Parse(status.ServerURL)
	if err != nil {
		return err
	}

	err = normalizeServerURL(u)
	if err != nil {
		return err
	}

	globalFlags.ServerURL = u
	globalFlags.SkipTLSCertificateCheck = globalFlags.SkipTLSCertificateCheck || status.ServerInsecureTLS

	return nil
}

// SetupClients configures local and PMM Server API clients.
func SetupClients(globalFlags *flags.GlobalFlags) {
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
		err = applyAgentServerParams(globalFlags, status)
		if err != nil {
			logrus.Fatalf("Invalid PMM Server URL %q reported by local pmm-agent: %s.\n"+
				"Please use --server-url flag to specify PMM Server URL.", status.ServerURL, err)
		}
	} else {
		err := normalizeServerURL(globalFlags.ServerURL)
		if err != nil {
			logrus.Fatalf("Invalid PMM Server URL %q: %s.", globalFlags.ServerURL, err)
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

	// set error handlers for nginx responses if pmm-managed is down
	errorConsumer := servererror.NginxConsumer()
	transport.Consumers = map[string]runtime.Consumer{
		runtime.JSONMime:    runtime.JSONConsumer(),
		"application/zip":   runtime.ByteStreamConsumer(),
		runtime.HTMLMime:    errorConsumer,
		runtime.TextMime:    errorConsumer,
		runtime.DefaultMime: errorConsumer,
	}

	// disable HTTP/2, set TLS config
	apitransport.Configure(
		transport,
		globalFlags.ServerURL.Scheme,
		globalFlags.ServerURL.Hostname(),
		globalFlags.SkipTLSCertificateCheck,
	)

	inventoryClient.Default.SetTransport(transport)
	managementClient.Default.SetTransport(transport)
	serverClient.Default.SetTransport(transport)
}
