// pmm-managed
// Copyright (C) 2017 Percona LLC
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

package apitests

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
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

	"github.com/percona/pmm/api/alertmanager/amclient"
	inventoryClient "github.com/percona/pmm/api/inventorypb/json/client"
	backupsClient "github.com/percona/pmm/api/managementpb/backup/json/client"
	dbaasClient "github.com/percona/pmm/api/managementpb/dbaas/json/client"
	channelsClient "github.com/percona/pmm/api/managementpb/ia/json/client"
	managementClient "github.com/percona/pmm/api/managementpb/json/client"
	platformClient "github.com/percona/pmm/api/platformpb/json/client"
	serverClient "github.com/percona/pmm/api/serverpb/json/client"
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

	// True if -debug or -trace flag is passed.
	Debug bool

	// RunUpdateTest is true if PMM Server update should be tested.
	RunUpdateTest bool

	// RunSTTTests is true if STT tests should be run.
	RunSTTTests bool

	// RunIATests is true if IA tests should be run.
	RunIATests bool

	// Kubeconfig contains kubeconfig.
	Kubeconfig string
)

// ErrFromNginx is an error type for nginx HTML response.
type ErrFromNginx string

// Error implements error interface.
func (e *ErrFromNginx) Error() string {
	return "response from nginx: " + string(*e)
}

// GoString implements fmt.GoStringer interface.
func (e *ErrFromNginx) GoString() string {
	return fmt.Sprintf("ErrFromNginx(%q)", string(*e))
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
	errorConsumer := runtime.ConsumerFunc(func(reader io.Reader, data interface{}) error {
		b, _ := ioutil.ReadAll(reader)
		err := ErrFromNginx(string(b))
		return &err
	})
	transport.Consumers = map[string]runtime.Consumer{
		runtime.JSONMime:    runtime.JSONConsumer(),
		runtime.HTMLMime:    errorConsumer,
		runtime.TextMime:    errorConsumer,
		runtime.DefaultMime: errorConsumer,
	}

	// disable HTTP/2, set TLS config
	httpTransport := transport.Transport.(*http.Transport)
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
	rand.Seed(seed)
	gofakeit.SetGlobalFaker(gofakeit.New(seed))

	debugF := flag.Bool("pmm.debug", false, "Enable debug output [PMM_DEBUG].")
	traceF := flag.Bool("pmm.trace", false, "Enable trace output [PMM_TRACE].")
	serverURLF := flag.String("pmm.server-url", "https://admin:admin@localhost/", "PMM Server URL [PMM_SERVER_URL].")
	serverInsecureTLSF := flag.Bool("pmm.server-insecure-tls", false, "Skip PMM Server TLS certificate validation [PMM_SERVER_INSECURE_TLS].")
	runUpdateTestF := flag.Bool("pmm.run-update-test", false, "Run PMM Server update test [PMM_RUN_UPDATE_TEST].")
	kubeconfigF := flag.String("pmm.kubeconfig", "", "Pass kubeconfig file to run DBaaS tests.")

	// FIXME we should rethink it once https://jira.percona.com/browse/PMM-5106 is implemented
	runSTTTestsF := flag.Bool("pmm.run-stt-tests", false, "Run STT tests that require connected clients [PMM_RUN_STT_TESTS].")

	// TODO remove once IA is out of beta: https://jira.percona.com/browse/PMM-7001
	runIATestsF := flag.Bool("pmm.run-ia-tests", false, "Run IA tests that require connected clients [PMM_RUN_IA_TESTS].")

	testing.Init()
	flag.Parse()

	for envVar, f := range map[string]*flag.Flag{
		"PMM_DEBUG":               flag.Lookup("pmm.debug"),
		"PMM_TRACE":               flag.Lookup("pmm.trace"),
		"PMM_SERVER_URL":          flag.Lookup("pmm.server-url"),
		"PMM_SERVER_INSECURE_TLS": flag.Lookup("pmm.server-insecure-tls"),
		"PMM_RUN_UPDATE_TEST":     flag.Lookup("pmm.run-update-test"),
		"PMM_RUN_STT_TESTS":       flag.Lookup("pmm.run-stt-tests"),
		"PMM_KUBECONFIG":          flag.Lookup("pmm.kubeconfig"),
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
	RunSTTTests = *runSTTTestsF
	RunIATests = *runIATestsF

	var cancel context.CancelFunc
	Context, cancel = context.WithCancel(context.Background())

	// handle termination signals
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		s := <-signals
		signal.Stop(signals)
		logrus.Warnf("Got %s, shutting down...", unix.SignalName(s.(syscall.Signal)))
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

	if *kubeconfigF != "" {
		data, err := ioutil.ReadFile(*kubeconfigF)
		if err != nil {
			logrus.Fatalf("Failed to read kubeconfig: %s", err)
		}
		Kubeconfig = string(data)
	}

	transport := Transport(BaseURL, *serverInsecureTLSF)
	alertmanagerTransport := Transport(BaseURL, *serverInsecureTLSF)
	alertmanagerTransport.BasePath = "/alertmanager/api/v2"
	transport.Consumers["application/zip"] = runtime.ByteStreamConsumer()
	inventoryClient.Default = inventoryClient.New(transport, nil)
	managementClient.Default = managementClient.New(transport, nil)
	dbaasClient.Default = dbaasClient.New(transport, nil)
	serverClient.Default = serverClient.New(transport, nil)
	amclient.Default = amclient.New(alertmanagerTransport, nil)
	channelsClient.Default = channelsClient.New(transport, nil)
	backupsClient.Default = backupsClient.New(transport, nil)
	platformClient.Default = platformClient.New(transport, nil)

	// do not run tests if server is not available
	_, err = serverClient.Default.Server.Readiness(nil)
	if err != nil {
		panic(err)
	}
}

// check interfaces
var (
	_ error          = (*ErrFromNginx)(nil)
	_ fmt.GoStringer = (*ErrFromNginx)(nil)
)
