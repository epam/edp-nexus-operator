package nexus

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepoClient_Get(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/service/rest/v1/repositories/go/proxy/go-success":
			rw.WriteHeader(http.StatusOK)
			rw.Write([]byte(`{"res":"success"}`))
		case "/service/rest/v1/repositories/go/proxy/go-not-found":
			rw.WriteHeader(http.StatusNotFound)
			rw.Write([]byte(`{"res":"not found"}`))
		case "/service/rest/v1/repositories/go/proxy/go-error":
			rw.WriteHeader(http.StatusInternalServerError)
			rw.Write([]byte(`{"res":"error"}`))
		default:
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.Path)
		}
	}))
	defer server.Close()

	tests := []struct {
		name    string
		id      string
		want    interface{}
		wantErr require.ErrorAssertionFunc
	}{
		{
			name:    "success",
			id:      "go-success",
			want:    map[string]interface{}{"res": "success"},
			wantErr: require.NoError,
		},
		{
			name: "not found",
			id:   "go-not-found",
			want: nil,
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.ErrorIs(t, err, ErrNotFound)
			},
		},
		{
			name: "error",
			id:   "go-error",
			want: nil,
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to get repository")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewRepoClient(ClientConfig{BaseURL: server.URL})

			got, err := s.Get(context.Background(), tt.id, FormatGo, TypeProxy)

			tt.wantErr(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRepoClient_Update(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rawdata, err := io.ReadAll(req.Body)
		require.NoError(t, err)

		data := map[string]string{}
		err = json.Unmarshal(rawdata, &data)
		require.NoError(t, err)

		switch data["res"] {
		case "success":
			rw.WriteHeader(http.StatusOK)
		case "error":
			rw.WriteHeader(http.StatusInternalServerError)
		default:
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.Path)
		}
	}))
	defer server.Close()

	tests := []struct {
		name    string
		data    map[string]string
		wantErr require.ErrorAssertionFunc
	}{
		{
			name:    "success",
			data:    map[string]string{"res": "success"},
			wantErr: require.NoError,
		},
		{
			name: "error",
			data: map[string]string{"res": "error"},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to create repository")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewRepoClient(ClientConfig{BaseURL: server.URL})

			err := s.Create(context.Background(), FormatGo, TypeProxy, tt.data)

			tt.wantErr(t, err)
		})
	}
}

func TestRepoClient_Update1(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rawdata, err := io.ReadAll(req.Body)
		require.NoError(t, err)

		data := map[string]string{}
		err = json.Unmarshal(rawdata, &data)
		require.NoError(t, err)

		switch data["res"] {
		case "success":
			rw.WriteHeader(http.StatusOK)
		case "error":
			rw.WriteHeader(http.StatusInternalServerError)
		default:
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.Path)
		}
	}))
	defer server.Close()

	tests := []struct {
		name    string
		data    map[string]string
		wantErr require.ErrorAssertionFunc
	}{
		{
			name:    "success",
			data:    map[string]string{"res": "success"},
			wantErr: require.NoError,
		},
		{
			name: "error",
			data: map[string]string{"res": "error"},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to update repository")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewRepoClient(ClientConfig{BaseURL: server.URL})

			err := s.Update(context.Background(), "go", FormatGo, TypeProxy, tt.data)

			tt.wantErr(t, err)
		})
	}
}

func TestRepoClient_Delete(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/service/rest/v1/repositories/go-success":
			rw.WriteHeader(http.StatusOK)
			rw.Write([]byte(`{"res":"success"}`))
		case "/service/rest/v1/repositories/go-not-found":
			rw.WriteHeader(http.StatusNotFound)
			rw.Write([]byte(`{"res":"not found"}`))
		case "/service/rest/v1/repositories/go-error":
			rw.WriteHeader(http.StatusInternalServerError)
			rw.Write([]byte(`{"res":"error"}`))
		default:
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.Path)
		}
	}))
	defer server.Close()

	tests := []struct {
		name    string
		id      string
		wantErr require.ErrorAssertionFunc
	}{
		{
			name:    "success",
			id:      "go-success",
			wantErr: require.NoError,
		},
		{
			name: "not found",
			id:   "go-not-found",
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.ErrorIs(t, err, ErrNotFound)
			},
		},
		{
			name: "error",
			id:   "go-error",
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to delete repository")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewRepoClient(ClientConfig{BaseURL: server.URL})

			err := s.Delete(context.Background(), tt.id)

			tt.wantErr(t, err)
		})
	}
}
