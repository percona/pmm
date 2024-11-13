// Copyright (C) 2023 Percona LLC
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

// Package agentpb contains pmm-agent<->pmm-managed protocol messages and helpers.
package agentpb

import "google.golang.org/protobuf/proto"

//go-sumtype:decl isAgentMessage_Payload
//go-sumtype:decl isServerMessage_Payload

//go-sumtype:decl AgentRequestPayload
//go-sumtype:decl AgentResponsePayload
//go-sumtype:decl ServerResponsePayload
//go-sumtype:decl ServerRequestPayload

//go-sumtype:decl isStartActionRequest_Params

// Code below uses the same order as payload types at AgentMessage / ServerMessage.

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

// A list of AgentMessage request payloads.

// AgentMessageRequestPayload returns the payload for the AgentMessageRequest.
func (m *Ping) AgentMessageRequestPayload() isAgentMessage_Payload { //nolint:ireturn
	return &AgentMessage_Ping{Ping: m}
}

// AgentMessageRequestPayload returns the payload for the AgentMessageRequest.
func (m *StateChangedRequest) AgentMessageRequestPayload() isAgentMessage_Payload { //nolint:ireturn
	return &AgentMessage_StateChanged{StateChanged: m}
}

// AgentMessageRequestPayload returns the payload for the AgentMessageRequest.
func (m *QANCollectRequest) AgentMessageRequestPayload() isAgentMessage_Payload { //nolint:ireturn
	return &AgentMessage_QanCollect{QanCollect: m}
}

// AgentMessageRequestPayload returns the payload for the AgentMessageRequest.
func (m *ActionResultRequest) AgentMessageRequestPayload() isAgentMessage_Payload { //nolint:ireturn
	return &AgentMessage_ActionResult{ActionResult: m}
}

// AgentMessageRequestPayload returns the payload for the AgentMessageRequest.
func (m *JobProgress) AgentMessageRequestPayload() isAgentMessage_Payload { //nolint:ireturn
	return &AgentMessage_JobProgress{JobProgress: m}
}

// AgentMessageRequestPayload returns the payload for the AgentMessageRequest.
func (m *JobResult) AgentMessageRequestPayload() isAgentMessage_Payload { //nolint:ireturn
	return &AgentMessage_JobResult{JobResult: m}
}

// A list of AgentMessage response payloads.

// AgentMessageResponsePayload returns the payload for the AgentMessageResponse.
func (m *Pong) AgentMessageResponsePayload() isAgentMessage_Payload { //nolint:ireturn
	return &AgentMessage_Pong{Pong: m}
}

// AgentMessageResponsePayload returns the payload for the AgentMessageResponse.
func (m *SetStateResponse) AgentMessageResponsePayload() isAgentMessage_Payload { //nolint:ireturn
	return &AgentMessage_SetState{SetState: m}
}

// AgentMessageResponsePayload returns the payload for the AgentMessageResponse.
func (m *StartActionResponse) AgentMessageResponsePayload() isAgentMessage_Payload { //nolint:ireturn
	return &AgentMessage_StartAction{StartAction: m}
}

// AgentMessageResponsePayload returns the payload for the AgentMessageResponse.
func (m *StopActionResponse) AgentMessageResponsePayload() isAgentMessage_Payload { //nolint:ireturn
	return &AgentMessage_StopAction{StopAction: m}
}

// AgentMessageResponsePayload returns the payload for the AgentMessageResponse.
func (m *CheckConnectionResponse) AgentMessageResponsePayload() isAgentMessage_Payload { //nolint:ireturn
	return &AgentMessage_CheckConnection{CheckConnection: m}
}

// AgentMessageResponsePayload returns the payload for the AgentMessageResponse.
func (m *ServiceInfoResponse) AgentMessageResponsePayload() isAgentMessage_Payload { //nolint:ireturn
	return &AgentMessage_ServiceInfo{ServiceInfo: m}
}

// AgentMessageResponsePayload returns the payload for the AgentMessageResponse.
func (m *JobStatusResponse) AgentMessageResponsePayload() isAgentMessage_Payload { //nolint:ireturn
	return &AgentMessage_JobStatus{JobStatus: m}
}

// AgentMessageResponsePayload returns the payload for the AgentMessageResponse.
func (m *StartJobResponse) AgentMessageResponsePayload() isAgentMessage_Payload { //nolint:ireturn
	return &AgentMessage_StartJob{StartJob: m}
}

// AgentMessageResponsePayload returns the payload for the AgentMessageResponse.
func (m *StopJobResponse) AgentMessageResponsePayload() isAgentMessage_Payload { //nolint:ireturn
	return &AgentMessage_StopJob{StopJob: m}
}

// AgentMessageResponsePayload returns the payload for the AgentMessageResponse.
func (m *JobProgress) AgentMessageResponsePayload() isAgentMessage_Payload { //nolint:ireturn
	return &AgentMessage_JobProgress{JobProgress: m}
}

// AgentMessageResponsePayload returns the payload for the AgentMessageResponse.
func (m *JobResult) AgentMessageResponsePayload() isAgentMessage_Payload { //nolint:ireturn
	return &AgentMessage_JobResult{JobResult: m}
}

// AgentMessageResponsePayload returns the payload for the AgentMessageResponse.
func (m *GetVersionsResponse) AgentMessageResponsePayload() isAgentMessage_Payload { //nolint:ireturn
	return &AgentMessage_GetVersions{GetVersions: m}
}

// AgentMessageResponsePayload returns the payload for the AgentMessageResponse.
func (m *PBMSwitchPITRResponse) AgentMessageResponsePayload() isAgentMessage_Payload { //nolint:ireturn
	return &AgentMessage_PbmSwitchPitr{PbmSwitchPitr: m}
}

// AgentMessageResponsePayload returns the payload for the AgentMessageResponse.
func (m *AgentLogsResponse) AgentMessageResponsePayload() isAgentMessage_Payload { //nolint:ireturn
	return &AgentMessage_AgentLogs{AgentLogs: m}
}

// A list of ServerMessage response payloads.

// ServerMessageResponsePayload returns the payload for the ServerMessageResponse.
func (m *Pong) ServerMessageResponsePayload() isServerMessage_Payload { //nolint:ireturn
	return &ServerMessage_Pong{Pong: m}
}

// ServerMessageResponsePayload returns the payload for the ServerMessageResponse.
func (m *StateChangedResponse) ServerMessageResponsePayload() isServerMessage_Payload { //nolint:ireturn
	return &ServerMessage_StateChanged{StateChanged: m}
}

// ServerMessageResponsePayload returns the payload for the ServerMessageResponse.
func (m *QANCollectResponse) ServerMessageResponsePayload() isServerMessage_Payload { //nolint:ireturn
	return &ServerMessage_QanCollect{QanCollect: m}
}

// ServerMessageResponsePayload returns the payload for the ServerMessageResponse.
func (m *ActionResultResponse) ServerMessageResponsePayload() isServerMessage_Payload { //nolint:ireturn
	return &ServerMessage_ActionResult{ActionResult: m}
}

// A list of ServerMessage request payloads.

// ServerMessageRequestPayload returns the payload for the ServerMessageRequestPayload.
func (m *Ping) ServerMessageRequestPayload() isServerMessage_Payload { //nolint:ireturn
	return &ServerMessage_Ping{Ping: m}
}

// ServerMessageRequestPayload returns the payload for the ServerMessageRequestPayload.
func (m *SetStateRequest) ServerMessageRequestPayload() isServerMessage_Payload { //nolint:ireturn
	return &ServerMessage_SetState{SetState: m}
}

// ServerMessageRequestPayload returns the payload for the ServerMessageRequestPayload.
func (m *StartActionRequest) ServerMessageRequestPayload() isServerMessage_Payload { //nolint:ireturn
	return &ServerMessage_StartAction{StartAction: m}
}

// ServerMessageRequestPayload returns the payload for the ServerMessageRequestPayload.
func (m *StopActionRequest) ServerMessageRequestPayload() isServerMessage_Payload { //nolint:ireturn
	return &ServerMessage_StopAction{StopAction: m}
}

// ServerMessageRequestPayload returns the payload for the ServerMessageRequestPayload.
func (m *CheckConnectionRequest) ServerMessageRequestPayload() isServerMessage_Payload { //nolint:ireturn
	return &ServerMessage_CheckConnection{CheckConnection: m}
}

// ServerMessageRequestPayload returns the payload for the ServerMessageRequestPayload.
func (m *StartJobRequest) ServerMessageRequestPayload() isServerMessage_Payload { //nolint:ireturn
	return &ServerMessage_StartJob{StartJob: m}
}

// ServerMessageRequestPayload returns the payload for the ServerMessageRequestPayload.
func (m *StopJobRequest) ServerMessageRequestPayload() isServerMessage_Payload { //nolint:ireturn
	return &ServerMessage_StopJob{StopJob: m}
}

// ServerMessageRequestPayload returns the payload for the ServerMessageRequestPayload.
func (m *JobStatusRequest) ServerMessageRequestPayload() isServerMessage_Payload { //nolint:ireturn
	return &ServerMessage_JobStatus{JobStatus: m}
}

// ServerMessageRequestPayload returns the payload for the ServerMessageRequestPayload.
func (m *GetVersionsRequest) ServerMessageRequestPayload() isServerMessage_Payload { //nolint:ireturn
	return &ServerMessage_GetVersions{GetVersions: m}
}

// ServerMessageRequestPayload returns the payload for the ServerMessageRequestPayload.
func (m *PBMSwitchPITRRequest) ServerMessageRequestPayload() isServerMessage_Payload { //nolint:ireturn
	return &ServerMessage_PbmSwitchPitr{PbmSwitchPitr: m}
}

// ServerMessageRequestPayload returns the payload for the ServerMessageRequestPayload.
func (m *AgentLogsRequest) ServerMessageRequestPayload() isServerMessage_Payload { //nolint:ireturn
	return &ServerMessage_AgentLogs{AgentLogs: m}
}

// ServerMessageRequestPayload returns the payload for the ServerMessageRequestPayload.
func (m *ServiceInfoRequest) ServerMessageRequestPayload() isServerMessage_Payload { //nolint:ireturn
	return &ServerMessage_ServiceInfo{ServiceInfo: m}
}

// in alphabetical order.
func (*ActionResultRequest) sealed()     {}
func (*ActionResultResponse) sealed()    {}
func (*AgentLogsRequest) sealed()        {}
func (*AgentLogsResponse) sealed()       {}
func (*CheckConnectionRequest) sealed()  {}
func (*CheckConnectionResponse) sealed() {}
func (*GetVersionsRequest) sealed()      {}
func (*GetVersionsResponse) sealed()     {}
func (*JobProgress) sealed()             {}
func (*JobResult) sealed()               {}
func (*JobStatusRequest) sealed()        {}
func (*JobStatusResponse) sealed()       {}
func (*PBMSwitchPITRRequest) sealed()    {}
func (*PBMSwitchPITRResponse) sealed()   {}
func (*Ping) sealed()                    {}
func (*Pong) sealed()                    {}
func (*QANCollectRequest) sealed()       {}
func (*QANCollectResponse) sealed()      {}
func (*ServiceInfoRequest) sealed()      {}
func (*ServiceInfoResponse) sealed()     {}
func (*SetStateRequest) sealed()         {}
func (*SetStateResponse) sealed()        {}
func (*StartActionRequest) sealed()      {}
func (*StartActionResponse) sealed()     {}
func (*StartJobRequest) sealed()         {}
func (*StartJobResponse) sealed()        {}
func (*StateChangedRequest) sealed()     {}
func (*StateChangedResponse) sealed()    {}
func (*StopActionRequest) sealed()       {}
func (*StopActionResponse) sealed()      {}
func (*StopJobRequest) sealed()          {}
func (*StopJobResponse) sealed()         {}

// check interfaces.
var (
	// A list of AgentMessage request payloads.
	_ AgentRequestPayload = (*Ping)(nil)
	_ AgentRequestPayload = (*StateChangedRequest)(nil)
	_ AgentRequestPayload = (*QANCollectRequest)(nil)
	_ AgentRequestPayload = (*ActionResultRequest)(nil)

	// A list of AgentMessage response payloads.
	_ AgentResponsePayload = (*Pong)(nil)
	_ AgentResponsePayload = (*SetStateResponse)(nil)
	_ AgentResponsePayload = (*StartActionResponse)(nil)
	_ AgentResponsePayload = (*StopActionResponse)(nil)
	_ AgentResponsePayload = (*CheckConnectionResponse)(nil)
	_ AgentResponsePayload = (*JobProgress)(nil)
	_ AgentResponsePayload = (*JobResult)(nil)
	_ AgentResponsePayload = (*StartJobResponse)(nil)
	_ AgentResponsePayload = (*StopJobResponse)(nil)
	_ AgentResponsePayload = (*JobStatusResponse)(nil)
	_ AgentResponsePayload = (*GetVersionsResponse)(nil)
	_ AgentResponsePayload = (*AgentLogsResponse)(nil)
	_ AgentResponsePayload = (*ServiceInfoResponse)(nil)

	// A list of ServerMessage response payloads.
	_ ServerResponsePayload = (*Pong)(nil)
	_ ServerResponsePayload = (*StateChangedResponse)(nil)
	_ ServerResponsePayload = (*QANCollectResponse)(nil)
	_ ServerResponsePayload = (*ActionResultResponse)(nil)

	// A list of ServerMessage request payloads.
	_ ServerRequestPayload = (*Ping)(nil)
	_ ServerRequestPayload = (*SetStateRequest)(nil)
	_ ServerRequestPayload = (*StartActionRequest)(nil)
	_ ServerRequestPayload = (*StopActionRequest)(nil)
	_ ServerRequestPayload = (*CheckConnectionRequest)(nil)
	_ ServerRequestPayload = (*StartJobRequest)(nil)
	_ ServerRequestPayload = (*StopJobRequest)(nil)
	_ ServerRequestPayload = (*JobStatusRequest)(nil)
	_ ServerRequestPayload = (*GetVersionsRequest)(nil)
	_ ServerRequestPayload = (*PBMSwitchPITRRequest)(nil)
	_ ServerRequestPayload = (*AgentLogsRequest)(nil)
	_ ServerRequestPayload = (*ServiceInfoRequest)(nil)
)

//go-sumtype:decl AgentParams

// AgentParams is a common interface for AgentProcess and BuiltinAgent parameters.
type AgentParams interface {
	proto.Message
	sealedAgentParams()
}

func (*SetStateRequest_AgentProcess) sealedAgentParams() {}
func (*SetStateRequest_BuiltinAgent) sealedAgentParams() {}
