package vault

import (
	"context"
	"fmt"
	"github.com/daanvinken/provider-radosgw/apis/v1alpha1"
	"github.com/daanvinken/provider-radosgw/internal/utils"
	vault "github.com/hashicorp/vault/api"
	k8s_auth "github.com/hashicorp/vault/api/auth/kubernetes"
	"github.com/pkg/errors"
	"os"
	"strings"
)

func NewVaultClient(config v1alpha1.VaultConfig) (*vault.Client, error) {
	clientConfig := vault.DefaultConfig()

	clientConfig.Address = config.Address

	client, err := vault.NewClient(clientConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to initialize Vault client: %w", err)
	}

	if os.Getenv("VAULT_TOKEN") != "" && os.Getenv("VAULT_ADDR") != "" {
		err = client.SetAddress(os.Getenv("VAULT_ADDR"))
		if err != nil {
			return &vault.Client{}, err
		}
	} else {
		k8sAuth, err := k8s_auth.NewKubernetesAuth(config.ServiceAccountName,
			k8s_auth.WithServiceAccountTokenPath(
				utils.Getenv("SA_TOKEN_PATH", "/var/run/secrets/kubernetes.io/serviceaccount/token")))

		if err != nil {
			return client, errors.Wrap(err, "failed to setup kubernetes auth for vault")
		}
		authInfo, err := client.Auth().Login(context.Background(), k8sAuth)
		if err != nil {
			return client, errors.Wrap(err, "failed to authenticate to vault")
		}

		if authInfo == nil {
			return client, fmt.Errorf("no auth info was returned after login")
		}
	}

	return client, err
}

func NewVaultClientForCephAdmins() (*vault.Client, error) {
	// This method is for in the early stage crossplane setup where we do not have access to VaultConfig yet
	// (which is part of the providerconfig)
	vaultConfig := v1alpha1.VaultConfig{
		ServiceAccountName: utils.Getenv("VAULT_CEPH_ADMIN_ROLE", "crossplane-ceph-admin"),
		Address:            utils.Getenv("VAULT_CEPH_ADMIN_ADDR", "http://localhost:8200"),
	}
	return NewVaultClient(vaultConfig)
}

func NewVaultClientWithPanic(config v1alpha1.VaultConfig) *vault.Client {
	client, err := NewVaultClient(config)
	if err != nil {
		panic(err)
	}
	return client
}

func WriteSecretsToVault(client *vault.Client, vaultConfig v1alpha1.VaultConfig, key *string, data *map[string]interface{}) error {
	if vaultConfig.KVVersion == "1" {
		err := client.KVv1(vaultConfig.MountPath).Put(context.TODO(), *key, *data)
		if err != nil {
			return errors.Wrapf(err, "failed to write to vault kv1 at '%s'", vaultConfig.MountPath)
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

func RemoveSecretFromVault(client *vault.Client, vaultConfig v1alpha1.VaultConfig, key *string) error {
	if vaultConfig.KVVersion == "1" {
		err := client.KVv1(vaultConfig.MountPath).Delete(context.TODO(), *key)
		if err != nil {
			return errors.Wrapf(err, "failed to delete from vault kv1 at '%s'", vaultConfig.MountPath)
		}
	} else if vaultConfig.KVVersion == "2" {
		err := client.KVv2(vaultConfig.MountPath).Delete(context.TODO(), *key)
		if err != nil {
			return errors.Wrapf(err, "failed to delete from vault kv2 at '%s'", vaultConfig.MountPath)
		}
	} else {
		return fmt.Errorf("unsupported KV version: %s", vaultConfig.KVVersion)
	}
	return nil
}

func ReadSecretsFromVault(client *vault.Client, vaultConfig v1alpha1.VaultConfig, key *string) (map[string]interface{}, error) {
	var secretData map[string]interface{}

	if vaultConfig.KVVersion == "1" {
		data, err := client.KVv1(vaultConfig.MountPath).Get(context.TODO(), *key)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read from vault kv1 at '%s'", vaultConfig.MountPath)
		}
		if data != nil {
			secretData = data.Data
		}
	} else if vaultConfig.KVVersion == "2" {
		data, err := client.KVv2(vaultConfig.MountPath).Get(context.TODO(), *key)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read from vault kv2 at '%s'", vaultConfig.MountPath)
		}
		if data != nil && data.Data != nil {
			secretData = data.Data["data"].(map[string]interface{})
		}
	} else {
		return nil, fmt.Errorf("unsupported KV version: %s", vaultConfig.KVVersion)
	}

	return secretData, nil
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
