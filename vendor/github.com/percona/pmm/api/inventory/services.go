package inventory

// Service is a common interface for all types of Services.
type Service interface {
	service()

	ID() string
	Name() string
}

func (*MySQLService) service()          {}
func (*AmazonRDSMySQLService) service() {}

func (s *MySQLService) ID() string          { return s.ServiceId }
func (s *AmazonRDSMySQLService) ID() string { return s.ServiceId }

func (s *MySQLService) Name() string          { return s.ServiceName }
func (s *AmazonRDSMySQLService) Name() string { return s.ServiceName }
