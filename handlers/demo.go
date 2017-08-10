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

package handlers

import (
	"io"
	"math/rand"
	"time"

	"golang.org/x/net/context"

	"github.com/Percona-Lab/pmm-managed/api"
	"github.com/Percona-Lab/pmm-managed/utils/logger"
)

type DemoServer struct{}

func (s *DemoServer) Ping(ctx context.Context, req *api.DemoPingRequest) (*api.DemoPingResponse, error) {
	return &api.DemoPingResponse{
		Type:   api.DemoPingType_PONG,
		Cookie: req.Cookie,
	}, nil
}

func (s *DemoServer) PingStream(stream api.Demo_PingStreamServer) (err error) {
	ctx, l := logger.Set(stream.Context(), logger.MakeRequestID())
	l.Debugf("Ping started")
	defer func() {
		l.Debugf("Ping stopped with error %s", err)
	}()

	// start pinger
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			resp := &api.DemoPingResponse{
				Type:   api.DemoPingType_PING,
				Cookie: uint64(time.Now().UnixNano()),
			}
			if pingErr := stream.Send(resp); pingErr != nil {
				l.Error(pingErr)
				return
			}
			time.Sleep(time.Duration(rand.Intn(int(time.Second))))
		}
	}()

	// ponger
	for {
		select {
		case <-ctx.Done():
			err = ctx.Err()
			return
		default:
		}

		var req *api.DemoPingRequest
		req, err = stream.Recv()
		if err != nil {
			if err == io.EOF {
				err = nil
			}
			return
		}

		switch req.Type {
		case api.DemoPingType_PING:
			l.Debugf("Received ping: %d", req.Cookie)
			pong := &api.DemoPingResponse{
				Type:   api.DemoPingType_PONG,
				Cookie: req.Cookie,
			}
			if err = stream.Send(pong); err != nil {
				return
			}
		case api.DemoPingType_PONG:
			d := time.Since(time.Unix(0, int64(req.Cookie)))
			l.Debugf("Received pong: %d (latency %v)", req.Cookie, d)
		}
	}
}

// check interface
var _ api.DemoServer = (*DemoServer)(nil)
