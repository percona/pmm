package main

import (
	"log"
	"net"
	"os"

	pbversion "github.com/Percona-Lab/qan-api/api/version"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type server struct{}

// HandleVersion implements version.VersionServer
func (s *server) HandleVersion(ctx context.Context, in *pbversion.VersionRequest) (*pbversion.VersionReply, error) {
	log.Println("Version is requested by:", in.Name)
	return &pbversion.VersionReply{Version: "2.0.0-alpha"}, nil
}

func main() {
	bind, ok := os.LookupEnv("QANAPI_BIND")
	if !ok {
		bind = "127.0.0.1:9001"
	}
	dsn, ok := os.LookupEnv("QANAPI_DSN")
	if !ok {
		dsn = "clickhouse://127.0.0.1:9000?debug=true&database=pmm&x-multi-statement=true"
	}

	db, err := NewDB(dsn)
	_ = db
	if err != nil {
		log.Fatal("DB error", err)
	}

	lis, err := net.Listen("tcp", bind)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pbversion.RegisterVersionServer(s, &server{})
	reflection.Register(s)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
