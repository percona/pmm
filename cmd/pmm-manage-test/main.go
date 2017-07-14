// pmm-managed
// Copyright (C) 2017 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"context"
	"flag"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/Percona-Lab/pmm-managed/api"
)

var (
	gRPCAddrF = flag.String("grpc-addr", "127.0.0.1:7771", "gRPC server address")
)

func main() {
	flag.Parse()

	conn, err := grpc.Dial(*gRPCAddrF, grpc.WithInsecure())
	if err != nil {
		logrus.Fatal(err)
	}
	defer conn.Close()

	c := api.NewBaseClient(conn)
	for {
		_, err = c.Version(context.Background(), &api.BaseVersionRequest{})
		if err != nil {
			logrus.Fatal(err)
		}
		time.Sleep(200 * time.Millisecond)
	}
}
