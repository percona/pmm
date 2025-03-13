package reader

import (
	"fmt"
	"log"
	"testing"
	"time"
)

func TestReader(t *testing.T) {
	// Specify the path to your MongoDB log file
	// filePath := getLogFilePath("mongodb://root:root-password@127.0.0.1:27017/admin")
	filePath := "/Users/jiri.ctvrtka/go/src/github.com/percona/pmm/agent/logs/mongod.log"
	// Create a new FileReader
	fr := NewFileReader(filePath)

	// Create a channel to receive new lines
	lineChannel := make(chan string)

	// Start the file reading in a goroutine
	go func() {
		err := fr.ReadFile(lineChannel)
		if err != nil {
			log.Fatalf("Error: %v", err)
		}
	}()

	ticker := time.NewTicker(10 * time.Second)
	var s []string
	for {
		select {
		case line := <-lineChannel:
			s = append(s, line)
		case <-ticker.C:
			fmt.Println("One minute passed, checking for new lines...")
			fmt.Println(s)
			s = nil
		}
	}
	// require.Error(t, nil)
}
