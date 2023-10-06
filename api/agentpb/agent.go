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

func (m *Ping) AgentMessageRequestPayload() isAgentMessage_Payload { //nolint:ireturn
	return &AgentMessage_Ping{Ping: m}
}

func (m *StateChangedRequest) AgentMessageRequestPayload() isAgentMessage_Payload { //nolint:ireturn
	return &AgentMessage_StateChanged{StateChanged: m}
}

func (m *QANCollectRequest) AgentMessageRequestPayload() isAgentMessage_Payload { //nolint:ireturn
	return &AgentMessage_QanCollect{QanCollect: m}
}

func (m *ActionResultRequest) AgentMessageRequestPayload() isAgentMessage_Payload { //nolint:ireturn
	return &AgentMessage_ActionResult{ActionResult: m}
}

func (m *JobProgress) AgentMessageRequestPayload() isAgentMessage_Payload { //nolint:ireturn
	return &AgentMessage_JobProgress{JobProgress: m}
}

func (m *JobResult) AgentMessageRequestPayload() isAgentMessage_Payload { //nolint:ireturn
	return &AgentMessage_JobResult{JobResult: m}
}

// A list of AgentMessage response payloads.

func (m *Pong) AgentMessageResponsePayload() isAgentMessage_Payload { //nolint:ireturn
	return &AgentMessage_Pong{Pong: m}
}

func (m *SetStateResponse) AgentMessageResponsePayload() isAgentMessage_Payload { //nolint:ireturn
	return &AgentMessage_SetState{SetState: m}
}

func (m *StartActionResponse) AgentMessageResponsePayload() isAgentMessage_Payload { //nolint:ireturn
	return &AgentMessage_StartAction{StartAction: m}
}

func (m *StopActionResponse) AgentMessageResponsePayload() isAgentMessage_Payload { //nolint:ireturn
	return &AgentMessage_StopAction{StopAction: m}
}

func (m *CheckConnectionResponse) AgentMessageResponsePayload() isAgentMessage_Payload { //nolint:ireturn
	return &AgentMessage_CheckConnection{CheckConnection: m}
}

func (m *JobStatusResponse) AgentMessageResponsePayload() isAgentMessage_Payload { //nolint:ireturn
	return &AgentMessage_JobStatus{JobStatus: m}
}

func (m *StartJobResponse) AgentMessageResponsePayload() isAgentMessage_Payload { //nolint:ireturn
	return &AgentMessage_StartJob{StartJob: m}
}

func (m *StopJobResponse) AgentMessageResponsePayload() isAgentMessage_Payload { //nolint:ireturn
	return &AgentMessage_StopJob{StopJob: m}
}

func (m *JobProgress) AgentMessageResponsePayload() isAgentMessage_Payload { //nolint:ireturn
	return &AgentMessage_JobProgress{JobProgress: m}
}

func (m *JobResult) AgentMessageResponsePayload() isAgentMessage_Payload { //nolint:ireturn
	return &AgentMessage_JobResult{JobResult: m}
}

func (m *GetVersionsResponse) AgentMessageResponsePayload() isAgentMessage_Payload { //nolint:ireturn
	return &AgentMessage_GetVersions{GetVersions: m}
}

func (m *PBMSwitchPITRResponse) AgentMessageResponsePayload() isAgentMessage_Payload { //nolint:ireturn
	return &AgentMessage_PbmSwitchPitr{PbmSwitchPitr: m}
}

func (m *AgentLogsResponse) AgentMessageResponsePayload() isAgentMessage_Payload { //nolint:ireturn
	return &AgentMessage_AgentLogs{AgentLogs: m}
}

// A list of ServerMessage response payloads.

func (m *Pong) ServerMessageResponsePayload() isServerMessage_Payload { //nolint:ireturn
	return &ServerMessage_Pong{Pong: m}
}

func (m *StateChangedResponse) ServerMessageResponsePayload() isServerMessage_Payload { //nolint:ireturn
	return &ServerMessage_StateChanged{StateChanged: m}
}

func (m *QANCollectResponse) ServerMessageResponsePayload() isServerMessage_Payload { //nolint:ireturn
	return &ServerMessage_QanCollect{QanCollect: m}
}

func (m *ActionResultResponse) ServerMessageResponsePayload() isServerMessage_Payload { //nolint:ireturn
	return &ServerMessage_ActionResult{ActionResult: m}
}

// A list of ServerMessage request payloads.

func (m *Ping) ServerMessageRequestPayload() isServerMessage_Payload { //nolint:ireturn
	return &ServerMessage_Ping{Ping: m}
}

func (m *SetStateRequest) ServerMessageRequestPayload() isServerMessage_Payload { //nolint:ireturn
	return &ServerMessage_SetState{SetState: m}
}

func (m *StartActionRequest) ServerMessageRequestPayload() isServerMessage_Payload { //nolint:ireturn
	return &ServerMessage_StartAction{StartAction: m}
}

func (m *StopActionRequest) ServerMessageRequestPayload() isServerMessage_Payload { //nolint:ireturn
	return &ServerMessage_StopAction{StopAction: m}
}

func (m *CheckConnectionRequest) ServerMessageRequestPayload() isServerMessage_Payload { //nolint:ireturn
	return &ServerMessage_CheckConnection{CheckConnection: m}
}

func (m *StartJobRequest) ServerMessageRequestPayload() isServerMessage_Payload { //nolint:ireturn
	return &ServerMessage_StartJob{StartJob: m}
}

func (m *StopJobRequest) ServerMessageRequestPayload() isServerMessage_Payload { //nolint:ireturn
	return &ServerMessage_StopJob{StopJob: m}
}

func (m *JobStatusRequest) ServerMessageRequestPayload() isServerMessage_Payload { //nolint:ireturn
	return &ServerMessage_JobStatus{JobStatus: m}
}

func (m *GetVersionsRequest) ServerMessageRequestPayload() isServerMessage_Payload { //nolint:ireturn
	return &ServerMessage_GetVersions{GetVersions: m}
}

func (m *PBMSwitchPITRRequest) ServerMessageRequestPayload() isServerMessage_Payload { //nolint:ireturn
	return &ServerMessage_PbmSwitchPitr{PbmSwitchPitr: m}
}

func (m *AgentLogsRequest) ServerMessageRequestPayload() isServerMessage_Payload { //nolint:ireturn
	return &ServerMessage_AgentLogs{AgentLogs: m}
}

// in alphabetical order.
func (*ActionResultRequest) sealed()     {}
func (*ActionResultResponse) sealed()    {}
func (*CheckConnectionRequest) sealed()  {}
func (*CheckConnectionResponse) sealed() {}
func (*JobProgress) sealed()             {}
func (*JobResult) sealed()               {}
func (*JobStatusRequest) sealed()        {}
func (*JobStatusResponse) sealed()       {}
func (*AgentLogsRequest) sealed()        {}
func (*AgentLogsResponse) sealed()       {}
func (*Ping) sealed()                    {}
func (*Pong) sealed()                    {}
func (*QANCollectRequest) sealed()       {}
func (*QANCollectResponse) sealed()      {}
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
func (*GetVersionsRequest) sealed()      {}
func (*GetVersionsResponse) sealed()     {}
func (*PBMSwitchPITRRequest) sealed()    {}
func (*PBMSwitchPITRResponse) sealed()   {}

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
)

//go-sumtype:decl AgentParams

// AgentParams is a common interface for AgentProcess and BuiltinAgent parameters.
type AgentParams interface {
	proto.Message
	sealedAgentParams()
}

func (*SetStateRequest_AgentProcess) sealedAgentParams() {}
func (*SetStateRequest_BuiltinAgent) sealedAgentParams() {}
