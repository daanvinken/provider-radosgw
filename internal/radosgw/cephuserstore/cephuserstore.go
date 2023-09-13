package cephuserstore

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/daanvinken/provider-radosgw/apis/ceph/v1alpha1"
	apisv1alpha1 "github.com/daanvinken/provider-radosgw/apis/v1alpha1"
	internals3 "github.com/daanvinken/provider-radosgw/internal/s3"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
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

func (c *CephUserStore) Get(cephUser v1alpha1.CephUser) s3.Client {
	c.l.RLock()
	defer c.l.RUnlock()
	if _, ok := c.CephUserRecords[*cephUser.Spec.ForProvider.UID]; ok {
		return c.CephUserRecords[*cephUser.Spec.ForProvider.UID].s3Client
	}
	return s3.Client{}
}

func (c *CephUserStore) Create(cephUser v1alpha1.CephUser, pc *apisv1alpha1.ProviderConfig) error {
	c.l.RLock()
	s3Client, err := internals3.NewClient(context.Background(), corev1.Secret{}, &pc.Spec)
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

	// TODO can we close the client?
	//if err := userRecord.s3Client.Close(); err != nil {
	//	return errors.Wrapf(err, "failed to close S3 client for cephUserUID '%s'", *cephUser.Spec.ForProvider.UID)
	//}

	delete(c.CephUserRecords, *cephUser.Spec.ForProvider.UID)

	return nil
}
