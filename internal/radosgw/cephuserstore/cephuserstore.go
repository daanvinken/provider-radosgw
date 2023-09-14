package cephuserstore

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/daanvinken/provider-radosgw/apis/ceph/v1alpha1"
	apisv1alpha1 "github.com/daanvinken/provider-radosgw/apis/v1alpha1"
	internals3 "github.com/daanvinken/provider-radosgw/internal/s3"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sync"
	"time"
)

const (
	startLengthCephUserStore = 1000
)

type UserRecord struct {
	s3Client  s3.Client
	cephUser  v1alpha1.CephUser
	createdAt time.Time
}

type CephUserStore struct {
	CephUserRecords map[string]*UserRecord
	l               sync.RWMutex
}

func New() (c *CephUserStore) {
	c = &CephUserStore{CephUserRecords: make(map[string]*UserRecord, startLengthCephUserStore)}
	return
}

func (c *CephUserStore) GetByUID(cephUserUID string) (*s3.Client, error) {
	c.l.RLock()
	defer c.l.RUnlock()
	if _, ok := c.CephUserRecords[cephUserUID]; ok {
		return &c.CephUserRecords[cephUserUID].s3Client, nil
	}
	return &s3.Client{}, fmt.Errorf("CephUser with UID '%s' cannot be found", cephUserUID)
}

func (c *CephUserStore) create(cephUser v1alpha1.CephUser, creds *corev1.Secret, pc *apisv1alpha1.ProviderConfig) error {
	c.l.Lock()
	defer c.l.Unlock()
	s3Client, err := internals3.NewClient(context.Background(), *creds, &pc.Spec)
	if err != nil {
		return errors.Wrapf(err, "failed to create s3 client (cephUserUID = '%s')", cephUser.Spec.ForProvider.UID)
	}
	c.CephUserRecords[*cephUser.Spec.ForProvider.UID] =
		&UserRecord{
			s3Client:  *s3Client,
			cephUser:  cephUser,
			createdAt: time.Now(),
		}

	return nil
}

func (c *CephUserStore) Delete(cephUser v1alpha1.CephUser) error {
	c.l.Lock()
	defer c.l.Unlock()

	// Check if the user exists in the map
	_, exists := c.CephUserRecords[*cephUser.Spec.ForProvider.UID]
	if !exists {
		return nil
	}

	delete(c.CephUserRecords, *cephUser.Spec.ForProvider.UID)

	return nil
}

func (c *CephUserStore) Init(ctx context.Context, kubeClient client.Client) error {
	cephUserList := &v1alpha1.CephUserList{}
	var listOptions []client.ListOption

	if err := kubeClient.List(ctx, cephUserList, listOptions...); err != nil {
		// Handle the error here and wrap it
		return errors.Wrap(err, "Error listing CephUsers during initialization of CephUser clients")
	}

	var wg sync.WaitGroup
	//errorsLeft := 0
	//errChan := make(chan error, len(activeBackends))

	//TODO error handling with a channel
	for _, cephUser := range cephUserList.Items {
		wg.Add(1)
		//errorsLeft++

		cephUser := cephUser
		go func(user v1alpha1.CephUser) {
			defer wg.Done()
			pcRef := user.Spec.ProviderConfigReference.Name
			pc := apisv1alpha1.ProviderConfig{}
			if err := kubeClient.Get(ctx, client.ObjectKey{Name: pcRef}, &pc); err != nil {
				fmt.Printf("Error fetching ProviderConfig during initialization of CephUser clients: %v\n", err)
				//errChan <- err
			}
			credsSecret := corev1.Secret{}
			if err := kubeClient.Get(ctx, client.ObjectKey{Name: *user.Spec.ForProvider.CredentialsSecretName, Namespace: "crossplane"}, &credsSecret); err != nil {
				fmt.Printf("Error fetching ProviderConfig during initialization of CephUser clients: %v\n", err)
				//errChan <- err
			}
			err := c.create(cephUser, &credsSecret, &pc)
			if err != nil {
				fmt.Printf("Error creating CephUser: %v\n", err)
				//errChan <- err
			}
			//errorsLeft--
		}(cephUser)
	}

	wg.Wait()
	return nil
}
