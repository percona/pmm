// Copyright (C) 2023 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package management

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	managementv1 "github.com/percona/pmm/api/management/v1"
)

func TestListElastiCacheRegions(t *testing.T) {
	t.Run("DefaultPartition", func(t *testing.T) {
		regions := listElastiCacheRegions([]string{"aws"})
		require.NotEmpty(t, regions)
		assert.Contains(t, regions, "us-east-1")
		assert.Contains(t, regions, "eu-central-1")
	})

	t.Run("GovCloudPartition", func(t *testing.T) {
		regions := listElastiCacheRegions([]string{"aws-us-gov"})
		require.NotEmpty(t, regions)
		assert.Contains(t, regions, "us-gov-west-1")
	})

	t.Run("MultiplePartitions", func(t *testing.T) {
		regions := listElastiCacheRegions([]string{"aws", "aws-cn"})
		require.NotEmpty(t, regions)
		assert.Contains(t, regions, "us-east-1")
		assert.Contains(t, regions, "cn-north-1")
	})

	t.Run("EmptyPartitions", func(t *testing.T) {
		regions := listElastiCacheRegions([]string{})
		assert.Empty(t, regions)
	})

	t.Run("UnknownPartition", func(t *testing.T) {
		regions := listElastiCacheRegions([]string{"unknown"})
		assert.Empty(t, regions)
	})
}

func TestElastiCacheEngineMap(t *testing.T) {
	t.Run("Redis", func(t *testing.T) {
		engine, ok := elasticacheEngines["redis"]
		assert.True(t, ok)
		assert.Equal(t, managementv1.DiscoverElastiCacheEngine_DISCOVER_ELASTICACHE_ENGINE_REDIS, engine)
	})

	t.Run("Valkey", func(t *testing.T) {
		engine, ok := elasticacheEngines["valkey"]
		assert.True(t, ok)
		assert.Equal(t, managementv1.DiscoverElastiCacheEngine_DISCOVER_ELASTICACHE_ENGINE_VALKEY, engine)
	})

	t.Run("UnsupportedEngine", func(t *testing.T) {
		_, ok := elasticacheEngines["memcached"]
		assert.False(t, ok)
	})
}
