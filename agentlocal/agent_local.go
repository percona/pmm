// pmm-agent
// Copyright (C) 2018 Percona LLC
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

package agentlocal

import (
	"bytes"
	"context"
	_ "expvar" // register /debug/vars
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof" // register /debug/pprof
	"os"
	"strings"
	"sync"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/percona/pmm/api/agentlocalpb"
	"github.com/percona/pmm/api/agentpb"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	channelz "google.golang.org/grpc/channelz/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"

	"github.com/percona/pmm-agent/config"
)

const (
	shutdownTimeout = 1 * time.Second
)

// ErrReload is returned from Service.Run after request to reload configuration.
var ErrReload = errors.New("reload")

// Server represents local pmm-agent API server.
type Server struct {
	cfg            *config.Config
	supervisor     supervisor
	client         client
	configFilePath string

	l               *logrus.Entry
	reload          chan struct{}
	reloadCloseOnce sync.Once
}

// NewServer creates new server.
//
// Caller should call Run.
func NewServer(cfg *config.Config, supervisor supervisor, client client, configFilePath string) *Server {
	return &Server{
		cfg:            cfg,
		supervisor:     supervisor,
		client:         client,
		configFilePath: configFilePath,
		l:              logrus.WithField("component", "local-server"),
		reload:         make(chan struct{}),
	}
}

// Run runs gRPC and JSON servers with API and debug endpoints until ctx is canceled.
//
// Run exits when ctx is canceled, or when a request to reload configuration is received.
// In the latter case, the returned error is ErrReload.
func (s *Server) Run(ctx context.Context) error {
	defer s.l.Info("Done.")

	serverCtx, serverCancel := context.WithCancel(ctx)

	// Get random free port for gRPC server.
	// If we can't get once, panic since everything is seriously broken.
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		s.l.Panic(err)
	}
	address := l.Addr().String()
	if err = l.Close(); err != nil {
		s.l.Panic(err)
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		s.runGRPCServer(serverCtx, address)
	}()
	go func() {
		defer wg.Done()
		s.runJSONServer(serverCtx, address)
	}()

	var res error
	select {
	case <-ctx.Done():
		res = ctx.Err()
	case <-s.reload:
		res = ErrReload
	}
	serverCancel()
	wg.Wait()
	return res
}

// Status returns current pmm-agent status.
func (s *Server) Status(ctx context.Context, req *agentlocalpb.StatusRequest) (*agentlocalpb.StatusResponse, error) {
	connected := true
	md := s.client.GetAgentServerMetadata()
	if md == nil {
		connected = false
		md = new(agentpb.AgentServerMetadata)
	}

	var serverInfo *agentlocalpb.ServerInfo
	if u := s.cfg.Server.URL(); u != nil {
		serverInfo = &agentlocalpb.ServerInfo{
			Url:          u.String(),
			InsecureTls:  s.cfg.Server.InsecureTLS,
			Version:      md.ServerVersion,
			LastPingTime: nil, // TODO https://jira.percona.com/browse/PMM-3758
			Latency:      nil, // TODO https://jira.percona.com/browse/PMM-3758
			Connected:    connected,
		}
	}

	agentsInfo := s.supervisor.AgentsList()

	return &agentlocalpb.StatusResponse{
		AgentId:        s.cfg.ID,
		RunsOnNodeId:   md.AgentRunsOnNodeID,
		ServerInfo:     serverInfo,
		AgentsInfo:     agentsInfo,
		ConfigFilePath: s.configFilePath,
	}, nil
}

// Reload reloads pmm-agent and it configuration.
func (s *Server) Reload(ctx context.Context, req *agentlocalpb.ReloadRequest) (*agentlocalpb.ReloadResponse, error) {
	// sync errors with setup command

	_, _, err := config.Get(s.l)
	if err != nil {
		return nil, status.Error(codes.FailedPrecondition, "Failed to reload configuration: "+err.Error())
	}

	s.reloadCloseOnce.Do(func() {
		close(s.reload)
	})

	// client may or may not receive this response due to server shutdown
	return new(agentlocalpb.ReloadResponse), nil
}

// runGRPCServer runs gRPC server until context is canceled, then gracefully stops it.
func (s *Server) runGRPCServer(ctx context.Context, address string) {
	l := s.l.WithField("component", "local-server/gRPC")
	l.Debugf("Starting gRPC server on http://%s/ ...", address)

	gRPCServer := grpc.NewServer()
	agentlocalpb.RegisterAgentLocalServer(gRPCServer, s)

	if s.cfg.Debug {
		l.Debug("Reflection and channelz are enabled.")
		reflection.Register(gRPCServer)
		channelz.RegisterChannelzServiceToServer(gRPCServer)
	}

	// run server until it is stopped gracefully or not
	listener, err := net.Listen("tcp", address)
	if err != nil {
		l.Panic(err) // we can't recover from that
	}
	go func() {
		for {
			err = gRPCServer.Serve(listener)
			if err == nil || err == grpc.ErrServerStopped {
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
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	go func() {
		<-ctx.Done()
		gRPCServer.Stop()
	}()
	gRPCServer.GracefulStop()
	cancel()
}

// runJSONServer runs JSON proxy server (grpc-gateway) until context is canceled, then gracefully stops it.
func (s *Server) runJSONServer(ctx context.Context, grpcAddress string) {
	address := fmt.Sprintf("127.0.0.1:%d", s.cfg.ListenPort)
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
	registry.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	registry.MustRegister(prometheus.NewGoCollector())
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

	proxyMux := runtime.NewServeMux()
	opts := []grpc.DialOption{
		grpc.WithInsecure(),
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
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			l.Panic(err)
		}
		l.Info("Stopped.")
	}()

	<-ctx.Done()
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	if err := server.Shutdown(ctx); err != nil {
		l.Errorf("Failed to shutdown gracefully: %s", err)
		_ = server.Close()
	}
	cancel()
}

// check interfaces
var (
	_ agentlocalpb.AgentLocalServer = (*Server)(nil)
)
