/*
Copyright 2022 The Crossplane Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cephuser

import (
	"context"
	"fmt"
	radosgw_admin "github.com/ceph/go-ceph/rgw/admin"
	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/controller"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/daanvinken/provider-radosgw/apis/ceph/v1alpha1"
	apisv1alpha1 "github.com/daanvinken/provider-radosgw/apis/v1alpha1"
	pc_v1alpha1 "github.com/daanvinken/provider-radosgw/apis/v1alpha1"
	"github.com/daanvinken/provider-radosgw/internal/clients/radosgw"
	"github.com/daanvinken/provider-radosgw/internal/clients/vault"
	vault_sdk "github.com/hashicorp/vault/api"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"strings"
)

const (
	errNotCephUser            = "managed resource is not a CephUser custom resource"
	errTrackPCUsage           = "cannot track ProviderConfig usage"
	errGetPC                  = "cannot get ProviderConfig"
	errGetCreds               = "cannot get Ceph admin credentials from Vault"
	errCreateAdminVaultClient = "failed to initialize Vault client to retrieve Ceph admin credentials"
	errNewClient              = "cannot create new radosgw client"
	errGetCephUser            = "Failed to retrieve cephuser"
	errCreateCephUser         = "Failed to create cephuser"
	errDeleteCephUser         = "Failed to delete cephuser"
	errVaultCleanup           = "Failed to remove credentials from vault_sdk"
	errFetchSecretAdmin       = "unable to extract secret data for radosgw admin"
	errVaultClientCreate      = "failed to create vault_sdk client for storing ceph credentials"
	errListBuckets            = "error listing user's buckets"
	errUserStillHasBuckets    = "ceph user still owns buckets"

	inUseFinalizer = "cephuser-in-use.ceph.radosgw.crossplane.io"
)

// Setup adds a controller that reconciles CephUser managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.CephUserGroupKind)

	cps := []managed.ConnectionPublisher{managed.NewAPISecretPublisher(mgr.GetClient(), mgr.GetScheme())}

	if os.Getenv("VAULT_TOKEN") != "" && os.Getenv("VAULT_ADDR") != "" {
		fmt.Println("Using local dev mode as 'VAULT_TOKEN' and 'VAULT_ADDR' are set.")
	}

	vaultAdminClient, err := vault.NewVaultClientForCephAdmins()
	if err != nil {
		panic(errors.Wrap(err, errCreateAdminVaultClient))
	}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.CephUserGroupVersionKind),
		managed.WithExternalConnecter(&connector{
			kube:               mgr.GetClient(),
			usage:              resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
			newRadosgwClientFn: radosgw.NewRadosgwClient,
			newVaultClientFn:   vault.NewVaultClientWithPanic,
			vaultAdminClient:   vaultAdminClient,
			log:                o.Logger.WithValues("controller", name)}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
		managed.WithConnectionPublishers(cps...))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.CephUser{}).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

// A connector is expected to produce an ExternalClient when its Connect method
// is called.
type connector struct {
	kube               client.Client
	usage              resource.Tracker
	newRadosgwClientFn func(host string, credentials radosgw.Credentials) *radosgw_admin.API
	newVaultClientFn   func(config pc_v1alpha1.VaultConfig) *vault_sdk.Client
	log                logging.Logger
	vaultAdminClient   *vault_sdk.Client
}

// Connect typically produces an ExternalClient by:
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig.
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1alpha1.CephUser)
	if !ok {
		return nil, errors.New(errNotCephUser)
	}

	// These fmt statements should be removed in the real implementation.
	fmt.Printf("Connecting: %+v\n", cr.Name)

	if err := c.usage.Track(ctx, mg); err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	pc := &apisv1alpha1.ProviderConfig{}
	if err := c.kube.Get(ctx, types.NamespacedName{Name: cr.GetProviderConfigReference().Name}, pc); err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	radosgwCredentials, err := GetAdminCredentials(c.vaultAdminClient, pc)
	if err != nil {
		return nil, errors.Wrap(err, errFetchSecretAdmin)
	}

	return &external{
		rgwClient:   c.newRadosgwClientFn(pc.Spec.HostName, radosgwCredentials),
		kubeClient:  c.kube,
		log:         c.log,
		vaultClient: c.newVaultClientFn(pc.Spec.CredentialsVault),
	}, err
}

// An ExternalClient observes, then either creates, updates, or deletes an
// external resource to ensure it reflects the managed resource's desired state.
type external struct {
	kubeClient  client.Client
	rgwClient   *radosgw_admin.API
	vaultClient *vault_sdk.Client
	log         logging.Logger
}

func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.CephUser)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotCephUser)
	}

	// These fmt statements should be removed in the real implementation.
	fmt.Printf("Observing: %+v\n", cr.Name)

	// Create a new context and cancel it when we have either found the user or cannot find it.
	ctxC, cancel := context.WithCancel(ctx)
	defer cancel()

	cephUserExists, err := radosgw.CephUserExists(ctxC, c.rgwClient, *cr.Spec.ForProvider.UID)
	if err != nil {
		c.log.Info(errors.Wrap(err, errGetCephUser).Error())
	}

	if cephUserExists {
		return managed.ExternalObservation{
			// Return false when the external resource does not exist. This lets
			// the managed resource reconciler know that it needs to call Create to
			// (re)create the resource, or that it has successfully been deleted.
			ResourceExists: true,

			// Return false when the external resource exists, but it not up to date
			// with the desired managed resource state. This lets the managed
			// resource reconciler know that it needs to call Update.
			ResourceUpToDate: true,

			// Return any details that may be required to connect to the external
			// resource. These will be stored as the connection secret.
			ConnectionDetails: managed.ConnectionDetails{},
		}, nil
	}

	// cephUser not found anywhere.
	return managed.ExternalObservation{
		// Return false when the external resource does not exist. This lets
		// the managed resource reconciler know that it needs to call Create to
		// (re)create the resource, or that it has successfully been deleted.
		// If the cephUser's Disabled flag has been set, no further action is needed.
		ResourceExists: false,
	}, nil
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.CephUser)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotCephUser)
	}

	// These fmt statements should be removed in the real implementation.
	fmt.Printf("Creating: %+v\n", cr.Name)

	user := radosgw.GenerateCephUserInput(cr)
	_, err := c.rgwClient.CreateUser(ctx, *user)
	if resource.Ignore(isAlreadyExists, err) != nil {
		c.log.Info("Failed to create cephUser on radosgw", "cephUser_uid", cr.Spec.ForProvider.UID, "error", err.Error())
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateCephUser)

	}

	quota := radosgw.GenerateCephUserQuotaInput(cr)

	err = c.rgwClient.SetUserQuota(ctx, *quota)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, "failed to set userquota during creation")
	}

	credentialsData := map[string]interface{}{
		"access_key": user.Keys[0].AccessKey,
		"secret_key": user.Keys[0].SecretKey,
	}

	pc := &apisv1alpha1.ProviderConfig{}
	if err := c.kubeClient.Get(ctx, types.NamespacedName{Name: cr.GetProviderConfigReference().Name}, pc); err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errGetPC)
	}

	secretPath, err := vault.BuildCephUserSecretPath(*pc, *cr.Spec.ForProvider.UID)

	if err != nil {
		c.log.Info(fmt.Sprintf("Failed to build secret path for storing CephUser credentials: '%v+'", err))
		return managed.ExternalCreation{}, err
	}

	err = vault.WriteSecretsToVault(c.vaultClient, pc.Spec.CredentialsVault, &secretPath, &credentialsData)
	if err != nil {
		//TODO remove user from radosgw again to fix state. Actually use defer with context.
		return managed.ExternalCreation{}, err
	}

	err = c.kubeClient.Update(ctx, cr)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	if controllerutil.AddFinalizer(cr, inUseFinalizer) {
		err := c.kubeClient.Update(ctx, cr)
		if err != nil {
			return managed.ExternalCreation{}, err
		}
	}

	cr.Status.SetConditions(xpv1.Available())

	if err := c.kubeClient.Status().Update(ctx, cr); err != nil {
		c.log.Info("Failed to update cephUser", "backend name", "cephUser_uid", cr.Spec.ForProvider.UID)
	}

	return managed.ExternalCreation{}, nil
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.CephUser)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotCephUser)
	}

	// These fmt statements should be removed in the real implementation.
	fmt.Printf("Updating: %+v\n", cr.Name)

	return managed.ExternalUpdate{
		// Optionally return any details that may be required to connect to the
		// external resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*v1alpha1.CephUser)
	if !ok {
		return errors.New(errNotCephUser)
	}

	// These fmt statements should be removed in the real implementation.
	fmt.Printf("Deleting: %+v\n", cr.Name)

	pc := &apisv1alpha1.ProviderConfig{}
	if err := c.kubeClient.Get(ctx, types.NamespacedName{Name: cr.GetProviderConfigReference().Name}, pc); err != nil {
		return errors.Wrap(err, errGetPC)
	}

	secretPath, err := vault.BuildCephUserSecretPath(*pc, *cr.Spec.ForProvider.UID)

	hasBuckets, err := cephUserHasBuckets(c.rgwClient, cr)
	if err != nil {
		c.log.Info("Failed to verify if user still has buckets during deletion", "cephUser_uid", cr.Spec.ForProvider.UID, "error", err.Error())
		return errors.Wrap(err, errDeleteCephUser)
	}

	if !hasBuckets {
		if controllerutil.RemoveFinalizer(cr, inUseFinalizer) {
			err := c.kubeClient.Update(ctx, cr)
			if err != nil {
				c.log.Info("Failed to remove in-use finalizer on cephuser", "cephUser_uid", cr.Spec.ForProvider.UID, "error", err.Error())
				return errors.Wrap(err, errDeleteCephUser)
			}
		}
	} else {
		return fmt.Errorf(errUserStillHasBuckets)
	}

	user := radosgw.GenerateCephUserInput(cr)
	err = c.rgwClient.RemoveUser(ctx, *user)
	if err != nil {
		c.log.Info("Failed to remove cephUser on radosgw", "cephUser_uid", cr.Spec.ForProvider.UID, "error", err.Error())
		return errors.Wrap(err, errDeleteCephUser)

	}

	err = vault.RemoveSecretFromVault(c.vaultClient, pc.Spec.CredentialsVault, &secretPath)
	if err != nil {
		c.log.Info("Failed to remove credentials from Vault", "cephUser_uid", cr.Spec.ForProvider.UID, "error", err.Error())
		return errors.Wrap(err, errVaultCleanup)
	}
	return nil
}

// isAlreadyExists helper function to test for an already existing user
func isAlreadyExists(err error) bool {
	// TODO can we check for direct client error types
	if err == nil {
		return false
	}
	if strings.HasPrefix(err.Error(), "KeyExists") {
		return true
	}
	return false
}

func cephUserHasBuckets(radosgwClient *radosgw_admin.API, cephUser *v1alpha1.CephUser) (bool, error) {
	buckets, err := radosgwClient.ListUsersBuckets(context.TODO(), *cephUser.Spec.ForProvider.UID)
	if err != nil {
		return false, errors.Wrap(err, errListBuckets)
	}

	// Check if the user has any buckets
	return len(buckets) > 0, nil
}

func GetAdminCredentials(vaultAdminClient *vault_sdk.Client, pc *apisv1alpha1.ProviderConfig) (radosgw.Credentials, error) {
	kvSecret, err := vaultAdminClient.KVv1("k8s-cl03").Get(context.Background(), "crossplane/ceph/admin-credentials/"+pc.Name)
	if err != nil {
		return radosgw.Credentials{}, err
	}

	secretKey, ok := kvSecret.Data["secret_key"].(string)
	if !ok {
		return radosgw.Credentials{}, errors.New("failed to fetch 'secret_key'")
	}

	accessKey, ok := kvSecret.Data["access_key"].(string)
	if !ok {
		return radosgw.Credentials{}, errors.New("failed to fetch 'access_key'")
	}

	return radosgw.Credentials{
		AccessKey: accessKey,
		SecretKey: secretKey,
	}, err

}
