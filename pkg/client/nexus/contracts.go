package nexus

import (
	"github.com/datadrivers/go-nexus-client/nexus3/schema/security"
)

//go:generate mockery --name User --filename user_mock.go
type User interface {
	Get(id string) (*security.User, error)
}
