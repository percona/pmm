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

package om

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergeFacts(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)

	t.Run("precedence decides, not call order", func(t *testing.T) {
		t.Parallel()

		// The probe is listed after metrics for version, so it loses however late it runs.
		merged := mergeFacts([]SourceResult{
			{Source: sourceProbe, Facts: []Fact{
				{Service: "s1", Field: fieldVersion, Value: "7.0.1", Source: sourceProbe},
			}},
			{Source: sourceMetrics, Facts: []Fact{
				{Service: "s1", Field: fieldVersion, Value: "7.0.39", Source: sourceMetrics, ObservedAt: new(now)},
			}},
		}, defaultPrecedence)

		require.Contains(t, merged, "s1")
		assert.Equal(t, "7.0.39", merged["s1"][fieldVersion].Value)
		assert.Equal(t, sourceMetrics, merged["s1"][fieldVersion].Source)
	})

	t.Run("the fallback source wins when the preferred one is silent", func(t *testing.T) {
		t.Parallel()

		merged := mergeFacts([]SourceResult{
			{Source: sourceInventory, Facts: []Fact{
				{Service: "s1", Field: fieldEndpoint, Value: "host:27017", Source: sourceInventory},
			}},
			{Source: sourceMetrics, Facts: []Fact{}},
		}, defaultPrecedence)

		assert.Equal(t, "host:27017", merged["s1"][fieldEndpoint].Value)
		assert.Equal(t, sourceInventory, merged["s1"][fieldEndpoint].Source)
	})

	t.Run("a source not listed for a field cannot set it", func(t *testing.T) {
		t.Parallel()

		// exporter_up names metrics alone: reachability is not the probe's to assert.
		merged := mergeFacts([]SourceResult{
			{Source: sourceProbe, Facts: []Fact{
				{Service: "s1", Field: fieldExporterUp, Value: 1.0, Source: sourceProbe},
			}},
		}, defaultPrecedence)

		assert.NotContains(t, merged["s1"], fieldExporterUp)
	})

	t.Run("probe-only fields are reserved and merge untouched", func(t *testing.T) {
		t.Parallel()

		merged := mergeFacts([]SourceResult{
			{Source: sourceProbe, Facts: []Fact{
				{Service: "s1", Field: fieldInstalledVersion, Value: "7.0.40", Source: sourceProbe},
			}},
			{Source: sourceMetrics, Facts: []Fact{
				{Service: "s1", Field: fieldInstalledVersion, Value: "nonsense", Source: sourceMetrics},
			}},
		}, defaultPrecedence)

		assert.Equal(t, "7.0.40", merged["s1"][fieldInstalledVersion].Value)
	})

	t.Run("a nil value never shadows a real one", func(t *testing.T) {
		t.Parallel()

		merged := mergeFacts([]SourceResult{
			{Source: sourceMetrics, Facts: []Fact{
				{Service: "s1", Field: fieldVersion, Value: nil, Source: sourceMetrics},
			}},
			{Source: sourceProbe, Facts: []Fact{
				{Service: "s1", Field: fieldVersion, Value: "7.0.39", Source: sourceProbe},
			}},
		}, defaultPrecedence)

		assert.Equal(t, "7.0.39", merged["s1"][fieldVersion].Value)
	})

	t.Run("the freshest series wins when one service has several", func(t *testing.T) {
		t.Parallel()

		// A replica-set reconfiguration leaves the superseded mongodb_members_self behind,
		// and it survives in the lookback window under the same service_id. Measured in
		// the sandbox: PRIMARY at 0s, SECONDARY at 7500s, PRIMARY at 68477s. Rank cannot
		// separate them -- they are all the metrics source -- so recency has to.
		fresh := now
		stale := now.Add(-7500 * time.Second)
		ancient := now.Add(-68477 * time.Second)

		for _, order := range [][]Fact{
			{
				{Service: "s1", Field: fieldState, Value: "SECONDARY", Source: sourceMetrics, ObservedAt: new(stale)},
				{Service: "s1", Field: fieldState, Value: "PRIMARY", Source: sourceMetrics, ObservedAt: new(fresh)},
				{Service: "s1", Field: fieldState, Value: "PRIMARY", Source: sourceMetrics, ObservedAt: new(ancient)},
			},
			// The same facts the other way round: the answer must not depend on the order
			// VictoriaMetrics happened to return them in.
			{
				{Service: "s1", Field: fieldState, Value: "PRIMARY", Source: sourceMetrics, ObservedAt: new(fresh)},
				{Service: "s1", Field: fieldState, Value: "SECONDARY", Source: sourceMetrics, ObservedAt: new(stale)},
			},
		} {
			merged := mergeFacts([]SourceResult{{Source: sourceMetrics, Facts: order}}, defaultPrecedence)
			assert.Equal(t, "PRIMARY", merged["s1"][fieldState].Value)
			assert.Equal(t, fresh, *merged["s1"][fieldState].ObservedAt)
		}
	})

	t.Run("a higher-precedence source still wins over a fresher low-precedence one", func(t *testing.T) {
		t.Parallel()

		// Recency only breaks ties. It must not let the probe outrank metrics for version
		// just by having looked more recently.
		merged := mergeFacts([]SourceResult{
			{Source: sourceMetrics, Facts: []Fact{
				{Service: "s1", Field: fieldVersion, Value: "7.0.39", Source: sourceMetrics, ObservedAt: new(now.Add(-time.Hour))},
			}},
			{Source: sourceProbe, Facts: []Fact{
				{Service: "s1", Field: fieldVersion, Value: "7.0.1", Source: sourceProbe, ObservedAt: new(now)},
			}},
		}, defaultPrecedence)

		assert.Equal(t, "7.0.39", merged["s1"][fieldVersion].Value)
	})

	t.Run("provenance survives the merge", func(t *testing.T) {
		t.Parallel()

		observed := now.Add(-90 * time.Second)
		merged := mergeFacts([]SourceResult{
			{Source: sourceMetrics, Facts: []Fact{
				{Service: "s1", Field: fieldState, Value: "PRIMARY", Source: sourceMetrics, ObservedAt: new(observed)},
			}},
		}, defaultPrecedence)

		held := merged["s1"][fieldState]
		require.NotNil(t, held.ObservedAt)
		assert.Equal(t, observed, *held.ObservedAt)

		age, datable := held.age(now)
		assert.True(t, datable)
		assert.Equal(t, 90*time.Second, age)
	})
}

func TestFieldSetFreshness(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)
	maxAge := 30 * time.Second

	newFields := func(age time.Duration) fieldSet {
		observed := now.Add(-age)
		return fieldSet{
			now:    now,
			maxAge: maxAge,
			fields: map[string]MergedField{
				// Volatile: only meaningful while fresh.
				fieldExporterUp: {Value: 1.0, Source: sourceMetrics, ObservedAt: new(observed)},
				fieldState:      {Value: "PRIMARY", Source: sourceMetrics, ObservedAt: new(observed)},
				// Durable: describes the service, not the moment.
				fieldVersion: {Value: "7.0.39", Source: sourceMetrics, ObservedAt: new(observed)},
				fieldEdition: {Value: "Community", Source: sourceMetrics, ObservedAt: new(observed)},
			},
		}
	}

	t.Run("fresh volatile facts read through", func(t *testing.T) {
		t.Parallel()

		fields := newFields(5 * time.Second)
		assert.InDelta(t, 1.0, *fields.f64(fieldExporterUp), 0.001)
		assert.Equal(t, "PRIMARY", fields.str(fieldState))
		assert.Zero(t, fields.staleVolatile())
	})

	t.Run("stale volatile facts are dropped, durable ones are kept", func(t *testing.T) {
		t.Parallel()

		// This is the case that made a stopped database report UP: the sample survives in
		// the long lookback window, and only the age says it is not about now.
		fields := newFields(10 * time.Minute)
		assert.Nil(t, fields.f64(fieldExporterUp))
		assert.Empty(t, fields.str(fieldState))
		assert.Equal(t, "7.0.39", fields.str(fieldVersion), "a version does not go stale")
		assert.Equal(t, "Community", fields.str(fieldEdition))
		assert.Equal(t, 2, fields.staleVolatile())
	})

	t.Run("an undatable fact is never stale", func(t *testing.T) {
		t.Parallel()

		// Inventory is current by definition and carries no observed_at.
		fields := fieldSet{
			now:    now,
			maxAge: maxAge,
			fields: map[string]MergedField{
				fieldEndpoint: {Value: "host:27017", Source: sourceInventory},
			},
		}
		assert.Equal(t, "host:27017", fields.str(fieldEndpoint))
	})

	t.Run("a missing or mistyped field reads as absent", func(t *testing.T) {
		t.Parallel()

		fields := fieldSet{
			now:    now,
			maxAge: maxAge,
			fields: map[string]MergedField{fieldVersion: {Value: 42, Source: sourceMetrics}},
		}
		assert.Empty(t, fields.str(fieldVersion))
		assert.Empty(t, fields.str(fieldVendor))
		assert.Nil(t, fields.f64(fieldCPUUsage))
		assert.False(t, fields.truthy(fieldIsMongos))
	})
}
