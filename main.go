// qan-api2
// Copyright (C) 2019 Percona LLC
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
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/jmoiron/sqlx"
	"github.com/percona/pmm/api/qanpb"
	"github.com/percona/pmm/version"
	"golang.org/x/sys/unix"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/percona/qan-api2/models"
	aservice "github.com/percona/qan-api2/services/analytics"
	rservice "github.com/percona/qan-api2/services/receiver"
)

const shutdownTimeout = 3 * time.Second
const responseTimeout = 1 * time.Minute

// runGRPCServer runs gRPC server until context is canceled, then gracefully stops it.
func runGRPCServer(ctx context.Context, db *sqlx.DB, bind string) {
	lis, err := net.Listen("tcp", bind)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	mbm := models.NewMetricsBucket(db)
	rm := models.NewReporter(db)
	mm := models.NewMetrics(db)
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(
			func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
				newCtx, cancel := context.WithTimeout(ctx, responseTimeout)
				defer cancel()
				return handler(newCtx, req)
			},
		),
	)
	aserv := aservice.NewService(rm, mm)
	qanpb.RegisterCollectorServer(grpcServer, rservice.NewService(mbm))
	qanpb.RegisterProfileServer(grpcServer, aserv)
	qanpb.RegisterObjectDetailsServer(grpcServer, aserv)
	qanpb.RegisterMetricsNamesServer(grpcServer, aserv)
	qanpb.RegisterFiltersServer(grpcServer, aserv)
	reflection.Register(grpcServer)
	log.Printf("QAN-API gRPC serve: %v\n", bind)

	go func() {
		for {
			err = grpcServer.Serve(lis)
			if err == nil || err == grpc.ErrServerStopped {
				break
			}
			log.Printf("Failed to serve: %v", err)
		}
		log.Println("Server stopped.")
	}()

	<-ctx.Done()
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	go func() {
		<-ctx.Done()
		grpcServer.Stop()
	}()
	cancel()
}

// runJSONServer runs gRPC-gateway until context is canceled, then gracefully stops it.
func runJSONServer(ctx context.Context, grpcBind, jsonBind string) {
	log.Printf("Starting server on http://%s/ ...", jsonBind)

	proxyMux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithInsecure()}

	type registrar func(context.Context, *runtime.ServeMux, string, []grpc.DialOption) error
	for _, r := range []registrar{
		qanpb.RegisterObjectDetailsHandlerFromEndpoint,
		qanpb.RegisterProfileHandlerFromEndpoint,
		qanpb.RegisterMetricsNamesHandlerFromEndpoint,
		qanpb.RegisterFiltersHandlerFromEndpoint,
	} {
		if err := r(ctx, proxyMux, grpcBind, opts); err != nil {
			log.Panic(err)
		}
	}

	mux := http.NewServeMux()
	mux.Handle("/", proxyMux)

	server := &http.Server{
		Addr:     jsonBind,
		ErrorLog: log.New(os.Stderr, "runJSONServer: ", 0),
		Handler:  mux,
	}
	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Panic(err)
		}
		log.Println("Server stopped.")
	}()

	<-ctx.Done()
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Failed to shutdown gracefully: %v \n", err)
		server.Close()
	}
	cancel()
}

func main() {
	kingpin.Version(version.ShortInfo())
	grpcBind := kingpin.Flag("grpc-bind", "GRPC bind address and port").Envar("QANAPI_GRPC_BIND").Default("127.0.0.1:9911").String()
	jsonBind := kingpin.Flag("json-bind", "JSON bind address and port").Envar("QANAPI_JSON_BIND").Default("127.0.0.1:9922").String()
	dataRetention := kingpin.Flag("data-retention", "QAN data Retention (in days)").Envar("QANAPI_DATA_RETENTION").Default("30").Uint()
	dsn := kingpin.Flag("dsn", "ClickHouse database DSN").Envar("QANAPI_DSN").Default("clickhouse://127.0.0.1:9000?database=pmm&debug=true").String()
	kingpin.Parse()

	log.Printf("%s.", version.ShortInfo())

	db := NewDB(*dsn)

	ctx, cancel := context.WithCancel(context.Background())
	// handle termination signals
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, unix.SIGTERM, unix.SIGINT)
	go func() {
		s := <-signals
		signal.Stop(signals)
		log.Printf("Got %s, shutting down...\n", unix.SignalName(s.(unix.Signal)))
		cancel()
	}()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		runGRPCServer(ctx, db, *grpcBind)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		runJSONServer(ctx, *grpcBind, *jsonBind)
	}()

	ticker := time.NewTicker(24 * time.Hour)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			// Drop old partitions once in 24h.
			DropOldPartition(db, *dataRetention)
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
