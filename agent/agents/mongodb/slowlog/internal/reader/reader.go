package reader

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/mongo"
	"gopkg.in/mgo.v2/bson"
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

// ReadFile continuously reads the file, detects truncations, and sends new lines to the provided channel.
func (fr *FileReader) ReadFile(lineChannel chan<- string) error {
	var file *os.File
	var err error

	for {
		fr.fileMutex.Lock()
		file, err = os.Open(fr.filePath)
		if err != nil {
			if os.IsNotExist(err) {
				fr.fileMutex.Unlock()
				return fmt.Errorf("File does not exist: %s\n", fr.filePath)
			} else {
				fr.fileMutex.Unlock()
				return fmt.Errorf("error opening file: %v", err)
			}
		} else {
			info, err := file.Stat()
			if err != nil {
				fr.fileMutex.Unlock()
				return fmt.Errorf("error getting file info: %v", err)
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

			// Create a new scanner to read the file line by line
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				// Send each new line to the channel
				// TODO logs could be formated, so one json != one line
				lineChannel <- scanner.Text()
			}

			// Handle any errors from the scanner
			if err := scanner.Err(); err != nil {
				fr.fileMutex.Unlock()
				return fmt.Errorf("error reading file: %v", err)
			}

			// Update the file size to track truncations
			fr.fileSize = info.Size()

			file.Close()
		}
		fr.fileMutex.Unlock()

		time.Sleep(1 * time.Second)
	}
}
