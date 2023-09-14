package radosgw

import (
	"context"
	radosgw_admin "github.com/ceph/go-ceph/rgw/admin"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/daanvinken/provider-radosgw/apis/ceph/v1alpha1"
	"github.com/daanvinken/provider-radosgw/internal/credentials"
	"strings"
)

func GenerateCephUserInput(cephUser *v1alpha1.CephUser) *radosgw_admin.User {
	//quotaEnable := true
	//userQuotaSpec := radosgw_admin.QuotaSpec{
	//	QuotaType:  "user",
	//	UID:        *cephUser.Spec.ForProvider.UID,
	//	MaxSizeKb:  cephUser.Spec.ForProvider.UserQuotaMaxSizeKB,
	//	MaxObjects: cephUser.Spec.ForProvider.UserQuotaMaxObjects,
	//	Enabled:    &quotaEnable,
	//}

	createCephUserInput := &radosgw_admin.User{
		ID: *cephUser.Spec.ForProvider.UID,
		//MaxBuckets:  cephUser.Spec.ForProvider.UserQuotaMaxBuckets,
		//UserQuota:   userQuotaSpec,
		DisplayName: *cephUser.Spec.ForProvider.DisplayedName,
		Keys: []radosgw_admin.UserKeySpec{
			{
				AccessKey: credentials.GenerateRandomSecret(15),
				SecretKey: credentials.GenerateRandomSecret(26),
			},
		},
		// TODO fill all parameters and generate secrets
	}

	return createCephUserInput
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
