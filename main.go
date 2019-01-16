// qan-api
// Copyright (C) 2019 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package main

import (
	"log"
	"net"
	"os"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	pbcollector "github.com/Percona-Lab/qan-api/api/collector"
	pbversion "github.com/Percona-Lab/qan-api/api/version"
	"github.com/Percona-Lab/qan-api/models"
	rservice "github.com/Percona-Lab/qan-api/services/receiver"
)

type server struct{}

//nolint
var version string // will be set by pkg tool.

// HandleVersion implements version.VersionServer
func (s *server) HandleVersion(ctx context.Context, in *pbversion.VersionRequest) (*pbversion.VersionReply, error) {
	log.Println("Version is requested by:", in.Name)
	return &pbversion.VersionReply{Version: version}, nil
}

func main() {
	log.Printf("QAN-API version %v\n", version)
	bind, ok := os.LookupEnv("QANAPI_BIND")
	if !ok {
		bind = "127.0.0.1:9911"
	}
	dsn, ok := os.LookupEnv("QANAPI_DSN")
	if !ok {
		dsn = "clickhouse://127.0.0.1:9000?database=pmm"
	}

	db, err := NewDB(dsn)
	if err != nil {
		log.Fatal("DB error", err)
	}

	lis, err := net.Listen("tcp", bind)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	qcm := models.NewQueryClass(db)
	grpcServer := grpc.NewServer()
	pbversion.RegisterVersionServer(grpcServer, &server{})
	pbcollector.RegisterAgentServer(grpcServer, rservice.NewService(qcm))
	reflection.Register(grpcServer)
	log.Printf("QAN-API serve: %v\n", bind)

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}

}
