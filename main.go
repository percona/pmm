// qan-api
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

	"github.com/Percona-Lab/qan-api/models"
	aservice "github.com/Percona-Lab/qan-api/services/analytics"
	rservice "github.com/Percona-Lab/qan-api/services/receiver"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	pbqan "github.com/percona/pmm/api/qan"
	"golang.org/x/sys/unix"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const shutdownTimeout = 3 * time.Second

// runGRPCServer runs gRPC server until context is canceled, then gracefully stops it.
func runGRPCServer(ctx context.Context, dsn, bind string) {

	db, err := NewDB(dsn)
	if err != nil {
		log.Fatal("DB error", err)
	}

	lis, err := net.Listen("tcp", bind)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	mbm := models.NewMetricsBucket(db)
	rm := models.NewReporter(db)
	mm := models.NewMetrics(db)
	grpcServer := grpc.NewServer()
	pbqan.RegisterAgentServer(grpcServer, rservice.NewService(mbm))
	pbqan.RegisterProfileServer(grpcServer, aservice.NewService(rm, mm))
	pbqan.RegisterMetricsServer(grpcServer, aservice.NewService(rm, mm))
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

func addSwaggerHandler(mux *http.ServeMux) {
	// TODO embed swagger resources?
	pattern := "/swagger/"
	fileServer := http.StripPrefix(pattern, http.FileServer(http.Dir("api/swagger")))
	mux.HandleFunc(pattern, func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Set("Access-Control-Allow-Origin", "*")
		fileServer.ServeHTTP(rw, req)
	})
}

// runJSONServer runs gRPC-gateway until context is canceled, then gracefully stops it.
func runJSONServer(ctx context.Context, grpcBind, jsonBind string) {

	log.Printf("Starting server on http://%s/ ...", jsonBind)

	proxyMux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithInsecure()}

	type registrar func(context.Context, *runtime.ServeMux, string, []grpc.DialOption) error
	for _, r := range []registrar{
		pbqan.RegisterMetricsHandlerFromEndpoint,
		pbqan.RegisterProfileHandlerFromEndpoint,
	} {
		if err := r(ctx, proxyMux, grpcBind, opts); err != nil {
			log.Panic(err)
		}
	}

	mux := http.NewServeMux()
	swaggerBind, ok := os.LookupEnv("QANAPI_SWAGGER_BIND")
	if !ok {
		log.Printf("Swagger enabled. http://%s/swagger/\n", swaggerBind)
		addSwaggerHandler(mux)
	}

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
	}
	cancel()
}

func main() {
	grpcBind, ok := os.LookupEnv("QANAPI_BIND")
	if !ok {
		grpcBind = "127.0.0.1:9911"
	}
	jsonBind, ok := os.LookupEnv("QANAPI_JSON_BIND")
	if !ok {
		jsonBind = "127.0.0.1:9922"
	}
	dsn, ok := os.LookupEnv("QANAPI_DSN")
	if !ok {
		dsn = "clickhouse://127.0.0.1:9000?database=pmm&debug=true"
	}

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
		runGRPCServer(ctx, dsn, grpcBind)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		runJSONServer(ctx, grpcBind, jsonBind)
	}()
	wg.Wait()

}
