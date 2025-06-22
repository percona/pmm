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

	actionsv1 "github.com/percona/pmm/api/actions/v1"
	qanpb "github.com/percona/pmm/api/qan/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

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
	ticker := time.Tick(1 * time.Second)
	for {
		select {
		case <-ticker:
			action, err := m.actionServer.GetAction(ctx, &actionsv1.GetActionRequest{
				ActionId: actionID,
			})
			if err != nil {
				m.l.WithField("action_id", actionID).Errorf("Failed to get action: %v", err)
				return nil, err
			}
			if action.Done {
				m.l.WithField("action_id", actionID).Infof("Action completed with output: %s", action.Output)
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

func (m *Mcp) ShowTables(ctx context.Context, request mcpgo.CallToolRequest, args showTablesArgs) (*mcpgo.CallToolResult, error) {
	m.l.WithField("args", args).Info("Received show tables request")

	params, err := parseQueryParams(args.QueryID, args.PeriodStart, args.PeriodEnd, args.Filters)
	if err != nil {
		return nil, fmt.Errorf("failed to parse query parameters: %w", err)
	}

	metadata, err := m.resolveQueryMetadata(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve metadata for query_id %s: %w", args.QueryID, err)
	}

	// For show tables, we need to extract table names from the query metadata
	if len(metadata.TableNames) == 0 {
		return nil, fmt.Errorf("no tables found for query_id %s", args.QueryID)
	}

	m.l.WithFields(logrus.Fields{
		"query_id":    args.QueryID,
		"table_count": len(metadata.TableNames),
		"tables":      metadata.TableNames,
	}).Info("Getting DDL for all tables")

	// Prepare to collect results for all tables
	var allResults []string
	var allErrors []string

	// Process each table
	for _, tableName := range metadata.TableNames {
		var actionRequest actionsv1.StartServiceActionRequest
		switch metadata.ServiceType {
		case "mysql":
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
		case "postgresql":
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
			actionID = resp.GetMysqlShowCreateTable().ActionId
		case "postgresql":
			actionID = resp.GetPostgresqlShowCreateTable().ActionId
		}

		result := fmt.Sprintf("=== Table: %s ===\nAction ID: %s\nService: %s (%s)\nDatabase: %s\n",
			tableName, actionID, metadata.ServiceID, metadata.ServiceType, metadata.Database)
		allResults = append(allResults, result)
	}

	// Combine all results
	finalResult := fmt.Sprintf("Table DDL requests for query_id: %s\nTotal tables: %d\nPeriod: %s to %s\n\n",
		args.QueryID, len(metadata.TableNames), params.PeriodStart.Format(time.RFC3339), params.PeriodEnd.Format(time.RFC3339))

	if len(params.Filters) > 0 {
		filtersJson, _ := json.Marshal(params.Filters)
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

type showIndexArgs struct {
	QueryID     string `json:"query_id"`
	PeriodStart string `json:"period_start,omitempty"`
	PeriodEnd   string `json:"period_end,omitempty"`
	Filters     string `json:"filters,omitempty"`
}

func (m *Mcp) ShowIndex(ctx context.Context, request mcpgo.CallToolRequest, args showIndexArgs) (*mcpgo.CallToolResult, error) {
	m.l.WithField("args", args).Info("Received show index request")

	params, err := parseQueryParams(args.QueryID, args.PeriodStart, args.PeriodEnd, args.Filters)
	if err != nil {
		return nil, fmt.Errorf("failed to parse query parameters: %w", err)
	}

	metadata, err := m.resolveQueryMetadata(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve metadata for query_id %s: %w", args.QueryID, err)
	}

	// For show index, we need to extract table names from the query metadata
	if len(metadata.TableNames) == 0 {
		return nil, fmt.Errorf("no tables found for query_id %s", args.QueryID)
	}

	m.l.WithFields(logrus.Fields{
		"query_id":    args.QueryID,
		"table_count": len(metadata.TableNames),
		"tables":      metadata.TableNames,
	}).Info("Getting indexes for all tables")

	// Prepare to collect results for all tables
	var allResults []string
	var allErrors []string

	// Process each table
	for _, tableName := range metadata.TableNames {
		var actionRequest actionsv1.StartServiceActionRequest
		switch metadata.ServiceType {
		case "mysql":
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
		case "postgresql":
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
			actionID = resp.GetMysqlShowIndex().ActionId
		case "postgresql":
			actionID = resp.GetPostgresqlShowIndex().ActionId
		}

		result := fmt.Sprintf("=== Table: %s ===\nAction ID: %s\nService: %s (%s)\nDatabase: %s\n",
			tableName, actionID, metadata.ServiceID, metadata.ServiceType, metadata.Database)
		allResults = append(allResults, result)
	}

	// Combine all results
	finalResult := fmt.Sprintf("Index details requests for query_id: %s\nTotal tables: %d\nPeriod: %s to %s\n\n",
		args.QueryID, len(metadata.TableNames), params.PeriodStart.Format(time.RFC3339), params.PeriodEnd.Format(time.RFC3339))

	if len(params.Filters) > 0 {
		filtersJson, _ := json.Marshal(params.Filters)
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
