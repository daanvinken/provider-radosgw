package radosgw

import (
	"context"
	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/daanvinken/provider-radosgw/apis/v1alpha1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CephCredentialExtractor extracts credentials from common sources.
func CephCredentialExtractor(ctx context.Context, cd v1alpha1.ProviderCredentials, kubeClient client.Client, selector xpv1.CommonCredentialSelectors) ([]byte, []byte, error) {
	secret := &corev1.Secret{}
	ns := types.NamespacedName{Namespace: cd.SecretRef.Namespace, Name: cd.SecretRef.Name}
	if err := kubeClient.Get(ctx, ns, secret); err != nil {
		return nil, nil, errors.Wrap(err, "cannot get provider secret")
	}
	return secret.Data["access_key"], secret.Data["secret_key"], nil
}
