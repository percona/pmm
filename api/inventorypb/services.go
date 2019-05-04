package inventorypb

//go-sumtype:decl Service

// Service is a common interface for all types of Services.
type Service interface {
	sealedService() //nolint:unused

	// TODO Remove it.
	// Deprecated: use ServiceId field instead.
	ID() string
}

func (*MySQLService) sealedService()          {}
func (*AmazonRDSMySQLService) sealedService() {}
func (*MongoDBService) sealedService()        {}
func (*PostgreSQLService) sealedService()     {}

func (m *MySQLService) ID() string          { return m.ServiceId }
func (m *AmazonRDSMySQLService) ID() string { return m.ServiceId }
func (m *MongoDBService) ID() string        { return m.ServiceId }
func (m *PostgreSQLService) ID() string     { return m.ServiceId }
