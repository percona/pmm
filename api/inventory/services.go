package inventory

// Service is a common interface for all types of Services.
type Service interface {
	service()
}

func (*MySQLService) service()          {}
func (*AmazonRDSMySQLService) service() {}
