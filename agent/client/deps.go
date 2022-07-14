// Copyright 2019 Percona LLC
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

package client

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/percona/pmm/api/agentpb"
)

//go:generate ../../bin/mockery -name=connectionChecker -case=snake -inpkg -testonly
//go:generate ../../bin/mockery -name=supervisor -case=snake -inpkg -testonly
//go:generate ../../bin/mockery -name=defaultsFileParser -case=snake -inpkg -testonly

// connectionChecker is a subset of methods of connectionchecker.ConnectionChecker used by this package.
// We use it instead of real type for testing and to avoid dependency cycle.
type connectionChecker interface {
	Check(ctx context.Context, req *agentpb.CheckConnectionRequest, id uint32) *agentpb.CheckConnectionResponse
}

// softwareVersioner is a subset of methods of version.Versioner used by this package.
type softwareVersioner interface {
	MySQLdVersion() (string, error)
	XtrabackupVersion() (string, error)
	XbcloudVersion() (string, error)
	Qpress() (string, error)
}

// supervisor is a subset of methods of supervisor.Supervisor used by this package.
// We use it instead of real type for testing and to avoid dependency cycle.
type supervisor interface {
	Changes() <-chan *agentpb.StateChangedRequest
	QANRequests() <-chan *agentpb.QANCollectRequest
	SetState(*agentpb.SetStateRequest)
	// Collector added to use client as Prometheus collector
	prometheus.Collector
}
type defaultsFileParser interface {
	ParseDefaultsFile(req *agentpb.ParseDefaultsFileRequest) *agentpb.ParseDefaultsFileResponse
}
