// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             (unknown)
// source: uieventspb/v1/server.proto

package uieventspbv1

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
	UIEventsService_Store_FullMethodName = "/uieventspb.v1.UIEventsService/Store"
)

// UIEventsServiceClient is the client API for UIEventsService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type UIEventsServiceClient interface {
	// Store persists received UI events for further processing.
	Store(ctx context.Context, in *StoreRequest, opts ...grpc.CallOption) (*StoreResponse, error)
}

type uIEventsServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewUIEventsServiceClient(cc grpc.ClientConnInterface) UIEventsServiceClient {
	return &uIEventsServiceClient{cc}
}

func (c *uIEventsServiceClient) Store(ctx context.Context, in *StoreRequest, opts ...grpc.CallOption) (*StoreResponse, error) {
	out := new(StoreResponse)
	err := c.cc.Invoke(ctx, UIEventsService_Store_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// UIEventsServiceServer is the server API for UIEventsService service.
// All implementations must embed UnimplementedUIEventsServiceServer
// for forward compatibility
type UIEventsServiceServer interface {
	// Store persists received UI events for further processing.
	Store(context.Context, *StoreRequest) (*StoreResponse, error)
	mustEmbedUnimplementedUIEventsServiceServer()
}

// UnimplementedUIEventsServiceServer must be embedded to have forward compatible implementations.
type UnimplementedUIEventsServiceServer struct{}

func (UnimplementedUIEventsServiceServer) Store(context.Context, *StoreRequest) (*StoreResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Store not implemented")
}
func (UnimplementedUIEventsServiceServer) mustEmbedUnimplementedUIEventsServiceServer() {}

// UnsafeUIEventsServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to UIEventsServiceServer will
// result in compilation errors.
type UnsafeUIEventsServiceServer interface {
	mustEmbedUnimplementedUIEventsServiceServer()
}

func RegisterUIEventsServiceServer(s grpc.ServiceRegistrar, srv UIEventsServiceServer) {
	s.RegisterService(&UIEventsService_ServiceDesc, srv)
}

func _UIEventsService_Store_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(StoreRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UIEventsServiceServer).Store(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: UIEventsService_Store_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UIEventsServiceServer).Store(ctx, req.(*StoreRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// UIEventsService_ServiceDesc is the grpc.ServiceDesc for UIEventsService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var UIEventsService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "uieventspb.v1.UIEventsService",
	HandlerType: (*UIEventsServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Store",
			Handler:    _UIEventsService_Store_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "uieventspb/v1/server.proto",
}