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

package main

import (
	"bytes"
	"context"
	"database/sql"
	_ "expvar" // register /debug/vars
	"flag"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof" // register /debug/pprof
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_validator "github.com/grpc-ecosystem/go-grpc-middleware/validator"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/percona/pmm/api/agentpb"
	"github.com/percona/pmm/api/inventorypb"
	"github.com/percona/pmm/api/managementpb"
	"github.com/percona/pmm/api/serverpb"
	"github.com/percona/pmm/version"
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
	"google.golang.org/grpc"
	channelz "google.golang.org/grpc/channelz/service"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/reflection"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/services/agents"
	agentgrpc "github.com/percona/pmm-managed/services/agents/grpc"
	"github.com/percona/pmm-managed/services/grafana"
	"github.com/percona/pmm-managed/services/inventory"
	inventorygrpc "github.com/percona/pmm-managed/services/inventory/grpc"
	"github.com/percona/pmm-managed/services/logs"
	"github.com/percona/pmm-managed/services/management"
	managementgrpc "github.com/percona/pmm-managed/services/management/grpc"
	"github.com/percona/pmm-managed/services/prometheus"
	"github.com/percona/pmm-managed/services/qan"
	"github.com/percona/pmm-managed/services/server"
	"github.com/percona/pmm-managed/services/telemetry"
	"github.com/percona/pmm-managed/utils/interceptors"
	"github.com/percona/pmm-managed/utils/logger"
)

const (
	shutdownTimeout = 3 * time.Second

	gRPCAddr  = "127.0.0.1:7771"
	http1Addr = "127.0.0.1:7772"
	debugAddr = "127.0.0.1:7773"
)

var (
	// TODO Switch to kingpin for flags parsing: https://jira.percona.com/browse/PMM-3259
	prometheusConfigF = flag.String("prometheus-config", "", "Prometheus configuration file path")
	prometheusURLF    = flag.String("prometheus-url", "http://127.0.0.1:9090/prometheus/", "Prometheus base URL")
	promtoolF         = flag.String("promtool", "promtool", "promtool path")

	grafanaAddrF = flag.String("grafana-addr", "127.0.0.1:3000", "Grafana HTTP API address")
	qanAPIAddrF  = flag.String("qan-api-addr", "127.0.0.1:9911", "QAN API gRPC API address")

	postgresDBNameF     = flag.String("postgres-name", "", "PostgreSQL database name")
	postgresDBUsernameF = flag.String("postgres-username", "pmm-managed", "PostgreSQL database username")
	postgresDBPasswordF = flag.String("postgres-password", "pmm-managed", "PostgreSQL database password")

	debugF = flag.Bool("debug", false, "Enable debug logging")
	traceF = flag.Bool("trace", false, "Enable trace logging")
)

func addLogsHandler(mux *http.ServeMux, logs *logs.Logs) {
	l := logrus.WithField("component", "logs.zip")

	mux.HandleFunc("/logs.zip", func(rw http.ResponseWriter, req *http.Request) {
		// fail-safe
		ctx, cancel := context.WithTimeout(req.Context(), 10*time.Second)
		defer cancel()

		filename := fmt.Sprintf("pmm-server_%s.zip", time.Now().UTC().Format("2006-01-02_15-04"))
		rw.Header().Set(`Access-Control-Allow-Origin`, `*`)
		rw.Header().Set(`Content-Type`, `application/zip`)
		rw.Header().Set(`Content-Disposition`, `attachment; filename="`+filename+`"`)
		ctx = logger.Set(ctx, "logs")
		if err := logs.Zip(ctx, rw); err != nil {
			l.Error(err)
		}
	})
}

type serviceDependencies struct {
	db             *reform.DB
	prometheus     *prometheus.Service
	server         *server.Server
	agentsRegistry *agents.Registry
	logs           *logs.Logs
}

// runGRPCServer runs gRPC server until context is canceled, then gracefully stops it.
func runGRPCServer(ctx context.Context, deps *serviceDependencies) {
	l := logrus.WithField("component", "gRPC")
	l.Infof("Starting server on http://%s/ ...", gRPCAddr)

	gRPCServer := grpc.NewServer(
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			interceptors.Unary,
			grpc_validator.UnaryServerInterceptor(),
		)),
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			interceptors.Stream,
			grpc_validator.StreamServerInterceptor(),
		)),
	)

	serverpb.RegisterServerServer(gRPCServer, deps.server)

	agentpb.RegisterAgentServer(gRPCServer, agentgrpc.NewAgentServer(deps.agentsRegistry))

	nodesSvc := inventory.NewNodesService(deps.db)
	servicesSvc := inventory.NewServicesService(deps.db, deps.agentsRegistry)
	agentsSvc := inventory.NewAgentsService(deps.db, deps.agentsRegistry)

	inventorypb.RegisterNodesServer(gRPCServer, inventorygrpc.NewNodesServer(nodesSvc))
	inventorypb.RegisterServicesServer(gRPCServer, inventorygrpc.NewServicesServer(servicesSvc))
	inventorypb.RegisterAgentsServer(gRPCServer, inventorygrpc.NewAgentsServer(agentsSvc))

	nodeSvc := management.NewNodeService(deps.db, deps.agentsRegistry)
	serviceSvc := management.NewServiceService(deps.db, deps.agentsRegistry)
	mysqlSvc := management.NewMySQLService(deps.db, deps.agentsRegistry)
	mongodbSvc := management.NewMongoDBService(deps.db, deps.agentsRegistry)
	postgresqlSvc := management.NewPostgreSQLService(deps.db, deps.agentsRegistry)
	proxysqlSvc := management.NewProxySQLService(deps.db, deps.agentsRegistry)

	managementpb.RegisterNodeServer(gRPCServer, managementgrpc.NewManagementNodeServer(nodeSvc))
	managementpb.RegisterServiceServer(gRPCServer, managementgrpc.NewManagementServiceServer(serviceSvc))
	managementpb.RegisterMySQLServer(gRPCServer, managementgrpc.NewManagementMySQLServer(mysqlSvc))
	managementpb.RegisterMongoDBServer(gRPCServer, managementgrpc.NewManagementMongoDBServer(mongodbSvc))
	managementpb.RegisterPostgreSQLServer(gRPCServer, managementgrpc.NewManagementPostgreSQLServer(postgresqlSvc))
	managementpb.RegisterProxySQLServer(gRPCServer, managementgrpc.NewManagementProxySQLServer(proxysqlSvc))
	managementpb.RegisterActionsServer(gRPCServer, managementgrpc.NewActionsServer(deps.agentsRegistry, deps.db))

	if *debugF {
		l.Debug("Reflection and channelz are enabled.")
		reflection.Register(gRPCServer)
		channelz.RegisterChannelzServiceToServer(gRPCServer)
	}

	grpc_prometheus.EnableHandlingTimeHistogram()
	grpc_prometheus.Register(gRPCServer)

	// run server until it is stopped gracefully or not
	listener, err := net.Listen("tcp", gRPCAddr)
	if err != nil {
		l.Panic(err)
	}
	go func() {
		for {
			err = gRPCServer.Serve(listener)
			if err == nil || err == grpc.ErrServerStopped {
				break
			}
			l.Errorf("Failed to serve: %s", err)
		}
		l.Info("Server stopped.")
	}()

	<-ctx.Done()

	// try to stop server gracefully, then not
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	go func() {
		<-ctx.Done()
		gRPCServer.Stop()
	}()
	gRPCServer.GracefulStop()
	cancel()
}

// runHTTP1Server runs grpc-gateway and other HTTP 1.1 APIs (like auth_request and logs.zip)
// until context is canceled, then gracefully stops it.
func runHTTP1Server(ctx context.Context, logs *logs.Logs, authServer *grafana.AuthServer) {
	l := logrus.WithField("component", "JSON")
	l.Infof("Starting server on http://%s/ ...", http1Addr)

	proxyMux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithInsecure()}

	// TODO switch from RegisterXXXHandlerFromEndpoint to RegisterXXXHandler to avoid extra dials
	// (even if they dial to localhost)
	// https://jira.percona.com/browse/PMM-4326
	type registrar func(context.Context, *runtime.ServeMux, string, []grpc.DialOption) error
	for _, r := range []registrar{
		serverpb.RegisterServerHandlerFromEndpoint,

		inventorypb.RegisterNodesHandlerFromEndpoint,
		inventorypb.RegisterServicesHandlerFromEndpoint,
		inventorypb.RegisterAgentsHandlerFromEndpoint,

		managementpb.RegisterNodeHandlerFromEndpoint,
		managementpb.RegisterServiceHandlerFromEndpoint,
		managementpb.RegisterMySQLHandlerFromEndpoint,
		managementpb.RegisterMongoDBHandlerFromEndpoint,
		managementpb.RegisterPostgreSQLHandlerFromEndpoint,
		managementpb.RegisterProxySQLHandlerFromEndpoint,
		managementpb.RegisterActionsHandlerFromEndpoint,
	} {
		if err := r(ctx, proxyMux, gRPCAddr, opts); err != nil {
			l.Panic(err)
		}
	}

	mux := http.NewServeMux()
	addLogsHandler(mux, logs)
	mux.Handle("/auth_request", authServer)
	mux.Handle("/", proxyMux)

	server := &http.Server{
		Addr:     http1Addr,
		ErrorLog: log.New(os.Stderr, "runJSONServer: ", 0),
		Handler:  mux,
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
// TODO merge with HTTP1 server? https://jira.percona.com/browse/PMM-4326
func runDebugServer(ctx context.Context) {
	handler := promhttp.HandlerFor(prom.DefaultGatherer, promhttp.HandlerOpts{
		ErrorLog:      logrus.WithField("component", "metrics"),
		ErrorHandling: promhttp.ContinueOnError,
	})
	http.Handle("/debug/metrics", promhttp.InstrumentMetricHandler(prom.DefaultRegisterer, handler))

	l := logrus.WithField("component", "debug")

	handlers := []string{
		"/debug/metrics",  // by http.Handle above
		"/debug/vars",     // by expvar
		"/debug/requests", // by golang.org/x/net/trace imported by google.golang.org/grpc
		"/debug/events",   // by golang.org/x/net/trace imported by google.golang.org/grpc
		"/debug/pprof",    // by net/http/pprof
	}
	for i, h := range handlers {
		handlers[i] = "http://" + debugAddr + h
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
	l.Infof("Starting server on http://%s/debug\nRegistered handlers:\n\t%s", debugAddr, strings.Join(handlers, "\n\t"))

	server := &http.Server{
		Addr:     debugAddr,
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

func runTelemetryService(ctx context.Context, db *reform.DB) {
	// Do not report this instance as running for the first 5 minutes.
	// Among other things, that solves reporting during PMM Server building when we start pmm-managed.
	sleepCtx, sleepCancel := context.WithTimeout(ctx, 5*time.Minute)
	<-sleepCtx.Done()
	sleepCancel()

	if ctx.Err() != nil {
		return
	}

	svc := telemetry.NewService(db, version.Version)
	svc.Run(ctx)
}

func setupDatabase(ctx context.Context, sqlDB *sql.DB, prometheus *prometheus.Service, server *server.Server, l *logrus.Entry) bool {
	l.Infof("Migrating database...")
	err := models.SetupDB(sqlDB, &models.SetupDBParams{
		Logf:          l.Debugf,
		Username:      *postgresDBUsernameF,
		Password:      *postgresDBPasswordF,
		SetupFixtures: models.SetupFixtures,
	})
	if err != nil {
		l.Warnf("Failed to migrate database: %s.", err)
		return false
	}

	l.Infof("Updating settings...")
	if err = server.UpdateSettings(); err != nil {
		l.Warnf("Settings problem: %s.", err)
		return false
	}

	l.Infof("Checking Prometheus...")
	if err = prometheus.Check(ctx); err != nil {
		l.Warnf("Prometheus problem: %s.", err)
		return false
	}
	prometheus.UpdateConfiguration()

	l.Info("Setup completed.")
	return true
}

func getQANClient(ctx context.Context, db *reform.DB) *qan.Client {
	opts := []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithBackoffMaxDelay(time.Second),
		grpc.WithUserAgent("pmm-managed/" + version.Version),
	}

	// Without grpc.WithBlock() DialContext returns an error only if something very wrong with address or options;
	// it does not return an error of connection failure but tries to reconnect in the background.
	conn, err := grpc.DialContext(ctx, *qanAPIAddrF, opts...)
	if err != nil {
		logrus.Fatalf("Failed to connect QAN API %s: %s.", *qanAPIAddrF, err)
	}
	return qan.NewClient(conn, db)
}

func main() {
	log.SetFlags(0)
	log.Print(version.FullInfo())
	log.SetPrefix("stdlog: ")
	flag.Parse()

	if *postgresDBNameF == "" {
		log.Fatal("-postgres-name flag must be given explicitly.")
	}

	if *debugF {
		logrus.SetLevel(logrus.DebugLevel)
	}
	if *traceF {
		logrus.SetLevel(logrus.TraceLevel)
		logrus.SetReportCaller(true) // https://github.com/sirupsen/logrus/issues/954
		grpclog.SetLoggerV2(&logger.GRPC{Entry: logrus.WithField("component", "grpclog")})
	}

	logrus.Infof("Log level: %s.", logrus.GetLevel())

	l := logrus.WithField("component", "main")
	ctx, cancel := context.WithCancel(context.Background())
	ctx = logger.Set(ctx, "main")
	defer l.Info("Done.")

	// handle termination signals
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, unix.SIGTERM, unix.SIGINT)
	go func() {
		s := <-signals
		signal.Stop(signals)
		logrus.Warnf("Got %s, shutting down...", unix.SignalName(s.(unix.Signal)))
		cancel()
	}()

	sqlDB, err := models.OpenDB(*postgresDBNameF, *postgresDBUsernameF, *postgresDBPasswordF)
	if err != nil {
		l.Panicf("Failed to connect to database: %+v", err)
	}
	defer sqlDB.Close()
	db := reform.NewDB(sqlDB, postgresql.Dialect, nil)

	prometheus, err := prometheus.NewService(*prometheusConfigF, *promtoolF, db, *prometheusURLF)
	if err != nil {
		l.Panicf("Prometheus service problem: %+v", err)
	}
	go prometheus.Run(ctx)

	server := server.NewServer(db, prometheus, os.Environ())

	// try synchronously once, then retry in the background
	setupL := logrus.WithField("component", "setup")
	if !setupDatabase(ctx, sqlDB, prometheus, server, setupL) {
		go func() {
			const delay = 2 * time.Second
			for {
				setupL.Warnf("Retrying in %s.", delay)
				sleepCtx, sleepCancel := context.WithTimeout(ctx, delay)
				<-sleepCtx.Done()
				sleepCancel()

				if ctx.Err() != nil {
					return
				}

				if setupDatabase(ctx, sqlDB, prometheus, server, setupL) {
					return
				}
			}
		}()
	}

	qanClient := getQANClient(ctx, db)
	logs := logs.New(version.Version)

	agentsRegistry := agents.NewRegistry(db, prometheus, qanClient)
	prom.MustRegister(agentsRegistry)

	grafanaClient := grafana.NewClient(*grafanaAddrF)
	prom.MustRegister(grafanaClient)
	authServer := grafana.NewAuthServer(grafanaClient)

	deps := &serviceDependencies{
		db:             db,
		prometheus:     prometheus,
		server:         server,
		agentsRegistry: agentsRegistry,
		logs:           logs,
	}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		runGRPCServer(ctx, deps)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		runHTTP1Server(ctx, logs, authServer)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		runDebugServer(ctx)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		runTelemetryService(ctx, db)
	}()

	wg.Wait()
}
