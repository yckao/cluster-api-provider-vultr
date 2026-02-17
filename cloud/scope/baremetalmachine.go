/*

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

package scope

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	capierrors "sigs.k8s.io/cluster-api/errors" //nolint:staticcheck
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	infrav1 "github.com/vultr/cluster-api-provider-vultr/api/v1beta1"
)

// BareMetalMachineScopeParams defines the input parameters used to create a new BareMetalMachineScope.
type BareMetalMachineScopeParams struct {
	VultrAPIClients
	Client                client.Client
	Logger                logr.Logger
	Machine               *clusterv1.Machine
	Cluster               *clusterv1.Cluster
	VultrBareMetalMachine *infrav1.VultrBareMetalMachine
	VultrCluster          *infrav1.VultrCluster
}

// BareMetalMachineScope defines a scope defined around a bare metal machine and its cluster.
type BareMetalMachineScope struct {
	logr.Logger
	client      client.Client
	patchHelper *patch.Helper

	Machine               *clusterv1.Machine
	Cluster               *clusterv1.Cluster
	VultrBareMetalMachine *infrav1.VultrBareMetalMachine
	VultrCluster          *infrav1.VultrCluster
}

// NewBareMetalMachineScope creates a new Scope from the supplied parameters.
// This is meant to be called for each reconcile iteration.
func NewBareMetalMachineScope(params BareMetalMachineScopeParams) (*BareMetalMachineScope, error) {
	if params.Client == nil {
		return nil, errors.New("Client is required when creating a BareMetalMachineScope")
	}
	if params.Cluster == nil {
		return nil, errors.New("Cluster is required when creating a BareMetalMachineScope")
	}
	if params.Machine == nil {
		return nil, errors.New("Machine is required when creating a BareMetalMachineScope")
	}
	if params.VultrCluster == nil {
		return nil, errors.New("VultrCluster is required when creating a BareMetalMachineScope")
	}
	if params.VultrBareMetalMachine == nil {
		return nil, errors.New("VultrBareMetalMachine is required when creating a BareMetalMachineScope")
	}

	helper, err := patch.NewHelper(params.VultrBareMetalMachine, params.Client)
	if err != nil {
		return nil, errors.Wrap(err, "failed to init patch helper")
	}

	return &BareMetalMachineScope{
		client:                params.Client,
		Logger:                params.Logger,
		Cluster:               params.Cluster,
		Machine:               params.Machine,
		VultrCluster:          params.VultrCluster,
		VultrBareMetalMachine: params.VultrBareMetalMachine,
		patchHelper:           helper,
	}, nil
}

func (s *BareMetalMachineScope) Close() error {
	return s.patchHelper.Patch(context.TODO(), s.VultrBareMetalMachine)
}

// PatchObject persists the bare metal machine spec and status.
func (m *BareMetalMachineScope) PatchObject(ctx context.Context) error {
	return m.patchHelper.Patch(ctx, m.VultrBareMetalMachine)
}

// SetReady sets the VultrBareMetalMachine Ready Status.
func (m *BareMetalMachineScope) SetReady() {
	m.VultrBareMetalMachine.Status.Ready = true
}

// AddFinalizer adds a finalizer if not present and immediately patches the
// object to avoid any race conditions.
func (m *BareMetalMachineScope) AddFinalizer(ctx context.Context) error {
	if controllerutil.AddFinalizer(m.VultrBareMetalMachine, infrav1.GroupVersion.String()) {
		return m.Close()
	}

	return nil
}

// GetInstanceID returns the VultrBareMetalMachine instance id by parsing Spec.ProviderID.
func (m *BareMetalMachineScope) GetInstanceID() string {
	id := m.GetProviderID()

	split := strings.Split(id, "://")
	if len(split) != 2 { //nolint
		return ""
	}

	if split[0] != "vultr" {
		return ""
	}
	return split[1]
}

// GetProviderID returns the VultrBareMetalMachine providerID from the spec.
func (m *BareMetalMachineScope) GetProviderID() string {
	if m.VultrBareMetalMachine.Spec.ProviderID != nil {
		return *m.VultrBareMetalMachine.Spec.ProviderID
	}
	return ""
}

// SetProviderID sets the VultrBareMetalMachine providerID in spec from server id.
func (m *BareMetalMachineScope) SetProviderID(serverID string) {
	pid := fmt.Sprintf("vultr://%s", serverID)
	m.VultrBareMetalMachine.Spec.ProviderID = ptr.To(pid)
}

// Name returns the VultrBareMetalMachine name.
func (m *BareMetalMachineScope) Name() string {
	return m.VultrBareMetalMachine.Name
}

// Namespace returns the namespace name.
func (m *BareMetalMachineScope) Namespace() string {
	return m.VultrBareMetalMachine.Namespace
}

func (m *BareMetalMachineScope) GetBootstrapData() (string, error) {
	if m.Machine.Spec.Bootstrap.DataSecretName == nil {
		m.Info("Bootstrap data secret reference is nil")
		return "", errors.New("error retrieving bootstrap data: linked Machine's bootstrap.dataSecretName is nil")
	}

	secretName := *m.Machine.Spec.Bootstrap.DataSecretName
	key := types.NamespacedName{Namespace: m.Namespace(), Name: secretName}
	m.Info("Attempting to retrieve bootstrap data secret", "namespace", key.Namespace, "name", key.Name)

	secret := &corev1.Secret{}
	if err := m.client.Get(context.TODO(), key, secret); err != nil {
		m.Error(err, "Failed to retrieve bootstrap data secret", "namespace", key.Namespace, "name", key.Name)
		return "", errors.Wrapf(err, "failed to retrieve bootstrap data secret for VultrBareMetalMachine %s/%s", m.Namespace(), m.Name())
	}

	value, ok := secret.Data["value"]
	if !ok {
		m.Info("Bootstrap data secret missing 'value' key")
		return "", errors.New("error retrieving bootstrap data: secret value key is missing")
	}

	// Log the retrieved bootstrap data (truncated to avoid logging sensitive information)
	m.Info("Successfully retrieved bootstrap data", "value", string(value)[:min(50, len(value))])
	return string(value), nil
}

// IsControlPlane returns true if the machine is a control plane.
func (m *BareMetalMachineScope) IsControlPlane() bool {
	return util.IsControlPlaneMachine(m.Machine)
}

// Role returns the machine role from the labels.
func (m *BareMetalMachineScope) Role() string {
	if util.IsControlPlaneMachine(m.Machine) {
		return infrav1.APIServerRoleTagValue
	}
	return infrav1.NodeRoleTagValue
}

// GetInstanceStatus returns the VultrBareMetalMachine instance status from the status.
func (m *BareMetalMachineScope) GetInstanceStatus() *infrav1.SubscriptionStatus {
	return m.VultrBareMetalMachine.Status.SubscriptionStatus
}

// SetInstanceStatus sets the VultrBareMetalMachine subscription status.
func (m *BareMetalMachineScope) SetInstanceStatus(v infrav1.SubscriptionStatus) {
	m.VultrBareMetalMachine.Status.SubscriptionStatus = &v
}

// SetFailureMessage sets the VultrBareMetalMachine status error message.
func (m *BareMetalMachineScope) SetFailureMessage(v error) {
	m.VultrBareMetalMachine.Status.FailureMessage = ptr.To(v.Error())
}

// SetAddresses sets the address status.
func (m *BareMetalMachineScope) SetAddresses(addrs []corev1.NodeAddress) {
	m.VultrBareMetalMachine.Status.Addresses = addrs
}

// SetFailureReason sets the VultrBareMetalMachine status error reason.
func (m *BareMetalMachineScope) SetFailureReason(v capierrors.MachineStatusError) {
	m.VultrBareMetalMachine.Status.FailureReason = &v
}

// SetCPU sets the VultrBareMetalMachine CPU status.
func (m *BareMetalMachineScope) SetCPU(v int) {
	m.VultrBareMetalMachine.Status.CPU = v
}

// SetRAM sets the VultrBareMetalMachine RAM status.
func (m *BareMetalMachineScope) SetRAM(v string) {
	m.VultrBareMetalMachine.Status.RAM = v
}

// SetStorage sets the VultrBareMetalMachine Storage status.
func (m *BareMetalMachineScope) SetStorage(v string) {
	m.VultrBareMetalMachine.Status.Storage = v
}
