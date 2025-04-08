package shared

import "fmt"

// QueryIDWithSchema returns query ID with schema in format schema-queryID.
// It is used to fix: https://perconadev.atlassian.net/browse/PMM-12413.
func QueryIDWithSchema(schema, queryID string) string {
	if schema == "" {
		return queryID
	}

	return fmt.Sprintf("%s-%s", schema, queryID)
}
