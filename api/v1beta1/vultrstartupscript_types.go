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
	// StartupScriptFinalizer allows ReconcileVultrStartupScript to clean up Vultr resources before removing it from the apiserver.
	StartupScriptFinalizer = "vultrstartupscript.infrastructure.cluster.x-k8s.io"
)

// VultrStartupScriptSpec defines the desired state of VultrStartupScript.
type VultrStartupScriptSpec struct {
	// Name is the name of the startup script.
	Name string `json:"name"`

	// Script is the base64-encoded startup script content.
	Script string `json:"script"`

	// Type is the startup script type: "boot" or "pxe". Defaults to "boot".
	// +optional
	// +kubebuilder:default=boot
	Type string `json:"type,omitempty"`
}

// VultrStartupScriptStatus defines the observed state of VultrStartupScript.
type VultrStartupScriptStatus struct {
	// Ready denotes that the startup script is ready.
	// +optional
	Ready bool `json:"ready"`

	// ID is the Vultr startup script ID.
	// +optional
	ID string `json:"id,omitempty"`

	// Conditions defines current service state of the VultrStartupScript.
	// +optional
	Conditions clusterv1.Conditions `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:path=vultrstartupscripts,scope=Namespaced,categories=cluster-api
//+kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.ready",description="StartupScript is ready"
//+kubebuilder:printcolumn:name="ID",type="string",JSONPath=".status.id",description="Vultr StartupScript ID"

// VultrStartupScript is the Schema for the vultrstartupscripts API.
type VultrStartupScript struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VultrStartupScriptSpec   `json:"spec,omitempty"`
	Status VultrStartupScriptStatus `json:"status,omitempty"`
}

func (r *VultrStartupScript) GetConditions() clusterv1.Conditions {
	return r.Status.Conditions
}

func (r *VultrStartupScript) SetConditions(conditions clusterv1.Conditions) {
	r.Status.Conditions = conditions
}

//+kubebuilder:object:root=true

// VultrStartupScriptList contains a list of VultrStartupScript.
type VultrStartupScriptList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VultrStartupScript `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VultrStartupScript{}, &VultrStartupScriptList{})
}
