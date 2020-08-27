// Package agentpb contains pmm-agent<->pmm-managed protocol messages and helpers.
package agentpb

import (
	"github.com/golang/protobuf/proto"
)

//go-sumtype:decl isAgentMessage_Payload
//go-sumtype:decl isServerMessage_Payload

//go-sumtype:decl AgentRequestPayload
//go-sumtype:decl AgentResponsePayload
//go-sumtype:decl ServerResponsePayload
//go-sumtype:decl ServerRequestPayload

//go-sumtype:decl isStartActionRequest_Params

// code below uses the same order as payload types at AgentMessage / ServerMessage

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
func (m *ActionResultRequest) AgentMessageRequestPayload() isAgentMessage_Payload {
	return &AgentMessage_ActionResult{ActionResult: m}
}

// AgentMessage response payloads
func (m *Pong) AgentMessageResponsePayload() isAgentMessage_Payload {
	return &AgentMessage_Pong{Pong: m}
}
func (m *SetStateResponse) AgentMessageResponsePayload() isAgentMessage_Payload {
	return &AgentMessage_SetState{SetState: m}
}
func (m *StartActionResponse) AgentMessageResponsePayload() isAgentMessage_Payload {
	return &AgentMessage_StartAction{StartAction: m}
}
func (m *StopActionResponse) AgentMessageResponsePayload() isAgentMessage_Payload {
	return &AgentMessage_StopAction{StopAction: m}
}
func (m *CheckConnectionResponse) AgentMessageResponsePayload() isAgentMessage_Payload {
	return &AgentMessage_CheckConnection{CheckConnection: m}
}
func (m *DownloadFileChunkResponse) AgentMessageResponsePayload() isAgentMessage_Payload {
	return &AgentMessage_DownloadFileChunk{DownloadFileChunk: m}
}
func (m *DeleteFileResponse) AgentMessageResponsePayload() isAgentMessage_Payload {
	return &AgentMessage_DeleteFile{DeleteFile: m}
}
func (m *StartJobResponse) AgentMessageResponsePayload() isAgentMessage_Payload {
	return &AgentMessage_StartJob{StartJob: m}
}
func (m *StopJobResponse) AgentMessageResponsePayload() isAgentMessage_Payload {
	return &AgentMessage_StopJob{StopJob: m}
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
func (m *ActionResultResponse) ServerMessageResponsePayload() isServerMessage_Payload {
	return &ServerMessage_ActionResult{ActionResult: m}
}

// ServerMessage request payloads
func (m *Ping) ServerMessageRequestPayload() isServerMessage_Payload {
	return &ServerMessage_Ping{Ping: m}
}
func (m *SetStateRequest) ServerMessageRequestPayload() isServerMessage_Payload {
	return &ServerMessage_SetState{SetState: m}
}
func (m *StartActionRequest) ServerMessageRequestPayload() isServerMessage_Payload {
	return &ServerMessage_StartAction{StartAction: m}
}
func (m *StopActionRequest) ServerMessageRequestPayload() isServerMessage_Payload {
	return &ServerMessage_StopAction{StopAction: m}
}
func (m *CheckConnectionRequest) ServerMessageRequestPayload() isServerMessage_Payload {
	return &ServerMessage_CheckConnection{CheckConnection: m}
}
func (m *DownloadFileChunkRequest) ServerMessageRequestPayload() isServerMessage_Payload {
	return &ServerMessage_DownloadFileChunk{DownloadFileChunk: m}
}
func (m *DeleteFileRequest) ServerMessageRequestPayload() isServerMessage_Payload {
	return &ServerMessage_DeleteFile{DeleteFile: m}
}
func (m *StartJobRequest) ServerMessageRequestPayload() isServerMessage_Payload {
	return &ServerMessage_StartJob{StartJob: m}
}
func (m *StopJobRequest) ServerMessageRequestPayload() isServerMessage_Payload {
	return &ServerMessage_StopJob{StopJob: m}
}

// in alphabetical order
func (*ActionResultRequest) sealed()       {}
func (*ActionResultResponse) sealed()      {}
func (*CheckConnectionRequest) sealed()    {}
func (*CheckConnectionResponse) sealed()   {}
func (*DeleteFileRequest) sealed()         {}
func (*DeleteFileResponse) sealed()        {}
func (*DownloadFileChunkRequest) sealed()  {}
func (*DownloadFileChunkResponse) sealed() {}
func (*Ping) sealed()                      {}
func (*Pong) sealed()                      {}
func (*QANCollectRequest) sealed()         {}
func (*QANCollectResponse) sealed()        {}
func (*SetStateRequest) sealed()           {}
func (*SetStateResponse) sealed()          {}
func (*StartActionRequest) sealed()        {}
func (*StartActionResponse) sealed()       {}
func (*StartJobRequest) sealed()           {}
func (*StartJobResponse) sealed()          {}
func (*StateChangedRequest) sealed()       {}
func (*StateChangedResponse) sealed()      {}
func (*StopActionRequest) sealed()         {}
func (*StopActionResponse) sealed()        {}
func (*StopJobRequest) sealed()            {}
func (*StopJobResponse) sealed()           {}

// check interfaces
var (
	// AgentMessage request payloads
	_ AgentRequestPayload = (*Ping)(nil)
	_ AgentRequestPayload = (*StateChangedRequest)(nil)
	_ AgentRequestPayload = (*QANCollectRequest)(nil)
	_ AgentRequestPayload = (*ActionResultRequest)(nil)

	// AgentMessage response payloads
	_ AgentResponsePayload = (*Pong)(nil)
	_ AgentResponsePayload = (*SetStateResponse)(nil)
	_ AgentResponsePayload = (*StartActionResponse)(nil)
	_ AgentResponsePayload = (*StopActionResponse)(nil)
	_ AgentResponsePayload = (*CheckConnectionResponse)(nil)
	_ AgentResponsePayload = (*DownloadFileChunkResponse)(nil)
	_ AgentResponsePayload = (*DeleteFileResponse)(nil)
	_ AgentResponsePayload = (*StartJobResponse)(nil)
	_ AgentResponsePayload = (*StopJobResponse)(nil)

	// ServerMessage response payloads
	_ ServerResponsePayload = (*Pong)(nil)
	_ ServerResponsePayload = (*StateChangedResponse)(nil)
	_ ServerResponsePayload = (*QANCollectResponse)(nil)
	_ ServerResponsePayload = (*ActionResultResponse)(nil)

	// ServerMessage request payloads
	_ ServerRequestPayload = (*Ping)(nil)
	_ ServerRequestPayload = (*SetStateRequest)(nil)
	_ ServerRequestPayload = (*StartActionRequest)(nil)
	_ ServerRequestPayload = (*StopActionRequest)(nil)
	_ ServerRequestPayload = (*CheckConnectionRequest)(nil)
	_ ServerRequestPayload = (*DownloadFileChunkRequest)(nil)
	_ ServerRequestPayload = (*DeleteFileRequest)(nil)
	_ ServerRequestPayload = (*StartJobRequest)(nil)
	_ ServerRequestPayload = (*StopJobRequest)(nil)
)

//go-sumtype:decl AgentParams

// AgentParams is a common interface for AgentProcess and BuiltinAgent parameters.
type AgentParams interface {
	proto.Message
	sealedAgentParams() //nolint:unused
}

func (*SetStateRequest_AgentProcess) sealedAgentParams() {}
func (*SetStateRequest_BuiltinAgent) sealedAgentParams() {}
