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
	// FirewallGroupFinalizer allows ReconcileVultrFirewallGroup to clean up Vultr resources before removing it from the apiserver.
	FirewallGroupFinalizer = "vultrfirewallgroup.infrastructure.cluster.x-k8s.io"
)

// VultrFirewallRuleSpec defines a firewall rule to be created within the group.
type VultrFirewallRuleSpec struct {
	// IPType is the IP type: "v4" or "v6".
	IPType string `json:"ipType"`

	// Protocol is the protocol: "tcp", "udp", or "icmp".
	Protocol string `json:"protocol"`

	// Subnet is the IP address representing a subnet.
	Subnet string `json:"subnet"`

	// SubnetSize is the number of bits for the netmask.
	SubnetSize int `json:"subnetSize"`

	// Port is the port or port range (e.g. "8080" or "8080:8085").
	// +optional
	Port string `json:"port,omitempty"`

	// Source is the source string if applicable.
	// +optional
	Source string `json:"source,omitempty"`

	// Notes is optional notes for the rule.
	// +optional
	Notes string `json:"notes,omitempty"`
}

// VultrFirewallGroupSpec defines the desired state of VultrFirewallGroup.
type VultrFirewallGroupSpec struct {
	// Description is an optional description for the firewall group.
	// +optional
	Description string `json:"description,omitempty"`

	// Rules is a list of firewall rules to create in this group.
	// +optional
	Rules []VultrFirewallRuleSpec `json:"rules,omitempty"`
}

// VultrFirewallGroupStatus defines the observed state of VultrFirewallGroup.
type VultrFirewallGroupStatus struct {
	// Ready denotes that the firewall group is ready.
	// +optional
	Ready bool `json:"ready"`

	// ID is the Vultr firewall group ID.
	// +optional
	ID string `json:"id,omitempty"`

	// RuleCount is the number of rules in the firewall group.
	// +optional
	RuleCount int `json:"ruleCount,omitempty"`

	// Conditions defines current service state of the VultrFirewallGroup.
	// +optional
	Conditions clusterv1.Conditions `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:path=vultrfirewallgroups,scope=Namespaced,categories=cluster-api
//+kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.ready",description="FirewallGroup is ready"
//+kubebuilder:printcolumn:name="ID",type="string",JSONPath=".status.id",description="Vultr FirewallGroup ID"
//+kubebuilder:printcolumn:name="Rules",type="integer",JSONPath=".status.ruleCount",description="Number of firewall rules"

// VultrFirewallGroup is the Schema for the vultrfirewallgroups API.
type VultrFirewallGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VultrFirewallGroupSpec   `json:"spec,omitempty"`
	Status VultrFirewallGroupStatus `json:"status,omitempty"`
}

func (r *VultrFirewallGroup) GetConditions() clusterv1.Conditions {
	return r.Status.Conditions
}

func (r *VultrFirewallGroup) SetConditions(conditions clusterv1.Conditions) {
	r.Status.Conditions = conditions
}

//+kubebuilder:object:root=true

// VultrFirewallGroupList contains a list of VultrFirewallGroup.
type VultrFirewallGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VultrFirewallGroup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VultrFirewallGroup{}, &VultrFirewallGroupList{})
}
