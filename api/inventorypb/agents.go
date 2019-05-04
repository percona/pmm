package inventorypb

//go-sumtype:decl Agent

// Agent is a common interface for all types of Agents.
type Agent interface {
	sealedAgent() //nolint:unused

	// TODO Remove it.
	// Deprecated: use AgentId field instead.
	ID() string
}

func (*PMMAgent) sealedAgent()                {}
func (*NodeExporter) sealedAgent()            {}
func (*MySQLdExporter) sealedAgent()          {}
func (*RDSExporter) sealedAgent()             {}
func (*ExternalExporter) sealedAgent()        {}
func (*MongoDBExporter) sealedAgent()         {}
func (*QANMySQLPerfSchemaAgent) sealedAgent() {}
func (*QANMongoDBProfilerAgent) sealedAgent() {}
func (*QANMySQLSlowlogAgent) sealedAgent()    {}
func (*PostgresExporter) sealedAgent()        {}

func (m *PMMAgent) ID() string                { return m.AgentId }
func (m *NodeExporter) ID() string            { return m.AgentId }
func (m *MySQLdExporter) ID() string          { return m.AgentId }
func (m *RDSExporter) ID() string             { return m.AgentId }
func (m *ExternalExporter) ID() string        { return m.AgentId }
func (m *MongoDBExporter) ID() string         { return m.AgentId }
func (m *QANMySQLPerfSchemaAgent) ID() string { return m.AgentId }
func (m *QANMongoDBProfilerAgent) ID() string { return m.AgentId }
func (m *QANMySQLSlowlogAgent) ID() string    { return m.AgentId }
func (m *PostgresExporter) ID() string        { return m.AgentId }
