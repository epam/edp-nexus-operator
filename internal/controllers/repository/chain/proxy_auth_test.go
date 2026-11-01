package chain

import (
	"context"
	"testing"

	"github.com/datadrivers/go-nexus-client/nexus3/schema/repository"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/epam/edp-nexus-operator/api/common"
	"github.com/epam/edp-nexus-operator/pkg/client/nexus"
)

const testNamespace = "ns"

func newFakeClient(t *testing.T, data map[string][]byte) client.Client {
	t.Helper()

	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))

	return fake.NewClientBuilder().WithScheme(scheme).WithObjects(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "cred", Namespace: testNamespace},
		Data:       data,
	}).Build()
}

func ref(name, key string) *common.SecretKeySelector {
	s := &common.SecretKeySelector{Key: key}
	s.Name = name

	return s
}

func TestResolveAuthentication(t *testing.T) {
	t.Parallel()

	preemptive := true

	tests := []struct {
		name       string
		auth       *nexus.ProxyAuth
		maven      bool
		secretData map[string][]byte
		want       *repository.HTTPClientAuthenticationWithPreemptive
		wantSecret string
		wantErr    string
	}{
		{
			name: "plain password",
			auth: &nexus.ProxyAuth{Type: "username", Username: "alice", Password: "pw-1"},
			want: &repository.HTTPClientAuthenticationWithPreemptive{Type: "username", Username: "alice", Password: "pw-1"},
		},
		{
			name:       "password reference",
			auth:       &nexus.ProxyAuth{Type: "username", Username: "alice", PasswordRef: ref("cred", "password")},
			secretData: map[string][]byte{"password": []byte("pw-1")},
			want:       &repository.HTTPClientAuthenticationWithPreemptive{Type: "username", Username: "alice", Password: "pw-1"},
			wantSecret: "pw-1",
		},
		{
			name: "ntlm with password reference",
			auth: &nexus.ProxyAuth{
				Type: "ntlm", Username: "alice", NTLMHost: "h", NTLMDomain: "d", PasswordRef: ref("cred", "password"),
			},
			secretData: map[string][]byte{"password": []byte("pw-1")},
			want: &repository.HTTPClientAuthenticationWithPreemptive{
				Type: "ntlm", Username: "alice", NTLMHost: "h", NTLMDomain: "d", Password: "pw-1",
			},
			wantSecret: "pw-1",
		},
		{
			name:       "maven password reference keeps preemptive",
			maven:      true,
			auth:       &nexus.ProxyAuth{Type: "username", Username: "alice", PasswordRef: ref("cred", "password"), Preemptive: &preemptive},
			secretData: map[string][]byte{"password": []byte("pw-1")},
			want: &repository.HTTPClientAuthenticationWithPreemptive{
				Type: "username", Username: "alice", Password: "pw-1", Preemptive: &preemptive,
			},
			wantSecret: "pw-1",
		},
		{
			name:  "maven preemptive only",
			maven: true,
			auth:  &nexus.ProxyAuth{Type: "username", Preemptive: &preemptive},
			want:  &repository.HTTPClientAuthenticationWithPreemptive{Type: "username", Preemptive: &preemptive},
		},
		{
			name:    "password and password reference",
			auth:    &nexus.ProxyAuth{Type: "username", Username: "alice", Password: "pw-1", PasswordRef: ref("cred", "password")},
			wantErr: "password and passwordRef are mutually exclusive",
		},
		{
			name:    "username only",
			auth:    &nexus.ProxyAuth{Type: "username", Username: "alice"},
			wantErr: "one of password or passwordRef must be set",
		},
		{
			name:    "maven username only",
			maven:   true,
			auth:    &nexus.ProxyAuth{Type: "username", Username: "alice"},
			wantErr: "username requires password or passwordRef, and a password requires username",
		},
		{
			name:       "maven password reference without username",
			maven:      true,
			auth:       &nexus.ProxyAuth{Type: "username", PasswordRef: ref("cred", "password")},
			secretData: map[string][]byte{"password": []byte("pw-1")},
			wantErr:    "username requires password or passwordRef, and a password requires username",
		},
		{
			name:    "maven password without username",
			maven:   true,
			auth:    &nexus.ProxyAuth{Type: "username", Password: "pw-1"},
			wantErr: "username requires password or passwordRef, and a password requires username",
		},
		{
			name:    "reference with empty key",
			auth:    &nexus.ProxyAuth{Type: "username", Username: "alice", PasswordRef: ref("cred", "")},
			wantErr: "passwordRef name and key must not be empty",
		},
		{
			name:    "reference with empty name",
			auth:    &nexus.ProxyAuth{Type: "username", Username: "alice", PasswordRef: ref("", "password")},
			wantErr: "passwordRef name and key must not be empty",
		},
		{
			name:       "missing key",
			auth:       &nexus.ProxyAuth{Type: "username", Username: "alice", PasswordRef: ref("cred", "absent")},
			secretData: map[string][]byte{"password": []byte("pw-1")},
			wantErr:    "secret cred does not contain key absent",
		},
		{
			name:       "missing secret",
			auth:       &nexus.ProxyAuth{Type: "username", Username: "alice", PasswordRef: ref("absent", "password")},
			secretData: map[string][]byte{"password": []byte("pw-1")},
			wantErr:    "unable to get secret absent",
		},
		{
			name:       "empty value",
			auth:       &nexus.ProxyAuth{Type: "username", Username: "alice", PasswordRef: ref("cred", "password")},
			secretData: map[string][]byte{"password": {}},
			wantErr:    "secret cred key password is empty",
		},
		{
			name:       "trailing newline",
			auth:       &nexus.ProxyAuth{Type: "username", Username: "alice", PasswordRef: ref("cred", "password")},
			secretData: map[string][]byte{"password": []byte("pw-1\n")},
			wantErr:    "secret cred key password has leading or trailing whitespace",
		},
		{
			name:       "leading space",
			auth:       &nexus.ProxyAuth{Type: "username", Username: "alice", PasswordRef: ref("cred", "password")},
			secretData: map[string][]byte{"password": []byte(" pw-1")},
			wantErr:    "secret cred key password has leading or trailing whitespace",
		},
		{
			name:       "whitespace inside is kept",
			auth:       &nexus.ProxyAuth{Type: "username", Username: "alice", PasswordRef: ref("cred", "password")},
			secretData: map[string][]byte{"password": []byte("pw 1")},
			want:       &repository.HTTPClientAuthenticationWithPreemptive{Type: "username", Username: "alice", Password: "pw 1"},
			wantSecret: "pw 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := resolveAuthentication(
				context.Background(), newFakeClient(t, tt.secretData), testNamespace, tt.auth, tt.maven,
			)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				require.Nil(t, got)

				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.want, got.wire)
			require.Equal(t, tt.wantSecret, got.secretPassword)
		})
	}
}

func TestResolveAuthentication_errorsDoNotContainValue(t *testing.T) {
	t.Parallel()

	auth := &nexus.ProxyAuth{Type: "username", Username: "alice", PasswordRef: ref("cred", "password")}

	_, err := resolveAuthentication(
		context.Background(),
		newFakeClient(t, map[string][]byte{"password": []byte("s3cret\n")}),
		testNamespace, auth, false,
	)
	require.Error(t, err)
	require.NotContains(t, err.Error(), "s3cret")
}

func TestRedactError(t *testing.T) {
	t.Parallel()

	require.NoError(t, redactError(nil, "pw-1"))

	err := redactError(errBody("bad request: password=pw-1 rejected, pw-1"), "pw-1")
	require.EqualError(t, err, "bad request: password=<redacted> rejected, <redacted>")

	orig := errBody("bad request")
	require.Equal(t, orig, redactError(orig, ""))
}

func TestRedactError_jsonEscapedValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		password string
		echo     string
		want     string
	}{
		{
			name:     "quote and backslash",
			password: `p"w\1`,
			echo:     `{"message":"bad p\"w\\1"} and p"w\1`,
			want:     `{"message":"bad <redacted>"} and <redacted>`,
		},
		{
			name:     "html characters, echo without HTML escaping",
			password: `p"w&x<y`,
			echo:     `{"message":"bad p\"w&x<y"} and p"w&x<y`,
			want:     `{"message":"bad <redacted>"} and <redacted>`,
		},
		{
			name:     "html characters, echo with HTML escaping",
			password: `p"w&x<y`,
			echo:     `{"message":"bad p\"w\u0026x\u003cy"} and p"w&x<y`,
			want:     `{"message":"bad <redacted>"} and <redacted>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := redactError(errBody(tt.echo), tt.password)
			require.EqualError(t, err, tt.want)
		})
	}
}

type errBody string

func (e errBody) Error() string { return string(e) }
