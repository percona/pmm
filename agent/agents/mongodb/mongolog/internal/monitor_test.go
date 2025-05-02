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

package mongolog

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/percona/percona-toolkit/src/go/mongolib/proto"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"

	"github.com/percona/pmm/agent/utils/filereader"
)

const (
	delay         = 5 * time.Millisecond
	timeout       = 30 * time.Second
	timeToCollect = 10 * time.Second
)

func TestCollector(t *testing.T) {
	logrus.SetLevel(logrus.TraceLevel)
	defer logrus.SetLevel(logrus.InfoLevel)

	tests, err := testFileNames()
	require.NoError(t, err)
	for _, test := range tests {
		t.Run(test, func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()

			hash, err := generateRandomHash()
			require.NoError(t, err)
			destination := fmt.Sprintf("./testdata/mongo_%s.log", hash)

			l := logrus.WithField("test", t.Name())

			file, err := os.Create(destination) //nolint:gosec
			require.NoError(t, err)
			file.Close()                 //nolint:errcheck
			defer os.Remove(destination) //nolint:errcheck

			reader, err := filereader.NewContinuousFileReader(destination, l)
			require.NoError(t, err)

			monitor := NewMonitor(destination, reader, l)

			docsChan := make(chan proto.SystemProfile, collectorChanCapacity)
			defer close(docsChan)

			doneChan := make(chan struct{})
			defer close(doneChan)

			errChan := make(chan error, 1)
			go readSourceWriteDestination(ctx, errChan, fmt.Sprintf("./testdata/logs/%s.log", test), destination, delay)

			var wg sync.WaitGroup
			wg.Add(2)
			monitor.Start(ctx, docsChan, doneChan, &wg)

			var data []proto.SystemProfile
			go func() {
				defer wg.Done()
				for {
					select {
					case <-ctx.Done():
						return
					case <-doneChan:
						return
					case row, ok := <-docsChan:
						if !ok {
							return
						}
						data = append(data, row)
					}
				}
			}()

			err = <-errChan
			require.NoError(t, err)

			// All data are written right now, lets wait
			// several more seconds to ensure all data are collected.
			<-time.After(timeToCollect)
			doneChan <- struct{}{}
			monitor.Stop()
			wg.Wait()

			expectedFile := fmt.Sprintf("./testdata/expected/%s", test)
			if os.Getenv("REFRESH_TEST_DATA") != "" {
				writeData(data, expectedFile)
				return
			}

			expectedData, err := readData(expectedFile)
			require.NoError(t, err)

			require.Equal(t, reorderData(expectedData), reorderData(data))
		})
	}
}

func generateRandomHash() (string, error) {
	randomBytes := make([]byte, 32)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(randomBytes)
	return hex.EncodeToString(hash[:]), nil
}

func testFileNames() ([]string, error) {
	files, err := os.ReadDir("./testdata/logs")
	if err != nil {
		return nil, err
	}

	var names []string //nolint:prealloc
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		name := file.Name()
		ext := filepath.Ext(name)
		names = append(names, strings.TrimSuffix(name, ext))
	}

	return names, nil
}

func reorderData(data []proto.SystemProfile) []proto.SystemProfile {
	var res []proto.SystemProfile //nolint:prealloc
	for _, d := range data {
		d.Ts = d.Ts.UTC()

		// all bson.D needs to be reordered
		d.Command = reorderBSOND(d.Command)
		d.OriginatingCommand = reorderBSOND(d.OriginatingCommand)
		d.UpdateObj = reorderBSOND(d.UpdateObj)

		res = append(res, d)
	}

	return res
}

func reorderBSOND(data bson.D) bson.D {
	var res []bson.E //nolint:prealloc
	for _, d := range data {
		res = append(res, d)
	}

	sort.SliceStable(res, func(i, j int) bool {
		return res[i].Key < res[j].Key
	})

	return res
}

func dataToJSON(data []proto.SystemProfile) ([]byte, error) {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, err
	}

	return jsonData, nil
}

func writeData(data []proto.SystemProfile, name string) error {
	file, err := os.Create(fmt.Sprintf("%s.json", name))
	if err != nil {
		return err
	}
	defer file.Close() //nolint:errcheck

	jsonData, err := dataToJSON(data)
	if err != nil {
		return err
	}
	_, err = file.Write(jsonData)
	if err != nil {
		return err
	}

	return nil
}

func readData(name string) ([]proto.SystemProfile, error) {
	file, err := os.Open(fmt.Sprintf("%s.json", name))
	if err != nil {
		return nil, err
	}
	defer file.Close() //nolint:errcheck

	var data []proto.SystemProfile
	err = json.NewDecoder(file).Decode(&data)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func readSourceWriteDestination(ctx context.Context, errChan chan error, source, destination string, delay time.Duration) {
	srcFile, err := os.Open(source)
	if err != nil {
		errChan <- err
		return
	}
	scanner := bufio.NewScanner(srcFile)
	var lines []string
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			errChan <- ctx.Err()
			return
		default:
		}
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		errChan <- err
		return
	}
	srcFile.Close() //nolint:errcheck

	dstFile, err := os.Create(destination)
	if err != nil {
		errChan <- err
		return
	}
	defer dstFile.Close() //nolint:errcheck

	writer := bufio.NewWriter(dstFile)
	for _, line := range lines {
		select {
		case <-ctx.Done():
			errChan <- ctx.Err()
			return
		default:
		}
		_, err := writer.WriteString(line + "\n")
		if err != nil {
			errChan <- err
			return
		}
		err = writer.Flush()
		if err != nil {
			errChan <- err
			return
		}
		time.Sleep(delay)
	}

	if err := scanner.Err(); err != nil {
		errChan <- err
		return
	}

	errChan <- nil
}
