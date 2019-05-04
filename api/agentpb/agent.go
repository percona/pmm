package agentpb

import (
	"github.com/golang/protobuf/proto"
)

//go-sumtype:decl isServerMessage_Payload
//go-sumtype:decl isAgentMessage_Payload

// Workaround for https://github.com/golang/protobuf/issues/261.
// Useful for helper functions.
// TODO Refactor code to use
// ServerRequestPayload, ServerResponsePayload, AgentRequestPayload, AgentResponsePayload instead.
type (
	ServerMessagePayload = isServerMessage_Payload
	AgentMessagePayload  = isAgentMessage_Payload
)

//go-sumtype:decl ServerRequestPayload
//go-sumtype:decl ServerResponsePayload
//go-sumtype:decl AgentRequestPayload
//go-sumtype:decl AgentResponsePayload

type ServerRequestPayload interface {
	server()
	request()
}

type ServerResponsePayload interface {
	server()
	response()
}

type AgentRequestPayload interface {
	agent()
	request()
}

type AgentResponsePayload interface {
	agent()
	response()
}

// code below uses the same order as definitions in agent.proto (except ping/pong)

func (*Ping) agent()    {}
func (*Ping) server()   {}
func (*Ping) request()  {}
func (*Pong) agent()    {}
func (*Pong) server()   {}
func (*Pong) response() {}

// AgentMessage request payloads
func (*StateChangedRequest) agent()   {}
func (*StateChangedRequest) request() {}
func (*QANCollectRequest) agent()     {}
func (*QANCollectRequest) request()   {}

// AgentMessage response payloads
func (*SetStateResponse) agent()    {}
func (*SetStateResponse) response() {}

// ServerMessage response payloads
func (*StateChangedResponse) server()   {}
func (*StateChangedResponse) response() {}
func (*QANCollectResponse) server()     {}
func (*QANCollectResponse) response()   {}

// ServerMessage request payloads
func (*SetStateRequest) server()  {}
func (*SetStateRequest) request() {}

//go-sumtype:decl AgentParams

// AgentParams is a common interface for AgentProcess and BuiltinAgent parameters.
type AgentParams interface {
	proto.Message
	agentParams()
}

func (*SetStateRequest_AgentProcess) agentParams() {}
func (*SetStateRequest_BuiltinAgent) agentParams() {}
