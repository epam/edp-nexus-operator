package nexus

import (
	"context"

	"github.com/datadrivers/go-nexus-client/nexus3/schema"
	"github.com/datadrivers/go-nexus-client/nexus3/schema/blobstore"
	"github.com/datadrivers/go-nexus-client/nexus3/schema/security"
)

type User interface {
	Get(id string) (*security.User, error)
	Create(user security.User) error
	Update(id string, user security.User) error
	Delete(id string) error
}

type Role interface {
	Get(id string) (*security.Role, error)
	Create(role security.Role) error
	Update(id string, role security.Role) error
	Delete(id string) error
}

type Repository interface {
	Get(ctx context.Context, id, format, repoType string) (interface{}, error)
	Create(ctx context.Context, format, repoType string, data interface{}) error
	Update(ctx context.Context, id, format, repoType string, data interface{}) error
	Delete(ctx context.Context, id string) error
}

type Script interface {
	Get(name string) (*schema.Script, error)
	Create(script *schema.Script) error
	Update(script *schema.Script) error
	Delete(name string) error
	RunWithPayload(name, payload string) error
}

type FileBlobStore interface {
	Get(name string) (*blobstore.File, error)
	Create(bs *blobstore.File) error
	Update(name string, bs *blobstore.File) error
	Delete(name string) error
}

type S3BlobStore interface {
	Get(name string) (*blobstore.S3, error)
	Create(bs *blobstore.S3) error
	Update(name string, bs *blobstore.S3) error
	Delete(name string) error
}

type NexusCleanupPolicyManager interface {
	Get(ctx context.Context, name string) (*NexusCleanupPolicy, error)
	Create(ctx context.Context, policy *NexusCleanupPolicy) error
	Update(ctx context.Context, name string, policy *NexusCleanupPolicy) error
	Delete(ctx context.Context, name string) error
}
