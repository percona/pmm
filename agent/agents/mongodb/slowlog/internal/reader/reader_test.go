package reader

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"testing"
	"time"
)

func TestReader(t *testing.T) {
	// Specify the path to your MongoDB log file
	filePath := "../../../../../testdata/mongo/var/log/mongodb/mongo.log"
	// Create a new FileReader
	fr := NewFileReader(filePath)

	linesInLogFile := countLinesInFile(t, filePath)

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
			fmt.Println("tick")
			if len(s) == linesInLogFile {
				return
			}
		}
	}
}

func countLinesInFile(t *testing.T, filePath string) int {
	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		t.Fatalf("Error opening file %s: %v", filePath, err)
	}
	defer file.Close()

	// Create a scanner to read through the file line by line
	scanner := bufio.NewScanner(file)
	lineCount := 0

	// Loop through each line and increment the count
	for scanner.Scan() {
		lineCount++
	}

	// Check for errors in scanning
	if err := scanner.Err(); err != nil {
		t.Fatalf("Error reading file %s: %v", filePath, err)
	}

	return lineCount
}
