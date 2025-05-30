// Code generated by mockery. DO NOT EDIT.

package dbaas

import (
	context "context"

	controllerv1beta1 "github.com/percona-platform/dbaas-api/gen/controller"
	mock "github.com/stretchr/testify/mock"
	grpc "google.golang.org/grpc"
)

// mockDbaasClient is an autogenerated mock type for the dbaasClient type
type mockDbaasClient struct {
	mock.Mock
}

// Connect provides a mock function with given fields: ctx
func (_m *mockDbaasClient) Connect(ctx context.Context) error {
	ret := _m.Called(ctx)

	if len(ret) == 0 {
		panic("no return value specified for Connect")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Disconnect provides a mock function with given fields:
func (_m *mockDbaasClient) Disconnect() error {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Disconnect")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetKubeConfig provides a mock function with given fields: ctx, in, opts
func (_m *mockDbaasClient) GetKubeConfig(ctx context.Context, in *controllerv1beta1.GetKubeconfigRequest, opts ...grpc.CallOption) (*controllerv1beta1.GetKubeconfigResponse, error) {
	_va := make([]interface{}, len(opts))
	for _i := range opts {
		_va[_i] = opts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, in)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for GetKubeConfig")
	}

	var r0 *controllerv1beta1.GetKubeconfigResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *controllerv1beta1.GetKubeconfigRequest, ...grpc.CallOption) (*controllerv1beta1.GetKubeconfigResponse, error)); ok {
		return rf(ctx, in, opts...)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *controllerv1beta1.GetKubeconfigRequest, ...grpc.CallOption) *controllerv1beta1.GetKubeconfigResponse); ok {
		r0 = rf(ctx, in, opts...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*controllerv1beta1.GetKubeconfigResponse)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *controllerv1beta1.GetKubeconfigRequest, ...grpc.CallOption) error); ok {
		r1 = rf(ctx, in, opts...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetLogs provides a mock function with given fields: ctx, in, opts
func (_m *mockDbaasClient) GetLogs(ctx context.Context, in *controllerv1beta1.GetLogsRequest, opts ...grpc.CallOption) (*controllerv1beta1.GetLogsResponse, error) {
	_va := make([]interface{}, len(opts))
	for _i := range opts {
		_va[_i] = opts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, in)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for GetLogs")
	}

	var r0 *controllerv1beta1.GetLogsResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *controllerv1beta1.GetLogsRequest, ...grpc.CallOption) (*controllerv1beta1.GetLogsResponse, error)); ok {
		return rf(ctx, in, opts...)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *controllerv1beta1.GetLogsRequest, ...grpc.CallOption) *controllerv1beta1.GetLogsResponse); ok {
		r0 = rf(ctx, in, opts...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*controllerv1beta1.GetLogsResponse)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *controllerv1beta1.GetLogsRequest, ...grpc.CallOption) error); ok {
		r1 = rf(ctx, in, opts...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetResources provides a mock function with given fields: ctx, in, opts
func (_m *mockDbaasClient) GetResources(ctx context.Context, in *controllerv1beta1.GetResourcesRequest, opts ...grpc.CallOption) (*controllerv1beta1.GetResourcesResponse, error) {
	_va := make([]interface{}, len(opts))
	for _i := range opts {
		_va[_i] = opts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, in)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for GetResources")
	}

	var r0 *controllerv1beta1.GetResourcesResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *controllerv1beta1.GetResourcesRequest, ...grpc.CallOption) (*controllerv1beta1.GetResourcesResponse, error)); ok {
		return rf(ctx, in, opts...)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *controllerv1beta1.GetResourcesRequest, ...grpc.CallOption) *controllerv1beta1.GetResourcesResponse); ok {
		r0 = rf(ctx, in, opts...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*controllerv1beta1.GetResourcesResponse)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *controllerv1beta1.GetResourcesRequest, ...grpc.CallOption) error); ok {
		r1 = rf(ctx, in, opts...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// InstallPSMDBOperator provides a mock function with given fields: ctx, in, opts
func (_m *mockDbaasClient) InstallPSMDBOperator(ctx context.Context, in *controllerv1beta1.InstallPSMDBOperatorRequest, opts ...grpc.CallOption) (*controllerv1beta1.InstallPSMDBOperatorResponse, error) {
	_va := make([]interface{}, len(opts))
	for _i := range opts {
		_va[_i] = opts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, in)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for InstallPSMDBOperator")
	}

	var r0 *controllerv1beta1.InstallPSMDBOperatorResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *controllerv1beta1.InstallPSMDBOperatorRequest, ...grpc.CallOption) (*controllerv1beta1.InstallPSMDBOperatorResponse, error)); ok {
		return rf(ctx, in, opts...)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *controllerv1beta1.InstallPSMDBOperatorRequest, ...grpc.CallOption) *controllerv1beta1.InstallPSMDBOperatorResponse); ok {
		r0 = rf(ctx, in, opts...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*controllerv1beta1.InstallPSMDBOperatorResponse)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *controllerv1beta1.InstallPSMDBOperatorRequest, ...grpc.CallOption) error); ok {
		r1 = rf(ctx, in, opts...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// InstallPXCOperator provides a mock function with given fields: ctx, in, opts
func (_m *mockDbaasClient) InstallPXCOperator(ctx context.Context, in *controllerv1beta1.InstallPXCOperatorRequest, opts ...grpc.CallOption) (*controllerv1beta1.InstallPXCOperatorResponse, error) {
	_va := make([]interface{}, len(opts))
	for _i := range opts {
		_va[_i] = opts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, in)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for InstallPXCOperator")
	}

	var r0 *controllerv1beta1.InstallPXCOperatorResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *controllerv1beta1.InstallPXCOperatorRequest, ...grpc.CallOption) (*controllerv1beta1.InstallPXCOperatorResponse, error)); ok {
		return rf(ctx, in, opts...)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *controllerv1beta1.InstallPXCOperatorRequest, ...grpc.CallOption) *controllerv1beta1.InstallPXCOperatorResponse); ok {
		r0 = rf(ctx, in, opts...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*controllerv1beta1.InstallPXCOperatorResponse)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *controllerv1beta1.InstallPXCOperatorRequest, ...grpc.CallOption) error); ok {
		r1 = rf(ctx, in, opts...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// StartMonitoring provides a mock function with given fields: ctx, in, opts
func (_m *mockDbaasClient) StartMonitoring(ctx context.Context, in *controllerv1beta1.StartMonitoringRequest, opts ...grpc.CallOption) (*controllerv1beta1.StartMonitoringResponse, error) {
	_va := make([]interface{}, len(opts))
	for _i := range opts {
		_va[_i] = opts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, in)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for StartMonitoring")
	}

	var r0 *controllerv1beta1.StartMonitoringResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *controllerv1beta1.StartMonitoringRequest, ...grpc.CallOption) (*controllerv1beta1.StartMonitoringResponse, error)); ok {
		return rf(ctx, in, opts...)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *controllerv1beta1.StartMonitoringRequest, ...grpc.CallOption) *controllerv1beta1.StartMonitoringResponse); ok {
		r0 = rf(ctx, in, opts...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*controllerv1beta1.StartMonitoringResponse)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *controllerv1beta1.StartMonitoringRequest, ...grpc.CallOption) error); ok {
		r1 = rf(ctx, in, opts...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// StopMonitoring provides a mock function with given fields: ctx, in, opts
func (_m *mockDbaasClient) StopMonitoring(ctx context.Context, in *controllerv1beta1.StopMonitoringRequest, opts ...grpc.CallOption) (*controllerv1beta1.StopMonitoringResponse, error) {
	_va := make([]interface{}, len(opts))
	for _i := range opts {
		_va[_i] = opts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, in)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for StopMonitoring")
	}

	var r0 *controllerv1beta1.StopMonitoringResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *controllerv1beta1.StopMonitoringRequest, ...grpc.CallOption) (*controllerv1beta1.StopMonitoringResponse, error)); ok {
		return rf(ctx, in, opts...)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *controllerv1beta1.StopMonitoringRequest, ...grpc.CallOption) *controllerv1beta1.StopMonitoringResponse); ok {
		r0 = rf(ctx, in, opts...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*controllerv1beta1.StopMonitoringResponse)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *controllerv1beta1.StopMonitoringRequest, ...grpc.CallOption) error); ok {
		r1 = rf(ctx, in, opts...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// newMockDbaasClient creates a new instance of mockDbaasClient. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func newMockDbaasClient(t interface {
	mock.TestingT
	Cleanup(func())
},
) *mockDbaasClient {
	mock := &mockDbaasClient{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
