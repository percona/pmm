package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	slowlog "github.com/percona/go-mysql/log"
	parser "github.com/percona/go-mysql/log/slow"
	"github.com/percona/go-mysql/query"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	pb "github.com/Percona-Lab/qan-api/api/collector"
)

const agentUUID = "dc889ca7be92a66f0a00f616f69ffa7b"

type closedChannelError struct {
	error
}

func main() {
	logOpt := slowlog.Options{}
	slowLogPath := flag.String("slow-log", "logs/mysql-slow.log", "Path to MySQL slow log file")
	serverURL := flag.String("server-url", "127.0.0.1:80", "ULR of QAN-API Server")
	offset := flag.Uint64("offset", 0, "Start Offset of slowlog")

	maxQCtoSent := flag.Int("max-qc-to-sent", 100000, "Maximum query classes  to sent to QAN-API.")
	maxTimeForSent := flag.Duration("max-time-for-tx", 5*time.Second, "Maximum time to send .")
	newEventWait := flag.Duration("new-event-wait", 10*time.Second, "Time to wait for a new event in slow log.")
	flag.Parse()
	logOpt.StartOffset = *offset

	log.SetOutput(os.Stderr)

	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())

	conn, err := grpc.Dial(*serverURL, opts...)
	if err != nil {
		log.Fatalf("fail to dial: %v", err)
	}
	defer conn.Close()
	client := pb.NewAgentClient(conn)

	events := parseSlowLog(*slowLogPath, logOpt)

	ctx := context.TODO()
	stream, err := client.DataInterchange(ctx)
	if err != nil {
		log.Fatalf("%v.DataInterchange(), %v", client, err)
	}

	go func() {
		for {
			in, err := stream.Recv()
			if err != nil {
				continue
			}
			log.Printf("Got message from Api. Saved %d query class(es)", in.SavedAmount)
		}
	}()

	events = parseSlowLog(*slowLogPath, logOpt)
	fmt.Println("Parsing slowlog: ", *slowLogPath, "...")

	for {
		start := time.Now()
		err = bulkSend(stream, func(am *pb.AgentMessage) error {
			i := 0
			for e := range events {
				fingerprint := query.Fingerprint(e.Query)
				digest := query.Id(fingerprint)
				fmt.Printf("parsed QC: %v\n", digest)
				qc := &pb.QueryClass{
					Digest:           digest,
					DigestText:       fingerprint,
					DbSchema:         e.Db,
					DbUsername:       e.User,
					ClientHost:       e.Host,
					AgentUuid:        agentUUID,
					PeriodStart:      e.Ts.UnixNano(),
					Example:          e.Query,
					MQueryTimeSum:    float32(e.TimeMetrics["Query_time"]),
					MLockTimeSum:     float32(e.TimeMetrics["Lock_time"]),
					MRowsSentSum:     e.NumberMetrics["Rows_sent"],
					MRowsExaminedSum: e.NumberMetrics["Rows_examined"],
					MRowsAffectedSum: e.NumberMetrics["Rows_affected"],
					MBytesSentSum:    e.NumberMetrics["Bytes_sent"],
				}
				am.QueryClass = append(am.QueryClass, qc)
				// Pass last offset to restart reader when reached out end of slowlog.
				logOpt.StartOffset = e.OffsetEnd

				i++
				if i >= *maxQCtoSent || time.Since(start) > *maxTimeForSent {
					return nil
				}
			}
			// No new events in slowlog. Nothing to send to API.
			if i == 0 {
				return closedChannelError{errors.New("closed channel")}
			}
			// Reached end of slowlog. Send all what we have to API.
			return nil
		})

		if err != nil {
			if _, ok := err.(closedChannelError); !ok {
				log.Fatal("transaction error:", err)
			}
			// Channel is closed when reached end of the slowlog.
			// Wait and try read the slowlog again.
			time.Sleep(*newEventWait)
			events = parseSlowLog(*slowLogPath, logOpt)
		}
	}
}

func bulkSend(stream pb.Agent_DataInterchangeClient, fn func(*pb.AgentMessage) error) error {
	am := &pb.AgentMessage{}
	err := fn(am)
	if err != nil {
		return err
	}
	lenQC := len(am.QueryClass)
	if lenQC > 0 {
		if err := stream.Send(am); err != nil {
			return fmt.Errorf("sent error: %v", err)
		}
		fmt.Printf("send to qan %v QC\n", lenQC)
	}
	return nil
}

func parseSlowLog(filename string, o slowlog.Options) <-chan *slowlog.Event {
	file, err := os.Open(filepath.Clean(filename))
	if err != nil {
		log.Fatal("cannot open slowlog. Use --slow-log=/path/to/slow.log", err)
	}
	p := parser.NewSlowLogParser(file, o)
	go func() {
		err = p.Start()
		if err != nil {
			log.Fatal("cannot start parser", err)
		}
	}()
	return p.EventChan()
}
