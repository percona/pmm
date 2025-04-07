package collector

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/percona/percona-toolkit/src/go/mongolib/proto"
)

type FileReader struct {
	filePath  string
	fileSize  int64
	fileMutex sync.Mutex
}

func NewFileReader(filePath string) *FileReader {
	return &FileReader{
		filePath: filePath,
	}
}

func GetLogFilePath(client *mongo.Client) (string, error) {
	var result bson.M
	err := client.Database("admin").RunCommand(context.TODO(), bson.M{"getCmdLineOpts": 1}).Decode(&result)
	if err != nil {
		errors.Wrap(err, "failed to run command getCmdLineOpts")
	}

	if parsed, ok := result["parsed"].(bson.M); ok {
		if systemLog, ok := parsed["systemLog"].(bson.M); ok {
			if logPath, ok := systemLog["path"].(string); ok {
				return logPath, nil
			}
		}
	}

	if argv, ok := result["argv"].([]interface{}); ok {
		for i := 0; i < len(argv); i++ {
			if arg, ok := argv[i].(string); ok && arg == "--logpath" && i+1 < len(argv) {
				return argv[i+1].(string), nil
			}
		}
	}

	return "", errors.New("No log path found. Logs may be in Docker stdout.")
}

const slowQuery = "Slow query"

type SlowQuery struct {
	// Ctx  string `bson:"ctx"`
	Msg  string `bson:"msg"`
	Attr json.RawMessage
}

type systemProfile struct {
	proto.SystemProfile
	// Command bson.Raw `bson:"command,omitempty"`
	Command            bson.M `bson:"command"`
	OriginatingCommand bson.M `bson:"originatingCommand"`
}

// ReadFile continuously reads the file, detects truncations, and sends new lines to the provided channel.
func (fr *FileReader) ReadFile(ctx context.Context, docsChan chan<- proto.SystemProfile, doneChan <-chan struct{}) {
	var file *os.File
	var err error

	for {
		fr.fileMutex.Lock()
		file, err = os.Open(fr.filePath)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Printf("File does not exist: %s\n", fr.filePath)
				fr.fileMutex.Unlock()
				continue // fmt.Errorf("File does not exist: %s\n", fr.filePath)
			} else {
				fr.fileMutex.Unlock()
				fmt.Printf("error opening file: %v", err)
				continue // fmt.Errorf("error opening file: %v", err)
			}
		}

		info, err := file.Stat()
		if err != nil {
			fr.fileMutex.Unlock()
			fmt.Printf("error getting file info: %v", err)
			continue // fmt.Errorf("error getting file info: %v", err)
		}

		// Check if file has been truncated
		if info.Size() < fr.fileSize {
			// File has been truncated, reset reading position
			fmt.Println("File truncated, seeking to the end")
			file.Seek(0, io.SeekEnd)
		} else {
			// Continue reading from where we left off
			file.Seek(fr.fileSize, io.SeekCurrent)
		}

		fr.fileMutex.Unlock()

		// Create a new scanner to read the file line by line
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			// Send each new line to the channel
			// TODO logs could be formated, so one json != one line

			line := scanner.Text()
			var l SlowQuery
			var doc proto.SystemProfile
			if line == "" || !json.Valid([]byte(line)) {
				docsChan <- doc // TODO remove, test purpose
				continue
			}
			err := json.Unmarshal([]byte(line), &l)
			if err != nil {
				log.Print(err.Error())
				docsChan <- doc
				continue
			}
			if l.Msg != slowQuery {
				docsChan <- doc
				continue
			}
			var stats systemProfile
			err = json.Unmarshal(l.Attr, &stats)
			if err != nil {
				log.Print(err.Error())
				docsChan <- doc
				continue
			}

			doc = stats.SystemProfile

			var command bson.D
			for key, value := range stats.Command {
				command = append(command, bson.E{Key: key, Value: value})
			}

			doc.Command = command
			docsChan <- doc
		}

		// Handle any errors from the scanner
		if err := scanner.Err(); err != nil {
			fmt.Printf("error reading file: %v", err)
			continue // fmt.Errorf("error reading file: %v", err)
		}

		// Update the file size to track truncations
		fr.fileSize = info.Size()

		file.Close()

		select {
		// check if we should shutdown
		case <-ctx.Done():
			return
		case <-doneChan:
			return
		case <-time.After(1 * time.Second):
		}
	}
}
