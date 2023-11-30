package nexus

import (
	"context"

	"github.com/datadrivers/go-nexus-client/nexus3/schema/security"
)

//go:generate mockery --name User --filename user_mock.go
type User interface {
	Get(id string) (*security.User, error)
	Create(user security.User) error
	Update(id string, user security.User) error
	Delete(id string) error
}

//go:generate mockery --name Role --filename role_mock.go
type Role interface {
	Get(id string) (*security.Role, error)
	Create(role security.Role) error
	Update(id string, role security.Role) error
	Delete(id string) error
}

//go:generate mockery --name Repository --filename repository_mock.go
type Repository interface {
	Get(ctx context.Context, id, format, repoType string) (interface{}, error)
	Create(ctx context.Context, format, repoType string, data interface{}) error
	Update(ctx context.Context, id, format, repoType string, data interface{}) error
	Delete(ctx context.Context, id string) error
}
