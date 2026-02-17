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

package controller

import (
	"context"
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	infrav1 "github.com/vultr/cluster-api-provider-vultr/api/v1beta1"
)

// ErrResourceNotReady is returned when a referenced CR exists but is not yet ready.
var ErrResourceNotReady = errors.New("referenced resource is not ready")

// resolveVPCRef resolves a VultrVPC CR reference to a Vultr VPC ID.
func resolveVPCRef(ctx context.Context, c client.Client, namespace string, ref *corev1.LocalObjectReference) (string, error) {
	vpc := &infrav1.VultrVPC{}
	key := client.ObjectKey{Namespace: namespace, Name: ref.Name}
	if err := c.Get(ctx, key, vpc); err != nil {
		return "", fmt.Errorf("failed to get VultrVPC %q: %w", ref.Name, err)
	}
	if !vpc.Status.Ready {
		return "", ErrResourceNotReady
	}
	return vpc.Status.ID, nil
}

// resolveFirewallGroupRef resolves a VultrFirewallGroup CR reference to a Vultr firewall group ID.
func resolveFirewallGroupRef(ctx context.Context, c client.Client, namespace string, ref *corev1.LocalObjectReference) (string, error) {
	fwGroup := &infrav1.VultrFirewallGroup{}
	key := client.ObjectKey{Namespace: namespace, Name: ref.Name}
	if err := c.Get(ctx, key, fwGroup); err != nil {
		return "", fmt.Errorf("failed to get VultrFirewallGroup %q: %w", ref.Name, err)
	}
	if !fwGroup.Status.Ready {
		return "", ErrResourceNotReady
	}
	return fwGroup.Status.ID, nil
}

// resolveStartupScriptRef resolves a VultrStartupScript CR reference to a Vultr startup script ID.
func resolveStartupScriptRef(ctx context.Context, c client.Client, namespace string, ref *corev1.LocalObjectReference) (string, error) {
	script := &infrav1.VultrStartupScript{}
	key := client.ObjectKey{Namespace: namespace, Name: ref.Name}
	if err := c.Get(ctx, key, script); err != nil {
		return "", fmt.Errorf("failed to get VultrStartupScript %q: %w", ref.Name, err)
	}
	if !script.Status.Ready {
		return "", ErrResourceNotReady
	}
	return script.Status.ID, nil
}

// resolveReservedIPRef resolves a VultrReservedIP CR reference to the assigned IP address (subnet).
func resolveReservedIPRef(ctx context.Context, c client.Client, namespace string, ref *corev1.LocalObjectReference) (string, error) {
	rip := &infrav1.VultrReservedIP{}
	key := client.ObjectKey{Namespace: namespace, Name: ref.Name}
	if err := c.Get(ctx, key, rip); err != nil {
		return "", fmt.Errorf("failed to get VultrReservedIP %q: %w", ref.Name, err)
	}
	if !rip.Status.Ready {
		return "", ErrResourceNotReady
	}
	return rip.Status.Subnet, nil
}
