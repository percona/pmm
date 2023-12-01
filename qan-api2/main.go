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

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_validator "github.com/grpc-ecosystem/go-grpc-middleware/validator"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	grpc_gateway "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
	"google.golang.org/grpc"
	channelz "google.golang.org/grpc/channelz/service"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/encoding/protojson"
	"gopkg.in/alecthomas/kingpin.v2"

	qanpb "github.com/percona/pmm/api/qanpb"
	"github.com/percona/pmm/qan-api2/models"
	aservice "github.com/percona/pmm/qan-api2/services/analytics"
	rservice "github.com/percona/pmm/qan-api2/services/receiver"
	"github.com/percona/pmm/qan-api2/utils/interceptors"
	"github.com/percona/pmm/utils/logger"
	"github.com/percona/pmm/utils/sqlmetrics"
	"github.com/percona/pmm/version"
)

const (
	shutdownTimeout = 3 * time.Second
	defaultDsnF     = "clickhouse://%s?database=%s&block_size=%s&pool_size=%s"
	maxIdleConns    = 5
	maxOpenConns    = 10
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
		grpc.MaxRecvMsgSize(20*1024*1024), //nolint:gomnd

		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			interceptors.Unary,
			grpc_validator.UnaryServerInterceptor())),
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			interceptors.Stream,
			grpc_validator.StreamServerInterceptor())),
	)

	aserv := aservice.NewService(rm, mm)
	qanpb.RegisterCollectorServer(grpcServer, rservice.NewService(mbm))
	qanpb.RegisterProfileServer(grpcServer, aserv)
	qanpb.RegisterObjectDetailsServer(grpcServer, aserv)
	qanpb.RegisterMetricsNamesServer(grpcServer, aserv)
	qanpb.RegisterFiltersServer(grpcServer, aserv)
	reflection.Register(grpcServer)

	if l.Logger.GetLevel() >= logrus.DebugLevel {
		l.Debug("Reflection and channelz are enabled.")
		reflection.Register(grpcServer)
		channelz.RegisterChannelzServiceToServer(grpcServer)

		l.Debug("RPC response latency histogram enabled.")
		grpc_prometheus.EnableHandlingTimeHistogram()
	}
	grpc_prometheus.Register(grpcServer)

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
		MarshalOptions: protojson.MarshalOptions{ //nolint:exhaustivestruct
			UseEnumNumbers:  false,
			EmitUnpopulated: false,
			UseProtoNames:   true,
			Indent:          "  ",
		},
		UnmarshalOptions: protojson.UnmarshalOptions{ //nolint:exhaustivestruct
			DiscardUnknown: true,
		},
	}

	proxyMux := grpc_gateway.NewServeMux(
		grpc_gateway.WithMarshalerOption(grpc_gateway.MIMEWildcard, marshaller),
	)
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

	type registrar func(context.Context, *grpc_gateway.ServeMux, string, []grpc.DialOption) error
	for _, r := range []registrar{
		qanpb.RegisterObjectDetailsHandlerFromEndpoint,
		qanpb.RegisterProfileHandlerFromEndpoint,
		qanpb.RegisterMetricsNamesHandlerFromEndpoint,
		qanpb.RegisterFiltersHandlerFromEndpoint,
	} {
		if err := r(ctx, proxyMux, grpcBindF, opts); err != nil {
			l.Panic(err)
		}
	}

	mux := http.NewServeMux()
	mux.Handle("/", proxyMux)

	server := &http.Server{ //nolint:gosec
		Addr:     jsonBindF,
		ErrorLog: log.New(os.Stderr, "runJSONServer: ", 0),
		Handler:  mux,
	}
	go func() {
		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			l.Panic(err)
		}
		l.Println("Server stopped.")
	}()

	<-ctx.Done()
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	if err := server.Shutdown(ctx); err != nil { //nolint:contextcheck
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
	http.HandleFunc("/debug", func(rw http.ResponseWriter, req *http.Request) {
		rw.Write(buf.Bytes()) //nolint:errcheck
	})
	l.Infof("Starting server on http://%s/debug\nRegistered handlers:\n\t%s", debugBindF, strings.Join(handlers, "\n\t"))

	server := &http.Server{ //nolint:gosec
		Addr:     debugBindF,
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

func main() {
	log.SetFlags(0)
	log.SetPrefix("stdlog: ")

	kingpin.Version(version.ShortInfo())
	kingpin.HelpFlag.Short('h')
	grpcBindF := kingpin.Flag("grpc-bind", "GRPC bind address and port").Default("127.0.0.1:9911").String()
	jsonBindF := kingpin.Flag("json-bind", "JSON bind address and port").Default("127.0.0.1:9922").String()
	debugBindF := kingpin.Flag("listen-debug-addr", "Debug server listen address").Default("127.0.0.1:9933").String()
	dataRetentionF := kingpin.Flag("data-retention", "QAN data Retention (in days)").Default("30").Uint()
	dsnF := kingpin.Flag("dsn", "ClickHouse database DSN. Can be override with database/host/port options").Default(defaultDsnF).String()
	clickHouseDatabaseF := kingpin.Flag("clickhouse-name", "Clickhouse database name").Default("pmm").Envar("PERCONA_TEST_PMM_CLICKHOUSE_DATABASE").String()
	clickhouseAddrF := kingpin.Flag("clickhouse-addr", "Clickhouse database address").Default("127.0.0.1:9000").Envar("PERCONA_TEST_PMM_CLICKHOUSE_ADDR").String()
	clickhouseBlockSizeF := kingpin.Flag("clickhouse-block-size", "Number of rows that can be load from table in one cycle").
		Default("10000").Envar("PERCONA_TEST_PMM_CLICKHOUSE_BLOCK_SIZE").String()
	clickhousePoolSizeF := kingpin.Flag("clickhouse-pool-size", "Controls how much queries can be run simultaneously").
		Default("2").Envar("PERCONA_TEST_PMM_CLICKHOUSE_POOL_SIZE").String()

	debugF := kingpin.Flag("debug", "Enable debug logging").Bool()
	traceF := kingpin.Flag("trace", "Enable trace logging (implies debug)").Bool()

	kingpin.Parse()

	log.Printf("%s.", version.ShortInfo())

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

	var dsn string
	if *dsnF == defaultDsnF {
		dsn = fmt.Sprintf(defaultDsnF, *clickhouseAddrF, *clickHouseDatabaseF, *clickhouseBlockSizeF, *clickhousePoolSizeF)
	} else {
		dsn = *dsnF
	}

	l.Info("DSN: ", dsn)
	db := NewDB(dsn, maxIdleConns, maxOpenConns)

	prom.MustRegister(sqlmetrics.NewCollector("clickhouse", "qan-api2", db.DB))

	// handle termination signals
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, unix.SIGTERM, unix.SIGINT)
	go func() {
		s := <-signals
		signal.Stop(signals)
		log.Printf("Got %s, shutting down...\n", unix.SignalName(s.(unix.Signal))) //nolint:forcetypeassert
		cancel()
	}()

	var wg sync.WaitGroup

	// run ingestion in a separate goroutine
	mbm := models.NewMetricsBucket(db)
	prom.MustRegister(mbm)
	mbmCtx, mbmCancel := context.WithCancel(context.Background())
	wg.Add(1)
	go func() {
		defer wg.Done()
		mbm.Run(mbmCtx)
	}()

	wg.Add(1)
	go func() {
		defer func() {
			// stop ingestion only after gRPC server is fully stopped to properly insert the last batch
			mbmCancel()
			wg.Done()
		}()
		runGRPCServer(ctx, db, mbm, *grpcBindF)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		runJSONServer(ctx, *grpcBindF, *jsonBindF)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		runDebugServer(ctx, *debugBindF)
	}()

	ticker := time.NewTicker(24 * time.Hour)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			// Drop old partitions once in 24h.
			DropOldPartition(db, *clickHouseDatabaseF, *dataRetentionF)
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// nothing
			}
		}
	}()

	wg.Wait()
}
