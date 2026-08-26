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

package main

import (
	"bytes"
	"context"
	"errors"
	_ "expvar" // register /debug/vars
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof" //nolint:gosec
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"github.com/alecthomas/kingpin/v2"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_validator "github.com/grpc-ecosystem/go-grpc-middleware/validator"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	grpc_gateway "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/jmoiron/sqlx"
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
	"google.golang.org/grpc"
	channelz "google.golang.org/grpc/channelz/service"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/encoding/protojson"

	qanv1 "github.com/percona/pmm/api/qan/v1"
	"github.com/percona/pmm/qan-api2/models"
	aservice "github.com/percona/pmm/qan-api2/services/analytics"
	rservice "github.com/percona/pmm/qan-api2/services/receiver"
	"github.com/percona/pmm/qan-api2/utils/interceptors"
	"github.com/percona/pmm/utils/dsnutils"
	pmmerrors "github.com/percona/pmm/utils/errors"
	"github.com/percona/pmm/utils/logger"
	"github.com/percona/pmm/utils/sqlmetrics"
	"github.com/percona/pmm/version"
)

const (
	shutdownTimeout    = 3 * time.Second
	leaderCheckTimeout = 5 * time.Second
	// pmm-managed serves its HTTP API on this port, see http1Addr in pmm-managed's main.go.
	defaultLeaderCheckURL = "http://127.0.0.1:7772/v1/server/leaderHealthCheck"
	defaultDsnF           = "clickhouse://%s:%s@%s/%s"
	maxIdleConns          = 5
	maxOpenConns          = 10
)

// Variables rather than constants so that tests can shrink them. Only ever read.
var (
	defaultDropOldPartitionInterval = 24 * time.Hour
	// How soon to look again when this node did not apply retention because it is not the
	// leader, or because leadership could not be determined. Short, so that a node promoted
	// mid-cycle starts enforcing retention in minutes rather than a day.
	leaderRecheckInterval = 5 * time.Minute
)

// Outcome of one pass of the retention loop. A rising undetermined or failed count is how an
// operator learns that nothing is deleting anything, which otherwise only shows up much later
// as a full disk.
var mRetentionPasses = prom.NewCounterVec(prom.CounterOpts{
	Namespace: "qan_api2",
	Subsystem: "retention",
	Name:      "passes_total",
	Help:      "Total number of data retention passes by outcome.",
}, []string{"result"})

const (
	retentionApplied      = "applied"
	retentionFailed       = "failed"
	retentionFollower     = "follower"
	retentionUndetermined = "undetermined"
)

// runGRPCServer runs gRPC server until context is canceled, then gracefully stops it.
func runGRPCServer(ctx context.Context, db *sqlx.DB, mbm *models.MetricsBucket, bind string) {
	l := logrus.WithField("component", "gRPC")
	lis, err := net.Listen("tcp", bind)
	if err != nil {
		l.Fatalf("Cannot start gRPC server on: %v", err)
	}
	l.Infof("Starting server on http://%s/ ...", bind)

	rm := models.NewReporter(db)
	mm := models.NewMetrics(db)
	grpcServer := grpc.NewServer(
		// Do not increase that value. If larger requests are required (there are errors in logs),
		// implement request slicing on pmm-managed side:
		// send B/N requests with N buckets in each instead of 1 huge request with B buckets.
		grpc.MaxRecvMsgSize(20*1024*1024), //nolint:mnd

		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			interceptors.Unary,
			grpc_validator.UnaryServerInterceptor(),
		)),
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			interceptors.Stream,
			grpc_validator.StreamServerInterceptor(),
		)),
	)

	aserv := aservice.NewService(db, rm, mm)
	qanv1.RegisterCollectorServiceServer(grpcServer, rservice.NewService(mbm))
	qanv1.RegisterQANServiceServer(grpcServer, aserv)
	reflection.Register(grpcServer)

	if l.Logger.GetLevel() >= logrus.DebugLevel {
		l.Debug("Reflection and channelz are enabled.")
		channelz.RegisterChannelzServiceToServer(grpcServer)

		l.Debug("RPC response latency histogram enabled.")
		grpc_prometheus.EnableHandlingTimeHistogram()
	}

	// run server until it is stopped gracefully or not
	go func() {
		for {
			err = grpcServer.Serve(lis)
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
		grpcServer.Stop()
	}()
	grpcServer.GracefulStop()
	cancel()
}

// runJSONServer runs gRPC-gateway until context is canceled, then gracefully stops it.
func runJSONServer(ctx context.Context, grpcBindF, jsonBindF string) {
	l := logrus.WithField("component", "JSON")
	l.Infof("Starting server on http://%s/ ...", jsonBindF)

	marshaller := &grpc_gateway.JSONPb{
		MarshalOptions: protojson.MarshalOptions{
			UseEnumNumbers:  false,
			EmitUnpopulated: false, // PMM-14566
			UseProtoNames:   true,
			Indent:          "  ",
		},
		UnmarshalOptions: protojson.UnmarshalOptions{
			DiscardUnknown: true,
		},
	}

	proxyMux := grpc_gateway.NewServeMux(
		grpc_gateway.WithIncomingHeaderMatcher(customMatcher),
		grpc_gateway.WithMetadata(gatewayAnnotator),
		grpc_gateway.WithMarshalerOption(grpc_gateway.MIMEWildcard, marshaller),
		grpc_gateway.WithRoutingErrorHandler(pmmerrors.PMMRoutingErrorHandler),
	)
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

	type registrar func(context.Context, *grpc_gateway.ServeMux, string, []grpc.DialOption) error
	for _, r := range []registrar{
		qanv1.RegisterQANServiceHandlerFromEndpoint,
	} {
		err := r(ctx, proxyMux, grpcBindF, opts)
		if err != nil {
			l.Panic(err)
		}
	}

	mux := http.NewServeMux()
	mux.Handle("/", proxyMux)

	server := &http.Server{ //nolint:gosec
		Addr:     jsonBindF,
		ErrorLog: log.New(logrus.StandardLogger().WriterLevel(logrus.ErrorLevel), "runJSONServer: ", 0),
		Handler:  mux,
	}
	go func() {
		err := server.ListenAndServe()
		if !errors.Is(err, http.ErrServerClosed) {
			l.Panic(err)
		}
		l.Println("Server stopped.")
	}()

	<-ctx.Done()
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	err := server.Shutdown(ctx) //nolint:contextcheck
	if err != nil {
		l.Errorf("Failed to shutdown gracefully: %s \n", err)
		server.Close() //nolint:errcheck
	}
	cancel()
}

// runDebugServer runs debug server until context is canceled, then gracefully stops it.
func runDebugServer(ctx context.Context, debugBindF string) {
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
		handlers[i] = "http://" + debugBindF + h
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
	http.HandleFunc("/debug", func(rw http.ResponseWriter, _ *http.Request) {
		rw.Write(buf.Bytes()) //nolint:errcheck
	})
	l.Infof("Starting server on http://%s/debug\nRegistered handlers:\n\t%s", debugBindF, strings.Join(handlers, "\n\t"))

	server := &http.Server{ //nolint:gosec
		Addr:     debugBindF,
		ErrorLog: log.New(logrus.StandardLogger().WriterLevel(logrus.ErrorLevel), "runDebugServer: ", 0),
	}
	go func() {
		err := server.ListenAndServe()
		if !errors.Is(err, http.ErrServerClosed) {
			l.Panic(err)
		}
		l.Info("Server stopped.")
	}()

	<-ctx.Done()
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	err = server.Shutdown(ctx) //nolint:contextcheck
	if err != nil {
		l.Errorf("Failed to shutdown gracefully: %s", err)
	}
	cancel()
}

// runRetentionLoop calls drop for partitions older than the retention period once a day,
// until ctx is canceled.
//
// Only the leader drops (see isLeader). A node that does not looks again shortly after rather
// than waiting out the full interval, because leadership can move at any time.
//
// Nothing here can fail loudly on its own: a node that never applies retention shows up much
// later as a full disk. So every pass records its outcome in mRetentionPasses, and a failed drop
// is retried in minutes rather than waited out for a day.
func runRetentionLoop(ctx context.Context, drop func(context.Context) error, leaderCheckURL string) {
	l := logrus.WithField("component", "retention")
	client := &http.Client{Timeout: leaderCheckTimeout}

	for {
		// Measured from the start of the iteration, so the cadence does not slip by however
		// long the drop took.
		start := time.Now()
		delay := defaultDropOldPartitionInterval
		result := retentionApplied

		leader, err := shouldApplyRetention(ctx, client, leaderCheckURL)
		switch {
		case err != nil:
			// Deleting data on a guess is worse than deleting it a few minutes later. The
			// metric is what escalates; this line is what says why.
			result = retentionUndetermined
			l.Warnf("Not applying data retention, cannot tell whether this node is the leader: %s.", err)
			delay = leaderRecheckInterval
		case !leader:
			result = retentionFollower
			l.Debug("Not applying data retention, this node is not the leader.")
			delay = leaderRecheckInterval
		default:
			err = drop(ctx)
			if err != nil {
				result = retentionFailed
				l.Errorf("Failed to apply data retention, will retry: %s.", err)
				delay = leaderRecheckInterval
			}
		}
		mRetentionPasses.WithLabelValues(result).Inc()

		t := time.NewTimer(time.Until(start.Add(delay)))
		select {
		case <-ctx.Done():
			t.Stop()
			return
		case <-t.C:
		}
	}
}

func main() {
	log.SetFlags(0)

	kingpin.Version(version.ShortInfo())
	kingpin.HelpFlag.Short('h')
	grpcBindF := kingpin.Flag("grpc-bind", "GRPC bind address and port").Default("127.0.0.1:9911").String()
	jsonBindF := kingpin.Flag("json-bind", "JSON bind address and port").Default("127.0.0.1:9922").String()
	debugBindF := kingpin.Flag("listen-debug-addr", "Debug server listen address").Default("127.0.0.1:9933").String()
	dataRetentionF := kingpin.Flag("data-retention", "QAN data Retention (in days)").Default("30").Uint()
	leaderCheckURLF := kingpin.Flag("leader-check-url",
		"URL of the pmm-managed leader health check; data retention is applied only while it reports this node is the leader. Empty disables the check").
		Default(defaultLeaderCheckURL).String()
	dsnF := kingpin.Flag("dsn", "ClickHouse database DSN. Can be overridden with database/host/port options").Default(defaultDsnF).String()
	clickhouseDatabaseF := kingpin.Flag("clickhouse-name", "ClickHouse database name").Default("pmm").Envar("PMM_CLICKHOUSE_DATABASE").String()
	clickhouseAddrF := kingpin.Flag("clickhouse-addr", "ClickHouse database address").Default("127.0.0.1:9000").Envar("PMM_CLICKHOUSE_ADDR").String()
	clickhouseUserF := kingpin.Flag("clickhouse-user", "ClickHouse database user").Default("default").Envar("PMM_CLICKHOUSE_USER").String()
	clickhousePasswordF := kingpin.Flag("clickhouse-password", "ClickHouse database user password").Default("clickhouse").Envar("PMM_CLICKHOUSE_PASSWORD").String()

	clickhouseIsClusterF := kingpin.Flag("clickhouse-cluster", "Is ClickHouse a cluster").Default("false").Envar("PMM_CLICKHOUSE_IS_CLUSTER").Bool()
	clickhouseClusterNameF := kingpin.Flag("clickhouse-cluster-name", "ClickHouse cluster name").Default("").Envar("PMM_CLICKHOUSE_CLUSTER_NAME").String()

	debugF := kingpin.Flag("debug", "Enable debug logging").Bool()
	traceF := kingpin.Flag("trace", "Enable trace logging (implies debug)").Bool()

	kingpin.Parse()

	logger.SetupGlobalLogger()

	logrus.Printf("%s.", version.ShortInfo())
	logrus.Printf("Clickhouse address: %s", *clickhouseAddrF)

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

	var dsn string
	if *dsnF == defaultDsnF {
		dsn = fmt.Sprintf(defaultDsnF, *clickhouseUserF, *clickhousePasswordF, *clickhouseAddrF, *clickhouseDatabaseF)
	} else {
		dsn = *dsnF
	}
	l.Info("DSN: ", dsnutils.RedactDSN(dsn))

	db := NewDB(dsn, maxIdleConns, maxOpenConns, *clickhouseIsClusterF, *clickhouseClusterNameF)
	prom.MustRegister(sqlmetrics.NewCollector("clickhouse", "qan-api2", db.DB))

	// Seeded so that every outcome is a zero series from the start: an increase() over a
	// condition that has never happened must read as zero, not as no data.
	for _, result := range []string{retentionApplied, retentionFailed, retentionFollower, retentionUndetermined} {
		mRetentionPasses.WithLabelValues(result)
	}
	prom.MustRegister(mRetentionPasses)

	// handle termination signals
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, unix.SIGTERM, unix.SIGINT)
	go func() {
		s := <-signals
		signal.Stop(signals)
		l.Infof("Got %s, shutting down...\n", unix.SignalName(s.(unix.Signal))) //nolint:forcetypeassert
		cancel()
	}()

	var wg sync.WaitGroup

	// run ingestion in a separate goroutine
	mbm := models.NewMetricsBucket(db)
	prom.MustRegister(mbm)
	mbmCtx, mbmCancel := context.WithCancel(context.Background())

	wg.Go(func() {
		mbm.Run(mbmCtx)
	})

	wg.Add(1)
	go func() {
		defer func() {
			// stop ingestion only after gRPC server is fully stopped to properly insert the last batch
			mbmCancel()
			wg.Done()
		}()
		runGRPCServer(ctx, db, mbm, *grpcBindF)
	}()

	wg.Go(func() {
		runJSONServer(ctx, *grpcBindF, *jsonBindF)
	})

	wg.Go(func() {
		runDebugServer(ctx, *debugBindF)
	})

	wg.Go(func() {
		runRetentionLoop(ctx, func(ctx context.Context) error {
			return DropOldPartition(ctx, db, *clickhouseDatabaseF, *dataRetentionF)
		}, *leaderCheckURLF)
	})

	wg.Wait()
}

// customMatcher allows to pass custom headers to the backend gRPC server.
func customMatcher(key string) (string, bool) {
	switch key {
	case models.LBACHeaderName:
		return key, true
	default:
		return grpc_gateway.DefaultHeaderMatcher(key)
	}
}

// gatewayAnnotator is used to annotate the gRPC request with metadata from the HTTP request.
func gatewayAnnotator(_ context.Context, req *http.Request) metadata.MD {
	md := metadata.MD{}
	if filters := req.Header.Get(models.LBACHeaderName); filters != "" {
		md.Set(strings.ToLower(models.LBACHeaderName), filters)
		return md
	}
	return nil
}
