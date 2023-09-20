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

// Package server holds the "pmm server" command.
package server

import (
	"github.com/percona/pmm/admin/commands"
	"github.com/percona/pmm/admin/commands/pmm/server/docker"
)

// BaseCommand is used by Kong for CLI flags and commands and holds all server commands.
type BaseCommand struct {
	Docker docker.BaseCommand `cmd:"" help:"Local docker deployment of PMM server"`
}

// BeforeApply is run before the command is applied.
func (cmd *BaseCommand) BeforeApply() error {
	commands.SetupClientsEnabled = false
	return nil
}
