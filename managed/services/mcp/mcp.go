package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"

	actionsv1 "github.com/percona/pmm/api/actions/v1"
	qanpb "github.com/percona/pmm/api/qan/v1"
)

// availableQANColumns contains the comprehensive list of all available QAN metric columns
const availableQANColumns = "Core: load, num_queries, num_queries_with_errors, num_queries_with_warnings, query_time, lock_time, rows_sent, rows_examined, rows_affected, rows_read; " +
	"MySQL: merge_passes, innodb_io_r_ops, innodb_io_r_bytes, innodb_io_r_wait, innodb_rec_lock_wait, innodb_queue_wait, innodb_pages_distinct, query_length, bytes_sent, tmp_tables, tmp_disk_tables, tmp_table_sizes, qc_hit, full_scan, full_join, tmp_table, tmp_table_on_disk, filesort, filesort_on_disk, select_full_range_join, select_range, select_range_check, sort_range, sort_rows, sort_scan, no_index_used, no_good_index_used; " +
	"MongoDB: docs_returned, response_length, docs_scanned, docs_examined, keys_examined, locks_global_acquire_count_read_shared, locks_global_acquire_count_write_shared, locks_database_acquire_count_read_shared, locks_database_acquire_wait_count_read_shared, locks_database_time_acquiring_micros_read_shared, locks_collection_acquire_count_read_shared, storage_bytes_read, storage_time_reading_micros; " +
	"PostgreSQL: shared_blks_hit, shared_blks_read, shared_blks_dirtied, shared_blks_written, local_blks_hit, local_blks_read, local_blks_dirtied, local_blks_written, temp_blks_read, temp_blks_written, shared_blk_read_time, shared_blk_write_time, local_blk_read_time, local_blk_write_time, cpu_user_time, cpu_sys_time, plans_calls, wal_records, wal_fpi, wal_bytes, plan_time"

type Mcp struct {
	actionServer actionsv1.ActionsServiceServer
	qanClient    qanpb.QANServiceClient
	l            *logrus.Entry
}

func New(actionServer actionsv1.ActionsServiceServer, qanClient qanpb.QANServiceClient) *Mcp {
	return &Mcp{
		actionServer: actionServer,
		qanClient:    qanClient,
		l:            logrus.WithField("component", "mcp"),
	}
}

func (m *Mcp) Server() *server.MCPServer {
	// Implement server logic

	mcpServer := server.NewMCPServer(
		"pmm-mcp",
		"1.0.0",
		// server.WithResourceCapabilities(true, true),
		// server.WithPromptCapabilities(true),
		server.WithToolCapabilities(true),
		server.WithLogging(),
	)

	mcpServer.AddTool(mcpgo.NewTool("explain",
		mcpgo.WithDescription("Runs explain for the given query. Supports MySQL and MongoDB service types."),
		mcpgo.WithString("query_id",
			mcpgo.Description("Query ID"),
			mcpgo.Required(),
		),
		mcpgo.WithString("period_start",
			mcpgo.Description("Period start time (RFC3339 format, e.g., 2024-01-01T00:00:00Z). Optional, defaults to 24 hours ago"),
		),
		mcpgo.WithString("period_end",
			mcpgo.Description("Period end time (RFC3339 format, e.g., 2024-01-01T23:59:59Z). Optional, defaults to now"),
		),
		mcpgo.WithString("filters",
			mcpgo.Description("Additional filters in JSON format (e.g., {\"service_name\": [\"mysql-1\"], \"database\": [\"test\"]}). Optional"),
		),
	), mcpgo.NewTypedToolHandler(m.explain))

	mcpServer.AddTool(mcpgo.NewTool("show_tables_definition",
		mcpgo.WithDescription("Shows table DDL for all tables referenced in the query. Supports MySQL and PostgreSQL service types."),
		mcpgo.WithString("query_id",
			mcpgo.Description("Query ID"),
			mcpgo.Required(),
		),
		mcpgo.WithString("period_start",
			mcpgo.Description("Period start time (RFC3339 format). Optional, defaults to 24 hours ago"),
		),
		mcpgo.WithString("period_end",
			mcpgo.Description("Period end time (RFC3339 format). Optional, defaults to now"),
		),
		mcpgo.WithString("filters",
			mcpgo.Description("Additional filters in JSON format. Optional"),
		),
	), mcpgo.NewTypedToolHandler(m.ShowTables))

	mcpServer.AddTool(mcpgo.NewTool("show_index",
		mcpgo.WithDescription("Shows index details for all tables referenced in the query. Supports MySQL and PostgreSQL service types."),
		mcpgo.WithString("query_id",
			mcpgo.Description("Query ID"),
			mcpgo.Required(),
		),
		mcpgo.WithString("period_start",
			mcpgo.Description("Period start time (RFC3339 format). Optional, defaults to 24 hours ago"),
		),
		mcpgo.WithString("period_end",
			mcpgo.Description("Period end time (RFC3339 format). Optional, defaults to now"),
		),
		mcpgo.WithString("filters",
			mcpgo.Description("Additional filters in JSON format. Optional"),
		),
	), mcpgo.NewTypedToolHandler(m.ShowIndex))

	mcpServer.AddTool(mcpgo.NewTool("qan_get_report",
		mcpgo.WithDescription("Gets QAN metrics report grouped by queryid or other dimensions. Returns aggregated performance metrics for queries over a specified time period."),
		mcpgo.WithString("period_start",
			mcpgo.Description("Period start time (RFC3339 format, e.g., 2024-01-01T00:00:00Z). Required"),
			mcpgo.Required(),
		),
		mcpgo.WithString("period_end",
			mcpgo.Description("Period end time (RFC3339 format, e.g., 2024-01-01T23:59:59Z). Required"),
			mcpgo.Required(),
		),
		mcpgo.WithString("group_by",
			mcpgo.Description("Group by dimension (e.g., 'queryid', 'host', 'service_name'). Optional, defaults to 'queryid'"),
		),
		mcpgo.WithString("labels",
			mcpgo.Description("Labels/filters in JSON format (e.g., {\"service_name\": [\"mysql-1\"], \"database\": [\"test\"]}). Optional"),
		),
		mcpgo.WithString("columns",
			mcpgo.Description("Comma-separated list of metric columns to include. Required. Available columns: "+availableQANColumns),
			mcpgo.Required(),
		),
		mcpgo.WithString("order_by",
			mcpgo.Description("Order by metric. Required. Use column name for ascending order or prefix with '-' for descending (e.g., 'query_time' or '-query_time'). Available columns: "+availableQANColumns),
			mcpgo.Required(),
		),
		mcpgo.WithNumber("offset",
			mcpgo.Description("Pagination offset. Optional, defaults to 0"),
		),
		mcpgo.WithNumber("limit",
			mcpgo.Description("Number of results to return. Optional, defaults to 100"),
		),
		mcpgo.WithString("main_metric",
			mcpgo.Description("Main metric for calculations (e.g., 'query_time'). Optional. Available metrics: "+availableQANColumns),
		),
		mcpgo.WithString("search",
			mcpgo.Description("Search term to filter results. Optional"),
		),
	), mcpgo.NewTypedToolHandler(m.QANGetReport))

	mcpServer.AddTool(mcpgo.NewTool("qan_get_metrics",
		mcpgo.WithDescription("Gets detailed QAN metrics for a specific dimension value (e.g., specific query ID or host). Returns comprehensive performance metrics and statistics."),
		mcpgo.WithString("period_start",
			mcpgo.Description("Period start time (RFC3339 format, e.g., 2024-01-01T00:00:00Z). Required"),
			mcpgo.Required(),
		),
		mcpgo.WithString("period_end",
			mcpgo.Description("Period end time (RFC3339 format, e.g., 2024-01-01T23:59:59Z). Required"),
			mcpgo.Required(),
		),
		mcpgo.WithString("filter_by",
			mcpgo.Description("Filter by specific dimension value (e.g., query ID like '1D410B4BE5060972' or hostname). Required"),
			mcpgo.Required(),
		),
		mcpgo.WithString("group_by",
			mcpgo.Description("Group by dimension (e.g., 'queryid', 'host', 'service_name'). Required"),
			mcpgo.Required(),
		),
		mcpgo.WithString("labels",
			mcpgo.Description("Additional labels/filters in JSON format (e.g., {\"service_name\": [\"mysql-1\"]}). Optional"),
		),
		mcpgo.WithString("include_only_fields",
			mcpgo.Description("Comma-separated list of specific metric fields to include. Optional. Available fields: "+availableQANColumns),
		),
		mcpgo.WithBoolean("totals",
			mcpgo.Description("Include only totals, excluding N/A values. Optional, defaults to false"),
		),
	), mcpgo.NewTypedToolHandler(m.QANGetMetrics))

	return mcpServer
}

type explainArgs struct {
	QueryID     string `json:"query_id"`
	PeriodStart string `json:"period_start,omitempty"`
	PeriodEnd   string `json:"period_end,omitempty"`
	Filters     string `json:"filters,omitempty"`
}

type QueryMetadata struct {
	ServiceID   string
	ServiceType string
	PMMAgentID  string
	Database    string
	QueryText   string
	TableNames  []string
}

// QueryParams holds the common parameters for QAN queries
type QueryParams struct {
	QueryID     string
	PeriodStart time.Time
	PeriodEnd   time.Time
	Filters     map[string][]string
}

// parseQueryParams parses common query parameters from tool arguments
func parseQueryParams(queryID, periodStartStr, periodEndStr, filtersStr string) (*QueryParams, error) {
	params := &QueryParams{
		QueryID: queryID,
		Filters: make(map[string][]string),
	}

	// Parse time periods
	now := time.Now()
	if periodStartStr != "" {
		var err error
		params.PeriodStart, err = time.Parse(time.RFC3339, periodStartStr)
		if err != nil {
			return nil, fmt.Errorf("invalid period_start format: %w (expected RFC3339, e.g., 2024-01-01T00:00:00Z)", err)
		}
	} else {
		params.PeriodStart = now.Add(-24 * time.Hour) // Default: 24 hours ago
	}

	if periodEndStr != "" {
		var err error
		params.PeriodEnd, err = time.Parse(time.RFC3339, periodEndStr)
		if err != nil {
			return nil, fmt.Errorf("invalid period_end format: %w (expected RFC3339, e.g., 2024-01-01T23:59:59Z)", err)
		}
	} else {
		params.PeriodEnd = now // Default: now
	}

	// Validate time range
	if params.PeriodStart.After(params.PeriodEnd) {
		return nil, fmt.Errorf("period_start (%v) cannot be after period_end (%v)", params.PeriodStart, params.PeriodEnd)
	}

	// Parse filters
	if filtersStr != "" {
		if err := json.Unmarshal([]byte(filtersStr), &params.Filters); err != nil {
			return nil, fmt.Errorf("invalid filters JSON format: %w", err)
		}
	}

	return params, nil
}

// resolveQueryMetadata calls QAN APIs to get service details and query information from query_id
func (m *Mcp) resolveQueryMetadata(ctx context.Context, params *QueryParams) (*QueryMetadata, error) {
	// Get query examples to get service details
	exampleReq := &qanpb.GetQueryExampleRequest{
		PeriodStartFrom: timestamppb.New(params.PeriodStart),
		PeriodStartTo:   timestamppb.New(params.PeriodEnd),
		FilterBy:        params.QueryID,
		GroupBy:         "queryid",
		Limit:           10, // Get multiple examples to have more data
	}

	// Add filters to the request
	for key, values := range params.Filters {
		exampleReq.Labels = append(exampleReq.Labels, &qanpb.MapFieldEntry{
			Key:   key,
			Value: values,
		})
	}

	exampleResp, err := m.qanClient.GetQueryExample(ctx, exampleReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get query example for query_id %s: %w", params.QueryID, err)
	}

	if len(exampleResp.QueryExamples) == 0 {
		return nil, fmt.Errorf("no query examples found for query_id %s in the specified time period", params.QueryID)
	}

	example := exampleResp.QueryExamples[0]

	// Collect all unique table names from all examples
	tableNamesSet := make(map[string]bool)
	for _, ex := range exampleResp.QueryExamples {
		for _, table := range ex.Tables {
			if table != "" {
				tableNamesSet[table] = true
			}
		}
	}

	// Convert set to slice
	var allTableNames []string
	for tableName := range tableNamesSet {
		allTableNames = append(allTableNames, tableName)
	}

	// Extract service information from example
	metadata := &QueryMetadata{
		ServiceID:   example.ServiceId,
		ServiceType: example.ServiceType,
		Database:    example.Schema,
		QueryText:   example.Example,
		TableNames:  allTableNames,
	}

	m.l.WithFields(logrus.Fields{
		"query_id":     params.QueryID,
		"service_id":   metadata.ServiceID,
		"service_type": metadata.ServiceType,
		"database":     metadata.Database,
		"table_count":  len(metadata.TableNames),
		"tables":       metadata.TableNames,
		"period_start": params.PeriodStart,
		"period_end":   params.PeriodEnd,
		"filters":      params.Filters,
	}).Info("Resolved query metadata")

	return metadata, nil
}

func (m *Mcp) explain(ctx context.Context, req mcpgo.CallToolRequest, args explainArgs) (*mcpgo.CallToolResult, error) {
	m.l.WithField("args", args).Info("Received explain request")

	params, err := parseQueryParams(args.QueryID, args.PeriodStart, args.PeriodEnd, args.Filters)
	if err != nil {
		return nil, fmt.Errorf("failed to parse query parameters: %w", err)
	}

	metadata, err := m.resolveQueryMetadata(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve metadata for query_id %s: %w", args.QueryID, err)
	}

	var actionRequest actionsv1.StartServiceActionRequest
	switch metadata.ServiceType {
	case "mysql":
		// MySQL specific logic
		actionRequest = actionsv1.StartServiceActionRequest{
			Action: &actionsv1.StartServiceActionRequest_MysqlExplain{
				MysqlExplain: &actionsv1.StartMySQLExplainActionParams{
					PmmAgentId: metadata.PMMAgentID, // This might be empty, action server should resolve it
					ServiceId:  metadata.ServiceID,
					Database:   metadata.Database,
					QueryId:    args.QueryID,
				},
			},
		}
	case "mongodb":
		// MongoDB specific logic
		actionRequest = actionsv1.StartServiceActionRequest{
			Action: &actionsv1.StartServiceActionRequest_MongodbExplain{
				MongodbExplain: &actionsv1.StartMongoDBExplainActionParams{
					PmmAgentId: metadata.PMMAgentID, // This might be empty, action server should resolve it
					ServiceId:  metadata.ServiceID,
					Query:      metadata.QueryText,
				},
			},
		}
	default:
		return nil, fmt.Errorf("unsupported service type: %s", metadata.ServiceType)
	}

	action, err := m.actionServer.StartServiceAction(ctx, &actionRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to start action: %w", err)
	}

	var actionID string
	switch metadata.ServiceType {
	case "mysql":
		actionID = action.GetMysqlExplain().ActionId
	case "mongodb":
		actionID = action.GetMongodbExplain().ActionId
	}

	m.l.WithFields(logrus.Fields{
		"query_id":     args.QueryID,
		"action_id":    actionID,
		"service_type": metadata.ServiceType,
		"period_start": params.PeriodStart,
		"period_end":   params.PeriodEnd,
		"filters":      params.Filters,
	}).Info("Started explain action")

	return m.responseActionOutput(ctx, actionID)
}

func (m *Mcp) responseActionOutput(ctx context.Context, actionID string) (*mcpgo.CallToolResult, error) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			action, err := m.actionServer.GetAction(ctx, &actionsv1.GetActionRequest{
				ActionId: actionID,
			})
			if err != nil {
				m.l.WithField("action_id", actionID).Errorf("Failed to get action: %v", err)
				return nil, err
			}
			if action.Done {
				m.l.WithField("action_id", actionID).Infof("Action completed with output: %s", action.Output)
				if action.Error != "" {
					return nil, fmt.Errorf("action failed: %s", action.Error)
				}
				return &mcpgo.CallToolResult{
					Content: []mcpgo.Content{
						mcpgo.TextContent{
							Type: "text",
							Text: action.Output,
						},
					},
				}, nil
			}
		case <-ctx.Done():
			m.l.WithField("action_id", actionID).Info("Context done, stopping action polling")
			return nil, ctx.Err()
		}
	}
}

type showTablesArgs struct {
	QueryID     string `json:"query_id"`
	PeriodStart string `json:"period_start,omitempty"`
	PeriodEnd   string `json:"period_end,omitempty"`
	Filters     string `json:"filters,omitempty"`
}

// actionType represents the type of database action to perform
type actionType string

const (
	actionShowTables actionType = "show_tables"
	actionShowIndex  actionType = "show_index"
)

// processTablesAction is a helper method that processes tables for different action types
func (m *Mcp) processTablesAction(ctx context.Context, queryID, periodStart, periodEnd, filters string, action actionType, logMessage, resultPrefix string) (*mcpgo.CallToolResult, error) {
	params, err := parseQueryParams(queryID, periodStart, periodEnd, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to parse query parameters: %w", err)
	}

	metadata, err := m.resolveQueryMetadata(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve metadata for query_id %s: %w", queryID, err)
	}

	if len(metadata.TableNames) == 0 {
		return nil, fmt.Errorf("no tables found for query_id %s", queryID)
	}

	m.l.WithFields(logrus.Fields{
		"query_id":    queryID,
		"table_count": len(metadata.TableNames),
		"tables":      metadata.TableNames,
	}).Info(logMessage)

	// Prepare to collect results for all tables
	var allResults []string
	var allErrors []string

	// Process each table
	for _, tableName := range metadata.TableNames {
		var actionRequest actionsv1.StartServiceActionRequest
		switch metadata.ServiceType {
		case "mysql":
			switch action {
			case actionShowTables:
				actionRequest = actionsv1.StartServiceActionRequest{
					Action: &actionsv1.StartServiceActionRequest_MysqlShowCreateTable{
						MysqlShowCreateTable: &actionsv1.StartMySQLShowCreateTableActionParams{
							PmmAgentId: metadata.PMMAgentID,
							ServiceId:  metadata.ServiceID,
							TableName:  tableName,
							Database:   metadata.Database,
						},
					},
				}
			case actionShowIndex:
				actionRequest = actionsv1.StartServiceActionRequest{
					Action: &actionsv1.StartServiceActionRequest_MysqlShowIndex{
						MysqlShowIndex: &actionsv1.StartMySQLShowIndexActionParams{
							PmmAgentId: metadata.PMMAgentID,
							ServiceId:  metadata.ServiceID,
							TableName:  tableName,
							Database:   metadata.Database,
						},
					},
				}
			}
		case "postgresql":
			switch action {
			case actionShowTables:
				actionRequest = actionsv1.StartServiceActionRequest{
					Action: &actionsv1.StartServiceActionRequest_PostgresShowCreateTable{
						PostgresShowCreateTable: &actionsv1.StartPostgreSQLShowCreateTableActionParams{
							PmmAgentId: metadata.PMMAgentID,
							ServiceId:  metadata.ServiceID,
							TableName:  tableName,
							Database:   metadata.Database,
						},
					},
				}
			case actionShowIndex:
				actionRequest = actionsv1.StartServiceActionRequest{
					Action: &actionsv1.StartServiceActionRequest_PostgresShowIndex{
						PostgresShowIndex: &actionsv1.StartPostgreSQLShowIndexActionParams{
							PmmAgentId: metadata.PMMAgentID,
							ServiceId:  metadata.ServiceID,
							TableName:  tableName,
							Database:   metadata.Database,
						},
					},
				}
			}
		default:
			allErrors = append(allErrors, fmt.Sprintf("Table %s: unsupported service type: %s", tableName, metadata.ServiceType))
			continue
		}

		resp, err := m.actionServer.StartServiceAction(ctx, &actionRequest)
		if err != nil {
			allErrors = append(allErrors, fmt.Sprintf("Table %s: failed to start action: %v", tableName, err))
			continue
		}

		// Extract action ID from response
		var actionID string
		switch metadata.ServiceType {
		case "mysql":
			switch action {
			case actionShowTables:
				actionID = resp.GetMysqlShowCreateTable().ActionId
			case actionShowIndex:
				actionID = resp.GetMysqlShowIndex().ActionId
			}
		case "postgresql":
			switch action {
			case actionShowTables:
				actionID = resp.GetPostgresqlShowCreateTable().ActionId
			case actionShowIndex:
				actionID = resp.GetPostgresqlShowIndex().ActionId
			}
		}

		result := fmt.Sprintf("=== Table: %s ===\nAction ID: %s\nService: %s (%s)\nDatabase: %s\n",
			tableName, actionID, metadata.ServiceID, metadata.ServiceType, metadata.Database)
		allResults = append(allResults, result)
	}

	// Combine all results
	finalResult := fmt.Sprintf("%s for query_id: %s\nTotal tables: %d\nPeriod: %s to %s\n\n",
		resultPrefix, queryID, len(metadata.TableNames), params.PeriodStart.Format(time.RFC3339), params.PeriodEnd.Format(time.RFC3339))

	if len(params.Filters) > 0 {
		filtersJson, err := json.Marshal(params.Filters)
		if err != nil {
			m.l.WithError(err).Error("Failed to marshal filters for result output")
			return nil, fmt.Errorf("failed to marshal filters: %w", err)
		}
		finalResult += fmt.Sprintf("Filters: %s\n\n", string(filtersJson))
	}

	if len(allResults) > 0 {
		finalResult += "Successfully started actions:\n" + strings.Join(allResults, "\n")
	}

	if len(allErrors) > 0 {
		finalResult += "\nErrors encountered:\n"
		for _, errMsg := range allErrors {
			finalResult += fmt.Sprintf("- %s\n", errMsg)
		}
	}

	return &mcpgo.CallToolResult{
		Content: []mcpgo.Content{
			mcpgo.NewTextContent(finalResult),
		},
	}, nil
}

func (m *Mcp) ShowTables(ctx context.Context, request mcpgo.CallToolRequest, args showTablesArgs) (*mcpgo.CallToolResult, error) {
	m.l.WithField("args", args).Info("Received show tables request")

	return m.processTablesAction(ctx, args.QueryID, args.PeriodStart, args.PeriodEnd, args.Filters,
		actionShowTables, "Getting DDL for all tables", "Table DDL requests")
}

type showIndexArgs struct {
	QueryID     string `json:"query_id"`
	PeriodStart string `json:"period_start,omitempty"`
	PeriodEnd   string `json:"period_end,omitempty"`
	Filters     string `json:"filters,omitempty"`
}

func (m *Mcp) ShowIndex(ctx context.Context, request mcpgo.CallToolRequest, args showIndexArgs) (*mcpgo.CallToolResult, error) {
	m.l.WithField("args", args).Info("Received show index request")

	return m.processTablesAction(ctx, args.QueryID, args.PeriodStart, args.PeriodEnd, args.Filters,
		actionShowIndex, "Getting indexes for all tables", "Index details requests")
}

type qanGetReportArgs struct {
	PeriodStart string `json:"period_start"`
	PeriodEnd   string `json:"period_end"`
	GroupBy     string `json:"group_by,omitempty"`
	Labels      string `json:"labels,omitempty"`
	Columns     string `json:"columns,omitempty"`
	OrderBy     string `json:"order_by,omitempty"`
	Offset      int    `json:"offset,omitempty"`
	Limit       int    `json:"limit,omitempty"`
	MainMetric  string `json:"main_metric,omitempty"`
	Search      string `json:"search,omitempty"`
}

func (m *Mcp) QANGetReport(ctx context.Context, request mcpgo.CallToolRequest, args qanGetReportArgs) (*mcpgo.CallToolResult, error) {
	m.l.WithField("args", args).Info("Received QAN get report request")

	// Validate required fields
	if args.Columns == "" {
		return nil, fmt.Errorf("columns parameter is required")
	}
	if args.OrderBy == "" {
		return nil, fmt.Errorf("order_by parameter is required")
	}

	// Parse time periods
	periodStart, err := time.Parse(time.RFC3339, args.PeriodStart)
	if err != nil {
		return nil, fmt.Errorf("invalid period_start format: %w (expected RFC3339, e.g., 2024-01-01T00:00:00Z)", err)
	}

	periodEnd, err := time.Parse(time.RFC3339, args.PeriodEnd)
	if err != nil {
		return nil, fmt.Errorf("invalid period_end format: %w (expected RFC3339, e.g., 2024-01-01T23:59:59Z)", err)
	}

	// Validate time range
	if periodStart.After(periodEnd) {
		return nil, fmt.Errorf("period_start (%v) cannot be after period_end (%v)", periodStart, periodEnd)
	}

	// Build QAN request
	// Note: OrderBy supports '-' prefix for descending order (e.g., '-query_time')
	// The QAN service will handle this prefix internally
	qanReq := &qanpb.GetReportRequest{
		PeriodStartFrom: timestamppb.New(periodStart),
		PeriodStartTo:   timestamppb.New(periodEnd),
		GroupBy:         args.GroupBy,
		OrderBy:         args.OrderBy,
		Offset:          uint32(args.Offset),
		Limit:           uint32(args.Limit),
		MainMetric:      args.MainMetric,
		Search:          args.Search,
	}

	// Set default group_by if not provided
	if qanReq.GroupBy == "" {
		qanReq.GroupBy = "queryid"
	}

	// Set default limit if not provided
	if qanReq.Limit == 0 {
		qanReq.Limit = 100
	}

	// Parse labels if provided
	if args.Labels != "" {
		var labelsMap map[string][]string
		if err := json.Unmarshal([]byte(args.Labels), &labelsMap); err != nil {
			return nil, fmt.Errorf("invalid labels JSON format: %w", err)
		}

		for key, values := range labelsMap {
			qanReq.Labels = append(qanReq.Labels, &qanpb.ReportMapFieldEntry{
				Key:   key,
				Value: values,
			})
		}
	}

	// Parse columns if provided
	if args.Columns != "" {
		qanReq.Columns = strings.Split(args.Columns, ",")
		// Trim whitespace from column names
		for i, col := range qanReq.Columns {
			qanReq.Columns[i] = strings.TrimSpace(col)
		}
	}

	m.l.WithFields(logrus.Fields{
		"period_start": periodStart,
		"period_end":   periodEnd,
		"group_by":     qanReq.GroupBy,
		"order_by":     args.OrderBy,
		"limit":        qanReq.Limit,
		"offset":       qanReq.Offset,
		"columns":      len(qanReq.Columns),
		"labels":       len(qanReq.Labels),
	}).Info("Calling QAN GetReport")

	// Call QAN service
	resp, err := m.qanClient.GetReport(ctx, qanReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get QAN report: %w", err)
	}

	// Format response
	result := fmt.Sprintf("QAN Metrics Report\n==================\n\n")
	result += fmt.Sprintf("Period: %s to %s\n", periodStart.Format(time.RFC3339), periodEnd.Format(time.RFC3339))
	result += fmt.Sprintf("Group By: %s\n", qanReq.GroupBy)
	result += fmt.Sprintf("Order By: %s\n", args.OrderBy)
	result += fmt.Sprintf("Columns: %s\n", args.Columns)
	result += fmt.Sprintf("Total Rows: %d (showing %d-%d)\n\n", resp.TotalRows, resp.Offset+1, resp.Offset+uint32(len(resp.Rows)))

	if len(resp.Rows) == 0 {
		result += "No data found for the specified criteria.\n"
	} else {
		for i, row := range resp.Rows {
			result += fmt.Sprintf("Row %d (Rank: %d):\n", i+1, row.Rank)
			result += fmt.Sprintf("  Dimension: %s\n", row.Dimension)
			if row.Database != "" {
				result += fmt.Sprintf("  Database: %s\n", row.Database)
			}
			result += fmt.Sprintf("  Queries: %d (QPS: %.2f, Load: %.2f)\n", row.NumQueries, row.Qps, row.Load)

			if len(row.Metrics) > 0 {
				result += "  Metrics:\n"
				for metricName, metric := range row.Metrics {
					stats := metric.Stats
					result += fmt.Sprintf("    %s: avg=%.3fs, sum=%.3fs, cnt=%.0f, min=%.3fs, max=%.3fs, p99=%.3fs\n",
						metricName, stats.Avg, stats.Sum, stats.Cnt, stats.Min, stats.Max, stats.P99)
				}
			}

			if row.Fingerprint != "" {
				result += fmt.Sprintf("  Fingerprint: %s\n", row.Fingerprint)
			}
			result += "\n"
		}
	}

	return &mcpgo.CallToolResult{
		Content: []mcpgo.Content{
			mcpgo.NewTextContent(result),
		},
	}, nil
}

type qanGetMetricsArgs struct {
	PeriodStart       string `json:"period_start"`
	PeriodEnd         string `json:"period_end"`
	FilterBy          string `json:"filter_by"`
	GroupBy           string `json:"group_by"`
	Labels            string `json:"labels,omitempty"`
	IncludeOnlyFields string `json:"include_only_fields,omitempty"`
	Totals            bool   `json:"totals,omitempty"`
}

func (m *Mcp) QANGetMetrics(ctx context.Context, request mcpgo.CallToolRequest, args qanGetMetricsArgs) (*mcpgo.CallToolResult, error) {
	m.l.WithField("args", args).Info("Received QAN get metrics request")

	// Parse time periods
	periodStart, err := time.Parse(time.RFC3339, args.PeriodStart)
	if err != nil {
		return nil, fmt.Errorf("invalid period_start format: %w (expected RFC3339, e.g., 2024-01-01T00:00:00Z)", err)
	}

	periodEnd, err := time.Parse(time.RFC3339, args.PeriodEnd)
	if err != nil {
		return nil, fmt.Errorf("invalid period_end format: %w (expected RFC3339, e.g., 2024-01-01T23:59:59Z)", err)
	}

	// Validate time range
	if periodStart.After(periodEnd) {
		return nil, fmt.Errorf("period_start (%v) cannot be after period_end (%v)", periodStart, periodEnd)
	}

	// Build QAN request
	qanReq := &qanpb.GetMetricsRequest{
		PeriodStartFrom: timestamppb.New(periodStart),
		PeriodStartTo:   timestamppb.New(periodEnd),
		FilterBy:        args.FilterBy,
		GroupBy:         args.GroupBy,
		Totals:          args.Totals,
	}

	// Parse labels if provided
	if args.Labels != "" {
		var labelsMap map[string][]string
		if err := json.Unmarshal([]byte(args.Labels), &labelsMap); err != nil {
			return nil, fmt.Errorf("invalid labels JSON format: %w", err)
		}

		for key, values := range labelsMap {
			qanReq.Labels = append(qanReq.Labels, &qanpb.MapFieldEntry{
				Key:   key,
				Value: values,
			})
		}
	}

	// Parse include_only_fields if provided
	if args.IncludeOnlyFields != "" {
		qanReq.IncludeOnlyFields = strings.Split(args.IncludeOnlyFields, ",")
		// Trim whitespace from field names
		for i, field := range qanReq.IncludeOnlyFields {
			qanReq.IncludeOnlyFields[i] = strings.TrimSpace(field)
		}
	}

	m.l.WithFields(logrus.Fields{
		"period_start":        periodStart,
		"period_end":          periodEnd,
		"filter_by":           args.FilterBy,
		"group_by":            args.GroupBy,
		"totals":              args.Totals,
		"include_only_fields": len(qanReq.IncludeOnlyFields),
		"labels":              len(qanReq.Labels),
	}).Info("Calling QAN GetMetrics")

	// Call QAN service
	resp, err := m.qanClient.GetMetrics(ctx, qanReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get QAN metrics: %w", err)
	}

	// Format response
	result := fmt.Sprintf("QAN Detailed Metrics\n=====================\n\n")
	result += fmt.Sprintf("Period: %s to %s\n", periodStart.Format(time.RFC3339), periodEnd.Format(time.RFC3339))
	result += fmt.Sprintf("Filter By: %s (%s)\n", args.FilterBy, args.GroupBy)
	if args.Totals {
		result += "Mode: Totals only\n"
	}
	result += "\n"

	// Display metrics
	if len(resp.Metrics) > 0 {
		result += "Performance Metrics:\n"
		for metricName, metricValues := range resp.Metrics {
			result += fmt.Sprintf("  %s:\n", metricName)
			result += fmt.Sprintf("    Rate: %.3f/sec\n", metricValues.Rate)
			result += fmt.Sprintf("    Count: %.0f\n", metricValues.Cnt)
			result += fmt.Sprintf("    Sum: %.3f\n", metricValues.Sum)
			result += fmt.Sprintf("    Average: %.3f\n", metricValues.Avg)
			result += fmt.Sprintf("    Min: %.3f\n", metricValues.Min)
			result += fmt.Sprintf("    Max: %.3f\n", metricValues.Max)
			result += fmt.Sprintf("    P99: %.3f\n", metricValues.P99)
			if metricValues.PercentOfTotal > 0 {
				result += fmt.Sprintf("    Percent of Total: %.2f%%\n", metricValues.PercentOfTotal)
			}
			result += "\n"
		}
	}

	// Display text metrics
	if len(resp.TextMetrics) > 0 {
		result += "Text Metrics:\n"
		for key, value := range resp.TextMetrics {
			result += fmt.Sprintf("  %s: %s\n", key, value)
		}
		result += "\n"
	}

	// Display totals if available
	if len(resp.Totals) > 0 {
		result += "Totals:\n"
		for metricName, metricValues := range resp.Totals {
			result += fmt.Sprintf("  %s: sum=%.3f, avg=%.3f, cnt=%.0f\n",
				metricName, metricValues.Sum, metricValues.Avg, metricValues.Cnt)
		}
		result += "\n"
	}

	// Display fingerprint if available
	if resp.Fingerprint != "" {
		result += fmt.Sprintf("Query Fingerprint:\n%s\n\n", resp.Fingerprint)
	}

	// Display metadata if available
	if resp.Metadata != nil {
		result += "Metadata:\n"
		if resp.Metadata.ServiceName != "" {
			result += fmt.Sprintf("  Service: %s\n", resp.Metadata.ServiceName)
		}
		if resp.Metadata.ServiceType != "" {
			result += fmt.Sprintf("  Service Type: %s\n", resp.Metadata.ServiceType)
		}
		if resp.Metadata.Database != "" {
			result += fmt.Sprintf("  Database: %s\n", resp.Metadata.Database)
		}
		if resp.Metadata.NodeName != "" {
			result += fmt.Sprintf("  Node: %s\n", resp.Metadata.NodeName)
		}
		if resp.Metadata.NodeType != "" {
			result += fmt.Sprintf("  Node Type: %s\n", resp.Metadata.NodeType)
		}
	}

	// Display sparkline info if available
	if len(resp.Sparkline) > 0 {
		result += fmt.Sprintf("Sparkline Data Points: %d\n", len(resp.Sparkline))
	}

	return &mcpgo.CallToolResult{
		Content: []mcpgo.Content{
			mcpgo.NewTextContent(result),
		},
	}, nil
}
