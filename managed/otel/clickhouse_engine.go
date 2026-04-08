// Copyright (C) 2026 Percona LLC
//
// Licensed under the GNU Affero General Public License, Version 3 or later.

package otel

import (
	"os"
	"strings"
)

// clickhouseIsCluster mirrors qan-api2’s PMM_CLICKHOUSE_IS_CLUSTER flag (kingpin bool / env).
func clickhouseIsCluster() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("PMM_CLICKHOUSE_IS_CLUSTER")))
	return v == "1" || v == "true" || v == "t" || v == "yes"
}

// clickhouseClusterName returns PMM_CLICKHOUSE_CLUSTER_NAME (used for ON CLUSTER DDL), same as qan-api2.
func clickhouseClusterName() string {
	return strings.TrimSpace(os.Getenv("PMM_CLICKHOUSE_CLUSTER_NAME"))
}

// TableEngine returns MergeTree or ReplicatedMergeTree, matching qan-api2/migrations.GetEngine.
func TableEngine() string {
	if clickhouseIsCluster() {
		return "ReplicatedMergeTree"
	}
	return "MergeTree"
}

// ReplacingTableEngine returns ReplacingMergeTree or ReplicatedReplacingMergeTree for Coroot helper tables.
func ReplacingTableEngine() string {
	if clickhouseIsCluster() {
		return "ReplicatedReplacingMergeTree"
	}
	return "ReplacingMergeTree"
}
