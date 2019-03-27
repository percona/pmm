package agentpb

import (
	"context"

	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Workaround for https://github.com/golang/protobuf/issues/261.
// Useful for helper functions.
type (
	ServerMessagePayload = isServerMessage_Payload
	AgentMessagePayload  = isAgentMessage_Payload
)

type RequestPayload interface{ request() }
type ResponsePayload interface{ response() }

func (*Ping) request()                {}
func (*QANCollectRequest) request()   {}
func (*StateChangedRequest) request() {}
func (*SetStateRequest) request()     {}

func (*Pong) response()                 {}
func (*QANCollectResponse) response()   {}
func (*StateChangedResponse) response() {}
func (*SetStateResponse) response()     {}

// AgentParams is a common interface for AgentProcess and BuiltinAgent parameters.
type AgentParams interface {
	proto.Message
	agentParams()
}

func (*SetStateRequest_AgentProcess) agentParams() {}
func (*SetStateRequest_BuiltinAgent) agentParams() {}

const (
	mdAgentID       = "pmm-agent-id"
	mdAgentVersion  = "pmm-agent-version"
	mdAgentNodeID   = "pmm-agent-node-id"
	mdServerVersion = "pmm-server-version"
)

// AgentConnectMetadata represents metadata sent by pmm-agent with Connect RPC method.
type AgentConnectMetadata struct {
	ID      string
	Version string
}

// AgentServerMetadata represents metadata sent by pmm-managed to pmm-agent.
type AgentServerMetadata struct {
	AgentRunsOnNodeID string
	ServerVersion     string
}

func getValue(md metadata.MD, key string) string {
	vs := md.Get(key)
	if len(vs) == 1 {
		return vs[0]
	}
	return ""
}

// AddAgentConnectMetadata adds metadata to pmm-agent's Connect RPC call.
// Used by pmm-agent.
func AddAgentConnectMetadata(ctx context.Context, md *AgentConnectMetadata) context.Context {
	return metadata.AppendToOutgoingContext(ctx, mdAgentID, md.ID, mdAgentVersion, md.Version)
}

// GetAgentConnectMetadata returns pmm-agent's metadata.
// Used by pmm-managed.
func GetAgentConnectMetadata(ctx context.Context) AgentConnectMetadata {
	var res AgentConnectMetadata
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		res.ID = getValue(md, mdAgentID)
		res.Version = getValue(md, mdAgentVersion)
	}
	return res
}

// SendAgentServerMetadata sends metadata to pmm-agent.
// Used by pmm-managed.
func SendAgentServerMetadata(stream grpc.ServerStream, md AgentServerMetadata) error {
	header := metadata.Pairs(
		mdAgentNodeID, md.AgentRunsOnNodeID,
		mdServerVersion, md.ServerVersion,
	)
	if err := stream.SendHeader(header); err != nil {
		return status.Errorf(codes.Internal, "Can't send server metadata to client.")
	}

	return nil
}

// GetAgentServerMetadata receives metadata from pmm-managed.
// Used by pmm-agent.
func GetAgentServerMetadata(stream grpc.ClientStream) (AgentServerMetadata, error) {
	var res AgentServerMetadata
	md, err := stream.Header()
	if err != nil {
		return res, err
	}

	res.AgentRunsOnNodeID = getValue(md, mdAgentNodeID)
	res.ServerVersion = getValue(md, mdServerVersion)
	return res, nil
}
