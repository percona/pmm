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
