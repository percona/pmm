// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             (unknown)
// source: management/v1/backup/artifacts.proto

package backupv1

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
	ArtifactsService_ListArtifacts_FullMethodName      = "/backup.v1.ArtifactsService/ListArtifacts"
	ArtifactsService_DeleteArtifact_FullMethodName     = "/backup.v1.ArtifactsService/DeleteArtifact"
	ArtifactsService_ListPitrTimeranges_FullMethodName = "/backup.v1.ArtifactsService/ListPitrTimeranges"
)

// ArtifactsServiceClient is the client API for ArtifactsService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type ArtifactsServiceClient interface {
	// ListArtifacts returns a list of all backup artifacts.
	ListArtifacts(ctx context.Context, in *ListArtifactsRequest, opts ...grpc.CallOption) (*ListArtifactsResponse, error)
	// DeleteArtifact deletes specified artifact.
	DeleteArtifact(ctx context.Context, in *DeleteArtifactRequest, opts ...grpc.CallOption) (*DeleteArtifactResponse, error)
	// ListPitrTimeranges list the available MongoDB PITR timeranges in a given backup location
	ListPitrTimeranges(ctx context.Context, in *ListPitrTimerangesRequest, opts ...grpc.CallOption) (*ListPitrTimerangesResponse, error)
}

type artifactsServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewArtifactsServiceClient(cc grpc.ClientConnInterface) ArtifactsServiceClient {
	return &artifactsServiceClient{cc}
}

func (c *artifactsServiceClient) ListArtifacts(ctx context.Context, in *ListArtifactsRequest, opts ...grpc.CallOption) (*ListArtifactsResponse, error) {
	out := new(ListArtifactsResponse)
	err := c.cc.Invoke(ctx, ArtifactsService_ListArtifacts_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *artifactsServiceClient) DeleteArtifact(ctx context.Context, in *DeleteArtifactRequest, opts ...grpc.CallOption) (*DeleteArtifactResponse, error) {
	out := new(DeleteArtifactResponse)
	err := c.cc.Invoke(ctx, ArtifactsService_DeleteArtifact_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *artifactsServiceClient) ListPitrTimeranges(ctx context.Context, in *ListPitrTimerangesRequest, opts ...grpc.CallOption) (*ListPitrTimerangesResponse, error) {
	out := new(ListPitrTimerangesResponse)
	err := c.cc.Invoke(ctx, ArtifactsService_ListPitrTimeranges_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ArtifactsServiceServer is the server API for ArtifactsService service.
// All implementations must embed UnimplementedArtifactsServiceServer
// for forward compatibility
type ArtifactsServiceServer interface {
	// ListArtifacts returns a list of all backup artifacts.
	ListArtifacts(context.Context, *ListArtifactsRequest) (*ListArtifactsResponse, error)
	// DeleteArtifact deletes specified artifact.
	DeleteArtifact(context.Context, *DeleteArtifactRequest) (*DeleteArtifactResponse, error)
	// ListPitrTimeranges list the available MongoDB PITR timeranges in a given backup location
	ListPitrTimeranges(context.Context, *ListPitrTimerangesRequest) (*ListPitrTimerangesResponse, error)
	mustEmbedUnimplementedArtifactsServiceServer()
}

// UnimplementedArtifactsServiceServer must be embedded to have forward compatible implementations.
type UnimplementedArtifactsServiceServer struct{}

func (UnimplementedArtifactsServiceServer) ListArtifacts(context.Context, *ListArtifactsRequest) (*ListArtifactsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListArtifacts not implemented")
}

func (UnimplementedArtifactsServiceServer) DeleteArtifact(context.Context, *DeleteArtifactRequest) (*DeleteArtifactResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeleteArtifact not implemented")
}

func (UnimplementedArtifactsServiceServer) ListPitrTimeranges(context.Context, *ListPitrTimerangesRequest) (*ListPitrTimerangesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListPitrTimeranges not implemented")
}
func (UnimplementedArtifactsServiceServer) mustEmbedUnimplementedArtifactsServiceServer() {}

// UnsafeArtifactsServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ArtifactsServiceServer will
// result in compilation errors.
type UnsafeArtifactsServiceServer interface {
	mustEmbedUnimplementedArtifactsServiceServer()
}

func RegisterArtifactsServiceServer(s grpc.ServiceRegistrar, srv ArtifactsServiceServer) {
	s.RegisterService(&ArtifactsService_ServiceDesc, srv)
}

func _ArtifactsService_ListArtifacts_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListArtifactsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ArtifactsServiceServer).ListArtifacts(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: ArtifactsService_ListArtifacts_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ArtifactsServiceServer).ListArtifacts(ctx, req.(*ListArtifactsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ArtifactsService_DeleteArtifact_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DeleteArtifactRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ArtifactsServiceServer).DeleteArtifact(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: ArtifactsService_DeleteArtifact_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ArtifactsServiceServer).DeleteArtifact(ctx, req.(*DeleteArtifactRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ArtifactsService_ListPitrTimeranges_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListPitrTimerangesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ArtifactsServiceServer).ListPitrTimeranges(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: ArtifactsService_ListPitrTimeranges_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ArtifactsServiceServer).ListPitrTimeranges(ctx, req.(*ListPitrTimerangesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// ArtifactsService_ServiceDesc is the grpc.ServiceDesc for ArtifactsService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var ArtifactsService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "backup.v1.ArtifactsService",
	HandlerType: (*ArtifactsServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "ListArtifacts",
			Handler:    _ArtifactsService_ListArtifacts_Handler,
		},
		{
			MethodName: "DeleteArtifact",
			Handler:    _ArtifactsService_DeleteArtifact_Handler,
		},
		{
			MethodName: "ListPitrTimeranges",
			Handler:    _ArtifactsService_ListPitrTimeranges_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "management/v1/backup/artifacts.proto",
}