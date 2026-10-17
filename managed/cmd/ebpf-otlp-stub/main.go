// Copyright (C) 2023 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

// Command ebpf-otlp-stub sends minimal OTLP traces to PMM Server for Phase 1 pipeline validation (no eBPF).
//
// Example:
//
//	PMM_OTLP_URL=https://127.0.0.1:8443/otlp/v1/traces PMM_OTLP_INSECURE=1 go run ./managed/cmd/ebpf-otlp-stub
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func main() {
	err := run(context.Background())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	endpoint := os.Getenv("PMM_OTLP_URL")
	if endpoint == "" {
		return errors.New("set PMM_OTLP_URL to OTLP HTTP traces endpoint (e.g. https://pmm.example:8443/otlp/v1/traces)")
	}
	opts := []otlptracehttp.Option{otlptracehttp.WithEndpointURL(endpoint)}
	if os.Getenv("PMM_OTLP_INSECURE") == "1" {
		opts = append(opts, otlptracehttp.WithInsecure())
	}
	exp, err := otlptracehttp.New(ctx, opts...)
	if err != nil {
		return err
	}
	res, err := resource.New(
		ctx,
		resource.WithAttributes(
			attribute.String("service.name", "ebpf-otlp-stub"),
			attribute.String("pmm.node_id", getenv("PMM_NODE_ID", "stub-node")),
			attribute.String("pmm.agent_id", getenv("PMM_AGENT_ID", "stub-agent")),
			attribute.String("net.peer.name", getenv("PMM_PEER", "db.example:3306")),
			attribute.String("db.system", getenv("PMM_DB_SYSTEM", "mysql")),
			attribute.String("pmm.component_role", "app"),
		),
	)
	if err != nil {
		return err
	}
	tp := sdktrace.NewTracerProvider(sdktrace.WithBatcher(exp), sdktrace.WithResource(res))
	otel.SetTracerProvider(tp)
	defer func() { _ = tp.Shutdown(ctx) }()

	tr := tp.Tracer("pmm.ebpf.stub")
	_, span := tr.Start(ctx, "db.mysql.query")
	span.SetAttributes(
		attribute.String("pmm.map_edge_target", getenv("PMM_MAP_TARGET", "mysql-primary")),
	)
	span.End()
	time.Sleep(2 * time.Second) //nolint:mnd
	return nil
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
