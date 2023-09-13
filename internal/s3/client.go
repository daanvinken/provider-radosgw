package s3

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	apisv1alpha1 "github.com/daanvinken/provider-radosgw/apis/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

// Borrowed from https://github.com/linode/provider-ceph/blob/main/internal/s3/client.go :)
// Yes I'll give it back

const (
	defaultRegion = "us-east-1"

	accessKey = "access_key"
	secretKey = "secret_key"
)

func NewClient(ctx context.Context, secret corev1.Secret, pcSpec *apisv1alpha1.ProviderConfigSpec) (*s3.Client, error) {
	hostBase := resolveHostBase(pcSpec.HostName, pcSpec.UseHTTPS)

	endpointResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL: hostBase,
		}, nil
	})

	sessionConfig, err := config.LoadDefaultConfig(ctx, config.WithEndpointResolverWithOptions(endpointResolver))
	if err != nil {
		return nil, err
	}

	// By default make sure a region is specified, this is required for S3 operations
	region := defaultRegion
	sessionConfig.Region = aws.ToString(&region)

	sessionConfig.Credentials = aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(string(secret.Data[accessKey]), string(secret.Data[secretKey]), ""))

	return s3.NewFromConfig(sessionConfig, func(o *s3.Options) {
		o.UsePathStyle = true
	}), nil
}

func resolveHostBase(hostBase string, useHTTPS bool) string {
	httpsPrefix := "https://"
	httpPrefix := "http://"
	// Remove prefix in either case if it has been specified.
	// Let useHTTPS option take precedence.
	hostBase = strings.TrimPrefix(hostBase, httpPrefix)
	hostBase = strings.TrimPrefix(hostBase, httpsPrefix)

	if useHTTPS {
		return httpsPrefix + hostBase
	}

	return httpPrefix + hostBase
}
