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
	_ "expvar"
	"flag"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"

	"github.com/Percona-Lab/pmm-managed/api"
	"github.com/Percona-Lab/pmm-managed/handlers"
	"github.com/Percona-Lab/pmm-managed/service"
)

var (
	// TODO combine gRPC and REST ports?
	gRPCAddrF  = flag.String("listen-grpc-addr", "127.0.0.1:7771", "gRPC server listen address")
	restAddrF  = flag.String("listen-rest-addr", "127.0.0.1:7772", "REST server listen address")
	debugAddrF = flag.String("listen-debug-addr", "127.0.0.1:7773", "Debug server listen address")

	// certFileF = flag.String("cert-file", "cert.pem", "TLS certificate file for gRPC server")
	// keyFileF  = flag.String("key-file", "key.pem", "TLS key file for gRPC server")

	prometheusConfigF = flag.String("prometheus-config", "", "Prometheus configuration file path")
	prometheusURLF    = flag.String("prometheus-url", "http://127.0.0.1:9090/", "Prometheus base URL")
)

// TODO graceful shutdown
func runGRPCServer(ctx context.Context) {
	logrus.Infof("Starting gRPC server on http://%s/ ...", *gRPCAddrF)

	prometheusURL, err := url.Parse(*prometheusURLF)
	if err != nil {
		logrus.Panic(err)
	}
	gRPCServer := grpc.NewServer()
	server := &handlers.Server{
		Prometheus: &service.Prometheus{
			ConfigPath: *prometheusConfigF,
			URL:        prometheusURL,
		},
	}
	api.RegisterBaseServer(gRPCServer, server)
	api.RegisterAlertsServer(gRPCServer, server)

	l, err := net.Listen("tcp", *gRPCAddrF)
	if err != nil {
		logrus.Fatal(err)
	}
	go func() {
		if err = gRPCServer.Serve(l); err != grpc.ErrServerStopped {
			logrus.Panic(err)
		}
		logrus.Info("gRPC server stopped.")
	}()

	<-ctx.Done()
	gRPCServer.GracefulStop()
}

func runRESTServer(ctx context.Context) {
	logrus.Infof("Starting REST server on http://%s/ ...", *restAddrF)

	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithInsecure()}

	type registrar func(context.Context, *runtime.ServeMux, string, []grpc.DialOption) error
	for _, r := range []registrar{
		api.RegisterBaseHandlerFromEndpoint,
		api.RegisterAlertsHandlerFromEndpoint,
	} {
		err := r(ctx, mux, *gRPCAddrF, opts)
		if err != nil {
			logrus.Fatal(err)
		}
	}
	server := &http.Server{
		Addr:    *restAddrF,
		Handler: mux,
		// TLSConfig: &tls.Config{
		// 	Certificates: []tls.Certificate{*cert},
		// 	// TODO set RootCAs ?
		// 	// NextProtos: []string{"h2"},
		// 	// TODO set ServerName ?
		// 	InsecureSkipVerify: true,
		// },
		// TODO add timeouts
		ErrorLog: log.New(os.Stderr, "runRESTServer: ", 0),
	}
	go func() {
		err := server.ListenAndServe()
		if err != http.ErrServerClosed {
			logrus.Panic(err)
		}
		logrus.Info("REST server stopped.")
	}()
	<-ctx.Done()
	server.Shutdown(context.TODO())
}

// TODO graceful shutdown
func runDebugServer(ctx context.Context) {
	msg := `Starting debug server ...
            pprof    http://%s/debug/pprof
            expvar   http://%s/debug/vars
            requests http://%s/debug/requests
            events   http://%s/debug/events`
	logrus.Infof(msg, *debugAddrF, *debugAddrF, *debugAddrF, *debugAddrF)

	logger := log.New(os.Stderr, "runDebugServer: ", 0)
	server := &http.Server{
		Addr: *debugAddrF,
		// TODO add timeouts
		ErrorLog: logger,
	}
	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			logger.Panic(err)
		}
		logrus.Info("Debug server stopped.")
	}()

	<-ctx.Done()
	server.Shutdown(context.TODO())
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("stdlog: ")
	logrus.SetLevel(logrus.DebugLevel)
	grpclog.SetLogger(logrus.WithField("component", "grpclog"))
	flag.Parse()

	// cert, err := tls.LoadX509KeyPair(*certFileF, *keyFileF)
	// if err != nil {
	// 	logrus.Fatal(err)
	// }

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer logrus.Info("Done.")

	// handle termination signals: first one gracefully, force exit on the second one
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		s := <-signals
		logrus.Warnf("Got %v (%d) signal, shutting down...", s, s)
		cancel()

		s = <-signals
		logrus.Fatalf("Got %v (%d) signal, exiting!", s, s)
	}()

	// start servers, wait for them to exit
	var wg sync.WaitGroup
	wg.Add(3)
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
	wg.Wait()
}
