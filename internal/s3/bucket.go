package s3

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/daanvinken/provider-radosgw/apis/s3/v1alpha1"
	"github.com/pkg/errors"
)

func GenerateBucketInput(bucket *v1alpha1.Bucket) *s3.CreateBucketInput {
	createBucketInput := &s3.CreateBucketInput{
		//ACL:    s3types.BucketCannedACL(aws.ToString(bucket.Spec.ForProvider.ACL)),
		Bucket: aws.String(bucket.Name),
		//GrantFullControl:           bucket.Spec.ForProvider.GrantFullControl,
		//GrantRead:                  bucket.Spec.ForProvider.GrantRead,
		//GrantReadACP:               bucket.Spec.ForProvider.GrantReadACP,
		//GrantWrite:                 bucket.Spec.ForProvider.GrantWrite,
		//GrantWriteACP:              bucket.Spec.ForProvider.GrantWriteACP,
		//ObjectLockEnabledForBucket: aws.ToBool(bucket.Spec.ForProvider.ObjectLockEnabledForBucket),
		//ObjectOwnership:            s3types.ObjectOwnership(aws.ToString(bucket.Spec.ForProvider.ObjectOwnership)),
	}

	if bucket.Spec.ForProvider.LocationConstraint != "" {
		createBucketInput.CreateBucketConfiguration = &s3types.CreateBucketConfiguration{
			LocationConstraint: s3types.BucketLocationConstraint(bucket.Spec.ForProvider.LocationConstraint),
		}
	}

	return createBucketInput
}

func BucketExists(ctx context.Context, s3Backend s3.Client, bucketName string) (bool, error) {
	_, err := s3Backend.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(bucketName)})
	if err != nil {
		return false, resource.Ignore(isNotFound, err)
	}
	// Bucket exists, return true with no error.
	return true, nil
}

// isNotFound helper function to test for NotFound error
func isNotFound(err error) bool {
	var notFoundError *s3types.NotFound

	return errors.As(err, &notFoundError)
}
