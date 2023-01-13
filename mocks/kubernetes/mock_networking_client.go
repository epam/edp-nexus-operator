// Code generated by mockery v2.15.0. DO NOT EDIT.

package mock

import (
	mock "github.com/stretchr/testify/mock"
	rest "k8s.io/client-go/rest"

	v1 "k8s.io/client-go/kubernetes/typed/networking/v1"
)

// NetworkingV1Interface is an autogenerated mock type for the NetworkingV1Interface type
type NetworkingV1Interface struct {
	mock.Mock
}

// IngressClasses provides a mock function with given fields:
func (_m *NetworkingV1Interface) IngressClasses() v1.IngressClassInterface {
	ret := _m.Called()

	var r0 v1.IngressClassInterface
	if rf, ok := ret.Get(0).(func() v1.IngressClassInterface); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(v1.IngressClassInterface)
		}
	}

	return r0
}

// Ingresses provides a mock function with given fields: namespace
func (_m *NetworkingV1Interface) Ingresses(namespace string) v1.IngressInterface {
	ret := _m.Called(namespace)

	var r0 v1.IngressInterface
	if rf, ok := ret.Get(0).(func(string) v1.IngressInterface); ok {
		r0 = rf(namespace)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(v1.IngressInterface)
		}
	}

	return r0
}

// NetworkPolicies provides a mock function with given fields: namespace
func (_m *NetworkingV1Interface) NetworkPolicies(namespace string) v1.NetworkPolicyInterface {
	ret := _m.Called(namespace)

	var r0 v1.NetworkPolicyInterface
	if rf, ok := ret.Get(0).(func(string) v1.NetworkPolicyInterface); ok {
		r0 = rf(namespace)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(v1.NetworkPolicyInterface)
		}
	}

	return r0
}

// RESTClient provides a mock function with given fields:
func (_m *NetworkingV1Interface) RESTClient() rest.Interface {
	ret := _m.Called()

	var r0 rest.Interface
	if rf, ok := ret.Get(0).(func() rest.Interface); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(rest.Interface)
		}
	}

	return r0
}

type mockConstructorTestingTNewNetworkingV1Interface interface {
	mock.TestingT
	Cleanup(func())
}

// NewNetworkingV1Interface creates a new instance of NetworkingV1Interface. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewNetworkingV1Interface(t mockConstructorTestingTNewNetworkingV1Interface) *NetworkingV1Interface {
	mock := &NetworkingV1Interface{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
