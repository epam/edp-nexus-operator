package nexus

import (
	"context"

	"github.com/stretchr/testify/mock"
)

type Mock struct {
	mock.Mock
}

func (m *Mock) CreateUser(ctx context.Context, u *User) error {
	u.Password = ""
	return m.Called(u).Error(0)
}

func (m *Mock) UpdateUser(ctx context.Context, u *User) error {
	return m.Called(u).Error(0)
}

func (m *Mock) DeleteUser(ctx context.Context, ID string) error {
	return m.Called(ID).Error(0)
}
