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
	"github.com/crossplane/crossplane-runtime/pkg/connection"
	"github.com/crossplane/crossplane-runtime/pkg/controller"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/daanvinken/provider-radosgw/apis/ceph/v1alpha1"
	apisv1alpha1 "github.com/daanvinken/provider-radosgw/apis/v1alpha1"
	"github.com/daanvinken/provider-radosgw/internal/credentials"
	"github.com/daanvinken/provider-radosgw/internal/features"
	"github.com/daanvinken/provider-radosgw/internal/radosgw"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"net/http"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"strings"
)

const (
	errNotCephUser    = "managed resource is not a CephUser custom resource"
	errTrackPCUsage   = "cannot track ProviderConfig usage"
	errGetPC          = "cannot get ProviderConfig"
	errGetCreds       = "cannot get credentials"
	errNewClient      = "cannot create new radosgw client"
	errGetCephUser    = "Failed to retrieve cephuser"
	errCreateCephUser = "Failed to create cephuser"
	errDeleteCephUser = "Failed to delete cephuser"

	inUseFinalizer = "cephuser-in-use.ceph.radosgw.crossplane.io"
)

// A NoOpService does nothing.
type NoOpService struct{}

var (
	newNoOpService = func(_ []byte) (interface{}, error) { return &NoOpService{}, nil }
)

// Setup adds a controller that reconciles CephUser managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.CephUserGroupKind)

	cps := []managed.ConnectionPublisher{managed.NewAPISecretPublisher(mgr.GetClient(), mgr.GetScheme())}
	if o.Features.Enabled(features.EnableAlphaExternalSecretStores) {
		cps = append(cps, connection.NewDetailsManager(mgr.GetClient(), apisv1alpha1.StoreConfigGroupVersionKind))
	}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.CephUserGroupVersionKind),
		managed.WithExternalConnecter(&connector{
			kube:         mgr.GetClient(),
			usage:        resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
			newServiceFn: newNoOpService}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
		managed.WithConnectionPublishers(cps...))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		Owns(&corev1.Secret{}).
		WithOptions(o.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.CephUser{}).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

// A connector is expected to produce an ExternalClient when its Connect method
// is called.
type connector struct {
	kube             client.Client
	usage            resource.Tracker
	newRadosgwClient func(AccessKey string, SecretKey string, HttpClient http.Client, Endpoint string) (radosgw_admin.API, error)
	newServiceFn     func(creds []byte) (interface{}, error)
	log              logging.Logger
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

	if err := c.usage.Track(ctx, mg); err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	pc := &apisv1alpha1.ProviderConfig{}
	if err := c.kube.Get(ctx, types.NamespacedName{Name: cr.GetProviderConfigReference().Name}, pc); err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	cd := pc.Spec.Credentials
	b_AccessKey, b_SecretKey, err := credentials.CephCredentialExtractor(ctx, cd, c.kube)
	if err != nil {
		return nil, errors.Wrap(err, errGetCreds)
	}

	httpClient := &http.Client{}
	rgwClient, err := radosgw_admin.New(pc.Spec.HostName, string(b_AccessKey), string(b_SecretKey), httpClient)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}
	return &external{
		rgw_client: rgwClient,
		kubeClient: c.kube,
	}, err
}

// An ExternalClient observes, then either creates, updates, or deletes an
// external resource to ensure it reflects the managed resource's desired state.
type external struct {
	kubeClient client.Client
	rgw_client *radosgw_admin.API
	log        logging.Logger
}

func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.CephUser)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotCephUser)
	}

	// TODO verify the configured backend? Or should we do a seperate healthcheck ?

	// Create a new context and cancel it when we have either found the user or cannot find it.
	ctxC, cancel := context.WithCancel(ctx)
	defer cancel()

	cephUserExists, err := radosgw.CephUserExists(ctxC, c.rgw_client, *cr.Spec.ForProvider.UID)
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

	//fmt.Printf("Creating: %+v", cr)

	user := radosgw.GenerateCephUserInput(cr)
	_, err := c.rgw_client.CreateUser(ctx, *user)
	if resource.Ignore(isAlreadyExists, err) != nil {
		c.log.Info("Failed to create cephUser on radosgw", "cephUser_uid", cr.Spec.ForProvider.UID, "error", err.Error())
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateCephUser)

	}

	// TODO fetch the current crossplane namespace
	secretObject := credentials.CreateKubernetesSecretCephUser(
		user.Keys[0].AccessKey,
		user.Keys[0].SecretKey,
		"crossplane",
		*cr.Spec.ForProvider.UID,
	)

	err = c.kubeClient.Create(ctx, &secretObject)
	if err != nil {
		c.log.Info("Failed to store cephUser credentials to K8s api", "cephUser_uid", cr.Spec.ForProvider.UID, "error", err.Error())
		return managed.ExternalCreation{}, err
	}

	// TODO secret should be a child object ALSO THIS DOS NOT WORK ?
	cr.Spec.CredentialsSecretRef = corev1.LocalObjectReference{
		Name: secretObject.Name,
	}

	cr.Spec.ForProvider.CredentialsSecretName = &secretObject.Name

	//controllerutil.SetControllerReference(cr, secretObject, r.scheme)

	// Set the owner reference to make the Secret a child of the CephUser CR

	err = c.kubeClient.Update(ctx, cr)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	cr.Status.SetConditions(xpv1.Available())

	if err := c.kubeClient.Status().Update(ctx, cr); err != nil {
		c.log.Info("Failed to update cephUser", "backend name", "cephUser_uid", cr.Spec.ForProvider.UID)
	}

	// TODO should we add the finalizer here already? Maybe only on adding at first bucket
	if controllerutil.AddFinalizer(cr, inUseFinalizer) {
		err := c.kubeClient.Update(ctx, cr)
		if err != nil {
			return managed.ExternalCreation{}, err
		}
	}

	//return managed.ExternalCreation{
	//	// Optionally return any details that may be required to connect to the
	//	// external resource. These will be stored as the connection secret.
	//	ConnectionDetails: managed.ConnectionDetails{},
	//}, nil
	return managed.ExternalCreation{}, nil
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	_, ok := mg.(*v1alpha1.CephUser)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotCephUser)
	}

	fmt.Println("Updating but not implemented.")

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

	// TODO write helper function to check for existing buckets using AWS sdk
	// However maybe we can do this by just properly setting buckets as child objects

	// Also need to verify how radosgw hanles deletion while buckets exist under a user
	if controllerutil.RemoveFinalizer(cr, inUseFinalizer) {
		err := c.kubeClient.Update(ctx, cr)
		if err != nil {
			c.log.Info("Failed to remove in-use finalizer on cephuser", "cephUser_uid", cr.Spec.ForProvider.UID, "error", err.Error())
			return errors.Wrap(err, errDeleteCephUser)
		}
	}

	user := radosgw.GenerateCephUserInput(cr)
	err := c.rgw_client.RemoveUser(ctx, *user)
	if err != nil {
		c.log.Info("Failed to remove cephUser on radosgw", "cephUser_uid", cr.Spec.ForProvider.UID, "error", err.Error())
		return errors.Wrap(err, errDeleteCephUser)

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
