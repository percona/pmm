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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/percona/percona-toolkit/src/go/mongolib/proto"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
)

const (
	delay = 5 * time.Millisecond
)

func TestCollector(t *testing.T) {
	logrus.SetLevel(logrus.TraceLevel)
	defer logrus.SetLevel(logrus.InfoLevel)

	destination := "testdata/mongo.log"
	timeout := 30 * time.Second

	tests, err := testFileNames()
	require.NoError(t, err)
	for _, test := range tests {
		t.Run(test, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()

			ctr := New(destination, logrus.WithField("component", "collector-test"))

			file, err := os.Create(destination)
			require.NoError(t, err)
			file.Close()
			defer os.Remove(destination)

			docsChan, err := ctr.Start(ctx)
			require.NoError(t, err)

			errChan := make(chan error, 1)
			go readSourceWriteDestination(errChan, ctx, fmt.Sprintf("testdata/%s.log", test), destination, delay)

			var data []proto.SystemProfile
			go func() {
				for {
					select {
					case <-ctx.Done():
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

			// lets triple collector wait duration to be sure we got data and can stop
			<-time.After(3 * collectorWaitDuration)
			ctx.Done()
			ctr.Stop()

			expectedFile := fmt.Sprintf("testdata/expected/%s", test)
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

func testFileNames() ([]string, error) {
	files, err := os.ReadDir("./testdata")
	if err != nil {
		return nil, err
	}

	var names []string
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
	for k, v := range data {
		data[k].Ts = v.Ts.UTC()

		// all bson.D needs to be reordered
		data[k].Command = reorderBSOND(v.Command)
		data[k].OriginatingCommand = reorderBSOND(v.OriginatingCommand)
		data[k].UpdateObj = reorderBSOND(v.UpdateObj)
	}

	return data
}

func reorderBSOND(data bson.D) bson.D {
	var res []bson.E
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
	defer file.Close()

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
	defer file.Close()

	var data []proto.SystemProfile
	err = json.NewDecoder(file).Decode(&data)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func readSourceWriteDestination(errChan chan error, ctx context.Context, source, destination string, delay time.Duration) {
	srcFile, err := os.Open(source)
	if err != nil {
		errChan <- err
		return
	}
	defer srcFile.Close()

	dstFile, err := os.Create(destination)
	if err != nil {
		errChan <- err
		return
	}
	defer dstFile.Close()

	scanner := bufio.NewScanner(srcFile)
	writer := bufio.NewWriter(dstFile)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			errChan <- ctx.Err()
			return
		default:
		}
		line := scanner.Text()
		_, err := writer.WriteString(line + "\n")
		if err != nil {
			errChan <- err
			return
		}
		writer.Flush()
		time.Sleep(delay)
	}

	if err := scanner.Err(); err != nil {
		errChan <- err
		return
	}

	errChan <- nil
}
