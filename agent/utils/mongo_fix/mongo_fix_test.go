package mongo_fix

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClientOptionsForDSN(t *testing.T) {
	tests := []struct {
		name             string
		dsn              string
		expectedUser     string
		expectedPassword string
	}{
		{
			name: "Escape username",
			dsn: (&url.URL{
				Scheme: "mongo",
				Host:   "localhost",
				Path:   "/db",
				User:   url.UserPassword("user+", "pass"),
			}).String(),
			expectedUser:     "user+",
			expectedPassword: "pass",
		},
		{
			name: "Escape password",
			dsn: (&url.URL{
				Scheme: "mongo",
				Host:   "localhost",
				Path:   "/db",
				User:   url.UserPassword("user", "pass+"),
			}).String(),
			expectedUser:     "user",
			expectedPassword: "pass+",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ClientOptionsForDSN(tt.dsn)
			assert.Nil(t, err)
			assert.Equal(t, got.Auth.Username, tt.expectedUser)
			assert.Equal(t, got.Auth.Password, tt.expectedPassword)
		})
	}
}
