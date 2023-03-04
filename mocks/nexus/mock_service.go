// Code generated by mockery v2.20.2. DO NOT EDIT.

package mocks

import (
	context "context"

	clientnexus "github.com/epam/edp-nexus-operator/v2/pkg/client/nexus"

	mock "github.com/stretchr/testify/mock"

	nexus "github.com/epam/edp-nexus-operator/v2/pkg/service/nexus"

	v1 "github.com/epam/edp-nexus-operator/v2/api/v1"
)

// Service is an autogenerated mock type for the Service type
type Service struct {
	mock.Mock
}

// ClientForNexusChild provides a mock function with given fields: ctx, child
func (_m *Service) ClientForNexusChild(ctx context.Context, child nexus.Child) (*clientnexus.Client, error) {
	ret := _m.Called(ctx, child)

	var r0 *clientnexus.Client
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, nexus.Child) (*clientnexus.Client, error)); ok {
		return rf(ctx, child)
	}
	if rf, ok := ret.Get(0).(func(context.Context, nexus.Child) *clientnexus.Client); ok {
		r0 = rf(ctx, child)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*clientnexus.Client)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, nexus.Child) error); ok {
		r1 = rf(ctx, child)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Configure provides a mock function with given fields: instance
func (_m *Service) Configure(instance *v1.Nexus) (*v1.Nexus, bool, error) {
	ret := _m.Called()

	var r0 *v1.Nexus
	if rf, ok := ret.Get(0).(func(v1.Nexus) *v1.Nexus); ok {
		r0 = rf(*instance)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1.Nexus)
		}
	}

	var r1 bool
	if rf, ok := ret.Get(1).(func(v1.Nexus) bool); ok {
		r1 = rf(*instance)
	} else {
		r1 = ret.Get(1).(bool)
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(v1.Nexus) error); ok {
		r2 = rf(*instance)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// ExposeConfiguration provides a mock function with given fields: ctx, instance
func (_m *Service) ExposeConfiguration(ctx context.Context, instance *v1.Nexus) (*v1.Nexus, error) {
	ret := _m.Called(ctx, instance)

	var r0 *v1.Nexus
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Nexus) (*v1.Nexus, error)); ok {
		return rf(ctx, instance)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Nexus) *v1.Nexus); ok {
		r0 = rf(ctx, instance)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1.Nexus)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *v1.Nexus) error); ok {
		r1 = rf(ctx, instance)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// IsDeploymentReady provides a mock function with given fields: instance
func (_m *Service) IsDeploymentReady(instance *v1.Nexus) (*bool, error) {
	ret := _m.Called()

	var r0 *bool
	if rf, ok := ret.Get(0).(func(v1.Nexus) *bool); ok {
		r0 = rf(*instance)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*bool)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(v1.Nexus) error); ok {
		r1 = rf(*instance)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
