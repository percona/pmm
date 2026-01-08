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
func (g *GRPC) Infoln(args ...interface{}) { g.Info(args...) }

// Warning prints log message with warning level.
func (g *GRPC) Warning(args ...interface{}) { g.Warn(args...) }

// Warningln similar to Warning.
func (g *GRPC) Warningln(args ...interface{}) { g.Warn(args...) }

// Warningf prints warning level log message wit given format.
func (g *GRPC) Warningf(format string, args ...interface{}) { g.Warnf(format, args...) }

// Errorln prints log message with error level.
func (g *GRPC) Errorln(args ...interface{}) { g.Error(args...) }

// Fatalln prints log message and exit program.
func (g *GRPC) Fatalln(args ...interface{}) { g.Fatal(args...) }

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
	msg       interface{}
	ctx       context.Context //nolint: containedctx
	info      *grpc.UnaryServerInfo
	isRequest bool
}

// NewGRPCMessageDumper creates gRPC message dumper for zap logger.
func NewGRPCMessageDumper(ctx context.Context, msg interface{}, info *grpc.UnaryServerInfo, isRequest bool) *GRPCMessageDumper {
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
