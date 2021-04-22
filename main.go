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
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof" // register /debug/pprof
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_validator "github.com/grpc-ecosystem/go-grpc-middleware/validator"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	grpc_gateway "github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/percona/pmm/api/agentpb"
	"github.com/percona/pmm/api/inventorypb"
	"github.com/percona/pmm/api/managementpb"
	azurev1beta1 "github.com/percona/pmm/api/managementpb/azure"
	backupv1beta1 "github.com/percona/pmm/api/managementpb/backup"
	dbaasv1beta1 "github.com/percona/pmm/api/managementpb/dbaas"
	iav1beta1 "github.com/percona/pmm/api/managementpb/ia"
	jobs1beta1 "github.com/percona/pmm/api/managementpb/jobs"
	"github.com/percona/pmm/api/serverpb"
	"github.com/percona/pmm/utils/sqlmetrics"
	"github.com/percona/pmm/version"
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	channelz "google.golang.org/grpc/channelz/service"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/reflection"
	"gopkg.in/alecthomas/kingpin.v2"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/services/agents"
	agentgrpc "github.com/percona/pmm-managed/services/agents/grpc"
	"github.com/percona/pmm-managed/services/alertmanager"
	"github.com/percona/pmm-managed/services/checks"
	"github.com/percona/pmm-managed/services/dbaas"
	"github.com/percona/pmm-managed/services/grafana"
	"github.com/percona/pmm-managed/services/inventory"
	inventorygrpc "github.com/percona/pmm-managed/services/inventory/grpc"
	"github.com/percona/pmm-managed/services/management"
	"github.com/percona/pmm-managed/services/management/backup"
	managementdbaas "github.com/percona/pmm-managed/services/management/dbaas"
	managementgrpc "github.com/percona/pmm-managed/services/management/grpc"
	"github.com/percona/pmm-managed/services/management/ia"
	"github.com/percona/pmm-managed/services/minio"
	"github.com/percona/pmm-managed/services/platform"
	"github.com/percona/pmm-managed/services/qan"
	"github.com/percona/pmm-managed/services/server"
	"github.com/percona/pmm-managed/services/supervisord"
	"github.com/percona/pmm-managed/services/telemetry"
	"github.com/percona/pmm-managed/services/victoriametrics"
	"github.com/percona/pmm-managed/services/vmalert"
	"github.com/percona/pmm-managed/utils/clean"
	"github.com/percona/pmm-managed/utils/interceptors"
	"github.com/percona/pmm-managed/utils/logger"
)

const (
	shutdownTimeout = 3 * time.Second

	gRPCAddr  = "127.0.0.1:7771"
	http1Addr = "127.0.0.1:7772"
	debugAddr = "127.0.0.1:7773"

	cleanInterval  = 10 * time.Minute
	cleanOlderThan = 30 * time.Minute
)

func addLogsHandler(mux *http.ServeMux, logs *supervisord.Logs) {
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
			l.Errorf("%+v", err)
		}
	})
}

type gRPCServerDeps struct {
	db                    *reform.DB
	vmdb                  *victoriametrics.Service
	server                *server.Server
	agentsRegistry        *agents.Registry
	grafanaClient         *grafana.Client
	checksService         *checks.Service
	dbaasControllerClient *dbaas.Client
	alertmanager          *alertmanager.Service
	vmalert               *vmalert.Service
	settings              *models.Settings
	alertsService         *ia.AlertsService
	templatesService      *ia.TemplatesService
	rulesService          *ia.RulesService
	jobsService           *agents.JobsService
	versionServiceClient  *managementdbaas.VersionServiceClient
	minio                 *minio.Service
}

// runGRPCServer runs gRPC server until context is canceled, then gracefully stops it.
func runGRPCServer(ctx context.Context, deps *gRPCServerDeps) {
	l := logrus.WithField("component", "gRPC")
	l.Infof("Starting server on http://%s/ ...", gRPCAddr)

	gRPCServer := grpc.NewServer(
		grpc.MaxRecvMsgSize(10*1024*1024),

		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			interceptors.Unary,
			interceptors.UnaryServiceEnabledInterceptor(),
			grpc_validator.UnaryServerInterceptor(),
		)),
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			interceptors.Stream,
			interceptors.StreamServiceEnabledInterceptor(),
			grpc_validator.StreamServerInterceptor(),
		)),
	)

	serverpb.RegisterServerServer(gRPCServer, deps.server)

	agentpb.RegisterAgentServer(gRPCServer, agentgrpc.NewAgentServer(deps.agentsRegistry))

	nodesSvc := inventory.NewNodesService(deps.db, deps.agentsRegistry, deps.vmdb)
	servicesSvc := inventory.NewServicesService(deps.db, deps.agentsRegistry, deps.vmdb)
	agentsSvc := inventory.NewAgentsService(deps.db, deps.agentsRegistry, deps.vmdb)

	inventorypb.RegisterNodesServer(gRPCServer, inventorygrpc.NewNodesServer(nodesSvc))
	inventorypb.RegisterServicesServer(gRPCServer, inventorygrpc.NewServicesServer(servicesSvc))
	inventorypb.RegisterAgentsServer(gRPCServer, inventorygrpc.NewAgentsServer(agentsSvc))

	nodeSvc := management.NewNodeService(deps.db, deps.agentsRegistry)
	serviceSvc := management.NewServiceService(deps.db, deps.agentsRegistry, deps.vmdb)
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
	managementpb.RegisterRDSServer(gRPCServer, management.NewRDSService(deps.db, deps.agentsRegistry))
	azurev1beta1.RegisterAzureDatabaseServer(gRPCServer, management.NewAzureDatabaseService(deps.db, deps.agentsRegistry))
	managementpb.RegisterHAProxyServer(gRPCServer, management.NewHAProxyService(deps.db, deps.agentsRegistry, deps.vmdb))
	managementpb.RegisterExternalServer(gRPCServer, management.NewExternalService(deps.db, deps.agentsRegistry, deps.vmdb))
	managementpb.RegisterAnnotationServer(gRPCServer, managementgrpc.NewAnnotationServer(deps.db, deps.grafanaClient))
	managementpb.RegisterSecurityChecksServer(gRPCServer, management.NewChecksAPIService(deps.checksService))
	jobs1beta1.RegisterJobsServer(gRPCServer, management.NewJobsAPIServer(deps.db, deps.jobsService))

	iav1beta1.RegisterChannelsServer(gRPCServer, ia.NewChannelsService(deps.db, deps.alertmanager))
	deps.templatesService.Collect(ctx)
	iav1beta1.RegisterTemplatesServer(gRPCServer, deps.templatesService)
	iav1beta1.RegisterRulesServer(gRPCServer, deps.rulesService)
	iav1beta1.RegisterAlertsServer(gRPCServer, deps.alertsService)

	backupv1beta1.RegisterBackupsServer(gRPCServer, backup.NewBackupsService(deps.db, deps.jobsService))
	backupv1beta1.RegisterLocationsServer(gRPCServer, backup.NewLocationsService(deps.db, deps.minio))
	backupv1beta1.RegisterArtifactsServer(gRPCServer, backup.NewArtifactsService(deps.db))

	// TODO Remove once changing settings.DBaaS.Enabled is possible via API.
	if deps.settings.DBaaS.Enabled {
		dbaasv1beta1.RegisterKubernetesServer(gRPCServer, managementdbaas.NewKubernetesServer(deps.db, deps.dbaasControllerClient))
		dbaasv1beta1.RegisterXtraDBClusterServer(gRPCServer, managementdbaas.NewXtraDBClusterService(deps.db, deps.dbaasControllerClient))
		dbaasv1beta1.RegisterPSMDBClusterServer(gRPCServer, managementdbaas.NewPSMDBClusterService(deps.db, deps.dbaasControllerClient))
		dbaasv1beta1.RegisterLogsAPIServer(gRPCServer, managementdbaas.NewLogsService(deps.db, deps.dbaasControllerClient))
		dbaasv1beta1.RegisterComponentsServer(gRPCServer, managementdbaas.NewComponentsService(deps.db, deps.dbaasControllerClient, deps.versionServiceClient))
	}

	if l.Logger.GetLevel() >= logrus.DebugLevel {
		l.Debug("Reflection and channelz are enabled.")
		reflection.Register(gRPCServer)
		channelz.RegisterChannelzServiceToServer(gRPCServer)

		l.Debug("RPC response latency histogram enabled.")
		grpc_prometheus.EnableHandlingTimeHistogram()
	}

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

type http1ServerDeps struct {
	logs       *supervisord.Logs
	authServer *grafana.AuthServer
}

// runHTTP1Server runs grpc-gateway and other HTTP 1.1 APIs (like auth_request and logs.zip)
// until context is canceled, then gracefully stops it.
func runHTTP1Server(ctx context.Context, deps *http1ServerDeps) {
	l := logrus.WithField("component", "JSON")
	l.Infof("Starting server on http://%s/ ...", http1Addr)

	marshaller := &grpc_gateway.JSONPb{
		OrigName:     true,
		EnumsAsInts:  false,
		EmitDefaults: false,
		Indent:       "  ",
	}

	// FIXME make that a default behavior: https://jira.percona.com/browse/PMM-6722
	if nicer, _ := strconv.ParseBool(os.Getenv("PERCONA_TEST_NICER_API")); nicer {
		l.Warn("Enabling nicer API with default/zero values in response.")
		marshaller.EmitDefaults = true
	}

	proxyMux := grpc_gateway.NewServeMux(
		grpc_gateway.WithMarshalerOption(grpc_gateway.MIMEWildcard, marshaller),
	)
	opts := []grpc.DialOption{grpc.WithInsecure(), grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(5 * 1024 * 1024))}

	// TODO switch from RegisterXXXHandlerFromEndpoint to RegisterXXXHandler to avoid extra dials
	// (even if they dial to localhost)
	// https://jira.percona.com/browse/PMM-4326
	type registrar func(context.Context, *grpc_gateway.ServeMux, string, []grpc.DialOption) error
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
		managementpb.RegisterRDSHandlerFromEndpoint,
		azurev1beta1.RegisterAzureDatabaseHandlerFromEndpoint,
		managementpb.RegisterHAProxyHandlerFromEndpoint,
		managementpb.RegisterExternalHandlerFromEndpoint,
		managementpb.RegisterAnnotationHandlerFromEndpoint,
		managementpb.RegisterSecurityChecksHandlerFromEndpoint,

		iav1beta1.RegisterAlertsHandlerFromEndpoint,
		iav1beta1.RegisterChannelsHandlerFromEndpoint,
		iav1beta1.RegisterRulesHandlerFromEndpoint,
		iav1beta1.RegisterTemplatesHandlerFromEndpoint,

		backupv1beta1.RegisterBackupsHandlerFromEndpoint,
		backupv1beta1.RegisterLocationsHandlerFromEndpoint,
		backupv1beta1.RegisterArtifactsHandlerFromEndpoint,

		jobs1beta1.RegisterJobsHandlerFromEndpoint,

		dbaasv1beta1.RegisterKubernetesHandlerFromEndpoint,
		dbaasv1beta1.RegisterXtraDBClusterHandlerFromEndpoint,
		dbaasv1beta1.RegisterPSMDBClusterHandlerFromEndpoint,
		dbaasv1beta1.RegisterLogsAPIHandlerFromEndpoint,
		dbaasv1beta1.RegisterComponentsHandlerFromEndpoint,
	} {
		if err := r(ctx, proxyMux, gRPCAddr, opts); err != nil {
			l.Panic(err)
		}
	}

	mux := http.NewServeMux()
	addLogsHandler(mux, deps.logs)
	mux.Handle("/auth_request", deps.authServer)
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
		rw.Write(buf.Bytes()) //nolint:errcheck
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

type setupDeps struct {
	sqlDB        *sql.DB
	dbUsername   string
	dbPassword   string
	supervisord  *supervisord.Service
	vmdb         *victoriametrics.Service
	vmalert      *vmalert.Service
	alertmanager *alertmanager.Service
	server       *server.Server
	l            *logrus.Entry
}

// setup migrates database and performs other setup tasks that depend on database.
func setup(ctx context.Context, deps *setupDeps) bool {
	deps.l.Infof("Migrating database...")
	db, err := models.SetupDB(deps.sqlDB, &models.SetupDBParams{
		Logf:          deps.l.Debugf,
		Username:      deps.dbUsername,
		Password:      deps.dbPassword,
		SetupFixtures: models.SetupFixtures,
	})
	if err != nil {
		deps.l.Warnf("Failed to migrate database: %s.", err)
		return false
	}

	// log and ignore validation errors; fail on other errors
	deps.l.Infof("Updating settings...")
	env := os.Environ()
	sort.Strings(env)
	if errs := deps.server.UpdateSettingsFromEnv(env); len(errs) != 0 {
		// This should be impossible in the normal workflow.
		// An invalid environment variable must be caught with pmm-managed-init
		// and the docker run must be terminated.
		deps.l.Errorln("Failed to update settings from environment:")
		for _, e := range errs {
			deps.l.Errorln(e)
		}
		return false
	}

	deps.l.Infof("Updating supervisord configuration...")
	settings, err := models.GetSettings(db.Querier)
	if err != nil {
		deps.l.Warnf("Failed to get settings: %+v.", err)
		return false
	}
	if err = deps.supervisord.UpdateConfiguration(settings); err != nil {
		deps.l.Warnf("Failed to update supervisord configuration: %+v.", err)
		return false
	}

	deps.l.Infof("Checking VictoriaMetrics...")
	if err = deps.vmdb.IsReady(ctx); err != nil {
		deps.l.Warnf("VictoriaMetrics problem: %+v.", err)
		return false
	}
	deps.vmdb.RequestConfigurationUpdate()

	deps.l.Infof("Checking VMAlert...")
	if err = deps.vmalert.IsReady(ctx); err != nil {
		deps.l.Warnf("VMAlert problem: %+v.", err)
		return false
	}
	deps.vmalert.RequestConfigurationUpdate()

	deps.l.Infof("Checking Alertmanager...")
	if err = deps.alertmanager.IsReady(ctx); err != nil {
		deps.l.Warnf("Alertmanager problem: %+v.", err)
		return false
	}
	deps.alertmanager.RequestConfigurationUpdate()

	deps.l.Info("Setup completed.")
	return true
}

func getQANClient(ctx context.Context, sqlDB *sql.DB, dbName, qanAPIAddr string) *qan.Client {
	opts := []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithBackoffMaxDelay(time.Second), //nolint:staticcheck
		grpc.WithUserAgent("pmm-managed/" + version.Version),
	}

	// Without grpc.WithBlock() DialContext returns an error only if something very wrong with address or options;
	// it does not return an error of connection failure but tries to reconnect in the background.
	conn, err := grpc.DialContext(ctx, qanAPIAddr, opts...)
	if err != nil {
		logrus.Fatalf("Failed to connect QAN API %s: %s.", qanAPIAddr, err)
	}

	l := logrus.WithField("component", "reform/qan")
	reformL := sqlmetrics.NewReform("postgres", dbName+"/qan", l.Tracef)
	prom.MustRegister(reformL)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reformL)
	return qan.NewClient(conn, db)
}

func getDBaaSControllerClient(ctx context.Context, dbaasControllerAPIAddr string, settings *models.Settings) *dbaas.Client {
	// TODO Remove once changing settings.DBaaS.Enabled is possible via API.
	if !settings.DBaaS.Enabled {
		return dbaas.NewClient(nil)
	}

	opts := []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithConnectParams(grpc.ConnectParams{Backoff: backoff.Config{MaxDelay: 10 * time.Second}, MinConnectTimeout: 10 * time.Second}),
		grpc.WithUserAgent("pmm-managed/" + version.Version),
	}

	// Without grpc.WithBlock() DialContext returns an error only if something very wrong with address or options;
	// it does not return an error of connection failure but tries to reconnect in the background.
	conn, err := grpc.DialContext(ctx, dbaasControllerAPIAddr, opts...)
	if err != nil {
		logrus.Fatalf("Failed to connect DBaaS Controller API %s: %s.", dbaasControllerAPIAddr, err)
	}

	return dbaas.NewClient(conn)
}

func main() {
	// empty version breaks much of pmm-managed logic
	if version.Version == "" {
		panic("pmm-managed version is not set during build.")
	}

	log.SetFlags(0)
	log.SetPrefix("stdlog: ")

	kingpin.Version(version.FullInfo())
	kingpin.HelpFlag.Short('h')

	victoriaMetricsURLF := kingpin.Flag("victoriametrics-url", "VictoriaMetrics base URL").
		Default("http://127.0.0.1:9090/prometheus/").String()
	victoriaMetricsVMAlertURLF := kingpin.Flag("victoriametrics-vmalert-url", "VictoriaMetrics VMAlert base URL").
		Default("http://127.0.0.1:8880/").String()
	victoriaMetricsConfigF := kingpin.Flag("victoriametrics-config", "VictoriaMetrics scrape configuration file path").
		Default("/etc/victoriametrics-promscrape.yml").String()

	grafanaAddrF := kingpin.Flag("grafana-addr", "Grafana HTTP API address").Default("127.0.0.1:3000").String()
	qanAPIAddrF := kingpin.Flag("qan-api-addr", "QAN API gRPC API address").Default("127.0.0.1:9911").String()
	dbaasControllerAPIAddrF := kingpin.Flag("dbaas-controller-api-addr", "DBaaS Controller gRPC API address").Default("127.0.0.1:20201").String()

	versionServiceAPIURLF := kingpin.Flag("version-service-api-url", "Version Service API URL").Default("https://check.percona.com/versions/v1").String()

	postgresAddrF := kingpin.Flag("postgres-addr", "PostgreSQL address").Default("127.0.0.1:5432").String()
	postgresDBNameF := kingpin.Flag("postgres-name", "PostgreSQL database name").Required().String()
	postgresDBUsernameF := kingpin.Flag("postgres-username", "PostgreSQL database username").Default("pmm-managed").String()
	postgresDBPasswordF := kingpin.Flag("postgres-password", "PostgreSQL database password").Default("pmm-managed").String()

	supervisordConfigDirF := kingpin.Flag("supervisord-config-dir", "Supervisord configuration directory").Required().String()

	debugF := kingpin.Flag("debug", "Enable debug logging").Envar("PMM_DEBUG").Bool()
	traceF := kingpin.Flag("trace", "Enable trace logging (implies debug)").Envar("PMM_TRACE").Bool()

	kingpin.Parse()

	logger.SetupGlobalLogger()
	if *debugF {
		logrus.SetLevel(logrus.DebugLevel)
	}
	if *traceF {
		logrus.SetLevel(logrus.TraceLevel)
		grpclog.SetLoggerV2(&logger.GRPC{Entry: logrus.WithField("component", "grpclog")})
		logrus.SetReportCaller(true)
	}
	logrus.Infof("Log level: %s.", logrus.GetLevel())

	l := logrus.WithField("component", "main")
	ctx, cancel := context.WithCancel(context.Background())
	ctx = logger.Set(ctx, "main")
	defer l.Info("Done.")

	sqlDB, err := models.OpenDB(*postgresAddrF, *postgresDBNameF, *postgresDBUsernameF, *postgresDBPasswordF)
	if err != nil {
		l.Panicf("Failed to connect to database: %+v", err)
	}
	defer sqlDB.Close() //nolint:errcheck
	prom.MustRegister(sqlmetrics.NewCollector("postgres", *postgresDBNameF, sqlDB))
	reformL := sqlmetrics.NewReform("postgres", *postgresDBNameF, logrus.WithField("component", "reform").Tracef)
	prom.MustRegister(reformL)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reformL)

	cleaner := clean.New(db)
	externalRules := vmalert.NewExternalRules()

	vmParams, err := models.NewVictoriaMetricsParams(victoriametrics.BasePrometheusConfigPath)
	if err != nil {
		l.Panicf("cannot load victoriametrics params problem: %+v", err)
	}
	vmdb, err := victoriametrics.NewVictoriaMetrics(*victoriaMetricsConfigF, db, *victoriaMetricsURLF, vmParams)
	if err != nil {
		l.Panicf("VictoriaMetrics service problem: %+v", err)
	}
	vmalert, err := vmalert.NewVMAlert(externalRules, *victoriaMetricsVMAlertURLF)
	if err != nil {
		l.Panicf("VictoriaMetrics VMAlert service problem: %+v", err)
	}
	prom.MustRegister(vmalert)

	qanClient := getQANClient(ctx, sqlDB, *postgresDBNameF, *qanAPIAddrF)

	agentsRegistry := agents.NewRegistry(db, qanClient, vmdb)
	prom.MustRegister(agentsRegistry)

	alertmanager := alertmanager.New(db)
	// Alertmanager is special due to being added to PMM with invalid /etc/alertmanager.yml.
	// Generate configuration file before reloading with supervisord, checking status, etc.
	alertmanager.GenerateBaseConfigs()

	pmmUpdateCheck := supervisord.NewPMMUpdateChecker(logrus.WithField("component", "supervisord/pmm-update-checker"))

	logs := supervisord.NewLogs(version.FullInfo(), pmmUpdateCheck)
	supervisord := supervisord.New(*supervisordConfigDirF, pmmUpdateCheck, vmParams)

	telemetry, err := telemetry.NewService(db, version.Version)
	if err != nil {
		l.Fatalf("Could not create telemetry service: %s", err)
	}

	awsInstanceChecker := server.NewAWSInstanceChecker(db, telemetry)
	grafanaClient := grafana.NewClient(*grafanaAddrF)
	prom.MustRegister(grafanaClient)

	checksService, err := checks.New(agentsRegistry, alertmanager, db)
	if err != nil {
		l.Fatalf("Could not create checks service: %s", err)
	}

	prom.MustRegister(checksService)

	platformService, err := platform.New(db)
	if err != nil {
		l.Fatalf("Could not create platform service: %s", err)
	}

	jobsService := agents.NewJobsService(db, agentsRegistry)

	// Integrated alerts services
	templatesService := ia.NewTemplatesService(db)
	rulesService := ia.NewRulesService(db, templatesService, vmalert, alertmanager)
	alertsService := ia.NewAlertsService(db, alertmanager, templatesService)

	versionService := managementdbaas.NewVersionServiceClient(*versionServiceAPIURLF)

	serverParams := &server.Params{
		DB:                   db,
		VMDB:                 vmdb,
		VMAlert:              vmalert,
		AgentsRegistry:       agentsRegistry,
		Alertmanager:         alertmanager,
		ChecksService:        checksService,
		Supervisord:          supervisord,
		TelemetryService:     telemetry,
		PlatformService:      platformService,
		AwsInstanceChecker:   awsInstanceChecker,
		GrafanaClient:        grafanaClient,
		VMAlertExternalRules: externalRules,
		RulesService:         rulesService,
	}

	server, err := server.NewServer(serverParams)
	if err != nil {
		l.Panicf("Server problem: %+v", err)
	}

	// handle unix signals
	terminationSignals := make(chan os.Signal, 1)
	signal.Notify(terminationSignals, unix.SIGTERM, unix.SIGINT)
	updateSignals := make(chan os.Signal, 1)
	signal.Notify(updateSignals, unix.SIGHUP)
	go func() {
		for {
			select {
			case s := <-terminationSignals:
				signal.Stop(terminationSignals)
				l.Warnf("Got %s, shutting down...", unix.SignalName(s.(unix.Signal)))
				cancel()
				return
			case s := <-updateSignals:
				l.Infof("Got %s, reloading configuration...", unix.SignalName(s.(unix.Signal)))
				err := server.UpdateConfigurations()
				if err != nil {
					l.Warnf("Couldn't reload configuration: %s", err)
				} else {
					l.Info("Configuration reloaded")
				}
			}
		}
	}()

	// try synchronously once, then retry in the background
	deps := &setupDeps{
		sqlDB:        sqlDB,
		dbUsername:   *postgresDBUsernameF,
		dbPassword:   *postgresDBPasswordF,
		supervisord:  supervisord,
		vmdb:         vmdb,
		vmalert:      vmalert,
		alertmanager: alertmanager,
		server:       server,
		l:            logrus.WithField("component", "setup"),
	}
	if !setup(ctx, deps) {
		go func() {
			const delay = 2 * time.Second
			for {
				deps.l.Warnf("Retrying in %s.", delay)
				sleepCtx, sleepCancel := context.WithTimeout(ctx, delay)
				<-sleepCtx.Done()
				sleepCancel()

				if ctx.Err() != nil {
					return
				}

				if setup(ctx, deps) {
					return
				}
			}
		}()
	}
	settings, err := models.GetSettings(sqlDB)
	if err != nil {
		l.Fatalf("Failed to get settings: %+v.", err)
	}

	dbaasControllerClient := getDBaaSControllerClient(ctx, *dbaasControllerAPIAddrF, settings)

	authServer := grafana.NewAuthServer(grafanaClient, awsInstanceChecker)

	l.Info("Starting services...")
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		authServer.Run(ctx)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		vmalert.Run(ctx)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		vmdb.Run(ctx)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		alertmanager.Run(ctx)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		checksService.Run(ctx)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		platformService.Run(ctx)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		supervisord.Run(ctx)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		telemetry.Run(ctx)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		runGRPCServer(ctx, &gRPCServerDeps{
			db:                    db,
			vmdb:                  vmdb,
			server:                server,
			agentsRegistry:        agentsRegistry,
			grafanaClient:         grafanaClient,
			checksService:         checksService,
			dbaasControllerClient: dbaasControllerClient,
			alertmanager:          alertmanager,
			vmalert:               vmalert,
			settings:              settings,
			alertsService:         alertsService,
			templatesService:      templatesService,
			rulesService:          rulesService,
			jobsService:           jobsService,
			versionServiceClient:  versionService,
			minio:                 &minio.Service{},
		})
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		runHTTP1Server(ctx, &http1ServerDeps{
			logs:       logs,
			authServer: authServer,
		})
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		runDebugServer(ctx)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		cleaner.Run(ctx, cleanInterval, cleanOlderThan)
	}()

	wg.Wait()
}
