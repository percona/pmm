// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package apitransport configures the HTTP transports of the go-openapi runtimes pmm-admin and
// pmm-agent use to call the PMM Server and pmm-agent APIs.
//
// It exists so that those settings are applied in exactly one place. Every go-openapi runtime
// is built on http.DefaultTransport, so a call site configuring its transport in place would
// reconfigure that process-wide global, leaking the settings - a disabled certificate check
// included - into every other HTTP client in the process.
package apitransport

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"

	httptransport "github.com/go-openapi/runtime/client"

	"github.com/percona/pmm/utils/tlsconfig"
)

// Configure gives rt a private HTTP transport for a PMM Server reachable under scheme at
// hostname. JSON APIs are used over HTTP/1.1, and an https target is validated against
// Percona's TLS baseline unless insecureTLS asks for the certificate check to be skipped,
// which is what --server-insecure-tls does.
func Configure(rt *httptransport.Runtime, scheme, hostname string, insecureTLS bool) {
	httpTransport := clone(rt)

	if scheme == "https" {
		httpTransport.TLSClientConfig = tlsconfig.Get()
		httpTransport.TLSClientConfig.ServerName = hostname
		httpTransport.TLSClientConfig.InsecureSkipVerify = insecureTLS
	}

	rt.Transport = httpTransport
}

// ConfigureLocal gives rt a private HTTP transport for the pmm-agent local API, which is
// served over plain HTTP on the loopback interface and so needs no TLS configuration.
func ConfigureLocal(rt *httptransport.Runtime) {
	rt.Transport = clone(rt)
}

// SetAuth configures rt's authentication from u, the userinfo component of a PMM Server URL.
// PMM Server accepts a service token or an API key as a bearer token; any other username is
// sent as HTTP Basic Auth credentials. Does nothing if u is nil.
func SetAuth(rt *httptransport.Runtime, u *url.Userinfo) {
	if u == nil {
		return
	}

	user := u.Username()
	password, _ := u.Password()
	if user == "service_token" || user == "api_key" {
		rt.DefaultAuthentication = httptransport.BearerToken(password)
	} else {
		rt.DefaultAuthentication = httptransport.BasicAuth(user, password)
	}
}

// clone returns a copy of rt's HTTP transport which speaks HTTP/1.1 only. Working on a copy is
// what keeps PMM's settings off http.DefaultTransport.
func clone(rt *httptransport.Runtime) *http.Transport {
	base, ok := rt.Transport.(*http.Transport)
	if !ok {
		panic(fmt.Sprintf("cannot configure a %T: an *http.Transport is required", rt.Transport))
	}

	httpTransport := base.Clone()

	// A non-nil TLSNextProto is the documented way to disable HTTP/2, and it takes
	// precedence over the ForceAttemptHTTP2 that Clone carries over from the default.
	httpTransport.TLSNextProto = make(map[string]func(string, *tls.Conn) http.RoundTripper)

	return httpTransport
}
