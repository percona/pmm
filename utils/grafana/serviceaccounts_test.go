package grafana

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	stringsgen "github.com/percona/pmm/utils/strings"
)

func Test_sanitizeSAName(t *testing.T) {
	// max possible length without hashing
	len180, err := stringsgen.GenerateRandomString(180)
	require.NoError(t, err)
	require.Equal(t, len180, SanitizeSAName(len180))

	// too long length - postfix hashed
	len200, err := stringsgen.GenerateRandomString(200)
	require.NoError(t, err)
	len200sanitized := SanitizeSAName(len200)
	require.Equal(t, fmt.Sprintf("%s%s", len200[:148], len200sanitized[148:]), len200sanitized)
}
