/*
Copyright 2024.

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

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

const (
	// VPCFinalizer allows ReconcileVultrVPC to clean up Vultr resources before removing it from the apiserver.
	VPCFinalizer = "vultrvpc.infrastructure.cluster.x-k8s.io"
)

// VultrVPCSpec defines the desired state of VultrVPC.
type VultrVPCSpec struct {
	// Region is the Vultr region for the VPC.
	Region string `json:"region"`

	// Description is an optional description for the VPC.
	// +optional
	Description string `json:"description,omitempty"`

	// V4Subnet is the IPv4 network address (e.g. 10.99.0.0).
	// +optional
	V4Subnet string `json:"v4Subnet,omitempty"`

	// V4SubnetMask is the number of bits for the netmask (e.g. 24).
	// +optional
	V4SubnetMask int `json:"v4SubnetMask,omitempty"`
}

// VultrVPCStatus defines the observed state of VultrVPC.
type VultrVPCStatus struct {
	// Ready denotes that the VPC is ready.
	// +optional
	Ready bool `json:"ready"`

	// ID is the Vultr VPC ID.
	// +optional
	ID string `json:"id,omitempty"`

	// Conditions defines current service state of the VultrVPC.
	// +optional
	Conditions clusterv1.Conditions `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:path=vultrvpcs,scope=Namespaced,categories=cluster-api
//+kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.ready",description="VPC is ready"
//+kubebuilder:printcolumn:name="ID",type="string",JSONPath=".status.id",description="Vultr VPC ID"

// VultrVPC is the Schema for the vultrvpcs API.
type VultrVPC struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VultrVPCSpec   `json:"spec,omitempty"`
	Status VultrVPCStatus `json:"status,omitempty"`
}

func (r *VultrVPC) GetConditions() clusterv1.Conditions {
	return r.Status.Conditions
}

func (r *VultrVPC) SetConditions(conditions clusterv1.Conditions) {
	r.Status.Conditions = conditions
}

//+kubebuilder:object:root=true

// VultrVPCList contains a list of VultrVPC.
type VultrVPCList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VultrVPC `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VultrVPC{}, &VultrVPCList{})
}
