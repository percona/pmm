package inventory

// Agent is a common interface for all types of Agents.
type Agent interface {
	agent()
}

func (*MySQLdExporter) agent() {}
