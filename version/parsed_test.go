package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsed(t *testing.T) {
	data := []struct {
		s string
		p *Parsed
	}{
		{
			s: "2.0.0-beta4",
			p: &Parsed{Major: 2, Minor: 0, Patch: 0, Rest: "-beta4"},
		},
		{
			s: "2.0.0-beta4-2-gff76039-dirty",
			p: &Parsed{Major: 2, Minor: 0, Patch: 0, Rest: "-beta4-2-gff76039-dirty"},
		},
		{
			s: "2.0.0",
			p: &Parsed{Major: 2, Minor: 0, Patch: 0},
		},
		{
			s: "2.1.2",
			p: &Parsed{Major: 2, Minor: 1, Patch: 2},
		},
		{
			s: "2.1.3",
			p: &Parsed{Major: 2, Minor: 1, Patch: 3},
		},
		{
			s: "3.0.0",
			p: &Parsed{Major: 3, Minor: 0, Patch: 0},
		},
	}
	for i, expected := range data {
		t.Run(expected.s, func(t *testing.T) {
			actual, err := Parse(expected.s)
			require.NoError(t, err)
			assert.Equal(t, *expected.p, *actual)
			assert.Equal(t, expected.s, actual.String())

			for j := 0; j < i; j++ {
				assert.True(t, data[j].p.Less(actual), "%s is expected to be less than %s", data[j].p, actual)
			}
			for j := i + 1; j < len(data); j++ {
				assert.False(t, data[j].p.Less(actual), "%s is expected to be not less than %s", data[j].p, actual)
			}
		})
	}
}
