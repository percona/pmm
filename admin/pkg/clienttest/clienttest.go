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

// Package clienttest provides helpers for tests which reconfigure the generated PMM Server API
// clients.
package clienttest

import (
	"testing"

	inventoryClient "github.com/percona/pmm/api/inventory/v1/json/client"
	managementClient "github.com/percona/pmm/api/management/v1/json/client"
	serverClient "github.com/percona/pmm/api/server/v1/json/client"
)

// RestoreDefaults puts the package-level PMM Server API clients back the way they were once the
// test is done. SetupClients (admin/commands/base) reconfigures them for the whole test binary,
// so without this a later test would talk to whatever server this one pointed them at - most
// often a closed port, failing a test which has nothing to do with the one that moved them.
//
// It lives here rather than next to either caller so that a generated client added later is
// listed in one place: a copy which forgot the new client would restore only part of the state
// and reintroduce exactly the leak this guards against.
func RestoreDefaults(t *testing.T) {
	t.Helper()

	inventory := inventoryClient.Default.Transport
	management := managementClient.Default.Transport
	server := serverClient.Default.Transport

	t.Cleanup(func() {
		inventoryClient.Default.SetTransport(inventory)
		managementClient.Default.SetTransport(management)
		serverClient.Default.SetTransport(server)
	})
}
