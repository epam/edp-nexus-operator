package nexus

import (
	"github.com/datadrivers/go-nexus-client/nexus3/schema/security"
)

//go:generate mockery --name User --filename user_mock.go
type User interface {
	Get(id string) (*security.User, error)
}

//go:generate mockery --name Role --filename role_mock.go
type Role interface {
	Get(id string) (*security.Role, error)
	Create(role security.Role) error
	Update(id string, role security.Role) error
	Delete(id string) error
}
