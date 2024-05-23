package encryption

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDatabaseConnection_Connect(t *testing.T) {
	dbConnection := DatabaseConnection{
		Host:     "127.0.0.1",
		Port:     5432,
		User:     "pmm-agent",
		Password: "pmm-agent-password",
		EncryptedItems: []EncryptedItem{
			{Database: "pmm-agent", Table: "accounts", Columns: []string{"username", "password"}},
		},
	}

	assert.NoError(t, dbConnection.Connect())
}
