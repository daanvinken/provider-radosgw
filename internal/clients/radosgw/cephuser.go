package radosgw

import (
	"context"
	radosgw_admin "github.com/ceph/go-ceph/rgw/admin"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/daanvinken/provider-radosgw/apis/ceph/v1alpha1"
	"github.com/daanvinken/provider-radosgw/internal/utils"
	"net/http"
	"strings"
)

type Credentials struct {
	AccessKey string
	SecretKey string
}

func NewRadosgwClient(host string, creds Credentials) *radosgw_admin.API {
	httpClient := &http.Client{}
	rgwClient, err := radosgw_admin.New(host, creds.AccessKey, creds.SecretKey, httpClient)
	if err != nil {
		panic(err)
	}
	return rgwClient
}

func GenerateCephUserInput(cephUser *v1alpha1.CephUser) *radosgw_admin.User {

	createCephUserInput := &radosgw_admin.User{
		ID:          *cephUser.Spec.ForProvider.UID,
		MaxBuckets:  cephUser.Spec.ForProvider.UserQuotaMaxBuckets,
		DisplayName: *cephUser.Spec.ForProvider.DisplayedName,
		Keys: []radosgw_admin.UserKeySpec{
			{
				AccessKey: utils.GenerateRandomSecret(15),
				SecretKey: utils.GenerateRandomSecret(26),
			},
		},
	}

	return createCephUserInput
}

func GenerateCephUserQuotaInput(cephUser *v1alpha1.CephUser) *radosgw_admin.QuotaSpec {
	quotaEnable := true
	userQuotaSpec := &radosgw_admin.QuotaSpec{
		QuotaType:  "user",
		UID:        *cephUser.Spec.ForProvider.UID,
		MaxSizeKb:  cephUser.Spec.ForProvider.UserQuotaMaxSizeKB,
		MaxObjects: cephUser.Spec.ForProvider.UserQuotaMaxObjects,
		Enabled:    &quotaEnable,
	}
	return userQuotaSpec
}

func CephUserExists(ctx context.Context, radosgwclient *radosgw_admin.API, UID string) (bool, error) {
	_, err := radosgwclient.GetUser(ctx, radosgw_admin.User{ID: UID})
	if err != nil {
		return false, resource.Ignore(isNotFound, err)
	}
	return true, nil
}

// isNotFound helper function to test for NotFound error
func isNotFound(err error) bool {
	if strings.HasPrefix(err.Error(), "NoSuchUser") {
		return true
	}
	return false
}
