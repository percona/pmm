package agentpb

import (
	"context"

	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc/metadata"
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
	mdID      = "pmm-agent-id"
	mdVersion = "pmm-agent-version"
)

// AgentConnectMetadata represents metadata sent by pmm-agent with Connect RPC method.
type AgentConnectMetadata struct {
	ID      string
	Version string
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
	return metadata.AppendToOutgoingContext(ctx, mdID, md.ID, mdVersion, md.Version)
}

// GetAgentConnectMetadata returns pmm-agent's metadata.
// Used by pmm-managed.
func GetAgentConnectMetadata(ctx context.Context) AgentConnectMetadata {
	var res AgentConnectMetadata
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		res.ID = getValue(md, mdID)
		res.Version = getValue(md, mdVersion)
	}
	return res
}
