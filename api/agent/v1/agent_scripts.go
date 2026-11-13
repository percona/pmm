// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package agentv1

// AgentMessageResponsePayload implements AgentResponsePayload interface.
func (m *RunOperationResponse) AgentMessageResponsePayload() isAgentMessage_Payload { //nolint:ireturn
	return &AgentMessage_RunOperation{RunOperation: m}
}

// AgentMessageResponsePayload implements AgentResponsePayload interface.
func (m *ReadJobFileResponse) AgentMessageResponsePayload() isAgentMessage_Payload { //nolint:ireturn
	return &AgentMessage_ReadJobFile{ReadJobFile: m}
}

// ServerMessageRequestPayload implements ServerRequestPayload interface.
func (m *RunOperationRequest) ServerMessageRequestPayload() isServerMessage_Payload { //nolint:ireturn
	return &ServerMessage_RunOperation{RunOperation: m}
}

// ServerMessageRequestPayload implements ServerRequestPayload interface.
func (m *ReadJobFileRequest) ServerMessageRequestPayload() isServerMessage_Payload { //nolint:ireturn
	return &ServerMessage_ReadJobFile{ReadJobFile: m}
}

// AgentMessageRequestPayload implements AgentRequestPayload interface.
func (m *JobMessageRequest) AgentMessageRequestPayload() isAgentMessage_Payload { //nolint:ireturn
	return &AgentMessage_JobMessage{JobMessage: m}
}

// AgentMessageResponsePayload lets a job hand an acknowledged message to the runner's Send, which
// is typed for the unacknowledged job messages.
func (m *JobMessageRequest) AgentMessageResponsePayload() isAgentMessage_Payload { //nolint:ireturn
	return &AgentMessage_JobMessage{JobMessage: m}
}

// ServerMessageResponsePayload implements ServerResponsePayload interface.
func (m *JobMessageResponse) ServerMessageResponsePayload() isServerMessage_Payload { //nolint:ireturn
	return &ServerMessage_JobMessage{JobMessage: m}
}

func (*JobMessageRequest) sealed()    {}
func (*JobMessageResponse) sealed()   {}
func (*RunOperationRequest) sealed()  {}
func (*RunOperationResponse) sealed() {}
func (*ReadJobFileRequest) sealed()   {}
func (*ReadJobFileResponse) sealed()  {}

var (
	_ AgentRequestPayload   = (*JobMessageRequest)(nil)
	_ AgentResponsePayload  = (*JobMessageRequest)(nil)
	_ ServerResponsePayload = (*JobMessageResponse)(nil)
	_ AgentResponsePayload  = (*RunOperationResponse)(nil)
	_ AgentResponsePayload  = (*ReadJobFileResponse)(nil)
	_ ServerRequestPayload  = (*RunOperationRequest)(nil)
	_ ServerRequestPayload  = (*ReadJobFileRequest)(nil)
)
