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
	_ "expvar"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/percona/pmm/api/agent"
	"github.com/percona/pmm/api/inventory"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"gopkg.in/reform.v1"
	reformMySQL "gopkg.in/reform.v1/dialects/mysql"

	"github.com/percona/pmm-managed/api"
	"github.com/percona/pmm-managed/handlers"
	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/services/agents"
	"github.com/percona/pmm-managed/services/consul"
	"github.com/percona/pmm-managed/services/grafana"
	"github.com/percona/pmm-managed/services/logs"
	"github.com/percona/pmm-managed/services/mysql"
	"github.com/percona/pmm-managed/services/postgresql"
	"github.com/percona/pmm-managed/services/prometheus"
	"github.com/percona/pmm-managed/services/qan"
	"github.com/percona/pmm-managed/services/rds"
	"github.com/percona/pmm-managed/services/remote"
	"github.com/percona/pmm-managed/services/supervisor"
	"github.com/percona/pmm-managed/services/telemetry"
	"github.com/percona/pmm-managed/utils/interceptors"
	"github.com/percona/pmm-managed/utils/logger"
	"github.com/percona/pmm-managed/utils/ports"
)

const (
	shutdownTimeout = 3 * time.Second

	// TODO set during build
	Version = "2.0.0-dev"
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

	consulAddrF  = flag.String("consul-addr", "127.0.0.1:8500", "Consul HTTP API address")
	grafanaAddrF = flag.String("grafana-addr", "127.0.0.1:3000", "Grafana HTTP API address")

	dbNameF     = flag.String("db-name", "", "Database name")
	dbUsernameF = flag.String("db-username", "pmm-managed", "Database username")
	dbPasswordF = flag.String("db-password", "pmm-managed", "Database password")

	agentMySQLdExporterF    = flag.String("agent-mysqld-exporter", "/usr/local/percona/pmm-client/mysqld_exporter", "mysqld_exporter path")
	agentPostgresExporterF  = flag.String("agent-postgres-exporter", "/usr/local/percona/pmm-client/postgres_exporter", "postgres_exporter path")
	agentRDSExporterF       = flag.String("agent-rds-exporter", "/usr/sbin/rds_exporter", "rds_exporter path")
	agentRDSExporterConfigF = flag.String("agent-rds-exporter-config", "/etc/percona-rds-exporter.yml", "rds_exporter configuration file path")
	agentQANBaseF           = flag.String("agent-qan-base", "/usr/local/percona/qan-agent", "qan-agent installation base path")

	debugF = flag.Bool("debug", false, "Enable debug logging")
)

func addSwaggerHandler(mux *http.ServeMux) {
	// TODO embed swagger resources?
	pattern := "/swagger/"
	fileServer := http.StripPrefix(pattern, http.FileServer(http.Dir("api/swagger")))
	mux.HandleFunc(pattern, func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Set("Access-Control-Allow-Origin", "*")
		fileServer.ServeHTTP(rw, req)
	})
}

func addLogsHandler(mux *http.ServeMux, logs *logs.Logs) {
	l := logrus.WithField("component", "logs.zip")

	mux.HandleFunc("/logs.zip", func(rw http.ResponseWriter, req *http.Request) {
		// fail-safe
		ctx, cancel := context.WithTimeout(req.Context(), 10*time.Second)
		defer cancel()

		t := time.Now().UTC()
		filename := fmt.Sprintf("pmm-server_%4d-%02d-%02d-%02d-%02d.zip", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute())

		rw.Header().Set(`Access-Control-Allow-Origin`, `*`)
		rw.Header().Set(`Content-Type`, `application/zip`)
		rw.Header().Set(`Content-Disposition`, `attachment; filename="`+filename+`"`)
		ctx, _ = logger.Set(ctx, "logs")
		if err := logs.Zip(ctx, rw); err != nil {
			l.Error(err)
		}
	})
}

func makePortsRegistry(db *reform.DB) (*ports.Registry, error) {
	// collect already reserved ports
	rows, err := db.Query("SELECT listen_port FROM agents")
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer rows.Close()

	var reserved []uint16
	for rows.Next() {
		var port uint16
		if err = rows.Scan(&port); err != nil {
			return nil, errors.WithStack(err)
		}
		reserved = append(reserved, port)
	}
	if err = rows.Err(); err != nil {
		return nil, errors.WithStack(err)
	}
	registry := ports.NewRegistry(10000, 10999, reserved)
	return registry, err
}

type serviceDependencies struct {
	prometheus    *prometheus.Service
	supervisor    *supervisor.Supervisor
	db            *reform.DB
	portsRegistry *ports.Registry
	qan           *qan.Service
}

func makeRDSService(ctx context.Context, deps *serviceDependencies) (*rds.Service, error) {
	rdsConfig := rds.ServiceConfig{
		MySQLdExporterPath:    *agentMySQLdExporterF,
		RDSExporterPath:       *agentRDSExporterF,
		RDSExporterConfigPath: *agentRDSExporterConfigF,

		Prometheus:    deps.prometheus,
		Supervisor:    deps.supervisor,
		DB:            deps.db,
		PortsRegistry: deps.portsRegistry,
		QAN:           deps.qan,
	}
	rdsService, err := rds.NewService(&rdsConfig)
	if err != nil {
		return nil, err
	}

	err = deps.db.InTransaction(func(tx *reform.TX) error {
		return rdsService.ApplyPrometheusConfiguration(ctx, tx.Querier)
	})
	if err != nil {
		return nil, err
	}
	err = deps.db.InTransaction(func(tx *reform.TX) error {
		return rdsService.Restore(ctx, tx)
	})
	if err != nil {
		return nil, err
	}

	return rdsService, nil
}

func makeMySQLService(ctx context.Context, deps *serviceDependencies) (*mysql.Service, error) {
	serviceConfig := mysql.ServiceConfig{
		MySQLdExporterPath: *agentMySQLdExporterF,

		Prometheus:    deps.prometheus,
		Supervisor:    deps.supervisor,
		DB:            deps.db,
		PortsRegistry: deps.portsRegistry,
		QAN:           deps.qan,
	}
	mysqlService, err := mysql.NewService(&serviceConfig)
	if err != nil {
		return nil, err
	}

	err = deps.db.InTransaction(func(tx *reform.TX) error {
		return mysqlService.ApplyPrometheusConfiguration(ctx, tx.Querier)
	})
	if err != nil {
		return nil, err
	}
	err = deps.db.InTransaction(func(tx *reform.TX) error {
		return mysqlService.Restore(ctx, tx)
	})
	if err != nil {
		return nil, err
	}

	return mysqlService, nil
}

func makePostgreSQLService(ctx context.Context, deps *serviceDependencies) (*postgresql.Service, error) {
	serviceConfig := postgresql.ServiceConfig{
		PostgresExporterPath: *agentPostgresExporterF,

		Prometheus:    deps.prometheus,
		Supervisor:    deps.supervisor,
		DB:            deps.db,
		PortsRegistry: deps.portsRegistry,
	}
	postgresqlService, err := postgresql.NewService(&serviceConfig)
	if err != nil {
		return nil, err
	}

	err = deps.db.InTransaction(func(tx *reform.TX) error {
		return postgresqlService.ApplyPrometheusConfiguration(ctx, tx.Querier)
	})
	if err != nil {
		return nil, err
	}
	err = deps.db.InTransaction(func(tx *reform.TX) error {
		return postgresqlService.Restore(ctx, tx)
	})
	if err != nil {
		return nil, err
	}

	return postgresqlService, nil
}

type grpcServerDependencies struct {
	*serviceDependencies
	consulClient *consul.Client
	rds          *rds.Service
	mysql        *mysql.Service
	postgres     *postgresql.Service
	remote       *remote.Service
	logs         *logs.Logs
}

// runGRPCServer runs gRPC server until context is canceled, then gracefully stops it.
func runGRPCServer(ctx context.Context, deps *grpcServerDependencies) {
	l := logrus.WithField("component", "gRPC")
	l.Infof("Starting server on http://%s/ ...", *gRPCAddrF)

	grafana := grafana.NewClient(*grafanaAddrF)

	gRPCServer := grpc.NewServer(
		grpc.UnaryInterceptor(interceptors.Unary),
		grpc.StreamInterceptor(interceptors.Stream),
	)
	api.RegisterBaseServer(gRPCServer, &handlers.BaseServer{PMMVersion: Version})
	api.RegisterDemoServer(gRPCServer, &handlers.DemoServer{})
	api.RegisterScrapeConfigsServer(gRPCServer, &handlers.ScrapeConfigsServer{
		Prometheus: deps.prometheus,
	})
	api.RegisterRDSServer(gRPCServer, &handlers.RDSServer{
		RDS: deps.rds,
	})
	api.RegisterMySQLServer(gRPCServer, &handlers.MySQLServer{
		MySQL: deps.mysql,
	})
	api.RegisterPostgreSQLServer(gRPCServer, &handlers.PostgreSQLServer{
		PostgreSQL: deps.postgres,
	})
	api.RegisterRemoteServer(gRPCServer, &handlers.RemoteServer{
		Remote: deps.remote,
	})
	api.RegisterLogsServer(gRPCServer, &handlers.LogsServer{
		Logs: deps.logs,
	})
	api.RegisterAnnotationsServer(gRPCServer, &handlers.AnnotationsServer{
		Grafana: grafana,
	})

	// PMM 2.0 APIs
	store := agents.NewStore()
	agent.RegisterAgentServer(gRPCServer, &handlers.AgentServer{
		Store: store,
	})
	inventory.RegisterNodesServer(gRPCServer, &handlers.NodesServer{
		Store: store,
	})
	inventory.RegisterServicesServer(gRPCServer, &handlers.ServicesServer{
		Store: store,
	})
	inventory.RegisterAgentsServer(gRPCServer, &handlers.AgentsServer{
		Store: store,
	})

	if *debugF {
		l.Debug("Reflection enabled.")
		reflection.Register(gRPCServer)
	}

	grpc_prometheus.Register(gRPCServer)
	grpc_prometheus.EnableHandlingTimeHistogram()

	listener, err := net.Listen("tcp", *gRPCAddrF)
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
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	go func() {
		<-ctx.Done()
		gRPCServer.Stop()
	}()
	gRPCServer.GracefulStop()
	cancel()
}

// runRESTServer runs REST proxy server until context is canceled, then gracefully stops it.
func runRESTServer(ctx context.Context, logs *logs.Logs) {
	l := logrus.WithField("component", "REST")
	l.Infof("Starting server on http://%s/ ...", *restAddrF)

	proxyMux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithInsecure()}

	type registrar func(context.Context, *runtime.ServeMux, string, []grpc.DialOption) error
	for _, r := range []registrar{
		api.RegisterBaseHandlerFromEndpoint,
		api.RegisterDemoHandlerFromEndpoint,
		api.RegisterScrapeConfigsHandlerFromEndpoint,
		api.RegisterRDSHandlerFromEndpoint,
		api.RegisterMySQLHandlerFromEndpoint,
		api.RegisterPostgreSQLHandlerFromEndpoint,
		api.RegisterRemoteHandlerFromEndpoint,
		api.RegisterLogsHandlerFromEndpoint,
		api.RegisterAnnotationsHandlerFromEndpoint,

		inventory.RegisterNodesHandlerFromEndpoint,
		inventory.RegisterAgentsHandlerFromEndpoint,
	} {
		if err := r(ctx, proxyMux, *gRPCAddrF, opts); err != nil {
			l.Panic(err)
		}
	}

	mux := http.NewServeMux()
	if *swaggerF == "rest" {
		l.Printf("Swagger enabled. http://%s/swagger/", *restAddrF)
		addSwaggerHandler(mux)
	}
	addLogsHandler(mux, logs)
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
		l.Printf("Swagger enabled. http://%s/swagger/", *debugAddrF)
		addSwaggerHandler(http.DefaultServeMux)
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

	uuid, err := getTelemetryUUID(consulClient)
	if err != nil {
		l.Panicf("cannot get/set telemetry UUID in Consul: %s", err)
	}

	svc := telemetry.NewService(uuid, Version)
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
	log.Printf("pmm-managed %s", Version)
	log.SetPrefix("stdlog: ")
	flag.Parse()

	if *dbNameF == "" {
		log.Fatal("-db-name flag must be given explicitly.")
	}

	if *debugF {
		logrus.SetLevel(logrus.DebugLevel)
		// grpclog.SetLoggerV2(&logger.GRPC{Entry: logrus.WithField("component", "grpclog")})
	}

	if *swaggerF != "rest" && *swaggerF != "debug" && *swaggerF != "off" {
		flag.Usage()
		log.Fatalf("Unexpected value %q for -swagger flag.", *swaggerF)
	}

	l := logrus.WithField("component", "main")
	ctx, cancel := context.WithCancel(context.Background())
	ctx, _ = logger.Set(ctx, "main")
	defer l.Info("Done.")

	// handle termination signals
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		s := <-signals
		signal.Stop(signals)
		l.Warnf("Got %v (%d) signal, shutting down...", s, s)
		cancel()
	}()

	consulClient, err := consul.NewClient(*consulAddrF)
	if err != nil {
		l.Panic(err)
	}

	prometheus, err := prometheus.NewService(*prometheusConfigF, *prometheusURLF, *promtoolF, consulClient)
	if err == nil {
		err = prometheus.Check(ctx)
	}
	if err != nil {
		l.Panicf("Prometheus service problem: %+v", err)
	}

	supervisor := supervisor.New(l)

	qan, err := qan.NewService(ctx, *agentQANBaseF, supervisor)
	if err != nil {
		l.Panicf("QAN service problem: %+v", err)
	}

	sqlDB, err := models.OpenDB(*dbNameF, *dbUsernameF, *dbPasswordF, l.Debugf)
	if err != nil {
		l.Panic(err)
	}
	defer sqlDB.Close()
	db := reform.NewDB(sqlDB, reformMySQL.Dialect, nil)

	portsRegistry, err := makePortsRegistry(db)
	if err != nil {
		l.Panic(err)
	}

	deps := &serviceDependencies{
		prometheus:    prometheus,
		supervisor:    supervisor,
		qan:           qan,
		db:            db,
		portsRegistry: portsRegistry,
	}
	rds, err := makeRDSService(ctx, deps)
	if err != nil {
		l.Panicf("RDS service problem: %+v", err)
	}

	mysqlService, err := makeMySQLService(ctx, deps)
	if err != nil {
		l.Panicf("MySQL service problem: %+v", err)
	}

	postgres, err := makePostgreSQLService(ctx, deps)
	if err != nil {
		l.Panicf("PostgreSQL service problem: %+v", err)
	}

	remoteService, err := remote.NewService(&remote.ServiceConfig{
		DB: deps.db,
	})
	if err != nil {
		l.Panicf("Remote service problem: %+v", err)
	}

	logs := logs.New(Version, consulClient, db, rds, nil)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		runGRPCServer(ctx, &grpcServerDependencies{
			serviceDependencies: deps,
			rds:                 rds,
			postgres:            postgres,
			mysql:               mysqlService,
			remote:              remoteService,
			consulClient:        consulClient,
			logs:                logs,
		})
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		runRESTServer(ctx, logs)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		runDebugServer(ctx)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		runTelemetryService(ctx, consulClient)
	}()

	wg.Wait()
}
