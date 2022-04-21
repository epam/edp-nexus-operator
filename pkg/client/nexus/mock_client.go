package nexus

import (
	"context"

	"github.com/stretchr/testify/mock"
)

type Mock struct {
	mock.Mock
}

func (_m *Mock) CreateUser(ctx context.Context, u *User) error {
	u.Password = ""
	return _m.Called(u).Error(0)
}

func (_m *Mock) UpdateUser(ctx context.Context, u *User) error {
	return _m.Called(u).Error(0)
}

func (_m *Mock) DeleteUser(ctx context.Context, ID string) error {
	return _m.Called(ID).Error(0)
}

// AreDefaultScriptsDeclared provides a mock function with given fields: listOfScripts
func (_m *Mock) AreDefaultScriptsDeclared(listOfScripts map[string]string) (bool, error) {
	ret := _m.Called(listOfScripts)

	var r0 bool
	if rf, ok := ret.Get(0).(func(map[string]string) bool); ok {
		r0 = rf(listOfScripts)
	} else {
		r0 = ret.Get(0).(bool)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(map[string]string) error); ok {
		r1 = rf(listOfScripts)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// DeclareDefaultScripts provides a mock function with given fields: listOfScripts
func (_m *Mock) DeclareDefaultScripts(listOfScripts map[string]string) error {
	ret := _m.Called(listOfScripts)

	var r0 error
	if rf, ok := ret.Get(0).(func(map[string]string) error); ok {
		r0 = rf(listOfScripts)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// IsNexusRestApiReady provides a mock function with given fields:
func (_m *Mock) IsNexusRestApiReady() (bool, int, error) {
	ret := _m.Called()

	var r0 bool
	if rf, ok := ret.Get(0).(func() bool); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(bool)
	}

	var r1 int
	if rf, ok := ret.Get(1).(func() int); ok {
		r1 = rf()
	} else {
		r1 = ret.Get(1).(int)
	}

	var r2 error
	if rf, ok := ret.Get(2).(func() error); ok {
		r2 = rf()
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// RunScript provides a mock function with given fields: scriptName, parameters
func (_m *Mock) RunScript(scriptName string, parameters map[string]interface{}) ([]byte, error) {
	ret := _m.Called(scriptName, parameters)

	var r0 []byte
	if rf, ok := ret.Get(0).(func(string, map[string]interface{}) []byte); ok {
		r0 = rf(scriptName, parameters)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, map[string]interface{}) error); ok {
		r1 = rf(scriptName, parameters)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

func (_m *Mock) GetUser(ctx context.Context, email string) (*User, error) {
	called := _m.Called(email)
	if err := called.Error(1); err != nil {
		return nil, err
	}

	return called.Get(0).(*User), nil
}
