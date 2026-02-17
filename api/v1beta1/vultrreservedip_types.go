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
	// ReservedIPFinalizer allows ReconcileVultrReservedIP to clean up Vultr resources before removing it from the apiserver.
	ReservedIPFinalizer = "vultrreservedip.infrastructure.cluster.x-k8s.io"
)

// VultrReservedIPSpec defines the desired state of VultrReservedIP.
type VultrReservedIPSpec struct {
	// Region is the Vultr region for the reserved IP.
	Region string `json:"region"`

	// IPType is the IP type: "v4" or "v6".
	IPType string `json:"ipType"`

	// Label is an optional label for the reserved IP.
	// +optional
	Label string `json:"label,omitempty"`
}

// VultrReservedIPStatus defines the observed state of VultrReservedIP.
type VultrReservedIPStatus struct {
	// Ready denotes that the reserved IP is ready.
	// +optional
	Ready bool `json:"ready"`

	// ID is the Vultr reserved IP ID.
	// +optional
	ID string `json:"id,omitempty"`

	// Subnet is the assigned IP address.
	// +optional
	Subnet string `json:"subnet,omitempty"`

	// SubnetSize is the number of bits for the netmask.
	// +optional
	SubnetSize int `json:"subnetSize,omitempty"`

	// InstanceID is the ID of the attached instance, if any.
	// +optional
	InstanceID string `json:"instanceID,omitempty"`

	// Conditions defines current service state of the VultrReservedIP.
	// +optional
	Conditions clusterv1.Conditions `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:path=vultrreservedips,scope=Namespaced,categories=cluster-api
//+kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.ready",description="ReservedIP is ready"
//+kubebuilder:printcolumn:name="ID",type="string",JSONPath=".status.id",description="Vultr ReservedIP ID"
//+kubebuilder:printcolumn:name="Subnet",type="string",JSONPath=".status.subnet",description="Assigned IP address"

// VultrReservedIP is the Schema for the vultrreservedips API.
type VultrReservedIP struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VultrReservedIPSpec   `json:"spec,omitempty"`
	Status VultrReservedIPStatus `json:"status,omitempty"`
}

func (r *VultrReservedIP) GetConditions() clusterv1.Conditions {
	return r.Status.Conditions
}

func (r *VultrReservedIP) SetConditions(conditions clusterv1.Conditions) {
	r.Status.Conditions = conditions
}

//+kubebuilder:object:root=true

// VultrReservedIPList contains a list of VultrReservedIP.
type VultrReservedIPList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VultrReservedIP `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VultrReservedIP{}, &VultrReservedIPList{})
}
