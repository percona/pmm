// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             (unknown)
// source: serverpb/server.proto

package serverpb

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
	Server_Version_FullMethodName          = "/server.Server/Version"
	Server_Readiness_FullMethodName        = "/server.Server/Readiness"
	Server_CheckUpdates_FullMethodName     = "/server.Server/CheckUpdates"
	Server_StartUpdate_FullMethodName      = "/server.Server/StartUpdate"
	Server_UpdateStatus_FullMethodName     = "/server.Server/UpdateStatus"
	Server_GetSettings_FullMethodName      = "/server.Server/GetSettings"
	Server_ChangeSettings_FullMethodName   = "/server.Server/ChangeSettings"
	Server_AWSInstanceCheck_FullMethodName = "/server.Server/AWSInstanceCheck"
)

// ServerClient is the client API for Server service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type ServerClient interface {
	// Version returns PMM Server versions.
	Version(ctx context.Context, in *VersionRequest, opts ...grpc.CallOption) (*VersionResponse, error)
	// Readiness returns an error when Server components being restarted are not ready yet.
	// Use this API for checking the health of Docker containers and for probing Kubernetes readiness.
	Readiness(ctx context.Context, in *ReadinessRequest, opts ...grpc.CallOption) (*ReadinessResponse, error)
	// CheckUpdates checks for available PMM Server updates.
	CheckUpdates(ctx context.Context, in *CheckUpdatesRequest, opts ...grpc.CallOption) (*CheckUpdatesResponse, error)
	// StartUpdate starts PMM Server update.
	StartUpdate(ctx context.Context, in *StartUpdateRequest, opts ...grpc.CallOption) (*StartUpdateResponse, error)
	// UpdateStatus returns PMM Server update status.
	UpdateStatus(ctx context.Context, in *UpdateStatusRequest, opts ...grpc.CallOption) (*UpdateStatusResponse, error)
	// GetSettings returns current PMM Server settings.
	GetSettings(ctx context.Context, in *GetSettingsRequest, opts ...grpc.CallOption) (*GetSettingsResponse, error)
	// ChangeSettings changes PMM Server settings.
	ChangeSettings(ctx context.Context, in *ChangeSettingsRequest, opts ...grpc.CallOption) (*ChangeSettingsResponse, error)
	// AWSInstanceCheck checks AWS EC2 instance ID.
	AWSInstanceCheck(ctx context.Context, in *AWSInstanceCheckRequest, opts ...grpc.CallOption) (*AWSInstanceCheckResponse, error)
}

type serverClient struct {
	cc grpc.ClientConnInterface
}

func NewServerClient(cc grpc.ClientConnInterface) ServerClient {
	return &serverClient{cc}
}

func (c *serverClient) Version(ctx context.Context, in *VersionRequest, opts ...grpc.CallOption) (*VersionResponse, error) {
	out := new(VersionResponse)
	err := c.cc.Invoke(ctx, Server_Version_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *serverClient) Readiness(ctx context.Context, in *ReadinessRequest, opts ...grpc.CallOption) (*ReadinessResponse, error) {
	out := new(ReadinessResponse)
	err := c.cc.Invoke(ctx, Server_Readiness_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *serverClient) CheckUpdates(ctx context.Context, in *CheckUpdatesRequest, opts ...grpc.CallOption) (*CheckUpdatesResponse, error) {
	out := new(CheckUpdatesResponse)
	err := c.cc.Invoke(ctx, Server_CheckUpdates_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *serverClient) StartUpdate(ctx context.Context, in *StartUpdateRequest, opts ...grpc.CallOption) (*StartUpdateResponse, error) {
	out := new(StartUpdateResponse)
	err := c.cc.Invoke(ctx, Server_StartUpdate_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *serverClient) UpdateStatus(ctx context.Context, in *UpdateStatusRequest, opts ...grpc.CallOption) (*UpdateStatusResponse, error) {
	out := new(UpdateStatusResponse)
	err := c.cc.Invoke(ctx, Server_UpdateStatus_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *serverClient) GetSettings(ctx context.Context, in *GetSettingsRequest, opts ...grpc.CallOption) (*GetSettingsResponse, error) {
	out := new(GetSettingsResponse)
	err := c.cc.Invoke(ctx, Server_GetSettings_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *serverClient) ChangeSettings(ctx context.Context, in *ChangeSettingsRequest, opts ...grpc.CallOption) (*ChangeSettingsResponse, error) {
	out := new(ChangeSettingsResponse)
	err := c.cc.Invoke(ctx, Server_ChangeSettings_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *serverClient) AWSInstanceCheck(ctx context.Context, in *AWSInstanceCheckRequest, opts ...grpc.CallOption) (*AWSInstanceCheckResponse, error) {
	out := new(AWSInstanceCheckResponse)
	err := c.cc.Invoke(ctx, Server_AWSInstanceCheck_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ServerServer is the server API for Server service.
// All implementations must embed UnimplementedServerServer
// for forward compatibility
type ServerServer interface {
	// Version returns PMM Server versions.
	Version(context.Context, *VersionRequest) (*VersionResponse, error)
	// Readiness returns an error when Server components being restarted are not ready yet.
	// Use this API for checking the health of Docker containers and for probing Kubernetes readiness.
	Readiness(context.Context, *ReadinessRequest) (*ReadinessResponse, error)
	// CheckUpdates checks for available PMM Server updates.
	CheckUpdates(context.Context, *CheckUpdatesRequest) (*CheckUpdatesResponse, error)
	// StartUpdate starts PMM Server update.
	StartUpdate(context.Context, *StartUpdateRequest) (*StartUpdateResponse, error)
	// UpdateStatus returns PMM Server update status.
	UpdateStatus(context.Context, *UpdateStatusRequest) (*UpdateStatusResponse, error)
	// GetSettings returns current PMM Server settings.
	GetSettings(context.Context, *GetSettingsRequest) (*GetSettingsResponse, error)
	// ChangeSettings changes PMM Server settings.
	ChangeSettings(context.Context, *ChangeSettingsRequest) (*ChangeSettingsResponse, error)
	// AWSInstanceCheck checks AWS EC2 instance ID.
	AWSInstanceCheck(context.Context, *AWSInstanceCheckRequest) (*AWSInstanceCheckResponse, error)
	mustEmbedUnimplementedServerServer()
}

// UnimplementedServerServer must be embedded to have forward compatible implementations.
type UnimplementedServerServer struct{}

func (UnimplementedServerServer) Version(context.Context, *VersionRequest) (*VersionResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Version not implemented")
}

func (UnimplementedServerServer) Readiness(context.Context, *ReadinessRequest) (*ReadinessResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Readiness not implemented")
}

func (UnimplementedServerServer) CheckUpdates(context.Context, *CheckUpdatesRequest) (*CheckUpdatesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CheckUpdates not implemented")
}

func (UnimplementedServerServer) StartUpdate(context.Context, *StartUpdateRequest) (*StartUpdateResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method StartUpdate not implemented")
}

func (UnimplementedServerServer) UpdateStatus(context.Context, *UpdateStatusRequest) (*UpdateStatusResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateStatus not implemented")
}

func (UnimplementedServerServer) GetSettings(context.Context, *GetSettingsRequest) (*GetSettingsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetSettings not implemented")
}

func (UnimplementedServerServer) ChangeSettings(context.Context, *ChangeSettingsRequest) (*ChangeSettingsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ChangeSettings not implemented")
}

func (UnimplementedServerServer) AWSInstanceCheck(context.Context, *AWSInstanceCheckRequest) (*AWSInstanceCheckResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AWSInstanceCheck not implemented")
}
func (UnimplementedServerServer) mustEmbedUnimplementedServerServer() {}

// UnsafeServerServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ServerServer will
// result in compilation errors.
type UnsafeServerServer interface {
	mustEmbedUnimplementedServerServer()
}

func RegisterServerServer(s grpc.ServiceRegistrar, srv ServerServer) {
	s.RegisterService(&Server_ServiceDesc, srv)
}

func _Server_Version_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(VersionRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ServerServer).Version(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Server_Version_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ServerServer).Version(ctx, req.(*VersionRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Server_Readiness_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ReadinessRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ServerServer).Readiness(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Server_Readiness_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ServerServer).Readiness(ctx, req.(*ReadinessRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Server_CheckUpdates_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CheckUpdatesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ServerServer).CheckUpdates(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Server_CheckUpdates_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ServerServer).CheckUpdates(ctx, req.(*CheckUpdatesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Server_StartUpdate_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(StartUpdateRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ServerServer).StartUpdate(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Server_StartUpdate_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ServerServer).StartUpdate(ctx, req.(*StartUpdateRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Server_UpdateStatus_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UpdateStatusRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ServerServer).UpdateStatus(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Server_UpdateStatus_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ServerServer).UpdateStatus(ctx, req.(*UpdateStatusRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Server_GetSettings_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetSettingsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ServerServer).GetSettings(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Server_GetSettings_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ServerServer).GetSettings(ctx, req.(*GetSettingsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Server_ChangeSettings_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ChangeSettingsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ServerServer).ChangeSettings(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Server_ChangeSettings_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ServerServer).ChangeSettings(ctx, req.(*ChangeSettingsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Server_AWSInstanceCheck_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AWSInstanceCheckRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ServerServer).AWSInstanceCheck(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Server_AWSInstanceCheck_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ServerServer).AWSInstanceCheck(ctx, req.(*AWSInstanceCheckRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// Server_ServiceDesc is the grpc.ServiceDesc for Server service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Server_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "server.Server",
	HandlerType: (*ServerServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Version",
			Handler:    _Server_Version_Handler,
		},
		{
			MethodName: "Readiness",
			Handler:    _Server_Readiness_Handler,
		},
		{
			MethodName: "CheckUpdates",
			Handler:    _Server_CheckUpdates_Handler,
		},
		{
			MethodName: "StartUpdate",
			Handler:    _Server_StartUpdate_Handler,
		},
		{
			MethodName: "UpdateStatus",
			Handler:    _Server_UpdateStatus_Handler,
		},
		{
			MethodName: "GetSettings",
			Handler:    _Server_GetSettings_Handler,
		},
		{
			MethodName: "ChangeSettings",
			Handler:    _Server_ChangeSettings_Handler,
		},
		{
			MethodName: "AWSInstanceCheck",
			Handler:    _Server_AWSInstanceCheck_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "serverpb/server.proto",
}