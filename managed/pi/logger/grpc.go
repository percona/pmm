// Copyright (C) 2023 Percona LLC
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

package logger

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// GRPC is a compatibility wrapper between zap's sugared logger entry and gRPC logger interface.
type GRPC struct {
	*zap.SugaredLogger

	// Set to true for very verbose gRPC logging.
	Verbose bool
}

// V reports whether verbosity level l is at least the requested verbose level.
func (g *GRPC) V(_ int) bool {
	return g.Verbose
}

// Infoln prints log message with info level.
func (g *GRPC) Infoln(args ...any) { g.Info(args...) }

// Warning prints log message with warning level.
func (g *GRPC) Warning(args ...any) { g.Warn(args...) }

// Warningln similar to Warning.
func (g *GRPC) Warningln(args ...any) { g.Warn(args...) }

// Warningf prints warning level log message wit given format.
func (g *GRPC) Warningf(format string, args ...any) { g.Warnf(format, args...) }

// Errorln prints log message with error level.
func (g *GRPC) Errorln(args ...any) { g.Error(args...) }

// Fatalln prints log message and exit program.
func (g *GRPC) Fatalln(args ...any) { g.Fatal(args...) }

// Check interfaces.
var _ grpclog.LoggerV2 = (*GRPC)(nil)

//nolint:gochecknoglobals
var protoMarshalOpts = protojson.MarshalOptions{
	UseProtoNames:   true,
	UseEnumNumbers:  false,
	EmitUnpopulated: true,
}

// GRPCMessageDumper helper struct for dumping gRPC message using zap logger.
type GRPCMessageDumper struct {
	msg       any
	ctx       context.Context //nolint: containedctx
	info      *grpc.UnaryServerInfo
	isRequest bool
}

// NewGRPCMessageDumper creates gRPC message dumper for zap logger.
func NewGRPCMessageDumper(ctx context.Context, msg any, info *grpc.UnaryServerInfo, isRequest bool) *GRPCMessageDumper {
	return &GRPCMessageDumper{
		ctx:       ctx,
		msg:       msg,
		info:      info,
		isRequest: isRequest,
	}
}

// MarshalLogObject implements zapcore.ObjectMarshaler interface.
func (d *GRPCMessageDumper) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	if d.isRequest {
		if md, ok := metadata.FromIncomingContext(d.ctx); ok {
			zap.Any("metadata", md).AddTo(enc)
		}

		enc.AddString("method", d.info.FullMethod)
	}

	if d.msg == nil {
		return nil
	}

	protoMsg, ok := d.msg.(proto.Message)
	if !ok {
		enc.AddString("error", fmt.Sprintf("not proto.Message: %v", d.msg))
	} else {
		buf, err := protoMarshalOpts.Marshal(protoMsg)
		if err != nil {
			return err
		}

		enc.AddByteString("fields", buf)
	}

	return nil
}
