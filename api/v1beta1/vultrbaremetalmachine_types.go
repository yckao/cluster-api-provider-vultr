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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/errors" //nolint:staticcheck
)

const (
	// BareMetalMachineFinalizer allows ReconcileVultrBareMetalMachine to clean up Vultr resources associated with
	// VultrBareMetalMachine before removing it from the apiserver.
	BareMetalMachineFinalizer = "vultrbaremetalmachine.infrastructure.cluster.x-k8s.io"
)

// VultrBareMetalMachineSpec defines the desired state of VultrBareMetalMachine
type VultrBareMetalMachineSpec struct {
	// ProviderID is the unique identifier as specified by the cloud provider.
	// +optional
	ProviderID *string `json:"providerID,omitempty"`

	// PlanID is the id of Vultr bare metal plan.
	PlanID string `json:"planID,omitempty"`

	// The Vultr Region (DCID) the cluster lives on.
	// +kubebuilder:validation:Required
	Region string `json:"region"`

	// OsID is the Vultr OS ID to use when deploying the bare metal server.
	// +optional
	OsID int `json:"os_id,omitempty"`

	// ImageID is the Vultr image ID to use when deploying the bare metal server.
	// +optional
	ImageID string `json:"image_id,omitempty"`

	// AppID is the Vultr marketplace application ID to deploy on the bare metal server.
	// +optional
	AppID int `json:"app_id,omitempty"`

	// StartupScriptID is the Vultr startup script ID to execute on the bare metal server.
	// +optional
	StartupScriptID string `json:"script_id,omitempty"`

	// IPXEChainURL is the URL to chain-load iPXE from when booting the bare metal server.
	// +optional
	IPXEChainURL string `json:"ipxe_chain_url,omitempty"`

	// PersistentPxe enables persistent PXE booting for the bare metal server.
	// When true, the server will always boot from PXE.
	// +optional
	PersistentPxe bool `json:"persistent_pxe,omitempty"`

	// sshKey is the name of the ssh key to attach to the bare metal server.
	// +optional
	SSHKey []string `json:"sshKey,omitempty"`

	// VPCID is the id of the VPC to be attached after creation.
	// Note: Bare metal VPC attachment is performed after server creation.
	// +optional
	VPCID string `json:"vpc_id,omitempty"`

	// ReservedIPv4 is the reserved IPv4 address to assign to the bare metal server.
	// +optional
	ReservedIPv4 string `json:"reserved_ipv4,omitempty"`

	// MdiskMode is the RAID configuration for bare metal servers with multiple disks.
	// +optional
	MdiskMode string `json:"mdisk_mode,omitempty"`
}

// VultrBareMetalMachineStatus defines the observed state of VultrBareMetalMachine
type VultrBareMetalMachineStatus struct {

	// Ready represents the infrastructure is ready to be used or not.
	// +optional
	Ready bool `json:"ready"`

	// CPU represents the number of CPUs of the bare metal server.
	// +optional
	CPU int `json:"cpu,omitempty"`

	// RAM represents the amount of memory of the bare metal server.
	// +optional
	RAM string `json:"ram,omitempty"`

	// Storage represents the disk size of the bare metal server.
	// +optional
	Storage string `json:"storage,omitempty"`

	// Addresses contains the Vultr bare metal server associated addresses.
	Addresses []corev1.NodeAddress `json:"addresses,omitempty"`

	// SubscriptionStatus represents the status of subscription.
	// +optional
	SubscriptionStatus *SubscriptionStatus `json:"subscriptionStatus,omitempty"`

	// FailureReason will be set in the event that there is a terminal problem
	// reconciling the Machine and will contain a succinct value suitable
	// for machine interpretation.
	//
	// This field should not be set for transitive errors that a controller
	// faces that are expected to be fixed automatically over
	// time (like service outages), but instead indicate that something is
	// fundamentally wrong with the Machine's spec or the configuration of
	// the controller, and that manual intervention is required. Examples
	// of terminal errors would be invalid combinations of settings in the
	// spec, values that are unsupported by the controller, or the
	// responsible controller itself being critically misconfigured.
	//
	// Any transient errors that occur during the reconciliation of Machines
	// can be added as events to the Machine object and/or logged in the
	// controller's output.
	// +optional
	FailureReason *errors.MachineStatusError `json:"failureReason,omitempty"`

	// FailureMessage will be set in the event that there is a terminal problem
	// reconciling the Machine and will contain a more verbose string suitable
	// for logging and human consumption.
	//
	// This field should not be set for transitive errors that a controller
	// faces that are expected to be fixed automatically over
	// time (like service outages), but instead indicate that something is
	// fundamentally wrong with the Machine's spec or the configuration of
	// the controller, and that manual intervention is required. Examples
	// of terminal errors would be invalid combinations of settings in the
	// spec, values that are unsupported by the controller, or the
	// responsible controller itself being critically misconfigured.
	//
	// Any transient errors that occur during the reconciliation of Machines
	// can be added as events to the Machine object and/or logged in the
	// controller's output.
	// +optional
	FailureMessage *string `json:"failureMessage,omitempty"`

	// Conditions defines current service state of the VultrBareMetalMachine.
	// +optional
	Conditions clusterv1.Conditions `json:"conditions,omitempty"`
}

func (r *VultrBareMetalMachine) GetConditions() clusterv1.Conditions {
	return r.Status.Conditions
}

func (r *VultrBareMetalMachine) SetConditions(conditions clusterv1.Conditions) {
	r.Status.Conditions = conditions
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:path=vultrbaremetalmachines,scope=Namespaced,categories=cluster-api
//+kubebuilder:printcolumn:name="Cluster",type="string",JSONPath=".metadata.labels.cluster\\.x-k8s\\.io/cluster-name",description="Cluster to which this VultrBareMetalMachine belongs"
//+kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.subscriptionStatus",description="Vultr bare metal server state"
//+kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.ready",description="Machine ready status"
//+kubebuilder:printcolumn:name="InstanceID",type="string",JSONPath=".spec.providerID",description="Vultr bare metal server ID"
//+kubebuilder:printcolumn:name="Machine",type="string",JSONPath=".metadata.ownerReferences[?(@.kind==\"Machine\")].name",description="Machine object which owns with this VultrBareMetalMachine"

// VultrBareMetalMachine is the Schema for the vultrbaremetalmachines API
type VultrBareMetalMachine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VultrBareMetalMachineSpec   `json:"spec,omitempty"`
	Status VultrBareMetalMachineStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// VultrBareMetalMachineList contains a list of VultrBareMetalMachine
type VultrBareMetalMachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VultrBareMetalMachine `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VultrBareMetalMachine{}, &VultrBareMetalMachineList{})
}
