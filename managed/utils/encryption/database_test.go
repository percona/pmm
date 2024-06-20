package encryption

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDatabaseConnection(t *testing.T) {
	dbConnection := DatabaseConnection{
		Host:     "127.0.0.1",
		Port:     5432,
		User:     "postgres",
		Password: "",
	}
	c, err := dbConnection.Connect()
	require.NoError(t, err)
	require.NoError(t, c.Close())
}
