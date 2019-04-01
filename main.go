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

package main

import (
	"bytes"
	"context"
	"crypto/tls"
	_ "expvar" // register /debug/vars
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof" // register /debug/pprof
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/percona/pmm/api/agentlocalpb"
	"github.com/percona/pmm/api/agentpb"
	"github.com/percona/pmm/version"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
	"google.golang.org/grpc"
	channelz "google.golang.org/grpc/channelz/service"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/reflection"

	"github.com/percona/pmm-agent/agentlocal"
	"github.com/percona/pmm-agent/agents/supervisor"
	"github.com/percona/pmm-agent/config"
	"github.com/percona/pmm-agent/server"
	"github.com/percona/pmm-agent/utils/logger"
)

const (
	dialTimeout       = 10 * time.Second
	backoffMaxDelay   = 10 * time.Second
	clockDriftWarning = 5 * time.Second
	shutdownTimeout   = 3 * time.Second
	grpcAddr          = "127.0.0.1:7776"
)

func handleChanges(cancel context.CancelFunc, s *supervisor.Supervisor, channel *server.Channel, l *logrus.Entry) {
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		for state := range s.Changes() {
			res := channel.SendRequest(&agentpb.AgentMessage_StateChanged{
				StateChanged: &state,
			})
			if res == nil {
				l.Warn("Failed to send StateChanged request.")
			}
		}
		l.Info("Supervisor changes done.")
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for collect := range s.QANRequests() {
			res := channel.SendRequest(&agentpb.AgentMessage_QanCollect{
				QanCollect: &collect,
			})
			if res == nil {
				l.Warn("Failed to send QanCollect request.")
			}
		}
		l.Info("Supervisor QAN requests done.")
	}()

	wg.Wait()
	cancel()
}

func handleRequests(s *supervisor.Supervisor, channel *server.Channel, l *logrus.Entry) {
	for serverMessage := range channel.Requests() {
		var agentMessage *agentpb.AgentMessage
		switch payload := serverMessage.Payload.(type) {
		case *agentpb.ServerMessage_Ping:
			agentMessage = &agentpb.AgentMessage{
				Id: serverMessage.Id,
				Payload: &agentpb.AgentMessage_Pong{
					Pong: &agentpb.Pong{
						CurrentTime: ptypes.TimestampNow(),
					},
				},
			}

		case *agentpb.ServerMessage_SetState:
			s.SetState(payload.SetState)

			agentMessage = &agentpb.AgentMessage{
				Id: serverMessage.Id,
				Payload: &agentpb.AgentMessage_SetState{
					SetState: new(agentpb.SetStateResponse),
				},
			}

		default:
			l.Panicf("Unhandled server message payload: %s.", payload)
		}

		channel.SendResponse(agentMessage)
	}
}

func workLoop(ctx context.Context, cfg *config.Config, l *logrus.Entry, client agentpb.AgentClient, localSrv *agentlocal.AgentLocalServer) {
	// use separate context for stream to cancel it after supervisor is done sending last changes
	streamCtx, streamCancel := context.WithCancel(context.Background())
	streamCtx = agentpb.AddAgentConnectMetadata(streamCtx, &agentpb.AgentConnectMetadata{
		ID:      cfg.ID,
		Version: version.Version,
	})

	l.Info("Establishing two-way communication channel ...")
	stream, err := client.Connect(streamCtx)
	if err != nil {
		l.Errorf("Failed to establish two-way communication channel: %s.", err)
		streamCancel()
		return
	}

	channel := server.NewChannel(stream)
	prometheus.MustRegister(channel)
	defer func() {
		err = channel.Wait()
		switch err {
		case nil:
			l.Info("Two-way communication channel closed.")
		default:
			l.Errorf("Two-way communication channel closed: %s", err)
		}
	}()

	// So far nginx can handle all that itself without pmm-managed.
	// We need to send ping to ensure that pmm-managed is alive and that Agent ID is valid.
	start := time.Now()
	res := channel.SendRequest(&agentpb.AgentMessage_Ping{
		Ping: new(agentpb.Ping),
	})
	if res == nil {
		// error will be logged by channel code
		streamCancel()
		return
	}
	roundtrip := time.Since(start)
	serverTime, err := ptypes.Timestamp(res.(*agentpb.ServerMessage_Pong).Pong.CurrentTime)
	if err != nil {
		l.Errorf("Failed to decode Pong.current_time: %s.", err)
		streamCancel()
		return
	}
	l.Infof("Two-way communication channel established in %s.", roundtrip)
	clockDrift := serverTime.Sub(start) - roundtrip/2
	if clockDrift > clockDriftWarning || -clockDrift > clockDriftWarning {
		l.Warnf("Estimated clock drift: %s.", clockDrift)
	}

	md, err := agentpb.GetAgentServerMetadata(stream)
	if err != nil {
		l.Warnf("Can't get metadata from server: %v", err)
	}
	localSrv.SetMetadata(&md)

	s := supervisor.NewSupervisor(ctx, &cfg.Paths, &cfg.Ports)
	go handleChanges(streamCancel, s, channel, l)
	handleRequests(s, channel, l)
}

// runGRPCServer runs gRPC server until context is canceled, then gracefully stops it.
func runGRPCServer(ctx context.Context, cfg *config.Config, localSrv agentlocalpb.AgentLocalServer, gRPCAddr string) {
	l := logrus.WithField("component", "gRPC")
	l.Infof("Starting server on http://%s/ ...", gRPCAddr)

	gRPCServer := grpc.NewServer()
	agentlocalpb.RegisterAgentLocalServer(gRPCServer, localSrv)

	if cfg.Debug {
		l.Debug("Reflection and channelz are enabled.")
		reflection.Register(gRPCServer)
		channelz.RegisterChannelzServiceToServer(gRPCServer)
	}

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

// runJSONServer runs JSON proxy server (grpc-gateway) until context is canceled, then gracefully stops it.
func runJSONServer(ctx context.Context, cfg *config.Config, gRPCAddr string) {
	jsonAddr := fmt.Sprintf("127.0.0.1:%d", cfg.ListenPort)

	l := logrus.WithField("component", "JSON")
	l.Infof("Starting server on http://%s/ ...", jsonAddr)

	proxyMux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithInsecure()}

	if err := agentlocalpb.RegisterAgentLocalHandlerFromEndpoint(ctx, proxyMux, gRPCAddr, opts); err != nil {
		l.Panic(err)
	}

	handlers := []string{
		"/debug/vars",     // by expvar
		"/debug/requests", // by golang.org/x/net/trace imported by google.golang.org/grpc
		"/debug/events",   // by golang.org/x/net/trace imported by google.golang.org/grpc
		"/debug/pprof",    // by net/http/pprof
	}
	for i, h := range handlers {
		handlers[i] = "http://" + jsonAddr + h
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

	http.Handle("/", proxyMux)
	http.HandleFunc("/debug", func(rw http.ResponseWriter, req *http.Request) {
		if _, err := rw.Write(buf.Bytes()); err != nil {
			l.Warn(err)
		}
	})

	l.Infof("Starting server on http://%s/debug\nRegistered handlers:\n\t%s", jsonAddr, strings.Join(handlers, "\n\t"))

	server := &http.Server{
		Addr:     jsonAddr,
		ErrorLog: log.New(os.Stderr, "runJSONServer: ", 0),
		Handler:  http.DefaultServeMux,
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

func main() {
	// empty version breaks much of pmm-managed logic
	if version.Version == "" {
		panic("pmm-agent version is not set during build.")
	}

	cfg, err := config.Get(os.Args[1:], logrus.WithField("component", "config"))
	if err != nil {
		logrus.Fatal(err)
	}
	logrus.Debugf("Loaded configuration: %+v", cfg)

	if cfg.Debug {
		logrus.SetLevel(logrus.DebugLevel)
	}
	if cfg.Trace {
		logrus.SetLevel(logrus.TraceLevel)
		logrus.SetReportCaller(true)
		grpclog.SetLoggerV2(&logger.GRPC{Entry: logrus.WithField("component", "grpclog")})
	}

	ctx, cancel := context.WithCancel(context.Background())

	// handle termination signals
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, unix.SIGTERM, unix.SIGINT)
	go func() {
		s := <-signals
		signal.Stop(signals)
		logrus.Warnf("Got %s, shutting down...", unix.SignalName(s.(unix.Signal)))
		cancel()
	}()

	if cfg.Address == "" {
		logrus.Error("PMM Server address is not provided, halting.")
		<-ctx.Done()
		return
	}

	host, _, _ := net.SplitHostPort(cfg.Address)
	tlsConfig := &tls.Config{
		ServerName:         host,
		InsecureSkipVerify: cfg.InsecureTLS, //nolint:gosec
	}
	opts := []grpc.DialOption{
		grpc.WithBlock(),
		grpc.WithBackoffMaxDelay(backoffMaxDelay),
		grpc.WithUserAgent("pmm-agent/" + version.Version),
		grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)),
	}

	l := logrus.WithField("component", "connection")
	l.Infof("Connecting to %s ...", cfg.Address)
	dialCtx, dialCancel := context.WithTimeout(ctx, dialTimeout)
	conn, err := grpc.DialContext(dialCtx, cfg.Address, opts...)
	dialCancel()
	if err != nil {
		l.Fatalf("Failed to connect to %s: %s.", cfg.Address, err)
	}
	defer func() {
		err := conn.Close()
		switch err {
		case nil:
			l.Info("Connection closed.")
		default:
			l.Errorf("Connection closed: %s.", err)
		}
	}()

	l.Infof("Connected to %s.", cfg.Address)
	client := agentpb.NewAgentClient(conn)

	// TODO
	// if cfg.UUID == "" {
	// 	logrus.Info("Registering pmm-agent ...")
	// 	resp, err := client.Register(ctx, &api.RegisterRequest{})
	// 	if err != nil {
	// 		logrus.Fatalf("Failed to register pmm-agent: %s.", err)
	// 	}
	// 	cfg.UUID = resp.Uuid
	// 	logrus.Infof("pmm-agent registered: %s.", cfg.UUID)
	// }

	localServer := agentlocal.NewAgentLocalServer(cfg)

	go runGRPCServer(ctx, cfg, localServer, grpcAddr)
	go runJSONServer(ctx, cfg, grpcAddr)

	workLoop(ctx, cfg, l, client, localServer)
}
