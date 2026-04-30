// Copyright (C) 2026 Percona LLC

package grafana

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFromToForGrafanaImageRendererQuery_AbsoluteToEpochMs(t *testing.T) {
	from, to := fromToForGrafanaImageRendererQuery("2026-04-29T14:30:00Z", "2026-04-29T20:30:00Z")
	assert.NotContains(t, from, ":")
	assert.NotContains(t, to, ":")
	assert.Len(t, from, 13)
	assert.Len(t, to, 13)
	assert.NotEqual(t, from, "2026-04-29T14:30:00Z")
}

func TestFromToForGrafanaImageRendererQuery_RelativePassthrough(t *testing.T) {
	from, to := fromToForGrafanaImageRendererQuery("now-6h", "now")
	assert.Equal(t, "now-6h", from)
	assert.Equal(t, "now", to)
}
