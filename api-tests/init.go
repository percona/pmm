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

// Package apitests contains PMM Server API tests.
package apitests

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/go-openapi/runtime"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"

	actionsClient "github.com/percona/pmm/api/actions/v1/json/client"
	advisorClient "github.com/percona/pmm/api/advisors/v1/json/client"
	alertingClient "github.com/percona/pmm/api/alerting/v1/json/client"
	backupsClient "github.com/percona/pmm/api/backup/v1/json/client"
	inventoryClient "github.com/percona/pmm/api/inventory/v1/json/client"
	managementClient "github.com/percona/pmm/api/management/v1/json/client"
	platformClient "github.com/percona/pmm/api/platform/v1/json/client"
	serverClient "github.com/percona/pmm/api/server/v1/json/client"
	"github.com/percona/pmm/utils/tlsconfig"
)

//nolint:gochecknoglobals
var (
	// Context is canceled on SIGTERM or SIGINT. Tests should cleanup and exit.
	Context context.Context

	// BaseURL contains PMM Server base URL like https://admin:admin@127.0.0.1:8443/.
	BaseURL *url.URL

	// Hostname contains local hostname that is used for generating test data.
	Hostname string

	// Debug is true if -debug or -trace flag is passed.
	Debug bool

	// RunUpdateTest is true if PMM Server update should be tested.
	RunUpdateTest bool

	// RunAdvisorTests is true if Advisor tests should be run.
	RunAdvisorTests bool
)

// NginxError is an error type for nginx HTML response.
type NginxError string

// Error implements error interface.
func (e *NginxError) Error() string {
	return "response from nginx: " + string(*e)
}

// GoString implements fmt.GoStringer interface.
func (e *NginxError) GoString() string {
	return fmt.Sprintf("NginxError(%q)", string(*e))
}

// Transport returns configured Swagger transport for given URL.
func Transport(baseURL *url.URL, insecureTLS bool) *httptransport.Runtime {
	transport := httptransport.New(baseURL.Host, baseURL.Path, []string{baseURL.Scheme})
	if u := baseURL.User; u != nil {
		password, _ := u.Password()
		transport.DefaultAuthentication = httptransport.BasicAuth(u.Username(), password)
	}
	transport.SetLogger(logrus.WithField("component", "client"))
	transport.SetDebug(logrus.GetLevel() >= logrus.DebugLevel)
	transport.Context = context.Background() // not Context - do not cancel the whole transport

	// set error handlers for nginx responses if pmm-managed is down
	errorConsumer := runtime.ConsumerFunc(func(reader io.Reader, _ interface{}) error {
		b, _ := io.ReadAll(reader)
		err := NginxError(string(b))
		return &err
	})
	transport.Consumers = map[string]runtime.Consumer{
		runtime.JSONMime:    runtime.JSONConsumer(),
		runtime.HTMLMime:    errorConsumer,
		runtime.TextMime:    errorConsumer,
		runtime.DefaultMime: errorConsumer,
	}

	// disable HTTP/2, set TLS config
	httpTransport := transport.Transport.(*http.Transport) //nolint:forcetypeassert
	httpTransport.TLSNextProto = make(map[string]func(string, *tls.Conn) http.RoundTripper)
	if baseURL.Scheme == "https" {
		httpTransport.TLSClientConfig = tlsconfig.Get()
		httpTransport.TLSClientConfig.ServerName = baseURL.Hostname()
		httpTransport.TLSClientConfig.InsecureSkipVerify = insecureTLS
	}

	return transport
}

//nolint:gochecknoinits
func init() {
	seed := time.Now().UnixNano()
	gofakeit.SetGlobalFaker(gofakeit.New(seed))

	debugF := flag.Bool("pmm.debug", false, "Enable debug output [PMM_DEBUG].")
	traceF := flag.Bool("pmm.trace", false, "Enable trace output [PMM_TRACE].")
	serverURLF := flag.String("pmm.server-url", "https://admin:admin@localhost/", "PMM Server URL [PMM_SERVER_URL].")
	serverInsecureTLSF := flag.Bool("pmm.server-insecure-tls", false, "Skip PMM Server TLS certificate validation [PMM_SERVER_INSECURE_TLS].")
	runUpdateTestF := flag.Bool("pmm.run-update-test", false, "Run PMM Server update test [PMM_RUN_UPDATE_TEST].")

	// FIXME we should rethink it once https://jira.percona.com/browse/PMM-5106 is implemented
	runAdvisorsTestF := flag.Bool("pmm.run-advisor-tests", false, "Run Advisor tests that require connected clients [PMM_RUN_ADVISOR_TESTS].")

	testing.Init()
	flag.Parse()

	for envVar, f := range map[string]*flag.Flag{
		"PMM_DEBUG":               flag.Lookup("pmm.debug"),
		"PMM_TRACE":               flag.Lookup("pmm.trace"),
		"PMM_SERVER_URL":          flag.Lookup("pmm.server-url"),
		"PMM_SERVER_INSECURE_TLS": flag.Lookup("pmm.server-insecure-tls"),
		"PMM_RUN_UPDATE_TEST":     flag.Lookup("pmm.run-update-test"),
		"PMM_RUN_ADVISOR_TESTS":   flag.Lookup("pmm.run-advisor-tests"),
	} {
		env, ok := os.LookupEnv(envVar)
		if ok {
			err := f.Value.Set(env)
			if err != nil {
				logrus.Fatalf("Invalid ENV variable %s: %s", envVar, env)
			}
		}
	}

	if *debugF {
		logrus.SetLevel(logrus.DebugLevel)
	}
	if *traceF {
		logrus.SetLevel(logrus.TraceLevel)
		logrus.SetReportCaller(true)
	}
	Debug = *debugF || *traceF
	RunUpdateTest = *runUpdateTestF
	RunAdvisorTests = *runAdvisorsTestF

	var cancel context.CancelFunc
	Context, cancel = context.WithCancel(context.Background())

	// handle termination signals
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		s := <-signals
		signal.Stop(signals)
		logrus.Warnf("Got %s, shutting down...", unix.SignalName(s.(syscall.Signal))) //nolint:forcetypeassert
		cancel()
	}()

	var err error
	BaseURL, err = url.Parse(*serverURLF)
	if err != nil {
		logrus.Fatalf("Failed to parse PMM Server URL: %s.", err)
	}
	if BaseURL.Host == "" || BaseURL.Scheme == "" {
		logrus.Fatalf("Invalid PMM Server URL: %s", BaseURL.String())
	}
	if BaseURL.Path == "" {
		BaseURL.Path = "/"
	}
	logrus.Debugf("PMM Server URL: %s.", BaseURL)

	Hostname, err = os.Hostname()
	if err != nil {
		logrus.Fatalf("Failed to detect hostname: %s", err)
	}

	transport := Transport(BaseURL, *serverInsecureTLSF)
	transport.Consumers["application/zip"] = runtime.ByteStreamConsumer()
	inventoryClient.Default = inventoryClient.New(transport, nil)
	managementClient.Default = managementClient.New(transport, nil)
	serverClient.Default = serverClient.New(transport, nil)
	backupsClient.Default = backupsClient.New(transport, nil)
	platformClient.Default = platformClient.New(transport, nil)
	alertingClient.Default = alertingClient.New(transport, nil)
	advisorClient.Default = advisorClient.New(transport, nil)
	actionsClient.Default = actionsClient.New(transport, nil)

	// do not run tests if server is not available
	_, err = serverClient.Default.ServerService.Readiness(nil)
	if err != nil {
		logrus.Fatalf("Failed to pass the server readiness probe: %s", err)
	}
}

// check interfaces.
var (
	_ error          = (*NginxError)(nil)
	_ fmt.GoStringer = (*NginxError)(nil)
)
