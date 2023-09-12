//go:build !ignore_autogenerated
// +build !ignore_autogenerated

/*
Copyright 2020 The Crossplane Authors.

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

// Code generated by controller-gen. DO NOT EDIT.

package v1alpha1

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CephUser) DeepCopyInto(out *CephUser) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CephUser.
func (in *CephUser) DeepCopy() *CephUser {
	if in == nil {
		return nil
	}
	out := new(CephUser)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *CephUser) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CephUserList) DeepCopyInto(out *CephUserList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]CephUser, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CephUserList.
func (in *CephUserList) DeepCopy() *CephUserList {
	if in == nil {
		return nil
	}
	out := new(CephUserList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *CephUserList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CephUserObservation) DeepCopyInto(out *CephUserObservation) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CephUserObservation.
func (in *CephUserObservation) DeepCopy() *CephUserObservation {
	if in == nil {
		return nil
	}
	out := new(CephUserObservation)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CephUserParameters) DeepCopyInto(out *CephUserParameters) {
	*out = *in
	if in.UID != nil {
		in, out := &in.UID, &out.UID
		*out = new(string)
		**out = **in
	}
	if in.DisplayedName != nil {
		in, out := &in.DisplayedName, &out.DisplayedName
		*out = new(string)
		**out = **in
	}
	if in.UserQuotaMaxBuckets != nil {
		in, out := &in.UserQuotaMaxBuckets, &out.UserQuotaMaxBuckets
		*out = new(int)
		**out = **in
	}
	if in.UserQuotaMaxSizeKB != nil {
		in, out := &in.UserQuotaMaxSizeKB, &out.UserQuotaMaxSizeKB
		*out = new(int)
		**out = **in
	}
	if in.UserQuotaMaxObjects != nil {
		in, out := &in.UserQuotaMaxObjects, &out.UserQuotaMaxObjects
		*out = new(int64)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CephUserParameters.
func (in *CephUserParameters) DeepCopy() *CephUserParameters {
	if in == nil {
		return nil
	}
	out := new(CephUserParameters)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CephUserSpec) DeepCopyInto(out *CephUserSpec) {
	*out = *in
	in.ResourceSpec.DeepCopyInto(&out.ResourceSpec)
	in.ForProvider.DeepCopyInto(&out.ForProvider)
	out.CredentialsSecretRef = in.CredentialsSecretRef
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CephUserSpec.
func (in *CephUserSpec) DeepCopy() *CephUserSpec {
	if in == nil {
		return nil
	}
	out := new(CephUserSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CephUserStatus) DeepCopyInto(out *CephUserStatus) {
	*out = *in
	in.ResourceStatus.DeepCopyInto(&out.ResourceStatus)
	out.AtProvider = in.AtProvider
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CephUserStatus.
func (in *CephUserStatus) DeepCopy() *CephUserStatus {
	if in == nil {
		return nil
	}
	out := new(CephUserStatus)
	in.DeepCopyInto(out)
	return out
}
