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

package v1alpha1

import (
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
)

// CephUserParameters are the configurable fields of a CephUser.
type CephUserParameters struct {
	// The uid of the user (human readable string)
	UID *string `json:"uid"`

	// The displayed name
	DisplayedName *string `json:"displayedName"`

	// The max number of objects allowed for this user
	UserQuotaMaxBuckets *int `json:"userQuotaMaxBuckets"`

	// The maximum storage size (total) in MB
	UserQuotaMaxSizeKB *int `json:"userQuotaMaxSizeKB"`

	// The number of objects for this user
	UserQuotaMaxObjects *int64 `json:"userQuotaMaxObjects"`

	// Config for storing the created user its credentials in vault
	VaultCredentialsStore *VaultConfig `json:"vaultCredentialsStore"`
}

type VaultConfig struct {
	// The version of the Vault KV store to use ("1" or "2")
	KVVersion string `json:"kvVersion"`

	// The address of the Vault server (e.g., "https://vault.example.com:8200")
	Address string `json:"address"`

	// The vault human readable name
	Name string `json:"Name"`

	// The name of the Kubernetes service account authorized to access Vault
	ServiceAccountName string `json:"serviceAccountName"`

	// The mount path in Vault where the secrets engine is
	MountPath string `json:"mountPath"`

	// The secret path in Vault where the credentials are stored
	SecretPath string `json:"secretPath"`
}

// CephUserObservation are the observable fields of a CephUser.
type CephUserObservation struct {
	ObservableField string `json:"observableField,omitempty"`
}

// A CephUserSpec defines the desired state of a CephUser.
type CephUserSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       CephUserParameters `json:"forProvider"`
}

// A CephUserStatus represents the observed state of a CephUser.
type CephUserStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          CephUserObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A CephUser is an example API type.
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="CLUSTERNAME",type="string",JSONPath=".spec.providerConfigRef.name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,radosgw}
type CephUser struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CephUserSpec   `json:"spec"`
	Status CephUserStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CephUserList contains a list of CephUser
type CephUserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CephUser `json:"items"`
}

// CephUser type metadata.
var (
	CephUserKind             = reflect.TypeOf(CephUser{}).Name()
	CephUserGroupKind        = schema.GroupKind{Group: Group, Kind: CephUserKind}.String()
	CephUserKindAPIVersion   = CephUserKind + "." + SchemeGroupVersion.String()
	CephUserGroupVersionKind = SchemeGroupVersion.WithKind(CephUserKind)
)

func init() {
	SchemeBuilder.Register(&CephUser{}, &CephUserList{})
}
