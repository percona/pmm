package agent

import (
	"context"

	"google.golang.org/grpc/metadata"
)

// Workaround for https://github.com/golang/protobuf/issues/261.
// Useful for helper functions.
type (
	ServerMessagePayload = isServerMessage_Payload
	AgentMessagePayload  = isAgentMessage_Payload
)

const (
	mdUUID    = "pmm-agent-uuid"
	mdVersion = "pmm-agent-version"
)

// AgentConnectMetadata represents metadata sent by pmm-agent with Connect RPC method.
type AgentConnectMetadata struct {
	UUID    string
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
func AddAgentConnectMetadata(ctx context.Context, md *AgentConnectMetadata) context.Context {
	return metadata.AppendToOutgoingContext(ctx, mdUUID, md.UUID, mdVersion, md.Version)
}

// GetAgentConnectMetadata returns pmm-agent's metadata.
func GetAgentConnectMetadata(stream Agent_ConnectServer) *AgentConnectMetadata {
	res := new(AgentConnectMetadata)
	md, ok := metadata.FromIncomingContext(stream.Context())
	if ok {
		res.UUID = getValue(md, mdUUID)
		res.Version = getValue(md, mdVersion)
	}
	return res
}
