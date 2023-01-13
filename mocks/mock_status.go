package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Status struct {
	sw client.StatusWriter
}

func (st *Status) Status() client.StatusWriter {
	return st.sw
}

type StatusWriter struct {
	mock.Mock
}

func (s *StatusWriter) Create(ctx context.Context, obj client.Object, subResource client.Object, opts ...client.SubResourceCreateOption) error {
	panic("call to an unimplemented stub method used to satisfy the interface")
}

func (s *StatusWriter) Update(ctx context.Context, obj client.Object, opts ...client.SubResourceUpdateOption) error {

	called := s.Called()
	parent, ok := called.Get(0).(client.StatusWriter)
	if ok {
		return parent.Update(ctx, obj, opts...)
	}

	return called.Error(0)
}

func (s *StatusWriter) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.SubResourcePatchOption) error {
	called := s.Called(obj, patch, opts)
	parent, ok := called.Get(0).(client.StatusWriter)
	if ok {
		return parent.Patch(ctx, obj, patch, opts...)
	}

	return called.Error(0)
}
