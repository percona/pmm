package mcp

import (
	"encoding/json"
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

	// Test JSON marshaling and unmarshaling
	jsonData, err := json.Marshal(args)
	assert.NoError(t, err, "Should be able to marshal qanGetReportArgs to JSON")
	assert.NotEmpty(t, jsonData, "JSON data should not be empty")

	var unmarshaledArgs qanGetReportArgs
	err = json.Unmarshal(jsonData, &unmarshaledArgs)
	assert.NoError(t, err, "Should be able to unmarshal JSON back to qanGetReportArgs")

	// Verify that the unmarshaled struct matches the original
	assert.Equal(t, args.PeriodStart, unmarshaledArgs.PeriodStart)
	assert.Equal(t, args.PeriodEnd, unmarshaledArgs.PeriodEnd)
	assert.Equal(t, args.GroupBy, unmarshaledArgs.GroupBy)
	assert.Equal(t, args.Labels, unmarshaledArgs.Labels)
	assert.Equal(t, args.Columns, unmarshaledArgs.Columns)
	assert.Equal(t, args.OrderBy, unmarshaledArgs.OrderBy)
	assert.Equal(t, args.Offset, unmarshaledArgs.Offset)
	assert.Equal(t, args.Limit, unmarshaledArgs.Limit)
	assert.Equal(t, args.MainMetric, unmarshaledArgs.MainMetric)
	assert.Equal(t, args.Search, unmarshaledArgs.Search)

	// Verify the entire struct equality
	assert.Equal(t, args, unmarshaledArgs, "Original and unmarshaled structs should be identical")
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

func TestExtractTablesFromExplainJSON(t *testing.T) {
	tests := []struct {
		name     string
		jsonData string
		expected []string
		wantErr  bool
	}{
		{
			name: "with real_table_name",
			jsonData: `{
				"real_table_name": "users",
				"query_block": {
					"cost_info": {
						"query_cost": "3158.94"
					}
				}
			}`,
			expected: []string{"users"},
			wantErr:  false,
		},
		{
			name: "without real_table_name",
			jsonData: `{
				"query_block": {
					"cost_info": {
						"query_cost": "100.00"
					}
				}
			}`,
			expected: []string{},
			wantErr:  false,
		},
		{
			name: "empty real_table_name",
			jsonData: `{
				"real_table_name": ""
			}`,
			expected: []string{},
			wantErr:  false,
		},
		{
			name: "invalid JSON",
			jsonData: `{
				"invalid": json
			}`,
			expected: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractTablesFromExplainJSON(tt.jsonData)

			if tt.wantErr {
				assert.Error(t, err, "Expected error for test case: %s", tt.name)
				return
			}

			assert.NoError(t, err, "Unexpected error for test case: %s", tt.name)
			assert.Equal(t, tt.expected, got, "Result mismatch for test case: %s", tt.name)
		})
	}
}

func TestExtractTablesFromExplainJSON_RealWorldExample(t *testing.T) {
	realWorldJSON := `{
		"query_block": {
			"cost_info": {
				"query_cost": "3158.94"
			},
			"ordering_operation": {
				"nested_loop": [
					{
						"table": {
							"access_type": "ALL",
							"cost_info": {
								"data_read_per_join": "547K",
								"eval_cost": "226.00",
								"prefix_cost": "228.25",
								"read_cost": "2.25"
							},
							"filtered": "100.00",
							"possible_keys": [
								"PRIMARY",
								"user_id"
							],
							"rows_examined_per_scan": 2260,
							"rows_produced_per_join": 2260,
							"table_name": "o",
							"used_columns": [
								"id",
								"user_id",
								"order_number",
								"total_amount",
								"status",
								"order_date",
								"shipping_address",
								"notes"
							]
						}
					},
					{
						"table": {
							"access_type": "eq_ref",
							"cost_info": {
								"data_read_per_join": "6M",
								"eval_cost": "226.00",
								"prefix_cost": "1019.25",
								"read_cost": "565.00"
							},
							"filtered": "100.00",
							"key": "PRIMARY",
							"key_length": "4",
							"possible_keys": [
								"PRIMARY"
							],
							"ref": [
								"testdb.o.user_id"
							],
							"rows_examined_per_scan": 1,
							"rows_produced_per_join": 2260,
							"table_name": "u",
							"used_columns": [
								"id",
								"username",
								"email",
								"first_name",
								"last_name",
								"birth_date",
								"created_at",
								"updated_at",
								"is_active",
								"profile_data"
							],
							"used_key_parts": [
								"id"
							]
						}
					},
					{
						"table": {
							"access_type": "ref",
							"cost_info": {
								"data_read_per_join": "6M",
								"eval_cost": "611.34",
								"prefix_cost": "3158.95",
								"read_cost": "1528.35"
							},
							"filtered": "100.00",
							"key": "order_id",
							"key_length": "4",
							"possible_keys": [
								"order_id"
							],
							"ref": [
								"testdb.o.id"
							],
							"rows_examined_per_scan": 2,
							"rows_produced_per_join": 6113,
							"table_name": "oi",
							"used_columns": [
								"id",
								"order_id",
								"product_name",
								"quantity",
								"unit_price",
								"total_price",
								"created_at"
							],
							"used_key_parts": [
								"order_id"
							]
						}
					}
				],
				"using_filesort": true,
				"using_temporary_table": true
			},
			"select_id": 1
		},
		"real_table_name": "users",
		"warnings": [
			{
				"Code": 1003,
				"Level": "Note",
				"Message": "/* select#1 */ select \"testdb\".\"u\".\"id\" AS \"id\",\"testdb\".\"u\".\"username\" AS \"username\",\"testdb\".\"u\".\"email\" AS \"email\",\"testdb\".\"u\".\"first_name\" AS \"first_name\",\"testdb\".\"u\".\"last_name\" AS \"last_name\",\"testdb\".\"u\".\"birth_date\" AS \"birth_date\",\"testdb\".\"u\".\"created_at\" AS \"created_at\",\"testdb\".\"u\".\"updated_at\" AS \"updated_at\",\"testdb\".\"u\".\"is_active\" AS \"is_active\",\"testdb\".\"u\".\"profile_data\" AS \"profile_data\",\"testdb\".\"o\".\"id\" AS \"id\",\"testdb\".\"o\".\"user_id\" AS \"user_id\",\"testdb\".\"o\".\"order_number\" AS \"order_number\",\"testdb\".\"o\".\"total_amount\" AS \"total_amount\",\"testdb\".\"o\".\"status\" AS \"status\",\"testdb\".\"o\".\"order_date\" AS \"order_date\",\"testdb\".\"o\".\"shipping_address\" AS \"shipping_address\",\"testdb\".\"o\".\"notes\" AS \"notes\",\"testdb\".\"oi\".\"id\" AS \"id\",\"testdb\".\"oi\".\"order_id\" AS \"order_id\",\"testdb\".\"oi\".\"product_name\" AS \"product_name\",\"testdb\".\"oi\".\"quantity\" AS \"quantity\",\"testdb\".\"oi\".\"unit_price\" AS \"unit_price\",\"testdb\".\"oi\".\"total_price\" AS \"total_price\",\"testdb\".\"oi\".\"created_at\" AS \"created_at\" from \"testdb\".\"users\" \"u\" join \"testdb\".\"orders\" \"o\" join \"testdb\".\"order_items\" \"oi\" where ((\"testdb\".\"u\".\"id\" = \"testdb\".\"o\".\"user_id\") and (\"testdb\".\"oi\".\"order_id\" = \"testdb\".\"o\".\"id\")) order by rand() limit 5"
			}
		]
	}`

	expected := []string{"users"}

	got, err := extractTablesFromExplainJSON(realWorldJSON)
	assert.NoError(t, err, "Should not error when parsing real-world JSON")

	assert.Equal(t, expected, got, "Should extract the correct table name from real-world JSON")
}
