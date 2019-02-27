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

	"github.com/Percona-Lab/qan-api/models"
	aservice "github.com/Percona-Lab/qan-api/services/analitycs"
	rservice "github.com/Percona-Lab/qan-api/services/receiver"
	pbqan "github.com/percona/pmm/api/qan"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type server struct{}

//nolint
var version string // will be set by pkg tool.

// HandleVersion implements version.VersionServer
func (s *server) HandleVersion(ctx context.Context, in *pbqan.VersionRequest) (*pbqan.VersionReply, error) {
	log.Println("Version is requested by:", in.Name)
	return &pbqan.VersionReply{Version: version}, nil
}

func main() {
	log.Printf("QAN-API version %v\n", version)
	bind, ok := os.LookupEnv("QANAPI_BIND")
	if !ok {
		bind = "127.0.0.1:9911"
	}
	dsn, ok := os.LookupEnv("QANAPI_DSN")
	if !ok {
		dsn = "clickhouse://127.0.0.1:9000?database=pmm&debug=true"
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
	rm := models.NewReporter(db)
	mm := models.NewMetrics(db)
	grpcServer := grpc.NewServer()
	pbqan.RegisterVersionServer(grpcServer, &server{})
	pbqan.RegisterAgentServer(grpcServer, rservice.NewService(qcm))
	pbqan.RegisterProfileServer(grpcServer, aservice.NewService(rm, mm))
	pbqan.RegisterMetricsServer(grpcServer, aservice.NewService(rm, mm))
	reflection.Register(grpcServer)
	log.Printf("QAN-API serve: %v\n", bind)

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}

}
