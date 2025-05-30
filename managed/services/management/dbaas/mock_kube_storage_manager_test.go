// Code generated by mockery. DO NOT EDIT.

package dbaas

import mock "github.com/stretchr/testify/mock"

// mockKubeStorageManager is an autogenerated mock type for the kubeStorageManager type
type mockKubeStorageManager struct {
	mock.Mock
}

// DeleteClient provides a mock function with given fields: name
func (_m *mockKubeStorageManager) DeleteClient(name string) error {
	ret := _m.Called(name)

	if len(ret) == 0 {
		panic("no return value specified for DeleteClient")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(name)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetOrSetClient provides a mock function with given fields: name
func (_m *mockKubeStorageManager) GetOrSetClient(name string) (kubernetesClient, error) {
	ret := _m.Called(name)

	if len(ret) == 0 {
		panic("no return value specified for GetOrSetClient")
	}

	var r0 kubernetesClient
	var r1 error
	if rf, ok := ret.Get(0).(func(string) (kubernetesClient, error)); ok {
		return rf(name)
	}
	if rf, ok := ret.Get(0).(func(string) kubernetesClient); ok {
		r0 = rf(name)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(kubernetesClient)
		}
	}

	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(name)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// newMockKubeStorageManager creates a new instance of mockKubeStorageManager. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func newMockKubeStorageManager(t interface {
	mock.TestingT
	Cleanup(func())
},
) *mockKubeStorageManager {
	mock := &mockKubeStorageManager{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
