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
// Package main provides the entry point for the update application.
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
	_ "net/http/pprof" //nolint:gosec // register /debug/pprof
	"net/url"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/alecthomas/kingpin/v2"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_validator "github.com/grpc-ecosystem/go-grpc-middleware/validator"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	grpc_gateway "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/percona/pmm/managed/services/nomad"
	"github.com/pkg/errors"
	metrics "github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/semaphore"
	"golang.org/x/sys/unix"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	channelz "google.golang.org/grpc/channelz/service"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/encoding/protojson"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	accesscontrolv1 "github.com/percona/pmm/api/accesscontrol/v1beta1"
	actionsv1 "github.com/percona/pmm/api/actions/v1"
	advisorsv1 "github.com/percona/pmm/api/advisors/v1"
	agentpb "github.com/percona/pmm/api/agent/pb"
	agentv1 "github.com/percona/pmm/api/agent/v1"
	alertingv1 "github.com/percona/pmm/api/alerting/v1"
	backupv1 "github.com/percona/pmm/api/backup/v1"
	dumpv1beta1 "github.com/percona/pmm/api/dump/v1beta1"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	managementv1 "github.com/percona/pmm/api/management/v1"
	platformv1 "github.com/percona/pmm/api/platform/v1"
	serverv1 "github.com/percona/pmm/api/server/v1"
	uieventsv1 "github.com/percona/pmm/api/uievents/v1"
	userv1 "github.com/percona/pmm/api/user/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services/agents"
	agentgrpc "github.com/percona/pmm/managed/services/agents/grpc"
	"github.com/percona/pmm/managed/services/alerting"
	"github.com/percona/pmm/managed/services/backup"
	"github.com/percona/pmm/managed/services/checks"
	"github.com/percona/pmm/managed/services/config" //nolint:staticcheck
	"github.com/percona/pmm/managed/services/dump"
	"github.com/percona/pmm/managed/services/grafana"
	"github.com/percona/pmm/managed/services/ha"
	"github.com/percona/pmm/managed/services/inventory"
	inventorygrpc "github.com/percona/pmm/managed/services/inventory/grpc"
	"github.com/percona/pmm/managed/services/management"
	managementbackup "github.com/percona/pmm/managed/services/management/backup"
	"github.com/percona/pmm/managed/services/management/common"
	managementdump "github.com/percona/pmm/managed/services/management/dump"
	managementgrpc "github.com/percona/pmm/managed/services/management/grpc"
	"github.com/percona/pmm/managed/services/minio"
	"github.com/percona/pmm/managed/services/platform"
	"github.com/percona/pmm/managed/services/qan"
	"github.com/percona/pmm/managed/services/scheduler"
	"github.com/percona/pmm/managed/services/server"
	"github.com/percona/pmm/managed/services/supervisord"
	"github.com/percona/pmm/managed/services/telemetry"
	"github.com/percona/pmm/managed/services/telemetry/uievents"
	"github.com/percona/pmm/managed/services/user"
	"github.com/percona/pmm/managed/services/versioncache"
	"github.com/percona/pmm/managed/services/victoriametrics"
	"github.com/percona/pmm/managed/services/vmalert"
	"github.com/percona/pmm/managed/utils/clean"
	"github.com/percona/pmm/managed/utils/distribution"
	"github.com/percona/pmm/managed/utils/envvars"
	"github.com/percona/pmm/managed/utils/interceptors"
	platformClient "github.com/percona/pmm/managed/utils/platform"
	pmmerrors "github.com/percona/pmm/utils/errors"
	"github.com/percona/pmm/utils/logger"
	"github.com/percona/pmm/utils/sqlmetrics"
	"github.com/percona/pmm/version"
)

var (
	interfaceToBind = envvars.GetInterfaceToBind()
	gRPCAddr        = net.JoinHostPort(interfaceToBind, "7771")
	http1Addr       = net.JoinHostPort(interfaceToBind, "7772")
	debugAddr       = net.JoinHostPort(interfaceToBind, "7773")
)

const (
	shutdownTimeout    = 3 * time.Second
	gRPCMessageMaxSize = 100 * 1024 * 1024

	cleanInterval  = 10 * time.Minute
	cleanOlderThan = 30 * time.Minute

	defaultContextTimeout = 10 * time.Second
	pProfProfileDuration  = 30 * time.Second
	pProfTraceDuration    = 10 * time.Second

	clickhouseMaxIdleConns = 5
	clickhouseMaxOpenConns = 10

	distributionInfoFilePath = "/srv/pmm-distribution"
	osInfoFilePath           = "/proc/version"
)

var pprofSemaphore = semaphore.NewWeighted(1)

func addLogsHandler(mux *http.ServeMux, logs *server.Logs) {
	l := logrus.WithField("component", "logs.zip")

	mux.HandleFunc("/v1/server/logs.zip", func(rw http.ResponseWriter, req *http.Request) {
		var lineCount int64
		contextTimeout := defaultContextTimeout
		// increase context timeout if pprof query parameter exist in request
		pprofQueryParameter, err := strconv.ParseBool(req.FormValue("pprof"))
		if err != nil {
			l.Debug("Unable to read 'pprof' query param. Using default: pprof=false")
		}
		var pprofConfig *server.PprofConfig
		if pprofQueryParameter {
			if !pprofSemaphore.TryAcquire(1) {
				rw.WriteHeader(http.StatusLocked)
				_, err := rw.Write([]byte("Pprof is already running. Please try again later."))
				if err != nil {
					l.Errorf("%+v", err)
				}
				return
			}
			defer pprofSemaphore.Release(1)

			contextTimeout += pProfProfileDuration + pProfTraceDuration
			pprofConfig = &server.PprofConfig{
				ProfileDuration: pProfProfileDuration,
				TraceDuration:   pProfTraceDuration,
			}
		}

		lc := req.FormValue("line-count")
		if lc != "" {
			lineCount, err = strconv.ParseInt(lc, 10, 32)
			if err != nil {
				l.Error(err)
				http.Error(rw, fmt.Sprintf("Unable to parse line-count parameter: %s", err), http.StatusBadRequest)
				return
			}
		}

		// fail-safe
		ctx, cancel := context.WithTimeout(req.Context(), contextTimeout)
		defer cancel()

		filename := fmt.Sprintf("pmm-server_%s.zip", time.Now().UTC().Format("2006-01-02_15-04"))
		rw.Header().Set(`Access-Control-Allow-Origin`, `*`)
		rw.Header().Set(`Content-Type`, `application/zip`)
		rw.Header().Set(`Content-Disposition`, `attachment; filename="`+filename+`"`)

		ctx = logger.Set(ctx, "logs")
		if err := logs.Zip(ctx, rw, pprofConfig, int(lineCount)); err != nil {
			l.Errorf("%+v", err)
		}
	})
}

type gRPCServerDeps struct {
	db                   *reform.DB
	ha                   *ha.Service
	checksService        *checks.Service
	config               *config.Config
	agentsRegistry       *agents.Registry
	handler              *agents.Handler
	actions              *agents.ActionsService
	agentService         *agents.AgentService
	jobsService          *agents.JobsService
	connectionCheck      *agents.ConnectionChecker
	serviceInfoBroker    *agents.ServiceInfoBroker
	agentsStateUpdater   *agents.StateUpdater
	grafanaClient        *grafana.Client
	templatesService     *alerting.Service
	backupService        *backup.Service
	dumpService          *dump.Service
	compatibilityService *backup.CompatibilityService
	backupRemovalService *backup.RemovalService
	pbmPITRService       *backup.PBMPITRService
	vmClient             *metrics.Client
	minioClient          *minio.Client
	settings             *models.Settings
	platformClient       *platformClient.Client
	schedulerService     *scheduler.Service
	supervisord          *supervisord.Service
	server               *server.Server
	uieventsService      *uievents.Service
	versionCache         *versioncache.Service
	vmdb                 *victoriametrics.Service
	vmalert              *vmalert.Service
}

// runGRPCServer runs gRPC server until context is canceled, then gracefully stops it.
//
//nolint:lll
func runGRPCServer(ctx context.Context, deps *gRPCServerDeps) {
	l := logrus.WithField("component", "gRPC")
	l.Infof("Starting server on http://%s/ ...", gRPCAddr)

	grpcMetrics := grpc_prometheus.NewServerMetricsWithExtension(&interceptors.GRPCMetricsExtension{})
	prom.MustRegister(grpcMetrics)

	gRPCServer := grpc.NewServer(
		grpc.MaxRecvMsgSize(gRPCMessageMaxSize),

		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			interceptors.Unary(grpcMetrics.UnaryServerInterceptor()),
			interceptors.UnaryServiceEnabledInterceptor(),
			grpc_validator.UnaryServerInterceptor())),
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			interceptors.Stream(grpcMetrics.StreamServerInterceptor()),
			interceptors.StreamServiceEnabledInterceptor(),
			grpc_validator.StreamServerInterceptor())),
	)

	if l.Logger.GetLevel() >= logrus.DebugLevel {
		l.Debug("Reflection and channelz are enabled.")
		reflection.Register(gRPCServer)
		channelz.RegisterChannelzServiceToServer(gRPCServer)

		l.Debug("RPC response latency histogram enabled.")
		grpcMetrics.EnableHandlingTimeHistogram()
	}
	serverv1.RegisterServerServiceServer(gRPCServer, deps.server)
	agentv1.RegisterAgentServiceServer(gRPCServer, agentgrpc.NewAgentServer(deps.handler))
	agentpb.RegisterAgentServer(gRPCServer, agentgrpc.NewAgentPBServer(deps.handler))

	nodesSvc := inventory.NewNodesService(deps.db, deps.agentsRegistry, deps.agentsStateUpdater, deps.vmdb)
	agentsSvc := inventory.NewAgentsService(
		deps.db, deps.agentsRegistry, deps.agentsStateUpdater,
		deps.vmdb, deps.connectionCheck, deps.serviceInfoBroker, deps.agentService)

	mgmtBackupService := managementbackup.NewBackupsService(deps.db, deps.backupService, deps.compatibilityService, deps.schedulerService, deps.backupRemovalService, deps.pbmPITRService)
	mgmtRestoreService := managementbackup.NewRestoreService(deps.db, deps.backupService, deps.schedulerService)
	mgmtServices := common.NewMgmtServices(mgmtBackupService, mgmtRestoreService)

	servicesSvc := inventory.NewServicesService(deps.db, deps.agentsRegistry, deps.agentsStateUpdater, deps.vmdb, deps.versionCache, mgmtServices)
	inventoryv1.RegisterNodesServiceServer(gRPCServer, inventorygrpc.NewNodesServer(nodesSvc))
	inventoryv1.RegisterServicesServiceServer(gRPCServer, inventorygrpc.NewServicesServer(servicesSvc))
	inventoryv1.RegisterAgentsServiceServer(gRPCServer, inventorygrpc.NewAgentsServer(agentsSvc))

	managementSvc := management.NewManagementService(deps.db, deps.agentsRegistry, deps.agentsStateUpdater, deps.connectionCheck, deps.serviceInfoBroker, deps.vmdb, deps.versionCache, deps.grafanaClient, v1.NewAPI(*deps.vmClient))

	managementv1.RegisterManagementServiceServer(gRPCServer, managementSvc)
	actionsv1.RegisterActionsServiceServer(gRPCServer, managementgrpc.NewActionsServer(deps.actions, deps.db))
	advisorsv1.RegisterAdvisorServiceServer(gRPCServer, management.NewChecksAPIService(deps.checksService))

	accesscontrolv1.RegisterAccessControlServiceServer(gRPCServer, management.NewAccessControlService(deps.db))

	alertingv1.RegisterAlertingServiceServer(gRPCServer, deps.templatesService)

	backupv1.RegisterBackupServiceServer(gRPCServer, mgmtBackupService)
	backupv1.RegisterLocationsServiceServer(gRPCServer, managementbackup.NewLocationsService(deps.db, deps.minioClient))
	backupv1.RegisterRestoreServiceServer(gRPCServer, mgmtRestoreService)

	dumpv1beta1.RegisterDumpServiceServer(gRPCServer, managementdump.New(deps.db, deps.grafanaClient, deps.dumpService))

	userv1.RegisterUserServiceServer(gRPCServer, user.NewUserService(deps.db, deps.grafanaClient))

	platformService := platform.New(deps.platformClient, deps.db, deps.supervisord, deps.checksService, deps.grafanaClient)
	platformv1.RegisterPlatformServiceServer(gRPCServer, platformService)
	uieventsv1.RegisterUIEventsServiceServer(gRPCServer, deps.uieventsService)

	// run server until it is stopped gracefully or not
	listener, err := net.Listen("tcp", gRPCAddr)
	if err != nil {
		l.Panic(err)
	}
	go func() {
		for {
			err = gRPCServer.Serve(listener)
			if err == nil || errors.Is(err, grpc.ErrServerStopped) {
				break
			}
			l.Errorf("Failed to serve: %s", err)
		}
		l.Info("Server stopped.")
	}()

	<-ctx.Done()

	// try to stop server gracefully, then not
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout) //nolint:contextcheck
	go func() {
		<-ctx.Done()
		gRPCServer.Stop()
	}()
	gRPCServer.GracefulStop()
	cancel()
}

type http1ServerDeps struct {
	logs       *server.Logs
	authServer *grafana.AuthServer
}

// runHTTP1Server runs grpc-gateway and other HTTP 1.1 APIs (like auth_request and logs.zip)
// until context is canceled, then gracefully stops it.
func runHTTP1Server(ctx context.Context, deps *http1ServerDeps) {
	l := logrus.WithField("component", "JSON")
	l.Infof("Starting server on http://%s/ ...", http1Addr)

	marshaller := &grpc_gateway.JSONPb{
		MarshalOptions: protojson.MarshalOptions{
			UseEnumNumbers:  false,
			EmitUnpopulated: true,
			UseProtoNames:   true,
			Indent:          "  ",
		},
		UnmarshalOptions: protojson.UnmarshalOptions{
			DiscardUnknown: true,
		},
	}

	proxyMux := grpc_gateway.NewServeMux(
		grpc_gateway.WithMarshalerOption(grpc_gateway.MIMEWildcard, marshaller),
		grpc_gateway.WithErrorHandler(pmmerrors.PMMHTTPErrorHandler),
		grpc_gateway.WithRoutingErrorHandler(pmmerrors.PMMRoutingErrorHandler))

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(gRPCMessageMaxSize)),
	}

	// TODO switch from RegisterXXXHandlerFromEndpoint to RegisterXXXHandler to avoid extra dials
	// (even if they dial to localhost)
	// https://jira.percona.com/browse/PMM-4326
	type registrar func(context.Context, *grpc_gateway.ServeMux, string, []grpc.DialOption) error
	for _, r := range []registrar{
		serverv1.RegisterServerServiceHandlerFromEndpoint,

		inventoryv1.RegisterNodesServiceHandlerFromEndpoint,
		inventoryv1.RegisterServicesServiceHandlerFromEndpoint,
		inventoryv1.RegisterAgentsServiceHandlerFromEndpoint,

		managementv1.RegisterManagementServiceHandlerFromEndpoint,
		actionsv1.RegisterActionsServiceHandlerFromEndpoint,
		advisorsv1.RegisterAdvisorServiceHandlerFromEndpoint,
		accesscontrolv1.RegisterAccessControlServiceHandlerFromEndpoint,

		alertingv1.RegisterAlertingServiceHandlerFromEndpoint,

		backupv1.RegisterBackupServiceHandlerFromEndpoint,
		backupv1.RegisterLocationsServiceHandlerFromEndpoint,
		backupv1.RegisterRestoreServiceHandlerFromEndpoint,

		dumpv1beta1.RegisterDumpServiceHandlerFromEndpoint,

		platformv1.RegisterPlatformServiceHandlerFromEndpoint,
		uieventsv1.RegisterUIEventsServiceHandlerFromEndpoint,

		userv1.RegisterUserServiceHandlerFromEndpoint,
	} {
		if err := r(ctx, proxyMux, gRPCAddr, opts); err != nil {
			l.Panic(err)
		}
	}

	mux := http.NewServeMux()
	addLogsHandler(mux, deps.logs)
	mux.Handle("/auth_request", deps.authServer)
	mux.Handle("/", proxyMux)

	server := &http.Server{ //nolint:gosec
		Addr:     http1Addr,
		ErrorLog: log.New(os.Stderr, "runJSONServer: ", 0),
		Handler:  mux,
	}
	go func() {
		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			l.Panic(err)
		}
		l.Info("Server stopped.")
	}()

	<-ctx.Done()
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	if err := server.Shutdown(ctx); err != nil { //nolint:contextcheck
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

	server := &http.Server{ //nolint:gosec
		Addr:     debugAddr,
		ErrorLog: log.New(os.Stderr, "runDebugServer: ", 0),
	}
	go func() {
		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			l.Panic(err)
		}
		l.Info("Server stopped.")
	}()

	<-ctx.Done()
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	if err := server.Shutdown(ctx); err != nil { //nolint:contextcheck
		l.Errorf("Failed to shutdown gracefully: %s", err)
	}
	cancel()
}

type setupDeps struct {
	sqlDB       *sql.DB
	ha          *ha.Service
	supervisord *supervisord.Service
	vmdb        *victoriametrics.Service
	vmalert     *vmalert.Service
	server      *server.Server
	l           *logrus.Entry
}

// setup performs setup tasks that depend on database.
func setup(ctx context.Context, deps *setupDeps) bool {
	l := reform.NewPrintfLogger(deps.l.Debugf)
	db := reform.NewDB(deps.sqlDB, postgresql.Dialect, l)

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
		deps.l.Warnf("Failed to get settings: %s.", err)
		return false
	}
	ssoDetails, err := models.GetPerconaSSODetails(ctx, db.Querier)
	if err != nil && !errors.Is(err, models.ErrNotConnectedToPortal) {
		deps.l.Warnf("Failed to get Percona SSO Details: %s.", err)
	}
	if err = deps.supervisord.UpdateConfiguration(settings, ssoDetails); err != nil {
		deps.l.Warnf("Failed to update supervisord configuration: %s.", err)
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

	deps.l.Info("Setup completed.")
	return true
}

func getQANClient(ctx context.Context, sqlDB *sql.DB, dbName, qanAPIAddr string) *qan.Client {
	bc := backoff.DefaultConfig
	bc.MaxDelay = time.Second

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithConnectParams(grpc.ConnectParams{
			Backoff: bc,
		}),
		grpc.WithUserAgent("pmm-managed/" + version.Version),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(gRPCMessageMaxSize)),
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

func migrateDB(ctx context.Context, sqlDB *sql.DB, params models.SetupDBParams) {
	l := logrus.WithField("component", "migration")
	params.Logf = l.Debugf
	params.SetupFixtures = models.SetupFixtures

	const timeout = 5 * time.Minute
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	for {
		select {
		case <-timeoutCtx.Done():
			l.Fatalf("Could not migrate DB: timeout")
		default:
		}
		l.Infof("Migrating database...")
		_, err := models.SetupDB(timeoutCtx, sqlDB, params)
		if err == nil {
			return
		}

		l.Warnf("Failed to migrate database: %s.", err)
		time.Sleep(time.Second)
	}
}

// newClickhouseDB return a new Clickhouse db.
func newClickhouseDB(dsn string, maxIdleConns, maxOpenConns int) (*sql.DB, error) {
	db, err := sql.Open("clickhouse", dsn)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to open connection to QAN DB")
	}

	db.SetConnMaxLifetime(0)
	db.SetMaxIdleConns(maxIdleConns)
	db.SetMaxOpenConns(maxOpenConns)

	return db, nil
}

func main() { //nolint:maintidx,cyclop
	// empty version breaks much of pmm-managed logic
	if version.Version == "" {
		panic("pmm-managed version is not set during build.")
	}

	log.SetFlags(0)
	log.SetPrefix("stdlog: ")

	kingpin.Version(version.FullInfo())
	kingpin.HelpFlag.Short('h')

	victoriaMetricsURLF := kingpin.Flag("victoriametrics-url", "VictoriaMetrics base URL").Envar("PMM_VM_URL").
		Default(models.VMBaseURL).String()
	victoriaMetricsVMAlertURLF := kingpin.Flag("victoriametrics-vmalert-url", "VictoriaMetrics VMAlert base URL").Envar("PMM_VM_ALERT_URL").
		Default("http://127.0.0.1:8880/").String()
	victoriaMetricsConfigF := kingpin.Flag("victoriametrics-config", "VictoriaMetrics scrape configuration file path").
		Default("/etc/victoriametrics-promscrape.yml").String()

	grafanaAddrF := kingpin.Flag("grafana-addr", "Grafana HTTP API address").Default("127.0.0.1:3000").String()
	qanAPIAddrF := kingpin.Flag("qan-api-addr", "QAN API gRPC API address").Default("127.0.0.1:9911").String()

	postgresAddrF := kingpin.Flag("postgres-addr", "PostgreSQL address").
		Default(models.DefaultPostgreSQLAddr).
		Envar("PMM_POSTGRES_ADDR").
		String()
	postgresDBNameF := kingpin.Flag("postgres-name", "PostgreSQL database name").
		Default("pmm-managed").
		Envar("PMM_POSTGRES_DBNAME").
		String()
	postgresDBUsernameF := kingpin.Flag("postgres-username", "PostgreSQL database username").
		Default("pmm-managed").
		Envar("PMM_POSTGRES_USERNAME").
		String()
	postgresSSLModeF := kingpin.Flag("postgres-ssl-mode", "PostgreSQL SSL mode").
		Default(models.DisableSSLMode).
		Envar("PMM_POSTGRES_SSL_MODE").
		Enum(models.DisableSSLMode, models.RequireSSLMode, models.VerifyCaSSLMode, models.VerifyFullSSLMode)
	postgresSSLCAPathF := kingpin.Flag("postgres-ssl-ca-path", "PostgreSQL SSL CA root certificate path").
		Envar("PMM_POSTGRES_SSL_CA_PATH").
		String()
	postgresDBPasswordF := kingpin.Flag("postgres-password", "PostgreSQL database password").
		Default("pmm-managed").
		Envar("PMM_POSTGRES_DBPASSWORD").
		String()
	postgresSSLKeyPathF := kingpin.Flag("postgres-ssl-key-path", "PostgreSQL SSL key path").
		Envar("PMM_POSTGRES_SSL_KEY_PATH").
		String()
	postgresSSLCertPathF := kingpin.Flag("postgres-ssl-cert-path", "PostgreSQL SSL certificate path").
		Envar("PMM_POSTGRES_SSL_CERT_PATH").
		String()

	haEnabled := kingpin.Flag("ha-enable", "Enable HA").
		Envar("PMM_TEST_HA_ENABLE").
		Bool()
	haBootstrap := kingpin.Flag("ha-bootstrap", "Bootstrap HA cluster").
		Envar("PMM_TEST_HA_BOOTSTRAP").
		Bool()
	haNodeID := kingpin.Flag("ha-node-id", "HA Node ID").
		Envar("PMM_TEST_HA_NODE_ID").
		String()
	haAdvertiseAddress := kingpin.Flag("ha-advertise-address", "HA Advertise address").
		Envar("PMM_TEST_HA_ADVERTISE_ADDRESS").
		String()
	haPeers := kingpin.Flag("ha-peers", "HA Peers").
		Envar("PMM_TEST_HA_PEERS").
		String()
	haRaftPort := kingpin.Flag("ha-raft-port", "HA raft port").
		Envar("PMM_TEST_HA_RAFT_PORT").
		Default("9760").
		Int()
	haGossipPort := kingpin.Flag("ha-gossip-port", "HA gossip port").
		Envar("PMM_TEST_HA_GOSSIP_PORT").
		Default("9761").
		Int()
	haGrafanaGossipPort := kingpin.Flag("ha-grafana-gossip-port", "HA Grafana gossip port").
		Envar("PMM_TEST_HA_GRAFANA_GOSSIP_PORT").
		Default("9762").
		Int()

	supervisordConfigDirF := kingpin.Flag("supervisord-config-dir", "Supervisord configuration directory").Required().String()

	logLevelF := kingpin.Flag("log-level", "Set logging level").Envar("PMM_LOG_LEVEL").Default("info").Enum("trace", "debug", "info", "warn", "error", "fatal")
	debugF := kingpin.Flag("debug", "Enable debug logging").Envar("PMM_DEBUG").Bool()
	traceF := kingpin.Flag("trace", "[DEPRECATED] Enable trace logging (implies debug)").Envar("PMM_TRACE").Bool()

	clickHouseDatabaseF := kingpin.Flag("clickhouse-name", "Clickhouse database name").Default("pmm").Envar("PMM_CLICKHOUSE_DATABASE").String()
	clickhouseAddrF := kingpin.Flag("clickhouse-addr", "Clickhouse database address").Default("127.0.0.1:9000").Envar("PMM_CLICKHOUSE_ADDR").String()

	watchtowerHostF := kingpin.Flag("watchtower-host", "Watchtower host").Default("http://watchtower:8080").Envar("PMM_WATCHTOWER_HOST").URL()

	kingpin.Parse()

	logger.SetupGlobalLogger()

	level := parseLoggerConfig(*logLevelF, *debugF, *traceF)

	logrus.SetLevel(level)

	if level == logrus.TraceLevel {
		grpclog.SetLoggerV2(&logger.GRPC{Entry: logrus.WithField("component", "grpclog")})
		logrus.SetReportCaller(true)
	}

	logrus.Infof("Log level: %s.", logrus.GetLevel())

	l := logrus.WithField("component", "main")
	ctx, cancel := context.WithCancel(context.Background())
	ctx = logger.Set(ctx, "main")
	defer l.Info("Done.")

	var nodes []string
	if *haPeers != "" {
		nodes = strings.Split(*haPeers, ",")
	}
	haParams := &models.HAParams{
		Enabled:           *haEnabled,
		Bootstrap:         *haBootstrap,
		NodeID:            *haNodeID,
		AdvertiseAddress:  *haAdvertiseAddress,
		Nodes:             nodes,
		RaftPort:          *haRaftPort,
		GossipPort:        *haGossipPort,
		GrafanaGossipPort: *haGrafanaGossipPort,
	}
	haService := ha.New(haParams)

	cfg := config.NewService()
	if err := cfg.Load(); err != nil {
		l.Panicf("Failed to load config: %+v", err)
	}
	// in order to reproduce postgres behaviour.
	if *postgresSSLModeF == models.RequireSSLMode && *postgresSSLCAPathF != "" {
		*postgresSSLModeF = models.VerifyCaSSLMode
	}
	ds := cfg.Config.Services.Telemetry.DataSources

	pmmdb := ds.PmmDBSelect
	pmmdb.Credentials.Username = *postgresDBUsernameF
	pmmdb.Credentials.Password = *postgresDBPasswordF
	pmmdb.DSN.Scheme = "postgres" // TODO: should be configurable
	pmmdb.DSN.Host = *postgresAddrF
	pmmdb.DSN.DB = *postgresDBNameF
	q := make(url.Values)
	q.Set("sslmode", *postgresSSLModeF)
	if *postgresSSLModeF != models.DisableSSLMode {
		q.Set("sslrootcert", *postgresSSLCAPathF)
		q.Set("sslcert", *postgresSSLCertPathF)
		q.Set("sslkey", *postgresSSLKeyPathF)
	}
	pmmdb.DSN.Params = q.Encode()

	grafanadb := ds.GrafanaDBSelect
	grafanadb.DSN.Scheme = "postgres"
	grafanadb.DSN.Host = *postgresAddrF
	grafanadb.DSN.DB = "grafana"
	grafanadb.DSN.Params = q.Encode()

	clickhouseDSN := "tcp://" + *clickhouseAddrF + "/" + *clickHouseDatabaseF

	qanDB := ds.QanDBSelect
	qanDB.DSN = clickhouseDSN

	ds.VM.Address = *victoriaMetricsURLF

	vmParams, err := models.NewVictoriaMetricsParams(
		models.BasePrometheusConfigPath,
		*victoriaMetricsURLF)
	if err != nil {
		l.Panicf("cannot load victoriametrics params problem: %+v", err)
	}

	setupParams := models.SetupDBParams{
		Address:     *postgresAddrF,
		Name:        *postgresDBNameF,
		Username:    *postgresDBUsernameF,
		Password:    *postgresDBPasswordF,
		SSLMode:     *postgresSSLModeF,
		SSLCAPath:   *postgresSSLCAPathF,
		SSLKeyPath:  *postgresSSLKeyPathF,
		SSLCertPath: *postgresSSLCertPathF,
	}

	sqlDB, err := models.OpenDB(setupParams)
	if err != nil {
		l.Panicf("Failed to connect to database: %+v", err)
	}
	defer sqlDB.Close() //nolint:errcheck

	if haService.Bootstrap() {
		migrateDB(ctx, sqlDB, setupParams)
	}

	prom.MustRegister(sqlmetrics.NewCollector("postgres", *postgresDBNameF, sqlDB))
	reformL := sqlmetrics.NewReform("postgres", *postgresDBNameF, logrus.WithField("component", "reform").Tracef)
	prom.MustRegister(reformL)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reformL)

	if haService.Bootstrap() {
		// Generate unique PMM Server ID if it's not already set.
		err = models.SetPMMServerID(db)
		if err != nil {
			l.Panicf("failed to set PMM Server ID")
		}
	}

	cleaner := clean.New(db)
	externalRules := vmalert.NewExternalRules()
	vmdb, err := victoriametrics.NewVictoriaMetrics(*victoriaMetricsConfigF, db, vmParams)
	if err != nil {
		l.Panicf("VictoriaMetrics service problem: %+v", err)
	}
	vmalert, err := vmalert.NewVMAlert(externalRules, *victoriaMetricsVMAlertURLF)
	if err != nil {
		l.Panicf("VictoriaMetrics VMAlert service problem: %+v", err)
	}
	prom.MustRegister(vmalert)

	minioClient := minio.New()

	qanClient := getQANClient(ctx, sqlDB, *postgresDBNameF, *qanAPIAddrF)

	agentsRegistry := agents.NewRegistry(db, vmParams)

	// TODO remove once PMM cluster will be Active-Active
	haService.AddLeaderService(ha.NewStandardService("agentsRegistry", func(ctx context.Context) error { return nil }, func() {
		agentsRegistry.KickAll(ctx)
	}))

	pbmPITRService := backup.NewPBMPITRService()
	backupRemovalService := backup.NewRemovalService(db, pbmPITRService)
	backupRetentionService := backup.NewRetentionService(db, backupRemovalService)
	prom.MustRegister(agentsRegistry)

	inventoryMetrics := inventory.NewInventoryMetrics(db, agentsRegistry)
	inventoryMetricsCollector := inventory.NewInventoryMetricsCollector(inventoryMetrics)
	prom.MustRegister(inventoryMetricsCollector)

	connectionCheck := agents.NewConnectionChecker(agentsRegistry)
	serviceInfoBroker := agents.NewServiceInfoBroker(agentsRegistry)

	updater := server.NewUpdater(*watchtowerHostF, gRPCMessageMaxSize, db)

	logs := server.NewLogs(version.FullInfo(), updater, vmParams)

	supervisord := supervisord.New(
		*supervisordConfigDirF,
		&models.Params{
			VMParams: vmParams,
			PGParams: &models.PGParams{
				Addr:        *postgresAddrF,
				DBName:      *postgresDBNameF,
				DBUsername:  *postgresDBUsernameF,
				DBPassword:  *postgresDBPasswordF,
				SSLMode:     *postgresSSLModeF,
				SSLCAPath:   *postgresSSLCAPathF,
				SSLKeyPath:  *postgresSSLKeyPathF,
				SSLCertPath: *postgresSSLCertPathF,
			},
			HAParams: haParams,
		})

	haService.AddLeaderService(ha.NewStandardService("pmm-agent-runner", func(ctx context.Context) error {
		return supervisord.StartSupervisedService("pmm-agent")
	}, func() {
		err := supervisord.StopSupervisedService("pmm-agent")
		if err != nil {
			l.Warnf("couldn't stop pmm-agent: %q", err)
		}
	}))

	platformAddress, err := envvars.GetPlatformAddress()
	if err != nil {
		l.Fatal(err)
	}

	platformClient, err := platformClient.NewClient(db, platformAddress)
	if err != nil {
		l.Fatalf("Could not create Percona Portal client: %s", err)
	}

	uieventsService := uievents.New()
	uieventsService.ScheduleCleanup(ctx)

	telemetryExtensions := map[telemetry.ExtensionType]telemetry.Extension{
		telemetry.UIEventsExtension: uieventsService,
	}

	dus := distribution.NewService(distributionInfoFilePath, osInfoFilePath, l)
	telemetry, err := telemetry.NewService(db, platformClient, version.Version, dus, cfg.Config.Services.Telemetry, telemetryExtensions)
	if err != nil {
		l.Fatalf("Could not create telemetry service: %s", err)
	}

	grafanaClient := grafana.NewClient(*grafanaAddrF)
	prom.MustRegister(grafanaClient)
	nomad, err := nomad.New(db)

	jobsService := agents.NewJobsService(db, agentsRegistry, backupRetentionService)
	agentsStateUpdater := agents.NewStateUpdater(db, agentsRegistry, vmdb, vmParams, nomad)
	agentsHandler := agents.NewHandler(db, qanClient, vmdb, agentsRegistry, agentsStateUpdater, jobsService)

	actionsService := agents.NewActionsService(qanClient, agentsRegistry)

	vmClient, err := metrics.NewClient(metrics.Config{Address: *victoriaMetricsURLF})
	if err != nil {
		l.Fatalf("Could not create Victoria Metrics client: %s", err)
	}

	clickhouseClient, err := newClickhouseDB(clickhouseDSN, clickhouseMaxIdleConns, clickhouseMaxOpenConns)
	if err != nil {
		l.Fatalf("Could not create Clickhouse client: %s", err)
	}

	checksService := checks.New(db, platformClient, actionsService, v1.NewAPI(vmClient), clickhouseClient)
	prom.MustRegister(checksService)

	alertingService, err := alerting.NewService(db, platformClient, grafanaClient)
	if err != nil {
		l.Fatalf("Could not create alerting service: %s", err)
	}
	alertingService.CollectTemplates(ctx)

	agentService := agents.NewAgentService(agentsRegistry)

	versioner := agents.NewVersionerService(agentsRegistry)
	compatibilityService := backup.NewCompatibilityService(db, versioner)
	backupService := backup.NewService(db, jobsService, agentService, compatibilityService, pbmPITRService)
	backupMetricsCollector := backup.NewMetricsCollector(db)
	prom.MustRegister(backupMetricsCollector)

	schedulerService := scheduler.New(db, backupService)
	versionCache := versioncache.New(db, versioner)

	dumpService := dump.New(db)

	serverParams := &server.Params{
		DB:                   db,
		VMDB:                 vmdb,
		VMAlert:              vmalert,
		AgentsStateUpdater:   agentsStateUpdater,
		ChecksService:        checksService,
		TemplatesService:     alertingService,
		Supervisord:          supervisord,
		TelemetryService:     telemetry,
		GrafanaClient:        grafanaClient,
		VMAlertExternalRules: externalRules,
		Updater:              updater,
		Dus:                  dus,
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
				l.Warnf("Got %s, shutting down...", unix.SignalName(s.(unix.Signal))) //nolint:forcetypeassert
				cancel()
				return
			case s := <-updateSignals:
				l.Infof("Got %s, reloading configuration...", unix.SignalName(s.(unix.Signal))) //nolint:forcetypeassert
				err := server.UpdateConfigurations(ctx)
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
		sqlDB:       sqlDB,
		ha:          haService,
		supervisord: supervisord,
		vmdb:        vmdb,
		vmalert:     vmalert,
		server:      server,
		l:           logrus.WithField("component", "setup"),
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

	authServer := grafana.NewAuthServer(grafanaClient, db)

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

	haService.AddLeaderService(ha.NewContextService("checks", func(ctx context.Context) error {
		checksService.Run(ctx)
		return nil
	}))

	wg.Add(1)
	go func() {
		defer wg.Done()
		supervisord.Run(ctx)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		updater.Run(ctx)
	}()

	wg.Add(1)
	haService.AddLeaderService(ha.NewContextService("telemetry", func(ctx context.Context) error {
		defer wg.Done()
		telemetry.Run(ctx)
		return nil
	}))

	haService.AddLeaderService(ha.NewContextService("scheduler", func(ctx context.Context) error {
		schedulerService.Run(ctx)
		return nil
	}))

	haService.AddLeaderService(ha.NewContextService("versionCache", func(ctx context.Context) error {
		versionCache.Run(ctx)
		return nil
	}))

	wg.Add(1)
	go func() {
		defer wg.Done()
		runGRPCServer(ctx,
			&gRPCServerDeps{
				actions:              actionsService,
				agentService:         agentService,
				agentsRegistry:       agentsRegistry,
				agentsStateUpdater:   agentsStateUpdater,
				backupRemovalService: backupRemovalService,
				backupService:        backupService,
				checksService:        checksService,
				compatibilityService: compatibilityService,
				config:               &cfg.Config,
				connectionCheck:      connectionCheck,
				db:                   db,
				dumpService:          dumpService,
				grafanaClient:        grafanaClient,
				handler:              agentsHandler,
				ha:                   haService,
				jobsService:          jobsService,
				minioClient:          minioClient,
				pbmPITRService:       pbmPITRService,
				platformClient:       platformClient,
				schedulerService:     schedulerService,
				server:               server,
				serviceInfoBroker:    serviceInfoBroker,
				settings:             settings,
				supervisord:          supervisord,
				templatesService:     alertingService,
				uieventsService:      uieventsService,
				versionCache:         versionCache,
				vmalert:              vmalert,
				vmClient:             &vmClient,
				vmdb:                 vmdb,
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

	haService.AddLeaderService(ha.NewContextService("cleaner", func(ctx context.Context) error {
		cleaner.Run(ctx, cleanInterval, cleanOlderThan)
		return nil
	}))

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := haService.Run(ctx)
		if err != nil {
			l.Panicf("cannot start high availability service: %+v", err)
		}
	}()

	wg.Wait()
}

func parseLoggerConfig(level string, debug, trace bool) logrus.Level {
	if trace {
		return logrus.TraceLevel
	}

	if debug {
		return logrus.DebugLevel
	}

	if level != "" {
		parsedLevel, err := logrus.ParseLevel(level)

		if err != nil {
			logrus.Warnf("Cannot parse logging level: %s, error: %v", level, err)
		} else {
			return parsedLevel
		}
	}

	return logrus.InfoLevel
}
