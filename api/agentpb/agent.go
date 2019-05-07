package agentpb

import (
	"github.com/golang/protobuf/proto"
)

//go-sumtype:decl isAgentMessage_Payload
//go-sumtype:decl isServerMessage_Payload

// code below uses the same order as definitions in agent.proto

//go-sumtype:decl AgentRequestPayload
//go-sumtype:decl AgentResponsePayload
//go-sumtype:decl ServerResponsePayload
//go-sumtype:decl ServerRequestPayload

// AgentRequestPayload represents agent's request payload.
type AgentRequestPayload interface {
	AgentMessageRequestPayload() isAgentMessage_Payload
	sealed()
}

// AgentResponsePayload represents agent's response payload.
type AgentResponsePayload interface {
	AgentMessageResponsePayload() isAgentMessage_Payload
	sealed()
}

// ServerResponsePayload represents server's response payload.
type ServerResponsePayload interface {
	ServerMessageResponsePayload() isServerMessage_Payload
	sealed()
}

// ServerRequestPayload represents server's request payload.
type ServerRequestPayload interface {
	ServerMessageRequestPayload() isServerMessage_Payload
	sealed()
}

// AgentMessage request payloads
func (m *Ping) AgentMessageRequestPayload() isAgentMessage_Payload {
	return &AgentMessage_Ping{Ping: m}
}
func (m *StateChangedRequest) AgentMessageRequestPayload() isAgentMessage_Payload {
	return &AgentMessage_StateChanged{StateChanged: m}
}
func (m *QANCollectRequest) AgentMessageRequestPayload() isAgentMessage_Payload {
	return &AgentMessage_QanCollect{QanCollect: m}
}
func (m *ActionResult) AgentMessageRequestPayload() isAgentMessage_Payload {
	return &AgentMessage_ActionResult{ActionResult: m}
}

// AgentMessage response payloads
func (m *Pong) AgentMessageResponsePayload() isAgentMessage_Payload {
	return &AgentMessage_Pong{Pong: m}
}
func (m *SetStateResponse) AgentMessageResponsePayload() isAgentMessage_Payload {
	return &AgentMessage_SetState{SetState: m}
}
func (m *ActionRunResponse) AgentMessageResponsePayload() isAgentMessage_Payload {
	return &AgentMessage_ActionRunResponse{ActionRunResponse: m}
}
func (m *ActionCancelResponse) AgentMessageResponsePayload() isAgentMessage_Payload {
	return &AgentMessage_ActionCancelResponse{ActionCancelResponse: m}
}

// ServerMessage response payloads
func (m *Pong) ServerMessageResponsePayload() isServerMessage_Payload {
	return &ServerMessage_Pong{Pong: m}
}
func (m *StateChangedResponse) ServerMessageResponsePayload() isServerMessage_Payload {
	return &ServerMessage_StateChanged{StateChanged: m}
}
func (m *QANCollectResponse) ServerMessageResponsePayload() isServerMessage_Payload {
	return &ServerMessage_QanCollect{QanCollect: m}
}

// ServerMessage request payloads
func (m *Ping) ServerMessageRequestPayload() isServerMessage_Payload {
	return &ServerMessage_Ping{Ping: m}
}
func (m *SetStateRequest) ServerMessageRequestPayload() isServerMessage_Payload {
	return &ServerMessage_SetState{SetState: m}
}
func (m *ActionRunRequest) ServerMessageRequestPayload() isServerMessage_Payload {
	return &ServerMessage_ActionRunRequest{ActionRunRequest: m}
}
func (m *ActionCancelRequest) ServerMessageRequestPayload() isServerMessage_Payload {
	return &ServerMessage_ActionCancelRequest{ActionCancelRequest: m}
}

func (*Ping) sealed()                   {}
func (m *StateChangedRequest) sealed()  {}
func (m *QANCollectRequest) sealed()    {}
func (*Pong) sealed()                   {}
func (m *SetStateResponse) sealed()     {}
func (m *StateChangedResponse) sealed() {}
func (m *QANCollectResponse) sealed()   {}
func (m *SetStateRequest) sealed()      {}
func (m *ActionRunRequest) sealed()     {}
func (m *ActionCancelRequest) sealed()  {}
func (m *ActionRunResponse) sealed()    {}
func (m *ActionCancelResponse) sealed() {}
func (m *ActionResult) sealed()         {}

// check interfaces
var (
	// AgentMessage request payloads
	_ AgentRequestPayload = (*Ping)(nil)
	_ AgentRequestPayload = (*StateChangedRequest)(nil)
	_ AgentRequestPayload = (*QANCollectRequest)(nil)
	_ AgentRequestPayload = (*ActionResult)(nil)

	// AgentMessage response payloads
	_ AgentResponsePayload = (*Pong)(nil)
	_ AgentResponsePayload = (*SetStateResponse)(nil)
	_ AgentResponsePayload = (*ActionRunResponse)(nil)
	_ AgentResponsePayload = (*ActionRunResponse)(nil)

	// ServerMessage response payloads
	_ ServerResponsePayload = (*Pong)(nil)
	_ ServerResponsePayload = (*StateChangedResponse)(nil)
	_ ServerResponsePayload = (*QANCollectResponse)(nil)

	// ServerMessage request payloads
	_ ServerRequestPayload = (*Ping)(nil)
	_ ServerRequestPayload = (*SetStateRequest)(nil)
	_ ServerRequestPayload = (*ActionRunRequest)(nil)
	_ ServerRequestPayload = (*ActionCancelRequest)(nil)
)

//go-sumtype:decl AgentParams

// AgentParams is a common interface for AgentProcess and BuiltinAgent parameters.
type AgentParams interface {
	proto.Message
	sealedAgentParams() //nolint:unused
}

func (*SetStateRequest_AgentProcess) sealedAgentParams() {}
func (*SetStateRequest_BuiltinAgent) sealedAgentParams() {}
