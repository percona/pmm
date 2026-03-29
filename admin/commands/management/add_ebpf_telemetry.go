// Copyright (C) 2026 Percona LLC
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

package management

import (
	"github.com/percona/pmm/admin/commands"
)

// AddEbpfTelemetryCommand registers an OTEL collector agent with Phase 1 eBPF pipeline labels.
// Real eBPF probes are phased in later; OTLP export path is the same as `management add otel`.
type AddEbpfTelemetryCommand struct {
	AddOtelCommand
}

// RunCmd sets default custom labels and delegates to AddOtelCommand.
// Label keys must match PMM inventory rules (alphanumeric + underscore; no dots) — see models.prepareLabels.
func (cmd *AddEbpfTelemetryCommand) RunCmd() (commands.Result, error) {
	labels := cmd.CustomLabels
	if labels == nil {
		labels = make(map[string]string)
	}
	if labels["pmm_ebpf_pipeline"] == "" {
		labels["pmm_ebpf_pipeline"] = "v1"
	}
	cmd.AddOtelCommand.CustomLabels = labels
	return cmd.AddOtelCommand.RunCmd()
}
