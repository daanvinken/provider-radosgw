package credentials

import (
	"context"
	"fmt"
	"github.com/daanvinken/provider-radosgw/apis/v1alpha1"
	vault "github.com/hashicorp/vault/api"
	"github.com/pkg/errors"
	"os"
	"strings"
)

// Two ways of gettign a vault client, as we cannot access ProviderConfig while setting up the CephAdmin credentials from vault
func NewVaultClient(config v1alpha1.VaultConfig) (*vault.Client, error) {
	clientConfig := vault.DefaultConfig()

	clientConfig.Address = config.Address

	client, err := vault.NewClient(clientConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to initialize Vault client: %w", err)
	}

	// TODO use actual serviceaccount
	//k8sAuth, err := k8s_auth.NewKubernetesAuth(os.Getenv("VAULT_ADMIN_ROLE_NAME"), k8s_auth.WithServiceAccountTokenPath(kubernetes.token), k8s_auth.WithMountPath("kubernetes/"+os.Getenv("VAULT_ADMIN_CLUSTER_NAME")))
	//authInfo, err := client.Auth().Login(context.Background(), k8sAuth)
	err = client.SetAddress("https://vault-k8s-office.services.nlzwo1o.adyen.com")
	if err != nil {
		return &vault.Client{}, err
	}

	// Set the Vault token obtained from the CLI
	client.SetToken(os.Getenv("VAULT_TOKEN"))
	if err != nil {
		return &vault.Client{}, err
	}
	//if authInfo == nil {
	//	return client, fmt.Errorf("no auth info was returned after login")
	//}

	return client, err
}

func NewVaultClientForCephAdmins() (*vault.Client, error) {
	clientConfig := vault.DefaultConfig()
	client, err := vault.NewClient(clientConfig)
	if err != nil {
		return &vault.Client{}, err
	}

	// TODO use actual serviceaccount
	//k8sAuth, err := k8s_auth.NewKubernetesAuth(os.Getenv("VAULT_ADMIN_ROLE_NAME"), k8s_auth.WithServiceAccountTokenPath(kubernetes.token), k8s_auth.WithMountPath("kubernetes/"+os.Getenv("VAULT_ADMIN_CLUSTER_NAME")))
	//authInfo, err := client.Auth().Login(context.Background(), k8sAuth)
	err = client.SetAddress("https://vault-k8s-office.services.nlzwo1o.adyen.com")
	if err != nil {
		return &vault.Client{}, err
	}

	// Set the Vault token obtained from the CLI
	client.SetToken(os.Getenv("VAULT_TOKEN"))
	if err != nil {
		return &vault.Client{}, err
	}
	//if authInfo == nil {
	//	return client, fmt.Errorf("no auth info was returned after login")
	//}

	return client, err

}

func WriteSecretsToVault(client *vault.Client, vaultConfig v1alpha1.VaultConfig, key *string, data *map[string]interface{}) error {
	if vaultConfig.KVVersion == "1" {
		err := client.KVv1(vaultConfig.MountPath).Put(context.TODO(), *key, *data)
		if err != nil {
			return errors.Wrapf(err, "failed to write to vault kv2 at '%s'", vaultConfig.MountPath)
		}
	} else if vaultConfig.KVVersion == "2" {
		_, err := client.KVv2(vaultConfig.MountPath).Put(context.TODO(), *key, *data)
		if err != nil {
			return errors.Wrapf(err, "failed to write to vault kv2 at '%s'", vaultConfig.MountPath)
		}
	} else {
		return fmt.Errorf("unsupported KV version: %d", vaultConfig.KVVersion)
	}
	return nil
}

func BuildCephUserSecretPath(pc v1alpha1.ProviderConfig, cephUserUID string) (string, error) {
	prefix := "ceph-"
	if !strings.HasPrefix(pc.Name, prefix) {
		return "", errors.New("provider config name does not start with 'ceph-'")
	}
	cephClusterName := pc.Name[len(prefix):]
	secretPath := pc.Spec.CredentialsVault.SecretPath + "/" + cephClusterName + "/users/" + cephUserUID
	return secretPath, nil
}
