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

package management

// AddOtelCommandGroup groups subcommands that create or update the single otel_collector per pmm-agent.
//
// TODO(otel): Add `pmm-admin remove otel` (symmetric to `add otel`, e.g. logs-only vs whole collector) — planned soon.
// Until then, remove the otel_collector agent with `pmm-admin inventory remove agent <agent-id>`.
type AddOtelCommandGroup struct {
	Logs   AddOtelLogsCommand   `cmd:"" name:"logs" help:"Add or update log file sources on the node OTEL collector"`
	Ebpf   AddOtelEbpfCommand   `cmd:"" name:"ebpf" help:"Enable eBPF pipeline labels on the node OTEL collector (OTLP export unchanged)"`
	Traces AddOtelTracesCommand `cmd:"" name:"traces" help:"Mark trace ingestion intent; collector already accepts OTLP traces on 4317/4318"`
}
