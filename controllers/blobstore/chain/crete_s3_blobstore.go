package chain

import (
	"context"
	"fmt"

	"github.com/datadrivers/go-nexus-client/nexus3/schema/blobstore"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	nexusApi "github.com/epam/edp-nexus-operator/api/v1alpha1"
	"github.com/epam/edp-nexus-operator/pkg/client/nexus"
	"github.com/epam/edp-nexus-operator/pkg/helper"
)

type CreateS3BlobStore struct {
	nexusS3BlobStoreApiClient nexus.S3BlobStore
	k8sClient                 client.Client
}

func NewCreateS3BlobStore(nexusS3BlobStoreApiClient nexus.S3BlobStore, k8sClient client.Client) *CreateS3BlobStore {
	return &CreateS3BlobStore{nexusS3BlobStoreApiClient: nexusS3BlobStoreApiClient, k8sClient: k8sClient}
}

func (c *CreateS3BlobStore) ServeRequest(ctx context.Context, blobStore *nexusApi.NexusBlobStore) error {
	if blobStore.Spec.S3 == nil {
		return nil
	}

	log := ctrl.LoggerFrom(ctx).WithValues("blobstore_name", blobStore.Spec.Name)
	log.Info("Start creating S3 blobstore")

	nexusBlobStore, err := c.specToS3Blobstore(ctx, &blobStore.Spec, blobStore.Namespace)
	if err != nil {
		return err
	}

	_, err = c.nexusS3BlobStoreApiClient.Get(blobStore.Spec.Name)
	if err != nil {
		if !nexus.IsErrNotFound(err) {
			return fmt.Errorf("failed to get blobstore: %w", err)
		}

		log.Info("Blobstore doesn't exist, creating new one")

		if err = c.nexusS3BlobStoreApiClient.Create(nexusBlobStore); err != nil {
			return fmt.Errorf("failed to create blobstore: %w", err)
		}

		log.Info("Blobstore has been created")

		return nil
	}

	log.Info("Updating blobstore")

	if err = c.nexusS3BlobStoreApiClient.Update(blobStore.Spec.Name, nexusBlobStore); err != nil {
		return fmt.Errorf("failed to update blobstore: %w", err)
	}

	log.Info("Blobstore has been updated")

	return nil
}

func (c *CreateS3BlobStore) specToS3Blobstore(
	ctx context.Context,
	spec *nexusApi.NexusBlobStoreSpec,
	namespace string,
) (*blobstore.S3, error) {
	specCopy := spec.DeepCopy()

	s3 := &blobstore.S3{
		Name: specCopy.Name,
		BucketConfiguration: blobstore.S3BucketConfiguration{
			Bucket: blobstore.S3Bucket{
				Region:     specCopy.S3.Bucket.Region,
				Name:       specCopy.S3.Bucket.Name,
				Prefix:     specCopy.S3.Bucket.Prefix,
				Expiration: specCopy.S3.Bucket.Expiration,
			},
		},
	}

	if specCopy.SoftQuota != nil {
		s3.SoftQuota = &blobstore.SoftQuota{
			Limit: specCopy.SoftQuota.Limit,
			Type:  specCopy.SoftQuota.Type,
		}
	}

	if specCopy.S3.Encryption != nil {
		s3.BucketConfiguration.Encryption = &blobstore.S3Encryption{
			Key:  specCopy.S3.Encryption.Key,
			Type: specCopy.S3.Encryption.Type,
		}
	}

	if specCopy.S3.AdvancedBucketConnection != nil {
		s3.BucketConfiguration.AdvancedBucketConnection = &blobstore.S3AdvancedBucketConnection{
			Endpoint:              specCopy.S3.AdvancedBucketConnection.Endpoint,
			SignerType:            specCopy.S3.AdvancedBucketConnection.SignerType,
			ForcePathStyle:        &specCopy.S3.AdvancedBucketConnection.ForcePathStyle,
			MaxConnectionPoolSize: &specCopy.S3.AdvancedBucketConnection.MaxConnectionPoolSize,
		}
	}

	if specCopy.S3.BucketSecurity != nil {
		bucketSecurity, err := c.s3BucketSecurityToS3Blobstore(ctx, specCopy.S3.BucketSecurity, namespace)
		if err != nil {
			return nil, err
		}

		s3.BucketConfiguration.BucketSecurity = bucketSecurity
	}

	return s3, nil
}

func (c *CreateS3BlobStore) s3BucketSecurityToS3Blobstore(
	ctx context.Context,
	s3BucketSecuritySpec *nexusApi.S3BucketSecurity,
	namespace string,
) (*blobstore.S3BucketSecurity, error) {
	bucketSecurity := &blobstore.S3BucketSecurity{
		AccessKeyID:     "",
		Role:            s3BucketSecuritySpec.Role,
		SecretAccessKey: "",
		SessionToken:    "",
	}

	accessKeyID, err := helper.GetValueFromSourceRef(ctx, &s3BucketSecuritySpec.AccessKeyID, namespace, c.k8sClient)
	if err != nil {
		return nil, fmt.Errorf("failed to get access key ID: %w", err)
	}

	bucketSecurity.AccessKeyID = accessKeyID

	secretAccessKey, err := helper.GetValueFromSourceRef(ctx, &s3BucketSecuritySpec.SecretAccessKey, namespace, c.k8sClient)
	if err != nil {
		return nil, fmt.Errorf("failed to get secret access key: %w", err)
	}

	bucketSecurity.SecretAccessKey = secretAccessKey

	if s3BucketSecuritySpec.SessionToken != nil {
		sessionToken, err := helper.GetValueFromSourceRef(ctx, s3BucketSecuritySpec.SessionToken, namespace, c.k8sClient)
		if err != nil {
			return nil, fmt.Errorf("failed to get session token: %w", err)
		}

		bucketSecurity.SessionToken = sessionToken
	}

	return bucketSecurity, nil
}
