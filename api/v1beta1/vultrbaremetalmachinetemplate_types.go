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
)

// VultrBareMetalMachineTemplateSpec defines the desired state of VultrBareMetalMachineTemplate
type VultrBareMetalMachineTemplateSpec struct {
	Template VultrBareMetalMachineTemplateResource `json:"template"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=vultrbaremetalmachinetemplates,scope=Namespaced,categories=cluster-api,shortName=vbmmt
// VultrBareMetalMachineTemplate is the Schema for the vultrbaremetalmachinetemplates API

type VultrBareMetalMachineTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec VultrBareMetalMachineTemplateSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true
// VultrBareMetalMachineTemplateList contains a list of VultrBareMetalMachineTemplate
type VultrBareMetalMachineTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VultrBareMetalMachineTemplate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VultrBareMetalMachineTemplate{}, &VultrBareMetalMachineTemplateList{})
}
