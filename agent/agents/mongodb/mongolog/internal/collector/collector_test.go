// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package collector

import (
	"bufio"
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/percona/percona-toolkit/src/go/mongolib/proto"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestCollector(t *testing.T) {
	timeout := 30 * time.Second

	logrus.SetLevel(logrus.TraceLevel)
	defer logrus.SetLevel(logrus.InfoLevel)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	filePath := "../../../../../testdata/mongo/var/log/mongodb/mongo.log"
	ctr := New(filePath, logrus.WithField("component", "collector-test"))

	// Start the collector
	var logs []proto.SystemProfile
	docsChan, err := ctr.Start(ctx)
	require.NoError(t, err)
	wg := &sync.WaitGroup{}
	wg.Add(1)
	<-time.After(time.Second)

	linesInLogFile := countLinesInFile(t, filePath)

	go func() {
		defer wg.Done()
		i := 0
		for log := range docsChan {
			select {
			case <-ctx.Done():
				return
			default:
			}
			logs = append(logs, log)
			i++
			if i >= linesInLogFile {
				return
			}
		}
	}()

	wg.Wait()
	ctr.Stop()
}

func countLinesInFile(t *testing.T, filePath string) int {
	// Open the file
	file, err := os.Open(filePath) //nolint:gosec
	if err != nil {
		t.Fatalf("Error opening file %s: %v", filePath, err)
	}
	defer file.Close() //nolint:errcheck

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
