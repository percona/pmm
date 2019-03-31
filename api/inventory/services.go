package inventorypb

// Service is a common interface for all types of Services.
type Service interface {
	service()

	ID() string
	Name() string
}

func (*MySQLService) service()          {}
func (*AmazonRDSMySQLService) service() {}
func (*MongoDBService) service()        {}
func (*PostgreSQLService) service()     {}

func (s *MySQLService) ID() string          { return s.ServiceId }
func (s *AmazonRDSMySQLService) ID() string { return s.ServiceId }
func (s *MongoDBService) ID() string        { return s.ServiceId }
func (s *PostgreSQLService) ID() string     { return s.ServiceId }

func (s *MySQLService) Name() string          { return s.ServiceName }
func (s *AmazonRDSMySQLService) Name() string { return s.ServiceName }
func (s *MongoDBService) Name() string        { return s.ServiceName }
func (s *PostgreSQLService) Name() string     { return s.ServiceName }
