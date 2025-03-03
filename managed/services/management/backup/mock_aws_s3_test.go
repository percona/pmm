// Code generated by mockery. DO NOT EDIT.

package backup

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
)

// mockAwsS3 is an autogenerated mock type for the awsS3 type
type mockAwsS3 struct {
	mock.Mock
}

// BucketExists provides a mock function with given fields: ctx, host, accessKey, secretKey, name
func (_m *mockAwsS3) BucketExists(ctx context.Context, host string, accessKey string, secretKey string, name string) (bool, error) {
	ret := _m.Called(ctx, host, accessKey, secretKey, name)

	if len(ret) == 0 {
		panic("no return value specified for BucketExists")
	}

	var r0 bool
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string, string, string) (bool, error)); ok {
		return rf(ctx, host, accessKey, secretKey, name)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, string, string, string) bool); ok {
		r0 = rf(ctx, host, accessKey, secretKey, name)
	} else {
		r0 = ret.Get(0).(bool)
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, string, string, string) error); ok {
		r1 = rf(ctx, host, accessKey, secretKey, name)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetBucketLocation provides a mock function with given fields: ctx, host, accessKey, secretKey, name
func (_m *mockAwsS3) GetBucketLocation(ctx context.Context, host string, accessKey string, secretKey string, name string) (string, error) {
	ret := _m.Called(ctx, host, accessKey, secretKey, name)

	if len(ret) == 0 {
		panic("no return value specified for GetBucketLocation")
	}

	var r0 string
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string, string, string) (string, error)); ok {
		return rf(ctx, host, accessKey, secretKey, name)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, string, string, string) string); ok {
		r0 = rf(ctx, host, accessKey, secretKey, name)
	} else {
		r0 = ret.Get(0).(string)
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, string, string, string) error); ok {
		r1 = rf(ctx, host, accessKey, secretKey, name)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// RemoveRecursive provides a mock function with given fields: ctx, endpoint, accessKey, secretKey, bucketName, prefix
func (_m *mockAwsS3) RemoveRecursive(ctx context.Context, endpoint string, accessKey string, secretKey string, bucketName string, prefix string) error {
	ret := _m.Called(ctx, endpoint, accessKey, secretKey, bucketName, prefix)

	if len(ret) == 0 {
		panic("no return value specified for RemoveRecursive")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string, string, string, string) error); ok {
		r0 = rf(ctx, endpoint, accessKey, secretKey, bucketName, prefix)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// newMockAwsS3 creates a new instance of mockAwsS3. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func newMockAwsS3(t interface {
	mock.TestingT
	Cleanup(func())
},
) *mockAwsS3 {
	mock := &mockAwsS3{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
