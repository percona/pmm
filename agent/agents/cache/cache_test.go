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

package cache

import (
	"reflect"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// random struct to test the cache.
type someType struct {
	int
	string
	innerStruct
}

type innerStruct struct{ float64 }

func TestCache(t *testing.T) {
	t.Parallel()
	set1 := map[int64]*someType{
		1: {},
		2: {},
		3: {},
		4: {},
		5: {},
	}

	set2 := map[int64]*someType{
		1: {},
		2: {},
		3: {},
		4: {},
		6: {},
		7: {},
	}

	t.Run("DoesntReachLimits", func(t *testing.T) {
		t.Parallel()
		c, err := New(make(map[int64]*someType), time.Second*60, 100, logrus.WithField("test", t.Name()))
		require.NoError(t, err)

		now1 := time.Now()
		_ = c.Set(set1)
		stats := c.Stats()
		actual := make(map[int64]*someType)
		_ = c.Get(actual)

		assert.True(t, reflect.DeepEqual(actual, set1))

		assert.Equal(t, uint(5), stats.Current)
		assert.Equal(t, uint(0), stats.UpdatedN)
		assert.Equal(t, uint(5), stats.AddedN)
		assert.Equal(t, uint(0), stats.RemovedN)
		assert.InDelta(t, 0, int(stats.Oldest.Sub(now1).Seconds()), 1)
		assert.InDelta(t, 0, int(stats.Newest.Sub(now1).Seconds()), 1)

		time.Sleep(time.Second * 1)

		now2 := time.Now()
		_ = c.Set(set2)
		stats = c.Stats()
		actual = make(map[int64]*someType)
		_ = c.Get(actual)

		expected := make(map[int64]*someType)
		for k, v := range set1 {
			expected[k] = v
		}
		expected[6] = &someType{}
		expected[7] = &someType{}

		assert.True(t, reflect.DeepEqual(actual, expected))
		assert.Equal(t, uint(7), stats.Current)
		assert.Equal(t, uint(4), stats.UpdatedN)
		assert.Equal(t, uint(7), stats.AddedN)
		assert.Equal(t, uint(0), stats.RemovedN)
		assert.InDelta(t, 0, int(stats.Oldest.Sub(now1).Seconds()), 0.01)
		assert.InDelta(t, 0, int(stats.Newest.Sub(now2).Seconds()), 0.01)
	})

	t.Run("ReachesTimeLimit", func(t *testing.T) {
		t.Parallel()
		c, err := New(make(map[int64]*someType), time.Second*1, 100, logrus.WithField("test", t.Name()))
		require.NoError(t, err)

		_ = c.Set(set1)
		time.Sleep(time.Second * 1)
		now := time.Now()
		_ = c.Set(set2)
		stats := c.Stats()
		actual := make(map[int64]*someType)
		_ = c.Get(actual)

		expected := make(map[int64]*someType)
		for k, v := range set2 {
			expected[k] = v
		}

		assert.True(t, reflect.DeepEqual(actual, expected))
		assert.Equal(t, uint(6), stats.Current)
		assert.Equal(t, uint(0), stats.UpdatedN)
		assert.Equal(t, uint(11), stats.AddedN)
		assert.Equal(t, uint(5), stats.RemovedN)
		assert.InDelta(t, 0, int(stats.Oldest.Sub(now).Seconds()), 0.01)
		assert.InDelta(t, 0, int(stats.Newest.Sub(now).Seconds()), 0.01)
	})

	t.Run("ReachesSizeLimit", func(t *testing.T) {
		t.Parallel()
		c, err := New(make(map[int64]*someType), time.Second*60, 5, logrus.WithField("test", t.Name()))
		require.NoError(t, err)

		_ = c.Set(set1)
		time.Sleep(time.Second * 1)
		now := time.Now()
		_ = c.Set(set2)
		stats := c.Stats()

		assert.Equal(t, uint(5), stats.Current)
		assert.InDelta(t, 0, int(stats.Oldest.Sub(now).Seconds()), 0.01)
		assert.InDelta(t, 0, int(stats.Newest.Sub(now).Seconds()), 0.01)
	})
}

func TestCacheErrors(t *testing.T) {
	t.Parallel()
	t.Run("WrongTypeOnNew", func(t *testing.T) {
		t.Parallel()
		var err error
		_, err = New(100, time.Second*100, 100, logrus.WithField("test", t.Name()))
		assert.Error(t, err)

		_, err = New([]float64{}, time.Second*100, 100, logrus.WithField("test", t.Name()))
		assert.Error(t, err)

		_, err = New(struct{}{}, time.Second*100, 100, logrus.WithField("test", t.Name()))
		assert.Error(t, err)
	})

	t.Run("WrongTypeOnRefresh", func(t *testing.T) {
		t.Parallel()
		c, _ := New(make(map[int]int), time.Second*100, 100, logrus.WithField("test", t.Name()))
		err := c.Set(map[int]string{1: "some string"})
		assert.Error(t, err)
	})

	t.Run("WrongTypeOnGet", func(t *testing.T) {
		t.Parallel()
		c, _ := New(make(map[int]int), time.Second*100, 100, logrus.WithField("test", t.Name()))
		dest := make(map[int]string)
		err := c.Get(dest)
		assert.Error(t, err)
	})
}
