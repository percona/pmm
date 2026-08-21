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
	"regexp"

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

// credentialPattern matches a URL userinfo component - "user:password@" - occurring anywhere in
// a string. It is a defensive final pass applied on top of structured parsing: url.URL.Redacted
// only recognises userinfo in a properly structured "scheme://user:pass@host" URL. A URL typed
// without "//" - a missing scheme being exactly the kind of mistake this code has to diagnose -
// parses as opaque instead, with "user:password" sitting in Opaque, in cleartext, which
// Redacted does not look at; a URL that fails to parse at all is not touched by it either.
//
// The password half deliberately allows "/": a scheme prefix such as "https:" would otherwise
// also satisfy "name:" and swallow it into the match, but the password can legitimately contain
// a slash, and excluding it let such a password slip through unredacted.
var credentialPattern = regexp.MustCompile(`([^/@:\s]+):(?:[^/@\s][^@\s]*)?@`)

// redactedServerURL returns raw with any password it carries replaced by a placeholder, so a
// PMM Server URL can be logged without leaking its credentials - however malformed the URL
// turns out to be. Structured parsing is tried first since it redacts a well-formed URL
// cleanly; credentialPattern is a defensive final pass over the result.
func redactedServerURL(raw string) string {
	candidate := raw

	u, err := url.Parse(raw)
	if err == nil {
		candidate = u.Redacted()
	}

	return credentialPattern.ReplaceAllString(candidate, "$1:xxxxx@")
}

// sanitizeURLError returns a safe-to-log form of err: url.Parse embeds the exact string it
// failed on - password included - in its own error text, so a *url.Error is reduced to its
// underlying reason instead of being logged as received.
func sanitizeURLError(err error) string {
	urlErr, ok := errors.AsType[*url.Error](err)
	if ok {
		return urlErr.Err.Error()
	}

	return err.Error()
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
			// status.ServerURL and err may both carry a password - reported by pmm-agent
			// in the first case, embedded by url.Parse's own error text in the second.
			logrus.Fatalf("Invalid PMM Server URL %q reported by local pmm-agent: %s.\n"+
				"Please use --server-url flag to specify PMM Server URL.",
				redactedServerURL(status.ServerURL), sanitizeURLError(err))
		}
	} else {
		err := normalizeServerURL(globalFlags.ServerURL)
		if err != nil {
			// globalFlags.ServerURL comes from kong's own url.Parse of --server-url, so a
			// scheme-less value (exactly what triggers this branch) hits the same opaque-URL
			// gap redactedServerURL exists for - hence going through it rather than Redacted.
			logrus.Fatalf("Invalid PMM Server URL %q: %s.",
				redactedServerURL(globalFlags.ServerURL.String()), sanitizeURLError(err))
		}
	}

	// use JSON APIs over HTTP/1.1
	transport := httptransport.New(globalFlags.ServerURL.Host, globalFlags.ServerURL.Path, []string{globalFlags.ServerURL.Scheme})
	apitransport.SetAuth(transport, globalFlags.ServerURL.User)
	transport.SetLogger(logrus.WithField("component", "server-transport"))
	transport.SetDebug(globalFlags.EnableDebug || globalFlags.EnableTrace)

	// set error handlers for nginx responses if pmm-managed is down
	transport.Consumers = servererror.Consumers(map[string]runtime.Consumer{
		"application/zip": runtime.ByteStreamConsumer(),
	})

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
