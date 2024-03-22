// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             (unknown)
// source: agentlocalpb/agentlocal.proto

package agentlocalpb

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
	AgentLocal_Status_FullMethodName = "/agentlocal.AgentLocal/Status"
	AgentLocal_Reload_FullMethodName = "/agentlocal.AgentLocal/Reload"
)

// AgentLocalClient is the client API for AgentLocal service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type AgentLocalClient interface {
	// Status returns current pmm-agent status.
	Status(ctx context.Context, in *StatusRequest, opts ...grpc.CallOption) (*StatusResponse, error)
	// Reload reloads pmm-agent configuration.
	Reload(ctx context.Context, in *ReloadRequest, opts ...grpc.CallOption) (*ReloadResponse, error)
}

type agentLocalClient struct {
	cc grpc.ClientConnInterface
}

func NewAgentLocalClient(cc grpc.ClientConnInterface) AgentLocalClient {
	return &agentLocalClient{cc}
}

func (c *agentLocalClient) Status(ctx context.Context, in *StatusRequest, opts ...grpc.CallOption) (*StatusResponse, error) {
	out := new(StatusResponse)
	err := c.cc.Invoke(ctx, AgentLocal_Status_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *agentLocalClient) Reload(ctx context.Context, in *ReloadRequest, opts ...grpc.CallOption) (*ReloadResponse, error) {
	out := new(ReloadResponse)
	err := c.cc.Invoke(ctx, AgentLocal_Reload_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// AgentLocalServer is the server API for AgentLocal service.
// All implementations must embed UnimplementedAgentLocalServer
// for forward compatibility
type AgentLocalServer interface {
	// Status returns current pmm-agent status.
	Status(context.Context, *StatusRequest) (*StatusResponse, error)
	// Reload reloads pmm-agent configuration.
	Reload(context.Context, *ReloadRequest) (*ReloadResponse, error)
	mustEmbedUnimplementedAgentLocalServer()
}

// UnimplementedAgentLocalServer must be embedded to have forward compatible implementations.
type UnimplementedAgentLocalServer struct{}

func (UnimplementedAgentLocalServer) Status(context.Context, *StatusRequest) (*StatusResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Status not implemented")
}

func (UnimplementedAgentLocalServer) Reload(context.Context, *ReloadRequest) (*ReloadResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Reload not implemented")
}
func (UnimplementedAgentLocalServer) mustEmbedUnimplementedAgentLocalServer() {}

// UnsafeAgentLocalServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to AgentLocalServer will
// result in compilation errors.
type UnsafeAgentLocalServer interface {
	mustEmbedUnimplementedAgentLocalServer()
}

func RegisterAgentLocalServer(s grpc.ServiceRegistrar, srv AgentLocalServer) {
	s.RegisterService(&AgentLocal_ServiceDesc, srv)
}

func _AgentLocal_Status_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(StatusRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AgentLocalServer).Status(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: AgentLocal_Status_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AgentLocalServer).Status(ctx, req.(*StatusRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _AgentLocal_Reload_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ReloadRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AgentLocalServer).Reload(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: AgentLocal_Reload_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AgentLocalServer).Reload(ctx, req.(*ReloadRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// AgentLocal_ServiceDesc is the grpc.ServiceDesc for AgentLocal service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var AgentLocal_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "agentlocal.AgentLocal",
	HandlerType: (*AgentLocalServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Status",
			Handler:    _AgentLocal_Status_Handler,
		},
		{
			MethodName: "Reload",
			Handler:    _AgentLocal_Reload_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "agentlocalpb/agentlocal.proto",
}