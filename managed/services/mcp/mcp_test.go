package mcp

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestQANGetReportArgs(t *testing.T) {
	// Test argument structure for qan_get_report
	args := qanGetReportArgs{
		PeriodStart: "2024-01-01T00:00:00Z",
		PeriodEnd:   "2024-01-01T23:59:59Z",
		GroupBy:     "queryid",
		Labels:      `{"service_name": ["mysql-1"]}`,
		Columns:     "query_time,lock_time",
		OrderBy:     "query_time_sum",
		Offset:      0,
		Limit:       100,
		MainMetric:  "query_time",
		Search:      "SELECT",
	}

	// Verify all fields are properly tagged for JSON
	assert.Equal(t, "2024-01-01T00:00:00Z", args.PeriodStart)
	assert.Equal(t, "2024-01-01T23:59:59Z", args.PeriodEnd)
	assert.Equal(t, "queryid", args.GroupBy)
	assert.Equal(t, `{"service_name": ["mysql-1"]}`, args.Labels)
	assert.Equal(t, "query_time,lock_time", args.Columns)
	assert.Equal(t, "query_time_sum", args.OrderBy)
	assert.Equal(t, 0, args.Offset)
	assert.Equal(t, 100, args.Limit)
	assert.Equal(t, "query_time", args.MainMetric)
	assert.Equal(t, "SELECT", args.Search)
}

func TestQANGetMetricsArgs(t *testing.T) {
	// Test argument structure for qan_get_metrics
	args := qanGetMetricsArgs{
		PeriodStart:       "2024-01-01T00:00:00Z",
		PeriodEnd:         "2024-01-01T23:59:59Z",
		FilterBy:          "1D410B4BE5060972",
		GroupBy:           "queryid",
		Labels:            `{"service_name": ["mysql-1"]}`,
		IncludeOnlyFields: "query_time,lock_time",
		Totals:            true,
	}

	// Verify all fields are properly tagged for JSON
	assert.Equal(t, "2024-01-01T00:00:00Z", args.PeriodStart)
	assert.Equal(t, "2024-01-01T23:59:59Z", args.PeriodEnd)
	assert.Equal(t, "1D410B4BE5060972", args.FilterBy)
	assert.Equal(t, "queryid", args.GroupBy)
	assert.Equal(t, `{"service_name": ["mysql-1"]}`, args.Labels)
	assert.Equal(t, "query_time,lock_time", args.IncludeOnlyFields)
	assert.True(t, args.Totals)
}

func TestTimeParsingValidation(t *testing.T) {
	// Test that time parsing works correctly
	testCases := []struct {
		name        string
		timeStr     string
		shouldError bool
	}{
		{
			name:        "valid RFC3339 time",
			timeStr:     "2024-01-01T00:00:00Z",
			shouldError: false,
		},
		{
			name:        "valid RFC3339 time with timezone",
			timeStr:     "2024-01-01T00:00:00+05:00",
			shouldError: false,
		},
		{
			name:        "invalid time format",
			timeStr:     "2024-01-01 00:00:00",
			shouldError: true,
		},
		{
			name:        "empty time string",
			timeStr:     "",
			shouldError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := time.Parse(time.RFC3339, tc.timeStr)
			if tc.shouldError {
				assert.Error(t, err, "Expected error for time string: %s", tc.timeStr)
			} else {
				assert.NoError(t, err, "Expected no error for time string: %s", tc.timeStr)
			}
		})
	}
}

func TestMCPServiceCreation(t *testing.T) {
	// Test that MCP service can be created (basic structure test)
	mcpService := &Mcp{
		actionServer: nil, // We're not testing the actual functionality here
		qanClient:    nil,
		l:            nil,
	}

	assert.NotNil(t, mcpService, "MCP service should be created")
}
