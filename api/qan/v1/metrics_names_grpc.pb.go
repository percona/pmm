// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             (unknown)
// source: qan/v1/metrics_names.proto

package qanv1

import (
	context "context"

	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

const (
	MetricsNamesService_GetMetricsNames_FullMethodName = "/qan.v1.MetricsNamesService/GetMetricsNames"
)

// MetricsNamesServiceClient is the client API for MetricsNamesService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type MetricsNamesServiceClient interface {
	// GetMetricsNames gets map of metrics names.
	GetMetricsNames(ctx context.Context, in *GetMetricsNamesRequest, opts ...grpc.CallOption) (*GetMetricsNamesResponse, error)
}

type metricsNamesServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewMetricsNamesServiceClient(cc grpc.ClientConnInterface) MetricsNamesServiceClient {
	return &metricsNamesServiceClient{cc}
}

func (c *metricsNamesServiceClient) GetMetricsNames(ctx context.Context, in *GetMetricsNamesRequest, opts ...grpc.CallOption) (*GetMetricsNamesResponse, error) {
	out := new(GetMetricsNamesResponse)
	err := c.cc.Invoke(ctx, MetricsNamesService_GetMetricsNames_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// MetricsNamesServiceServer is the server API for MetricsNamesService service.
// All implementations must embed UnimplementedMetricsNamesServiceServer
// for forward compatibility
type MetricsNamesServiceServer interface {
	// GetMetricsNames gets map of metrics names.
	GetMetricsNames(context.Context, *GetMetricsNamesRequest) (*GetMetricsNamesResponse, error)
	mustEmbedUnimplementedMetricsNamesServiceServer()
}

// UnimplementedMetricsNamesServiceServer must be embedded to have forward compatible implementations.
type UnimplementedMetricsNamesServiceServer struct{}

func (UnimplementedMetricsNamesServiceServer) GetMetricsNames(context.Context, *GetMetricsNamesRequest) (*GetMetricsNamesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetMetricsNames not implemented")
}
func (UnimplementedMetricsNamesServiceServer) mustEmbedUnimplementedMetricsNamesServiceServer() {}

// UnsafeMetricsNamesServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to MetricsNamesServiceServer will
// result in compilation errors.
type UnsafeMetricsNamesServiceServer interface {
	mustEmbedUnimplementedMetricsNamesServiceServer()
}

func RegisterMetricsNamesServiceServer(s grpc.ServiceRegistrar, srv MetricsNamesServiceServer) {
	s.RegisterService(&MetricsNamesService_ServiceDesc, srv)
}

func _MetricsNamesService_GetMetricsNames_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetMetricsNamesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MetricsNamesServiceServer).GetMetricsNames(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: MetricsNamesService_GetMetricsNames_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MetricsNamesServiceServer).GetMetricsNames(ctx, req.(*GetMetricsNamesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// MetricsNamesService_ServiceDesc is the grpc.ServiceDesc for MetricsNamesService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var MetricsNamesService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "qan.v1.MetricsNamesService",
	HandlerType: (*MetricsNamesServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetMetricsNames",
			Handler:    _MetricsNamesService_GetMetricsNames_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "qan/v1/metrics_names.proto",
}