// Code generated by mockery. DO NOT EDIT.

package management

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
)

// mockAuthProvider is an autogenerated mock type for the authProvider type
type mockAuthProvider struct {
	mock.Mock
}

// CreateServiceAccount provides a mock function with given fields: ctx
func (_m *mockAuthProvider) CreateServiceAccount(ctx context.Context) (int, error) {
	ret := _m.Called(ctx)

	var r0 int
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context) (int, error)); ok {
		return rf(ctx)
	}
	if rf, ok := ret.Get(0).(func(context.Context) int); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Get(0).(int)
	}

	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CreateServiceToken provides a mock function with given fields: ctx, serviceAccountID
func (_m *mockAuthProvider) CreateServiceToken(ctx context.Context, serviceAccountID int) (int, string, error) {
	ret := _m.Called(ctx, serviceAccountID)

	var r0 int
	var r1 string
	var r2 error
	if rf, ok := ret.Get(0).(func(context.Context, int) (int, string, error)); ok {
		return rf(ctx, serviceAccountID)
	}
	if rf, ok := ret.Get(0).(func(context.Context, int) int); ok {
		r0 = rf(ctx, serviceAccountID)
	} else {
		r0 = ret.Get(0).(int)
	}

	if rf, ok := ret.Get(1).(func(context.Context, int) string); ok {
		r1 = rf(ctx, serviceAccountID)
	} else {
		r1 = ret.Get(1).(string)
	}

	if rf, ok := ret.Get(2).(func(context.Context, int) error); ok {
		r2 = rf(ctx, serviceAccountID)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// DeleteServiceAccount provides a mock function with given fields: ctx, force
func (_m *mockAuthProvider) DeleteServiceAccount(ctx context.Context, force bool) (string, error) {
	ret := _m.Called(ctx, force)

	var r0 string
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, bool) (string, error)); ok {
		return rf(ctx, force)
	}
	if rf, ok := ret.Get(0).(func(context.Context, bool) string); ok {
		r0 = rf(ctx, force)
	} else {
		r0 = ret.Get(0).(string)
	}

	if rf, ok := ret.Get(1).(func(context.Context, bool) error); ok {
		r1 = rf(ctx, force)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// newMockAuthProvider creates a new instance of mockAuthProvider. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func newMockAuthProvider(t interface {
	mock.TestingT
	Cleanup(func())
},
) *mockAuthProvider {
	mock := &mockAuthProvider{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}