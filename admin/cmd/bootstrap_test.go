package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersionCompare(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name          string
		serverVersion string
		clientVersion string
		error         bool
	}{
		{
			name:          "equal server and client version",
			serverVersion: "2.30.0",
			clientVersion: "2.30.0",
			error:         false,
		},
		{
			name:          "mismatched patch version",
			serverVersion: "2.30.0",
			clientVersion: "2.30.1",
			error:         false,
		},
		{
			name:          "mismatched minor version",
			serverVersion: "2.29.1",
			clientVersion: "2.30.0",
			error:         true,
		},
		{
			name:          "mismatched major version",
			serverVersion: "1.19.0",
			clientVersion: "2.28.0",
			error:         true,
		},
		{
			name:          "server version ahead of client",
			serverVersion: "2.30.0",
			clientVersion: "2.28.0",
			error:         true,
		},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			err := compareVersions(c.clientVersion, c.serverVersion)
			if c.error {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
