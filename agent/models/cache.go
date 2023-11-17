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

import agentpb "github.com/percona/pmm/api/agentpb"

// Sender is a subset of methods of channel, cache.
type Sender interface {
	Send(resp *AgentResponse) error
	SendAndWaitResponse(payload agentpb.AgentRequestPayload) (agentpb.ServerResponsePayload, error)
}

// Cache represent cache methods.
type Cache interface {
	Sender
	Close()
	SetSender(s Sender)
}
