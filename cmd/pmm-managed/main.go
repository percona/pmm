// pmm-managed
// Copyright (C) 2017 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"bytes"
	_ "expvar"
	"flag"
	"html/template"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"

	"github.com/percona/pmm-managed/api"
	"github.com/percona/pmm-managed/handlers"
	"github.com/percona/pmm-managed/services/consul"
	"github.com/percona/pmm-managed/services/prometheus"
	"github.com/percona/pmm-managed/services/telemetry"
	"github.com/percona/pmm-managed/utils/interceptors"
	"github.com/percona/pmm-managed/utils/logger"
)

const (
	shutdownTimeout = 3 * time.Second

	// FIXME set it during build for PMM 1.4
	pmmVersion = "1.3.0"
)

var (
	// TODO we can combine gRPC and REST ports, but only with TLS
	// see https://github.com/grpc/grpc-go/issues/555
	// alternatively, we can try to use cmux: https://open.dgraph.io/post/cmux/
	gRPCAddrF  = flag.String("listen-grpc-addr", "127.0.0.1:7771", "gRPC server listen address")
	restAddrF  = flag.String("listen-rest-addr", "127.0.0.1:7772", "REST server listen address")
	debugAddrF = flag.String("listen-debug-addr", "127.0.0.1:7773", "Debug server listen address")

	swaggerF = flag.String("swagger", "off", "Server to serve Swagger: rest, debug or off")

	prometheusConfigF = flag.String("prometheus-config", "", "Prometheus configuration file path")
	prometheusURLF    = flag.String("prometheus-url", "http://127.0.0.1:9090/", "Prometheus base URL")
	promtoolF         = flag.String("promtool", "promtool", "promtool path")

	consulAddress = flag.String("consul-address", "localhost:8500", "consul endpoint address")
)

func addSwaggerHandler(mux *http.ServeMux, pattern string) {
	// TODO embed swagger resources?
	fileServer := http.StripPrefix(pattern, http.FileServer(http.Dir("api/swagger")))
	mux.HandleFunc(pattern, func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Set("Access-Control-Allow-Origin", "*")
		fileServer.ServeHTTP(rw, req)
	})
}

// runGRPCServer runs gRPC server until context is canceled, then gracefully stops it.
func runGRPCServer(ctx context.Context) {
	l := logrus.WithField("component", "gRPC")
	l.Infof("Starting server on http://%s/ ...", *gRPCAddrF)

	prometheus, err := prometheus.NewService(*prometheusConfigF, *prometheusURLF, *promtoolF)
	if err == nil {
		err = prometheus.Check(ctx)
	}
	if err != nil {
		l.Panicf("Prometheus problem: %s", err)
	}

	gRPCServer := grpc.NewServer(
		grpc.UnaryInterceptor(interceptors.Unary),
		grpc.StreamInterceptor(interceptors.Stream),
	)
	api.RegisterBaseServer(gRPCServer, &handlers.BaseServer{PMMVersion: pmmVersion})
	api.RegisterDemoServer(gRPCServer, &handlers.DemoServer{})
	// TODO api.RegisterAlertsServer(gRPCServer, &handlers.AlertsServer{
	// 	Prometheus: prometheus,
	// })
	api.RegisterScrapeJobsServer(gRPCServer, &handlers.ScrapeJobsServer{
		Prometheus: prometheus,
	})

	grpc_prometheus.Register(gRPCServer)
	grpc_prometheus.EnableHandlingTimeHistogram()

	listener, err := net.Listen("tcp", *gRPCAddrF)
	if err != nil {
		l.Panic(err)
	}
	go func() {
		for {
			err = gRPCServer.Serve(listener)
			if err == grpc.ErrServerStopped {
				break
			}
			l.Errorf("Failed to serve: %s", err)
		}
		l.Info("Server stopped.")
	}()

	<-ctx.Done()
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	go func() {
		<-ctx.Done()
		gRPCServer.Stop()
	}()
	gRPCServer.GracefulStop()
	cancel()
}

// runRESTServer runs REST proxy server until context is canceled, then gracefully stops it.
func runRESTServer(ctx context.Context) {
	l := logrus.WithField("component", "REST")
	l.Infof("Starting server on http://%s/ ...", *restAddrF)

	proxyMux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithInsecure()}

	type registrar func(context.Context, *runtime.ServeMux, string, []grpc.DialOption) error
	for _, r := range []registrar{
		api.RegisterBaseHandlerFromEndpoint,
		api.RegisterDemoHandlerFromEndpoint,
		// TODO api.RegisterAlertsHandlerFromEndpoint,
		api.RegisterScrapeJobsHandlerFromEndpoint,
	} {
		if err := r(ctx, proxyMux, *gRPCAddrF, opts); err != nil {
			l.Panic(err)
		}
	}

	mux := http.NewServeMux()
	if *swaggerF == "rest" {
		l.Printf("Swagger enabled.")
		addSwaggerHandler(mux, "/swagger/")
	}
	mux.Handle("/", proxyMux)

	server := &http.Server{
		Addr:     *restAddrF,
		ErrorLog: log.New(os.Stderr, "runRESTServer: ", 0),
		Handler:  mux,

		// TODO we probably will need it for TLS+HTTP/2, see https://github.com/philips/grpc-gateway-example/issues/11
		// TLSConfig: &tls.Config{
		// 	NextProtos: []string{"h2"},
		// },
	}
	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			l.Panic(err)
		}
		l.Info("Server stopped.")
	}()

	<-ctx.Done()
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	if err := server.Shutdown(ctx); err != nil {
		l.Errorf("Failed to shutdown gracefully: %s", err)
	}
	cancel()
}

// runDebugServer runs debug server until context is canceled, then gracefully stops it.
func runDebugServer(ctx context.Context) {
	l := logrus.WithField("component", "debug")

	http.Handle("/debug/metrics", promhttp.Handler())

	handlers := []string{"/debug/metrics", "/debug/vars", "/debug/requests", "/debug/events", "/debug/pprof"}
	if *swaggerF == "debug" {
		handlers = append(handlers, "/swagger")
		l.Printf("Swagger enabled.")
		addSwaggerHandler(http.DefaultServeMux, "/swagger/")
	}

	for i, h := range handlers {
		handlers[i] = "http://" + *debugAddrF + h
	}

	var buf bytes.Buffer
	err := template.Must(template.New("debug").Parse(`
	<html>
	<body>
	<ul>
	{{ range . }}
		<li><a href="{{ . }}">{{ . }}</a></li>
	{{ end }}
	</ul>
	</body>
	</html>
	`)).Execute(&buf, handlers)
	if err != nil {
		l.Panic(err)
	}
	http.HandleFunc("/debug", func(rw http.ResponseWriter, req *http.Request) {
		rw.Write(buf.Bytes())
	})
	l.Infof("Starting server on http://%s/debug\nRegistered handlers:\n\t%s", *debugAddrF, strings.Join(handlers, "\n\t"))

	server := &http.Server{
		Addr:     *debugAddrF,
		ErrorLog: log.New(os.Stderr, "runDebugServer: ", 0),
	}
	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			l.Panic(err)
		}
		l.Info("Server stopped.")
	}()

	<-ctx.Done()
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	if err := server.Shutdown(ctx); err != nil {
		l.Errorf("Failed to shutdown gracefully: %s", err)
	}
	cancel()
}

func runTelemetryService(ctx context.Context, consulClient *consul.Client) {
	l := logrus.WithField("component", "telemetry")

	disabledStr := strings.TrimSpace(strings.ToLower(os.Getenv("DISABLE_TELEMETRY")))
	if disabled, err := strconv.ParseBool(disabledStr); err == nil && disabled {
		l.Infof("Telemetry is disabled by DISABLE_TELEMETRY environment variable.")
		return
	}

	svc := &telemetry.Service{
		URL:        "https://v.percona.com/",
		PMMVersion: pmmVersion,
		Interval:   24 * time.Hour,
	}
	var err error
	if svc.UUID, err = getTelemetryUUID(consulClient); err != nil {
		l.Warnf("cannot get/set telemetry UUID in consul: %s", err)
		return
	}

	// Using this env var for compatibility with the Toolkit
	if telemetryEnvURL := os.Getenv("PERCONA_VERSION_CHECK_URL"); telemetryEnvURL != "" {
		l.Infof("PERCONA_VERSION_CHECK_URL env var is set")
		l.Infof("Using %s as the telemetry endpoint", telemetryEnvURL)
		svc.URL = telemetryEnvURL
	}

	l.Infof("Telemetry is enabled. UUID: %s", svc.UUID)
	svc.Run(ctx)
}

func getTelemetryUUID(consulClient *consul.Client) (string, error) {
	b, err := consulClient.GetKV("telemetry/uuid")
	if err != nil {
		return "", err
	}
	if len(b) > 0 {
		return string(b), nil
	}

	uuid, err := telemetry.GenerateUUID()
	if err != nil {
		return "", err
	}
	if err = consulClient.PutKV("telemetry/uuid", []byte(uuid)); err != nil {
		return "", err
	}
	return uuid, nil
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("stdlog: ")
	logrus.SetLevel(logrus.DebugLevel)
	grpclog.SetLoggerV2(&logger.GRPC{Entry: logrus.WithField("component", "grpclog")})
	flag.Parse()

	if *swaggerF != "rest" && *swaggerF != "debug" && *swaggerF != "off" {
		flag.Usage()
		log.Fatalf("Unexpected value %q for -swagger flag.", *swaggerF)
	}

	l := logrus.WithField("component", "main")
	ctx, cancel := context.WithCancel(context.Background())
	ctx, _ = logger.Set(ctx, "main")
	defer l.Info("Done.")

	// handle termination signals: first one gracefully, force exit on the second one
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		s := <-signals
		l.Warnf("Got %v (%d) signal, shutting down...", s, s)
		cancel()

		s = <-signals
		l.Panicf("Got %v (%d) signal, exiting!", s, s)
	}()

	consulClient, err := consul.NewClient(*consulAddress)
	if err != nil {
		l.Panic(err)
	}

	// start servers, wait for them to exit
	var wg sync.WaitGroup
	wg.Add(4)
	go func() {
		defer wg.Done()
		runGRPCServer(ctx)
	}()
	go func() {
		defer wg.Done()
		runRESTServer(ctx)
	}()
	go func() {
		defer wg.Done()
		runDebugServer(ctx)
	}()
	go func() {
		defer wg.Done()
		runTelemetryService(ctx, consulClient)
	}()
	wg.Wait()
}
