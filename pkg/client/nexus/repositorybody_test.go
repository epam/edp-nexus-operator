package nexus

import (
	"encoding/json"
	"testing"

	"github.com/datadrivers/go-nexus-client/nexus3/schema/repository"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"

	"github.com/epam/edp-nexus-operator/api/common"
	"github.com/epam/edp-nexus-operator/api/v1alpha1"
)

func TestWithAuthentication(t *testing.T) {
	t.Parallel()

	const hugeInt = 9007199254740993 // 2^53 + 1, not representable as float64

	passwordRef := &common.SecretKeySelector{Key: "password"}
	passwordRef.Name = "cred"

	data := &v1alpha1.GoProxyRepository{
		ProxySpec: v1alpha1.ProxySpec{
			Name:   "go-proxy",
			Online: true,
			Proxy: v1alpha1.Proxy{
				ContentMaxAge: hugeInt,
				RemoteURL:     "https://remote.example",
			},
			HTTPClient: v1alpha1.HTTPClient{
				AutoBlock: true,
				Authentication: &v1alpha1.HTTPClientAuthentication{
					Type:        "username",
					Username:    "alice",
					PasswordRef: passwordRef,
				},
			},
		},
	}

	got, err := WithAuthentication(data, &repository.HTTPClientAuthenticationWithPreemptive{
		Type:     repository.HTTPClientAuthenticationTypeUsername,
		Username: "alice",
		Password: "pw-1",
	})
	require.NoError(t, err)

	raw, err := json.Marshal(got)
	require.NoError(t, err)

	require.JSONEq(t, `{
		"name": "go-proxy",
		"online": true,
		"storage": {"blobStoreName": "", "strictContentTypeValidation": false},
		"proxy": {"contentMaxAge": 9007199254740993, "metadataMaxAge": 0, "remoteUrl": "https://remote.example"},
		"negativeCache": {"enabled": false, "timeToLive": 0},
		"httpClient": {
			"autoBlock": true,
			"blocked": false,
			"authentication": {"type": "username", "username": "alice", "password": "pw-1"}
		}
	}`, string(raw))
	require.NotContains(t, string(raw), "passwordRef")
	require.Contains(t, string(raw), "9007199254740993")

	require.Equal(t, passwordRef, data.Authentication.PasswordRef, "input is not modified")
	//nolint:staticcheck // covers the deprecated password field
	require.Empty(t, data.Authentication.Password, "input is not modified")
}

func TestWithAuthentication_noHTTPClient(t *testing.T) {
	t.Parallel()

	got, err := WithAuthentication(map[string]any{"name": "x"}, &repository.HTTPClientAuthenticationWithPreemptive{
		Type: repository.HTTPClientAuthenticationTypeUsername, Username: "alice", Password: "pw-1",
	})
	require.NoError(t, err)

	raw, err := json.Marshal(got)
	require.NoError(t, err)
	require.JSONEq(t, `{
		"name": "x",
		"httpClient": {"authentication": {"type": "username", "username": "alice", "password": "pw-1"}}
	}`, string(raw))
}

func TestWithAuthentication_matchesCRDBodyWithPlainPassword(t *testing.T) {
	t.Parallel()

	data := &v1alpha1.MavenProxyRepository{
		HTTPClient: v1alpha1.HTTPClientWithPreemptiveAuth{
			Authentication: &v1alpha1.HTTPClientAuthenticationWithPreemptive{
				Type: "username", Username: "alice", Password: "pw-1", Preemptive: ptr.To(true),
			},
		},
	}

	want, err := json.Marshal(data)
	require.NoError(t, err)

	got, err := WithAuthentication(data, &repository.HTTPClientAuthenticationWithPreemptive{
		Type: repository.HTTPClientAuthenticationTypeUsername, Username: "alice", Password: "pw-1", Preemptive: ptr.To(true),
	})
	require.NoError(t, err)

	raw, err := json.Marshal(got)
	require.NoError(t, err)
	require.JSONEq(t, string(want), string(raw))
}

func TestWithAuthentication_nilData(t *testing.T) {
	t.Parallel()

	auth := &repository.HTTPClientAuthenticationWithPreemptive{
		Type: repository.HTTPClientAuthenticationTypeUsername, Username: "alice", Password: "pw-1",
	}

	require.NotPanics(t, func() {
		got, err := WithAuthentication((*v1alpha1.GoProxyRepository)(nil), auth)
		require.ErrorContains(t, err, "not a JSON object")
		require.Nil(t, got)
	})
}
