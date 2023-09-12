package radosgw

import (
	"context"
	"github.com/daanvinken/provider-radosgw/apis/v1alpha1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// CephCredentialExtractor extracts credentials from common sources.
func CephCredentialExtractor(ctx context.Context, cd v1alpha1.ProviderCredentials, kubeClient client.Client) ([]byte, []byte, error) {
	secret := &corev1.Secret{}
	ns := types.NamespacedName{Namespace: cd.SecretRef.Namespace, Name: cd.SecretRef.Name}
	if err := kubeClient.Get(ctx, ns, secret); err != nil {
		return nil, nil, errors.Wrap(err, "cannot get provider secret")
	}
	return secret.Data["access_key"], secret.Data["secret_key"], nil
}

func generateRandomSecret(length int) string {
	rand.Seed(time.Now().UnixNano())
	secret := make([]byte, length)

	for i := 0; i < length; i++ {
		secret[i] = charset[rand.Intn(len(charset))]
	}
	return string(secret)
}

func CreateKubernetesSecretCephUser(accessKey string, secretKey string, namespace string, cephUserUID string) corev1.Secret {
	secretData := map[string][]byte{
		"access_key": []byte(accessKey),
		"secret_key": []byte(secretKey),
	}

	secretName := cephUserUID + "-credentials"
	return corev1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
		},
		Data: secretData,
		Type: corev1.SecretTypeOpaque,
	}
}
