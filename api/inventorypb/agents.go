package inventorypb

// Agent is a common interface for all types of Agents.
type Agent interface {
	agent()

	ID() string
}

func (*PMMAgent) agent()                {}
func (*NodeExporter) agent()            {}
func (*MySQLdExporter) agent()          {}
func (*RDSExporter) agent()             {}
func (*ExternalExporter) agent()        {}
func (*MongoDBExporter) agent()         {}
func (*QANMySQLPerfSchemaAgent) agent() {}
func (*QANMongoDBProfilerAgent) agent() {}
func (*QANMySQLSlowlogAgent) agent()    {}
func (*PostgresExporter) agent()        {}

func (a *PMMAgent) ID() string                { return a.AgentId }
func (a *NodeExporter) ID() string            { return a.AgentId }
func (a *MySQLdExporter) ID() string          { return a.AgentId }
func (a *RDSExporter) ID() string             { return a.AgentId }
func (a *ExternalExporter) ID() string        { return a.AgentId }
func (a *MongoDBExporter) ID() string         { return a.AgentId }
func (a *QANMySQLPerfSchemaAgent) ID() string { return a.AgentId }
func (a *QANMongoDBProfilerAgent) ID() string { return a.AgentId }
func (a *QANMySQLSlowlogAgent) ID() string    { return a.AgentId }
func (a *PostgresExporter) ID() string        { return a.AgentId }
