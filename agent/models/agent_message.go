// Copyright 2023 Percona LLC
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

// Package models contains client domain models and helpers.
package models

import (
	"google.golang.org/grpc/status"

	agentpb "github.com/percona/pmm/api/agentpb/v1"
)

// AgentRequest represents an request from agent.
// It is similar to agentpb.AgentMessage except it can contain only requests,
// and the payload is already unwrapped (XXX instead of AgentMessage_XXX).
type AgentRequest struct {
	ID      uint32
	Payload agentpb.AgentRequestPayload
}

// AgentResponse represents agent's response.
// It is similar to agentpb.AgentMessage except it can contain only responses,
// and the payload is already unwrapped (XXX instead of AgentMessage_XXX).
type AgentResponse struct {
	ID      uint32
	Status  *status.Status
	Payload agentpb.AgentResponsePayload
}
