// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             (unknown)
// source: managementpb/dbaas/db_clusters.proto

package dbaasv1beta1

import (
	context "context"

	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.64.0 or later.
const _ = grpc.SupportPackageIsVersion9

const (
	DBClusters_ListDBClusters_FullMethodName   = "/dbaas.v1beta1.DBClusters/ListDBClusters"
	DBClusters_GetDBCluster_FullMethodName     = "/dbaas.v1beta1.DBClusters/GetDBCluster"
	DBClusters_RestartDBCluster_FullMethodName = "/dbaas.v1beta1.DBClusters/RestartDBCluster"
	DBClusters_DeleteDBCluster_FullMethodName  = "/dbaas.v1beta1.DBClusters/DeleteDBCluster"
	DBClusters_ListS3Backups_FullMethodName    = "/dbaas.v1beta1.DBClusters/ListS3Backups"
	DBClusters_ListSecrets_FullMethodName      = "/dbaas.v1beta1.DBClusters/ListSecrets"
)

// DBClustersClient is the client API for DBClusters service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
//
// DBClusters service provides public methods for managing db clusters.
type DBClustersClient interface {
	// ListDBClusters returns a list of DB clusters.
	ListDBClusters(ctx context.Context, in *ListDBClustersRequest, opts ...grpc.CallOption) (*ListDBClustersResponse, error)
	// GetDBCluster returns parameters used to create a database cluster
	GetDBCluster(ctx context.Context, in *GetDBClusterRequest, opts ...grpc.CallOption) (*GetDBClusterResponse, error)
	// RestartDBCluster restarts DB cluster.
	RestartDBCluster(ctx context.Context, in *RestartDBClusterRequest, opts ...grpc.CallOption) (*RestartDBClusterResponse, error)
	// DeleteDBCluster deletes DB cluster.
	DeleteDBCluster(ctx context.Context, in *DeleteDBClusterRequest, opts ...grpc.CallOption) (*DeleteDBClusterResponse, error)
	// ListS3Backups lists backups stored on s3.
	ListS3Backups(ctx context.Context, in *ListS3BackupsRequest, opts ...grpc.CallOption) (*ListS3BackupsResponse, error)
	// ListSecrets returns a list of secrets from k8s
	ListSecrets(ctx context.Context, in *ListSecretsRequest, opts ...grpc.CallOption) (*ListSecretsResponse, error)
}

type dBClustersClient struct {
	cc grpc.ClientConnInterface
}

func NewDBClustersClient(cc grpc.ClientConnInterface) DBClustersClient {
	return &dBClustersClient{cc}
}

func (c *dBClustersClient) ListDBClusters(ctx context.Context, in *ListDBClustersRequest, opts ...grpc.CallOption) (*ListDBClustersResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(ListDBClustersResponse)
	err := c.cc.Invoke(ctx, DBClusters_ListDBClusters_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *dBClustersClient) GetDBCluster(ctx context.Context, in *GetDBClusterRequest, opts ...grpc.CallOption) (*GetDBClusterResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(GetDBClusterResponse)
	err := c.cc.Invoke(ctx, DBClusters_GetDBCluster_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *dBClustersClient) RestartDBCluster(ctx context.Context, in *RestartDBClusterRequest, opts ...grpc.CallOption) (*RestartDBClusterResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(RestartDBClusterResponse)
	err := c.cc.Invoke(ctx, DBClusters_RestartDBCluster_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *dBClustersClient) DeleteDBCluster(ctx context.Context, in *DeleteDBClusterRequest, opts ...grpc.CallOption) (*DeleteDBClusterResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(DeleteDBClusterResponse)
	err := c.cc.Invoke(ctx, DBClusters_DeleteDBCluster_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *dBClustersClient) ListS3Backups(ctx context.Context, in *ListS3BackupsRequest, opts ...grpc.CallOption) (*ListS3BackupsResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(ListS3BackupsResponse)
	err := c.cc.Invoke(ctx, DBClusters_ListS3Backups_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *dBClustersClient) ListSecrets(ctx context.Context, in *ListSecretsRequest, opts ...grpc.CallOption) (*ListSecretsResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(ListSecretsResponse)
	err := c.cc.Invoke(ctx, DBClusters_ListSecrets_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// DBClustersServer is the server API for DBClusters service.
// All implementations must embed UnimplementedDBClustersServer
// for forward compatibility.
//
// DBClusters service provides public methods for managing db clusters.
type DBClustersServer interface {
	// ListDBClusters returns a list of DB clusters.
	ListDBClusters(context.Context, *ListDBClustersRequest) (*ListDBClustersResponse, error)
	// GetDBCluster returns parameters used to create a database cluster
	GetDBCluster(context.Context, *GetDBClusterRequest) (*GetDBClusterResponse, error)
	// RestartDBCluster restarts DB cluster.
	RestartDBCluster(context.Context, *RestartDBClusterRequest) (*RestartDBClusterResponse, error)
	// DeleteDBCluster deletes DB cluster.
	DeleteDBCluster(context.Context, *DeleteDBClusterRequest) (*DeleteDBClusterResponse, error)
	// ListS3Backups lists backups stored on s3.
	ListS3Backups(context.Context, *ListS3BackupsRequest) (*ListS3BackupsResponse, error)
	// ListSecrets returns a list of secrets from k8s
	ListSecrets(context.Context, *ListSecretsRequest) (*ListSecretsResponse, error)
	mustEmbedUnimplementedDBClustersServer()
}

// UnimplementedDBClustersServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedDBClustersServer struct{}

func (UnimplementedDBClustersServer) ListDBClusters(context.Context, *ListDBClustersRequest) (*ListDBClustersResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListDBClusters not implemented")
}

func (UnimplementedDBClustersServer) GetDBCluster(context.Context, *GetDBClusterRequest) (*GetDBClusterResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetDBCluster not implemented")
}

func (UnimplementedDBClustersServer) RestartDBCluster(context.Context, *RestartDBClusterRequest) (*RestartDBClusterResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RestartDBCluster not implemented")
}

func (UnimplementedDBClustersServer) DeleteDBCluster(context.Context, *DeleteDBClusterRequest) (*DeleteDBClusterResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeleteDBCluster not implemented")
}

func (UnimplementedDBClustersServer) ListS3Backups(context.Context, *ListS3BackupsRequest) (*ListS3BackupsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListS3Backups not implemented")
}

func (UnimplementedDBClustersServer) ListSecrets(context.Context, *ListSecretsRequest) (*ListSecretsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListSecrets not implemented")
}
func (UnimplementedDBClustersServer) mustEmbedUnimplementedDBClustersServer() {}
func (UnimplementedDBClustersServer) testEmbeddedByValue()                    {}

// UnsafeDBClustersServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to DBClustersServer will
// result in compilation errors.
type UnsafeDBClustersServer interface {
	mustEmbedUnimplementedDBClustersServer()
}

func RegisterDBClustersServer(s grpc.ServiceRegistrar, srv DBClustersServer) {
	// If the following call pancis, it indicates UnimplementedDBClustersServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&DBClusters_ServiceDesc, srv)
}

func _DBClusters_ListDBClusters_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListDBClustersRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DBClustersServer).ListDBClusters(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: DBClusters_ListDBClusters_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DBClustersServer).ListDBClusters(ctx, req.(*ListDBClustersRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _DBClusters_GetDBCluster_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetDBClusterRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DBClustersServer).GetDBCluster(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: DBClusters_GetDBCluster_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DBClustersServer).GetDBCluster(ctx, req.(*GetDBClusterRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _DBClusters_RestartDBCluster_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RestartDBClusterRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DBClustersServer).RestartDBCluster(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: DBClusters_RestartDBCluster_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DBClustersServer).RestartDBCluster(ctx, req.(*RestartDBClusterRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _DBClusters_DeleteDBCluster_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DeleteDBClusterRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DBClustersServer).DeleteDBCluster(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: DBClusters_DeleteDBCluster_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DBClustersServer).DeleteDBCluster(ctx, req.(*DeleteDBClusterRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _DBClusters_ListS3Backups_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListS3BackupsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DBClustersServer).ListS3Backups(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: DBClusters_ListS3Backups_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DBClustersServer).ListS3Backups(ctx, req.(*ListS3BackupsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _DBClusters_ListSecrets_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListSecretsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DBClustersServer).ListSecrets(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: DBClusters_ListSecrets_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DBClustersServer).ListSecrets(ctx, req.(*ListSecretsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// DBClusters_ServiceDesc is the grpc.ServiceDesc for DBClusters service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var DBClusters_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "dbaas.v1beta1.DBClusters",
	HandlerType: (*DBClustersServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "ListDBClusters",
			Handler:    _DBClusters_ListDBClusters_Handler,
		},
		{
			MethodName: "GetDBCluster",
			Handler:    _DBClusters_GetDBCluster_Handler,
		},
		{
			MethodName: "RestartDBCluster",
			Handler:    _DBClusters_RestartDBCluster_Handler,
		},
		{
			MethodName: "DeleteDBCluster",
			Handler:    _DBClusters_DeleteDBCluster_Handler,
		},
		{
			MethodName: "ListS3Backups",
			Handler:    _DBClusters_ListS3Backups_Handler,
		},
		{
			MethodName: "ListSecrets",
			Handler:    _DBClusters_ListSecrets_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "managementpb/dbaas/db_clusters.proto",
}
