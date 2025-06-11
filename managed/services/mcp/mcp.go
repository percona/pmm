package mcp

import (
	"context"
	"fmt"
	"time"

	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/sirupsen/logrus"

	actionsv1 "github.com/percona/pmm/api/actions/v1"
)

type Mcp struct {
	actionServer actionsv1.ActionsServiceServer
	l            *logrus.Entry
}

func New(actionServer actionsv1.ActionsServiceServer) *Mcp {
	return &Mcp{
		actionServer: actionServer,
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
		mcpgo.WithDescription("Runs explain for the given query"),
		mcpgo.WithString("service_type",
			mcpgo.Description("Service type"),
			mcpgo.Enum("mysql", "mongodb"),
			mcpgo.Required(),
		),
		mcpgo.WithString("pmm_agent_id",
			mcpgo.Description("PMM Agent ID"),
		),
		mcpgo.WithString("service_id",
			mcpgo.Description("Service ID"),
			mcpgo.Required(),
		),
		mcpgo.WithString("query_id",
			mcpgo.Description("Query ID"),
			mcpgo.Required(),
		),
		mcpgo.WithString("query",
			mcpgo.Description("Query"),
			mcpgo.Required(),
		),
		mcpgo.WithString("database",
			mcpgo.Description("Database name"),
		),
	), mcpgo.NewTypedToolHandler(m.explain))

	mcpServer.AddTool(mcpgo.NewTool("show_tables_definition",
		mcpgo.WithDescription("Shows table DDL query"),
		mcpgo.WithString("service_type",
			mcpgo.Description("Service type"),
			mcpgo.Enum("mysql", "postgresql"),
			mcpgo.Required(),
		),
		mcpgo.WithString("pmm_agent_id",
			mcpgo.Description("PMM Agent ID"),
		),
		mcpgo.WithString("service_id",
			mcpgo.Description("Service ID"),
			mcpgo.Required(),
		),
		mcpgo.WithString("table_name",
			mcpgo.Description("Table name"),
			mcpgo.Required(),
		),
		mcpgo.WithString("database",
			mcpgo.Description("Database name"),
			mcpgo.Required(),
		),
	), mcpgo.NewTypedToolHandler(m.ShowTables))

	mcpServer.AddTool(mcpgo.NewTool("show_index",
		mcpgo.WithDescription("Shows index details for a given table"),
		mcpgo.WithString("service_type",
			mcpgo.Description("Service type"),
			mcpgo.Enum("mysql", "postgresql"),
			mcpgo.Required(),
		),
		mcpgo.WithString("pmm_agent_id",
			mcpgo.Description("PMM Agent ID"),
		),
		mcpgo.WithString("service_id",
			mcpgo.Description("Service ID"),
			mcpgo.Required(),
		),
		mcpgo.WithString("table_name",
			mcpgo.Description("Table name"),
			mcpgo.Required(),
		),
		mcpgo.WithString("database",
			mcpgo.Description("Database name"),
			mcpgo.Required(),
		),
	), mcpgo.NewTypedToolHandler(m.ShowIndex))
	return mcpServer
}

type explainArgs struct {
	PMMAgentID  string `json:"pmm_agent_id"`
	ServiceType string `json:"service_type"`
	ServiceID   string `json:"service_id"`
	Database    string `json:"database"`
	QueryID     string `json:"query_id"`
	Query       string `json:"query"`
}

func (m *Mcp) explain(ctx context.Context, req mcpgo.CallToolRequest, args explainArgs) (*mcpgo.CallToolResult, error) {
	m.l.WithField("args", args).Info("Received explain request")
	var actionRequest actionsv1.StartServiceActionRequest
	switch args.ServiceType {
	case "mysql":
		// MySQL specific logic
		actionRequest = actionsv1.StartServiceActionRequest{
			Action: &actionsv1.StartServiceActionRequest_MysqlExplain{
				MysqlExplain: &actionsv1.StartMySQLExplainActionParams{
					PmmAgentId: args.PMMAgentID,
					ServiceId:  args.ServiceID,
					Database:   args.Database,
					QueryId:    args.QueryID,
				},
			},
		}
	case "mongodb":
		// MongoDB specific logic
		actionRequest = actionsv1.StartServiceActionRequest{
			Action: &actionsv1.StartServiceActionRequest_MongodbExplain{
				MongodbExplain: &actionsv1.StartMongoDBExplainActionParams{
					PmmAgentId: args.PMMAgentID,
					ServiceId:  args.ServiceID,
					Query:      args.Query,
				},
			},
		}
	default:
		return nil, fmt.Errorf("unsupported service type: %s", args.ServiceType)
	}
	action, err := m.actionServer.StartServiceAction(ctx, &actionRequest)
	if err != nil {
		return nil, err
	}
	var actionID string
	switch args.ServiceType {
	case "mysql":
		actionID = action.GetMysqlExplain().ActionId
	case "mongodb":
		actionID = action.GetMongodbExplain().ActionId
	}
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
	PMMAgentID  string `json:"pmm_agent_id"`
	ServiceType string `json:"service_type"`
	ServiceID   string `json:"service_id"`
	Database    string `json:"database"`
	TableName   string `json:"table_name"`
}

func (m *Mcp) ShowTables(ctx context.Context, request mcpgo.CallToolRequest, args showTablesArgs) (*mcpgo.CallToolResult, error) {
	m.l.WithField("args", args).Info("Received show tables request")
	var actionRequest actionsv1.StartServiceActionRequest
	switch args.ServiceType {
	case "mysql":
		// MySQL specific logic
		actionRequest = actionsv1.StartServiceActionRequest{
			Action: &actionsv1.StartServiceActionRequest_MysqlShowCreateTable{
				MysqlShowCreateTable: &actionsv1.StartMySQLShowCreateTableActionParams{
					PmmAgentId: args.PMMAgentID,
					ServiceId:  args.ServiceID,
					Database:   args.Database,
					TableName:  args.TableName,
				},
			},
		}
	case "postgresql":
		// MongoDB specific logic
		actionRequest = actionsv1.StartServiceActionRequest{
			Action: &actionsv1.StartServiceActionRequest_PostgresShowCreateTable{
				PostgresShowCreateTable: &actionsv1.StartPostgreSQLShowCreateTableActionParams{
					PmmAgentId: args.PMMAgentID,
					ServiceId:  args.ServiceID,
					TableName:  args.TableName,
					Database:   args.Database,
				},
			},
		}
	}

	action, err := m.actionServer.StartServiceAction(ctx, &actionRequest)
	if err != nil {
		return nil, err
	}
	var actionID string
	switch args.ServiceType {
	case "mysql":
		actionID = action.GetMysqlShowCreateTable().ActionId
	case "postgresql":
		actionID = action.GetPostgresqlShowCreateTable().ActionId
	}
	return m.responseActionOutput(ctx, actionID)
}

type showIndexArgs struct {
	PMMAgentID  string `json:"pmm_agent_id"`
	ServiceType string `json:"service_type"`
	ServiceID   string `json:"service_id"`
	TableName   string `json:"table_name"`
	Database    string `json:"database"`
}

func (m *Mcp) ShowIndex(ctx context.Context, request mcpgo.CallToolRequest, args showIndexArgs) (*mcpgo.CallToolResult, error) {
	m.l.WithField("args", args).Info("Received show index request")

	var actionRequest actionsv1.StartServiceActionRequest
	switch args.ServiceType {
	case "mysql":
		// MySQL specific logic
		actionRequest = actionsv1.StartServiceActionRequest{
			Action: &actionsv1.StartServiceActionRequest_MysqlShowIndex{
				MysqlShowIndex: &actionsv1.StartMySQLShowIndexActionParams{
					PmmAgentId: args.PMMAgentID,
					ServiceId:  args.ServiceID,
					TableName:  args.TableName,
					Database:   args.Database,
				},
			},
		}
	case "postgresql":
		// PostgreSQL specific logic
		actionRequest = actionsv1.StartServiceActionRequest{
			Action: &actionsv1.StartServiceActionRequest_PostgresShowIndex{
				PostgresShowIndex: &actionsv1.StartPostgreSQLShowIndexActionParams{
					PmmAgentId: args.PMMAgentID,
					ServiceId:  args.ServiceID,
					TableName:  args.TableName,
					Database:   args.Database,
				},
			},
		}
	}

	action, err := m.actionServer.StartServiceAction(ctx, &actionRequest)
	if err != nil {
		return nil, err
	}
	var actionID string
	switch args.ServiceType {
	case "mysql":
		actionID = action.GetMysqlShowIndex().ActionId
	case "postgresql":
		actionID = action.GetPostgresqlShowIndex().ActionId
	}
	return m.responseActionOutput(ctx, actionID)
}
