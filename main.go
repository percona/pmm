package main

import (
	"context"
	"crypto/tls"
	"os"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/percona/pmm/api/agent"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/percona/pmm-agent/config"
)

var (
	Version = "2.0.0-dev"
)

const (
	dialTimeout     = 10 * time.Second
	backoffMaxDelay = 10 * time.Second
)

func workLoop(ctx context.Context, cfg *config.Config, client agent.AgentClient) {
	l := logrus.WithField("component", "conn")

	l.Info("Establishing two-way communication channel ...")
	stream, err := client.Connect(ctx)
	if err != nil {
		l.Fatal(err)
	}
	l.Info("Two-way communication channel established.")
	defer stream.CloseSend()
	var id uint32 = 1

	// connect request/response
	agentMessage := &agent.AgentMessage{
		Id: id,
		Payload: &agent.AgentMessage_Auth{
			Auth: &agent.AuthRequest{
				Uuid:    cfg.UUID,
				Version: Version,
			},
		},
	}
	l.Debugf("Send: %s.", agentMessage)
	if err = stream.Send(agentMessage); err != nil {
		l.Fatal(err)
	}
	serverMessage, err := stream.Recv()
	if err != nil {
		l.Fatal(err)
	}
	l.Debugf("Recv: %s.", serverMessage)

	for {
		serverMessage, err = stream.Recv()
		if err != nil {
			l.Fatal(err)
		}
		l.Debugf("Recv: %s.", serverMessage)

		agentMessage = nil
		switch payload := serverMessage.Payload.(type) {
		case *agent.ServerMessage_Ping:
			agentMessage = &agent.AgentMessage{
				Id: serverMessage.Id,
				Payload: &agent.AgentMessage_Ping{
					Ping: &agent.PingResponse{
						CurrentTime: ptypes.TimestampNow(),
					},
				},
			}

		case *agent.ServerMessage_State:
			for _, agent := range payload.State.AgentProcesses {
				switch agent.Type {
				default:
					l.Warnf("Got unhandled agent type %s (%d), ignoring.", agent.Type, agent.Type)
				}
				// l.Infof("Starting mysqld_exporter on 127.0.0.1:%d ...", exporter.ListenPort)
			}

			agentMessage = &agent.AgentMessage{
				Id: serverMessage.Id,
				Payload: &agent.AgentMessage_State{
					State: &agent.SetStateResponse{},
				},
			}

		default:
			l.Warn("Unexpected server message type.")
		}

		if agentMessage != nil {
			l.Debugf("Send: %s.", agentMessage)
			if err = stream.Send(agentMessage); err != nil {
				l.Errorf("Failed to send message: %s.", err)
				return
			}
		}
	}
}

func main() {
	var cfg config.Config
	app := config.Application(&cfg, Version)
	kingpin.MustParse(app.Parse(os.Args[1:]))

	logrus.Infof("Loaded configuration: %+v.", cfg)

	if cfg.Debug {
		logrus.SetLevel(logrus.DebugLevel)
		// grpclog.SetLoggerV2(&logger.GRPC{Entry: logrus.WithField("component", "gRPC")})
		logrus.Debug("Debug logging enabled.")
	}

	// TODO add signal handling, etc
	ctx := context.TODO()

	if cfg.Address == "" {
		logrus.Error("PMM Server address is not provided, halting.")
		<-ctx.Done()
		return
	}

	opts := []grpc.DialOption{
		grpc.WithWaitForHandshake(), // TODO check if we need it
		grpc.WithBlock(),            // TODO check if we need it
		grpc.WithBackoffMaxDelay(backoffMaxDelay),
		grpc.WithUserAgent("pmm-agent/" + Version),
	}
	if cfg.WithoutNginx {
		opts = append(opts, grpc.WithInsecure())
	} else {
		creds := credentials.NewTLS(&tls.Config{})
		opts = append(opts, grpc.WithTransportCredentials(creds))
	}

	logrus.Infof("Connecting to %s ...", cfg.Address)
	dialCtx, cancel := context.WithTimeout(ctx, dialTimeout)
	conn, err := grpc.DialContext(dialCtx, cfg.Address, opts...)
	cancel()
	if err != nil {
		logrus.Fatalf("Failed to connect to %s: %s.", cfg.Address, err)
	}
	logrus.Infof("Connected to %s.", cfg.Address)
	client := agent.NewAgentClient(conn)

	if cfg.UUID == "" {
		logrus.Info("Registering pmm-agent ...")
		resp, err := client.Register(ctx, &agent.RegisterRequest{})
		if err != nil {
			logrus.Fatalf("Failed to register pmm-agent: %s.", err)
		}
		cfg.UUID = resp.Uuid
		logrus.Infof("pmm-agent registered: %s.", cfg.UUID)
	}

	workLoop(ctx, &cfg, client)
}
