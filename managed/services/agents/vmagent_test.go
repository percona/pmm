package agents

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMaxScrapeSize(t *testing.T) {
	t.Run("by default 64MiB", func(t *testing.T) {
		actual := vmAgentConfig("")
		assert.Contains(t, actual.Args, "-promscrape.maxScrapeSize=64MiB")
	})
	t.Run("overridden with ENV", func(t *testing.T) {
		newValue := "16MiB"
		err := os.Setenv(maxScrapeSizeEnv, newValue)
		if err != nil {
			panic(err)
		}
		actual := vmAgentConfig("")
		assert.Contains(t, actual.Args, "-promscrape.maxScrapeSize="+newValue)
	})
}
