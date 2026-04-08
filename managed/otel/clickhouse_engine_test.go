// Copyright (C) 2026 Percona LLC
//
// Licensed under the GNU Affero General Public License, Version 3 or later.

package otel

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTableEngine_ClusterEnv(t *testing.T) {
	t.Setenv("PMM_CLICKHOUSE_IS_CLUSTER", "")
	t.Setenv("PMM_CLICKHOUSE_CLUSTER_NAME", "")
	assert.Equal(t, "MergeTree", TableEngine())
	assert.Equal(t, "ReplacingMergeTree", ReplacingTableEngine())

	t.Setenv("PMM_CLICKHOUSE_IS_CLUSTER", "true")
	assert.Equal(t, "ReplicatedMergeTree", TableEngine())
	assert.Equal(t, "ReplicatedReplacingMergeTree", ReplacingTableEngine())
}
