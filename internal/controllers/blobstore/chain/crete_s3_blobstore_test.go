package chain

import (
	"context"
	"errors"
	"testing"

	"github.com/datadrivers/go-nexus-client/nexus3/schema/blobstore"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/epam/edp-nexus-operator/api/common"
	nexusApi "github.com/epam/edp-nexus-operator/api/v1alpha1"
	"github.com/epam/edp-nexus-operator/pkg/client/nexus"
	"github.com/epam/edp-nexus-operator/pkg/client/nexus/mocks"
)

func TestCreateS3BlobStore_ServeRequest(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))

	tests := []struct {
		name                    string
		blobStore               *nexusApi.NexusBlobStore
		nexusBlobStoreApiClient func(t *testing.T) nexus.S3BlobStore
		k8sClient               func(t *testing.T) client.Client
		wantErr                 require.ErrorAssertionFunc
	}{
		{
			name: "blobstore doesn't exist, creating new one",
			blobStore: &nexusApi.NexusBlobStore{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-blobstore",
					Namespace: "default",
				},
				Spec: nexusApi.NexusBlobStoreSpec{
					Name: "test-blobstore",
					SoftQuota: &nexusApi.SoftQuota{
						Type:  nexusApi.SoftQuotaSpaceRemainingQuota,
						Limit: 100,
					},
					S3: &nexusApi.S3{
						Bucket: nexusApi.S3Bucket{
							Region:     "us-east-1",
							Name:       "test-bucket",
							Prefix:     "bucket-prefix",
							Expiration: 3,
						},
						Encryption: &nexusApi.S3Encryption{
							Type: nexusApi.S3BucketEncryptionTypeKms,
							Key:  "test-key",
						},
						BucketSecurity: &nexusApi.S3BucketSecurity{
							AccessKeyID: common.SourceRef{
								ConfigMapKeyRef: &common.ConfigMapKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "cm-with-keys",
									},
									Key: "access-key-id",
								},
							},
							SecretAccessKey: common.SourceRef{
								ConfigMapKeyRef: &common.ConfigMapKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "cm-with-keys",
									},
									Key: "access-key",
								},
							},
							Role: "test-role",
							SessionToken: &common.SourceRef{
								SecretKeyRef: &common.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "secret-with-token",
									},
									Key: "token",
								},
							},
						},
						AdvancedBucketConnection: &nexusApi.S3AdvancedBucketConnection{
							Endpoint:              "https://test-endpoint",
							SignerType:            nexusApi.S3SingerTypeS3,
							ForcePathStyle:        true,
							MaxConnectionPoolSize: 10,
						},
					},
				},
			},
			nexusBlobStoreApiClient: func(t *testing.T) nexus.S3BlobStore {
				m := mocks.NewMockS3BlobStore(t)

				m.On("Get", "test-blobstore").
					Return(nil, errors.New("not found"))

				m.On("Create", &blobstore.S3{
					Name: "test-blobstore",
					SoftQuota: &blobstore.SoftQuota{
						Type:  nexusApi.SoftQuotaSpaceRemainingQuota,
						Limit: 100,
					},
					BucketConfiguration: blobstore.S3BucketConfiguration{
						Bucket: blobstore.S3Bucket{
							Region:     "us-east-1",
							Name:       "test-bucket",
							Prefix:     "bucket-prefix",
							Expiration: 3,
						},
						Encryption: &blobstore.S3Encryption{
							Key:  "test-key",
							Type: nexusApi.S3BucketEncryptionTypeKms,
						},
						BucketSecurity: &blobstore.S3BucketSecurity{
							AccessKeyID:     "access-key-id",
							Role:            "test-role",
							SecretAccessKey: "access-key",
							SessionToken:    "token",
						},
						AdvancedBucketConnection: &blobstore.S3AdvancedBucketConnection{
							Endpoint:              "https://test-endpoint",
							SignerType:            nexusApi.S3SingerTypeS3,
							ForcePathStyle:        ptr.To(true),
							MaxConnectionPoolSize: ptr.To(int32(10)),
						},
					},
				}).Return(nil)

				return m
			},
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().WithScheme(scheme).WithObjects(
					&corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "cm-with-keys",
							Namespace: "default",
						},
						Data: map[string]string{
							"access-key-id": "access-key-id",
							"access-key":    "access-key",
						},
					},
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "secret-with-token",
							Namespace: "default",
						},
						Data: map[string][]byte{
							"token": []byte("token"),
						},
					},
				).Build()
			},
			wantErr: require.NoError,
		},
		{
			name: "blobstore exists, updating it",
			blobStore: &nexusApi.NexusBlobStore{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-blobstore",
					Namespace: "default",
				},
				Spec: nexusApi.NexusBlobStoreSpec{
					Name: "test-blobstore",
					S3: &nexusApi.S3{
						Bucket: nexusApi.S3Bucket{
							Name:   "test-bucket",
							Region: "us-east-1",
						},
					},
				},
			},
			nexusBlobStoreApiClient: func(t *testing.T) nexus.S3BlobStore {
				m := mocks.NewMockS3BlobStore(t)

				m.On("Get", "test-blobstore").
					Return(&blobstore.S3{
						Name: "test-blobstore",
					}, nil)

				m.On("Update", "test-blobstore", &blobstore.S3{
					Name: "test-blobstore",
					BucketConfiguration: blobstore.S3BucketConfiguration{
						Bucket: blobstore.S3Bucket{
							Name:   "test-bucket",
							Region: "us-east-1",
						},
					},
				}).Return(nil)

				return m
			},
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().WithScheme(scheme).Build()
			},
			wantErr: require.NoError,
		},
		{
			name: "failed to update blobstore",
			blobStore: &nexusApi.NexusBlobStore{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-blobstore",
					Namespace: "default",
				},
				Spec: nexusApi.NexusBlobStoreSpec{
					Name: "test-blobstore",
					S3: &nexusApi.S3{
						Bucket: nexusApi.S3Bucket{
							Name:   "test-bucket",
							Region: "us-east-1",
						},
					},
				},
			},
			nexusBlobStoreApiClient: func(t *testing.T) nexus.S3BlobStore {
				m := mocks.NewMockS3BlobStore(t)

				m.On("Get", "test-blobstore").
					Return(&blobstore.S3{
						Name: "test-blobstore",
					}, nil)

				m.On("Update", "test-blobstore", mock.Anything).
					Return(errors.New("failed to update blobstore"))

				return m
			},
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().WithScheme(scheme).Build()
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to update blobstore")
			},
		},
		{
			name: "failed to get session token",
			blobStore: &nexusApi.NexusBlobStore{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-blobstore",
					Namespace: "default",
				},
				Spec: nexusApi.NexusBlobStoreSpec{
					Name: "test-blobstore",
					SoftQuota: &nexusApi.SoftQuota{
						Type:  nexusApi.SoftQuotaSpaceRemainingQuota,
						Limit: 100,
					},
					S3: &nexusApi.S3{
						Bucket: nexusApi.S3Bucket{
							Name: "test-bucket",
						},
						BucketSecurity: &nexusApi.S3BucketSecurity{
							SessionToken: &common.SourceRef{
								SecretKeyRef: &common.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "secret-with-token",
									},
									Key: "token",
								},
							},
						},
					},
				},
			},
			nexusBlobStoreApiClient: func(t *testing.T) nexus.S3BlobStore {
				return mocks.NewMockS3BlobStore(t)
			},
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().WithScheme(scheme).Build()
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to get session token")
			},
		},
		{
			name: "failed to create blobstore",
			blobStore: &nexusApi.NexusBlobStore{
				Spec: nexusApi.NexusBlobStoreSpec{
					Name: "test-blobstore",
					S3: &nexusApi.S3{
						Bucket: nexusApi.S3Bucket{
							Name: "test-bucket",
						},
					},
				},
			},
			nexusBlobStoreApiClient: func(t *testing.T) nexus.S3BlobStore {
				m := mocks.NewMockS3BlobStore(t)

				m.On("Get", "test-blobstore").
					Return(nil, errors.New("not found"))

				m.On("Create", mock.Anything).Return(errors.New("failed to create blobstore"))

				return m
			},
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().WithScheme(scheme).Build()
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to create blobstore")
			},
		},
		{
			name: "failed to get blobstore",
			blobStore: &nexusApi.NexusBlobStore{
				Spec: nexusApi.NexusBlobStoreSpec{
					Name: "test-blobstore",
					S3: &nexusApi.S3{
						Bucket: nexusApi.S3Bucket{
							Name: "test-bucket",
						},
					},
				},
			},
			nexusBlobStoreApiClient: func(t *testing.T) nexus.S3BlobStore {
				m := mocks.NewMockS3BlobStore(t)

				m.On("Get", "test-blobstore").
					Return(nil, errors.New("failed to get blobstore"))

				return m
			},
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().WithScheme(scheme).Build()
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to get blobstore")
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := NewCreateS3BlobStore(tt.nexusBlobStoreApiClient(t), tt.k8sClient(t))

			err := c.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), tt.blobStore)
			tt.wantErr(t, err)
		})
	}
}
