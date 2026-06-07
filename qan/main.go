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
	"html/template"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof" //nolint:gosec
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/alecthomas/kingpin/v2"
	grpc_gateway "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/encoding/protojson"

	qanv1 "github.com/percona/pmm/api/qan/v1"
	"github.com/percona/pmm/qan/models"
	aservice "github.com/percona/pmm/qan/services/analytics"
	rservice "github.com/percona/pmm/qan/services/receiver"
	"github.com/percona/pmm/utils/logger"
	"github.com/percona/pmm/version"
)

const (
	shutdownTimeout = 3 * time.Second
	maxRecvMsgSize  = 20 * 1024 * 1024
)

// runGRPCServer runs the gRPC server until the context is canceled.
func runGRPCServer(ctx context.Context, bind string, collector qanv1.CollectorServiceServer, analytics qanv1.QANServiceServer) {
	l := logrus.WithField("component", "gRPC")
	lis, err := net.Listen("tcp", bind)
	if err != nil {
		l.Fatalf("Cannot start gRPC server: %v", err)
	}
	l.Infof("Starting gRPC server on %s ...", bind)

	grpcServer := grpc.NewServer(grpc.MaxRecvMsgSize(maxRecvMsgSize))
	qanv1.RegisterCollectorServiceServer(grpcServer, collector)
	qanv1.RegisterQANServiceServer(grpcServer, analytics)
	reflection.Register(grpcServer)

	go func() {
		err := grpcServer.Serve(lis)
		if err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			l.Errorf("Failed to serve: %v", err)
		}
	}()

	<-ctx.Done()
	grpcServer.GracefulStop()
	l.Info("Server stopped.")
}

// runJSONServer runs the gRPC-gateway JSON proxy until the context is canceled.
func runJSONServer(ctx context.Context, grpcBind, jsonBind string) {
	l := logrus.WithField("component", "JSON")
	l.Infof("Starting JSON server on %s ...", jsonBind)

	mux := grpc_gateway.NewServeMux(grpc_gateway.WithMarshalerOption(grpc_gateway.MIMEWildcard, &grpc_gateway.JSONPb{
		MarshalOptions:   protojson.MarshalOptions{UseProtoNames: true, EmitUnpopulated: false},
		UnmarshalOptions: protojson.UnmarshalOptions{DiscardUnknown: true},
	}))
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	err := qanv1.RegisterQANServiceHandlerFromEndpoint(ctx, mux, grpcBind, opts)
	if err != nil {
		l.Panic(err)
	}

	server := &http.Server{ //nolint:gosec
		Addr:     jsonBind,
		Handler:  mux,
		ErrorLog: log.New(logrus.StandardLogger().WriterLevel(logrus.ErrorLevel), "runJSONServer: ", 0),
	}
	go func() {
		err := server.ListenAndServe()
		if !errors.Is(err, http.ErrServerClosed) {
			l.Panic(err)
		}
	}()

	<-ctx.Done()
	stopCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	err = server.Shutdown(stopCtx) //nolint:contextcheck
	if err != nil {
		l.Errorf("Failed to shutdown gracefully: %v", err)
	}
}

// runDebugServer runs the Prometheus/pprof debug server until the context is canceled.
func runDebugServer(ctx context.Context, bind string) {
	l := logrus.WithField("component", "debug")

	handler := promhttp.HandlerFor(prom.DefaultGatherer, promhttp.HandlerOpts{
		ErrorLog:      l,
		ErrorHandling: promhttp.ContinueOnError,
	})
	http.Handle("/debug/metrics", promhttp.InstrumentMetricHandler(prom.DefaultRegisterer, handler))

	handlers := []string{"/debug/metrics", "/debug/vars", "/debug/pprof"}
	for i, h := range handlers {
		handlers[i] = "http://" + bind + h
	}
	var buf bytes.Buffer
	err := template.Must(template.New("debug").Parse(
		`<html><body><ul>{{ range . }}<li><a href="{{ . }}">{{ . }}</a></li>{{ end }}</ul></body></html>`,
	)).Execute(&buf, handlers)
	if err != nil {
		l.Panic(err)
	}
	http.HandleFunc("/debug", func(rw http.ResponseWriter, _ *http.Request) {
		rw.Write(buf.Bytes()) //nolint:errcheck
	})
	l.Infof("Starting debug server on http://%s/debug ...", bind)

	server := &http.Server{ //nolint:gosec
		Addr:     bind,
		ErrorLog: log.New(logrus.StandardLogger().WriterLevel(logrus.ErrorLevel), "runDebugServer: ", 0),
	}
	go func() {
		err := server.ListenAndServe()
		if !errors.Is(err, http.ErrServerClosed) {
			l.Panic(err)
		}
	}()

	<-ctx.Done()
	stopCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	err = server.Shutdown(stopCtx) //nolint:contextcheck
	if err != nil {
		l.Errorf("Failed to shutdown gracefully: %v", err)
	}
}

func main() {
	log.SetFlags(0)

	kingpin.Version(version.ShortInfo())
	kingpin.HelpFlag.Short('h')
	grpcBindF := kingpin.Flag("grpc-bind", "gRPC bind address and port").Default("127.0.0.1:9911").String()
	jsonBindF := kingpin.Flag("json-bind", "JSON (gRPC-gateway) bind address and port").Default("127.0.0.1:9922").String()
	debugBindF := kingpin.Flag("listen-debug-addr", "Debug server listen address").Default("127.0.0.1:9933").String()
	clickhouseDatabaseF := kingpin.Flag("clickhouse-name", "ClickHouse database name").Default("pmm").Envar("PMM_CLICKHOUSE_DATABASE").String()
	clickhouseAddrF := kingpin.Flag("clickhouse-addr", "ClickHouse database address").Default("127.0.0.1:9000").Envar("PMM_CLICKHOUSE_ADDR").String()
	clickhouseUserF := kingpin.Flag("clickhouse-user", "ClickHouse database user").Default("default").Envar("PMM_CLICKHOUSE_USER").String()
	clickhousePasswordF := kingpin.Flag("clickhouse-password", "ClickHouse database user password").Default("clickhouse").Envar("PMM_CLICKHOUSE_PASSWORD").String()
	clickhousePoolSizeF := kingpin.Flag("clickhouse-pool-size", "ClickHouse connection pool size").Default("10").Int()
	dataRetentionF := kingpin.Flag("data-retention", "QAN data retention (in days)").Default("30").Uint()
	debugF := kingpin.Flag("debug", "Enable debug logging").Bool()
	traceF := kingpin.Flag("trace", "Enable trace logging (implies debug)").Bool()
	kingpin.Parse()

	logger.SetupGlobalLogger()
	logrus.Printf("%s.", version.ShortInfo())
	if *debugF {
		logrus.SetLevel(logrus.DebugLevel)
	}
	if *traceF {
		logrus.SetLevel(logrus.TraceLevel)
		logrus.SetReportCaller(true)
	}

	l := logrus.WithField("component", "main")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer l.Info("Done.")

	conn := NewDB(*clickhouseAddrF, *clickhouseDatabaseF, *clickhouseUserF, *clickhousePasswordF, *clickhousePoolSizeF, *clickhousePoolSizeF, *dataRetentionF)
	defer conn.Close() //nolint:errcheck

	collector := rservice.NewService(models.NewIngestor(conn))
	analyticsSvc := aservice.NewService(conn)

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, unix.SIGTERM, unix.SIGINT)
	go func() {
		s := <-signals
		signal.Stop(signals)
		l.Infof("Got %s, shutting down...", unix.SignalName(s.(unix.Signal))) //nolint:forcetypeassert
		cancel()
	}()

	var wg sync.WaitGroup
	wg.Go(func() {
		runGRPCServer(ctx, *grpcBindF, collector, analyticsSvc)
	})
	wg.Go(func() {
		runJSONServer(ctx, *grpcBindF, *jsonBindF)
	})
	wg.Go(func() {
		runDebugServer(ctx, *debugBindF)
	})
	wg.Wait()
}
