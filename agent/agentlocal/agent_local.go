// Copyright 2019 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package agentlocal

import (
	"bytes"
	"context"
	_ "expvar" // register /debug/vars
	"html/template"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof" // register /debug/pprof
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	grpc_gateway "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	channelz "google.golang.org/grpc/channelz/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/percona/pmm/agent/config"
	"github.com/percona/pmm/api/agentlocalpb"
	"github.com/percona/pmm/api/agentpb"
	pmmerrors "github.com/percona/pmm/utils/errors"
	"github.com/percona/pmm/version"
)

const (
	shutdownTimeout = 1 * time.Second
)

// Server represents local pmm-agent API server.
type Server struct {
	cfg            *config.Config
	supervisor     supervisor
	client         client
	configFilepath string

	l               *logrus.Entry
	reload          chan struct{}
	reloadCloseOnce sync.Once

	agentlocalpb.UnimplementedAgentLocalServer
}

// NewServer creates new server.
//
// Caller should call Run.
func NewServer(cfg *config.Config, supervisor supervisor, client client, configFilepath string) *Server {
	return &Server{
		cfg:            cfg,
		supervisor:     supervisor,
		client:         client,
		configFilepath: configFilepath,
		l:              logrus.WithField("component", "local-server"),
		reload:         make(chan struct{}),
	}
}

// Run runs gRPC and JSON servers with API and debug endpoints until ctx is canceled.
//
// Run exits when ctx is canceled, or when a request to reload configuration is received.
func (s *Server) Run(ctx context.Context) {
	defer s.l.Info("Done.")

	serverCtx, serverCancel := context.WithCancel(ctx)

	// Get random free port for gRPC server.
	// If we can't get one, panic since everything is seriously broken.
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		s.l.Panic(err)
	}
	// l is closed by runGRPCServer

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		s.runGRPCServer(serverCtx, l)
	}()
	go func() {
		defer wg.Done()
		s.runJSONServer(serverCtx, l.Addr().String())
	}()

	select {
	case <-ctx.Done():
	case <-s.reload:
	}

	serverCancel()
	wg.Wait()
}

// Status returns current pmm-agent status.
func (s *Server) Status(ctx context.Context, req *agentlocalpb.StatusRequest) (*agentlocalpb.StatusResponse, error) {
	connected := true
	md := s.client.GetServerConnectMetadata()
	if md == nil {
		connected = false
		md = &agentpb.ServerConnectMetadata{}
	}

	var serverInfo *agentlocalpb.ServerInfo
	if u := s.cfg.Server.URL(); u != nil {
		serverInfo = &agentlocalpb.ServerInfo{
			Url:         u.String(),
			InsecureTls: s.cfg.Server.InsecureTLS,
			Connected:   connected,
			Version:     md.ServerVersion,
		}

		if req.GetNetworkInfo && connected {
			latency, clockDrift, err := s.client.GetNetworkInformation()
			if err != nil {
				s.l.Errorf("Can't get network info: %s", err)
			} else {
				serverInfo.Latency = durationpb.New(latency)
				serverInfo.ClockDrift = durationpb.New(clockDrift)
			}
		}
	}

	agentsInfo := s.supervisor.AgentsList()

	return &agentlocalpb.StatusResponse{
		AgentId:        s.cfg.ID,
		RunsOnNodeId:   md.AgentRunsOnNodeID,
		ServerInfo:     serverInfo,
		AgentsInfo:     agentsInfo,
		ConfigFilepath: s.configFilepath,
		AgentVersion:   version.Version,
	}, nil
}

// Reload reloads pmm-agent and it configuration.
func (s *Server) Reload(ctx context.Context, req *agentlocalpb.ReloadRequest) (*agentlocalpb.ReloadResponse, error) {
	// sync errors with setup command

	if _, _, err := config.Get(s.l); err != nil {
		return nil, status.Error(codes.FailedPrecondition, "Failed to reload configuration: "+err.Error())
	}

	s.reloadCloseOnce.Do(func() {
		close(s.reload)
	})

	// client may or may not receive this response due to server shutdown
	return &agentlocalpb.ReloadResponse{}, nil
}

// runGRPCServer runs gRPC server until context is canceled, then gracefully stops it.
func (s *Server) runGRPCServer(ctx context.Context, listener net.Listener) {
	l := s.l.WithField("component", "local-server/gRPC")
	l.Debugf("Starting gRPC server on http://%s/ ...", listener.Addr().String())

	gRPCServer := grpc.NewServer()
	agentlocalpb.RegisterAgentLocalServer(gRPCServer, s)

	if s.cfg.Debug {
		l.Debug("Reflection and channelz are enabled.")
		reflection.Register(gRPCServer)
		channelz.RegisterChannelzServiceToServer(gRPCServer)
	}

	// run server until it is stopped gracefully or not
	go func() {
		var err error
		for {
			err = gRPCServer.Serve(listener) // listener will be closed when this method returns
			if err == nil || errors.Is(err, grpc.ErrServerStopped) {
				break
			}
		}
		if err != nil {
			l.Errorf("Failed to serve: %s", err)
			return
		}
		l.Debug("Server stopped.")
	}()

	<-ctx.Done()

	// try to stop server gracefully, then not
	stopped := make(chan struct{})
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
	go func() {
		<-shutdownCtx.Done()
		gRPCServer.Stop()
		close(stopped)
	}()
	gRPCServer.GracefulStop()
	shutdownCancel()
	<-stopped // wait for Stop() to return
}

// runJSONServer runs JSON proxy server (grpc-gateway) until context is canceled, then gracefully stops it.
func (s *Server) runJSONServer(ctx context.Context, grpcAddress string) {
	address := net.JoinHostPort(s.cfg.ListenAddress, strconv.Itoa(int(s.cfg.ListenPort)))
	l := s.l.WithField("component", "local-server/JSON")
	l.Infof("Starting local API server on http://%s/ ...", address)

	handlers := []string{
		"/debug/metrics",  // by metricsHandler below
		"/debug/vars",     // by expvar
		"/debug/requests", // by golang.org/x/net/trace imported by google.golang.org/grpc
		"/debug/events",   // by golang.org/x/net/trace imported by google.golang.org/grpc
		"/debug/pprof",    // by net/http/pprof
	}
	for i, h := range handlers {
		handlers[i] = "http://" + address + h
	}
	l.Debugf("Debug handlers:\n\t%s", strings.Join(handlers, "\n\t"))

	var debugPage bytes.Buffer
	err := template.Must(template.New("").Parse(`
	<html>
	<body>
	<ul>
	{{ range . }}
		<li><a href="{{ . }}">{{ . }}</a></li>
	{{ end }}
	</ul>
	</body>
	</html>
	`)).Execute(&debugPage, handlers)
	if err != nil {
		l.Panic(err)
	}

	registry := prometheus.NewRegistry()
	registry.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	registry.MustRegister(collectors.NewGoCollector())
	registry.MustRegister(s.client)
	metricsHandler := promhttp.InstrumentMetricHandler(registry, promhttp.HandlerFor(registry, promhttp.HandlerOpts{
		ErrorLog:      l,
		ErrorHandling: promhttp.ContinueOnError,
	}))

	debugPageHandler := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if _, err := rw.Write(debugPage.Bytes()); err != nil {
			l.Warn(err)
		}
	})

	proxyMux := grpc_gateway.NewServeMux(
		grpc_gateway.WithMarshalerOption(grpc_gateway.MIMEWildcard, &grpc_gateway.JSONPb{
			MarshalOptions: protojson.MarshalOptions{
				EmitUnpopulated: true,
				Indent:          "  ",
				UseProtoNames:   true,
			},
			UnmarshalOptions: protojson.UnmarshalOptions{
				DiscardUnknown: true,
			},
		}),
		grpc_gateway.WithErrorHandler(pmmerrors.PMMHTTPErrorHandler))

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	}
	if err := agentlocalpb.RegisterAgentLocalHandlerFromEndpoint(ctx, proxyMux, grpcAddress, opts); err != nil {
		l.Panic(err)
	}

	mux := http.NewServeMux()
	mux.Handle("/debug/metrics", metricsHandler)
	mux.Handle("/debug/", http.DefaultServeMux)
	mux.Handle("/debug", debugPageHandler)
	mux.Handle("/", proxyMux)

	server := &http.Server{
		Addr:     address,
		Handler:  mux,
		ErrorLog: log.New(os.Stderr, "local-server/JSON: ", 0),
	}
	go func() {
		l.Info("Started.")
		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			l.Panic(err)
		}
		l.Info("Stopped.")
	}()

	<-ctx.Done()

	// try to stop server gracefully, then not
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	if err := server.Shutdown(ctx); err != nil {
		l.Errorf("Failed to shutdown gracefully: %s", err)
	}
	cancel()
	_ = server.Close() // call Close() in all cases
}

// check interfaces
var (
	_ agentlocalpb.AgentLocalServer = (*Server)(nil)
)
