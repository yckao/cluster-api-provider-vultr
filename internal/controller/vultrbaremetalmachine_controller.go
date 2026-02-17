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
	"encoding/json"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/annotations"
	"sigs.k8s.io/cluster-api/util/predicates"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/pkg/errors"
	infrav1 "github.com/vultr/cluster-api-provider-vultr/api/v1beta1"
	"github.com/vultr/cluster-api-provider-vultr/cloud/scope"
	"github.com/vultr/cluster-api-provider-vultr/cloud/services"
	"github.com/vultr/cluster-api-provider-vultr/util/reconciler"
	capierrors "sigs.k8s.io/cluster-api/errors" //nolint:staticcheck
)

// VultrBareMetalMachineReconciler reconciles a VultrBareMetalMachine object
type VultrBareMetalMachineReconciler struct {
	client.Client
	Recorder         record.EventRecorder
	ReconcileTimeout time.Duration
}

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=vultrbaremetalmachines,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=vultrbaremetalmachines/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=vultrbaremetalmachines/finalizers,verbs=update

func (r *VultrBareMetalMachineReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	ctx, cancel := context.WithTimeout(ctx, reconciler.DefaultedLoopTimeout(r.ReconcileTimeout))
	defer cancel()

	log := ctrl.LoggerFrom(ctx)

	// Fetch the VultrBareMetalMachine.
	vultrBareMetalMachine := &infrav1.VultrBareMetalMachine{}
	if err := r.Get(ctx, req.NamespacedName, vultrBareMetalMachine); err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// Fetch the Machine.
	machine, err := util.GetOwnerMachine(ctx, r.Client, vultrBareMetalMachine.ObjectMeta)
	if err != nil {
		return ctrl.Result{}, err
	}
	if machine == nil {
		log.Info("Machine Controller has not yet set OwnerRef")
		return ctrl.Result{}, nil
	}

	// Fetch the Cluster.
	cluster, err := util.GetClusterFromMetadata(ctx, r.Client, machine.ObjectMeta)
	if err != nil {
		log.Info("Machine is missing cluster label or cluster does not exist")
		return ctrl.Result{}, nil
	}

	// Fetch the VultrCluster.
	vultrCluster := &infrav1.VultrCluster{}
	vultrClusterName := client.ObjectKey{
		Namespace: vultrBareMetalMachine.Namespace,
		Name:      cluster.Spec.InfrastructureRef.Name,
	}
	if err := r.Get(ctx, vultrClusterName, vultrCluster); err != nil && !scope.MachineOnly() {
		log.Info("VultrCluster is not available yet.")
		return ctrl.Result{}, nil
	}

	// Return early if the object or Cluster is paused.
	if annotations.IsPaused(cluster, vultrCluster) && !scope.MachineOnly() {
		log.Info("VultrBareMetalMachine or linked Cluster is marked as paused. Won't reconcile")
		return ctrl.Result{}, nil
	}

	// Create the cluster scope.
	clusterScope, err := scope.NewClusterScope(scope.ClusterScopeParams{
		Client:       r.Client,
		Logger:       log,
		Cluster:      cluster,
		VultrCluster: vultrCluster,
	})
	if err != nil {
		return ctrl.Result{}, errors.Errorf("failed to create scope: %v", err)
	}

	// Create the bare metal machine scope
	machineScope, err := scope.NewBareMetalMachineScope(scope.BareMetalMachineScopeParams{
		Client:                r.Client,
		Logger:                log,
		Cluster:               cluster,
		Machine:               machine,
		VultrCluster:          vultrCluster,
		VultrBareMetalMachine: vultrBareMetalMachine,
	})
	if err != nil {
		return ctrl.Result{}, errors.Errorf("failed to create bare metal machine scope: %v", err)
	}

	defer func() {
		err := machineScope.Close()
		if err != nil && reterr == nil {
			reterr = err
		}
	}()

	if !vultrBareMetalMachine.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, machineScope, clusterScope)
	}

	return r.reconcileNormal(ctx, machineScope, clusterScope)

}

func (r *VultrBareMetalMachineReconciler) reconcileNormal(ctx context.Context, machineScope *scope.BareMetalMachineScope, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	machineScope.Info("Reconciling VultrBareMetalMachine")
	vultrBareMetalMachine := machineScope.VultrBareMetalMachine

	if vultrBareMetalMachine.Status.FailureReason != nil || vultrBareMetalMachine.Status.FailureMessage != nil {
		machineScope.Info("Error state detected, skipping reconciliation")
		return reconcile.Result{}, nil
	}

	// If the VultrBareMetalMachine doesn't have our finalizer, add it.
	controllerutil.AddFinalizer(machineScope.VultrBareMetalMachine, infrav1.BareMetalMachineFinalizer)

	if !machineScope.Cluster.Status.InfrastructureReady {
		machineScope.Info("Cluster infrastructure is not ready yet")
		return reconcile.Result{}, nil
	}

	// Make sure bootstrap data is available and populated.
	if machineScope.Machine.Spec.Bootstrap.DataSecretName == nil {
		machineScope.Info("Bootstrap data secret reference is not yet available")
		return reconcile.Result{}, nil
	}

	r.Recorder.Event(vultrBareMetalMachine, corev1.EventTypeNormal, "BareMetalServiceInitializing", "Initializing bare metal service")
	bmSvc := services.NewService(ctx, clusterScope)
	r.Recorder.Event(vultrBareMetalMachine, corev1.EventTypeNormal, "BareMetalServiceInitialized", "Bare metal service initialized")

	serverID := machineScope.GetInstanceID()
	r.Recorder.Eventf(vultrBareMetalMachine, corev1.EventTypeNormal, "BareMetalServerRetrieving", "Retrieving bare metal server with ID %s", serverID)
	server, err := bmSvc.GetBareMetalServer(serverID)
	if err != nil {
		return reconcile.Result{}, err
	}

	if server == nil {
		r.Recorder.Eventf(vultrBareMetalMachine, corev1.EventTypeNormal, "BareMetalServerCreating", "Server is nil attempting create %v", server)
		server, err = bmSvc.CreateBareMetalServer(machineScope)
		serverPayload, _ := json.Marshal(vultrBareMetalMachine)
		machineScope.Info("Created new bare metal server", "payload", string(serverPayload))
		if err != nil {
			err = errors.Errorf("Failed to create bare metal server for VultrBareMetalMachine %s/%s: %v", vultrBareMetalMachine.Namespace, vultrBareMetalMachine.Name, err)
			r.Recorder.Event(vultrBareMetalMachine, corev1.EventTypeWarning, "BareMetalServerCreatingError", err.Error())
			return reconcile.Result{}, err
		}
		r.Recorder.Eventf(vultrBareMetalMachine, corev1.EventTypeNormal, "BareMetalServerCreated", "Created new bare metal server - %s, payload: %s", server.Label, string(serverPayload))
	}

	machineScope.SetProviderID(server.ID)
	r.Recorder.Eventf(vultrBareMetalMachine, corev1.EventTypeNormal, "SetBareMetalStatus", "Setting Bare Metal Server Status %s", server.Label)
	machineScope.SetInstanceStatus(infrav1.SubscriptionStatus(server.Status))
	machineScope.SetCPU(server.CPUCount)
	machineScope.SetRAM(server.RAM)
	machineScope.SetStorage(server.Disk)

	if strings.Contains(server.Label, "control-plane") {
		r.Recorder.Eventf(vultrBareMetalMachine, corev1.EventTypeNormal, "AddBareMetalServerToVLB", "Server %s is a control plane node, adding to VLB", server.ID)
		err := bmSvc.AddBareMetalServerToVLB(clusterScope.APIServerLoadbalancersRef().ResourceID, server.ID)
		if err != nil {
			r.Recorder.Eventf(vultrBareMetalMachine, corev1.EventTypeWarning, "AddBareMetalServerToVLBFailed", "Failed to add server %s to VLB: %v", server.ID, err)
			return reconcile.Result{}, errors.Wrap(err, "failed to add bare metal server to VLB")
		}
		r.Recorder.Eventf(vultrBareMetalMachine, corev1.EventTypeNormal, "AddBareMetalServerToVLBSuccess", "Successfully added server %s to VLB", server.ID)
	}

	r.Recorder.Eventf(vultrBareMetalMachine, corev1.EventTypeNormal, "GetBareMetalServerAddress", "Getting address for server %s", server.ID)
	addrs, err := bmSvc.GetBareMetalServerAddress(server)
	if err != nil {
		r.Recorder.Eventf(vultrBareMetalMachine, corev1.EventTypeWarning, "GetBareMetalServerAddressFailed", "Failed to get address for server %s: %v", server.ID, err)
		machineScope.SetFailureMessage(errors.New("failed to get bare metal server address"))
		return reconcile.Result{}, err
	}
	r.Recorder.Eventf(vultrBareMetalMachine, corev1.EventTypeNormal, "GetBareMetalServerAddressSuccess", "Successfully retrieved address for server %s: %v", server.ID, addrs)
	machineScope.SetAddresses(addrs)

	switch infrav1.SubscriptionStatus(server.Status) {
	case infrav1.SubscriptionStatusPending:
		machineScope.Info("Bare metal server is pending", "server-id", machineScope.GetInstanceID())
		return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
	case infrav1.SubscriptionStatusActive:
		machineScope.Info("Bare metal server is active", "server-id", machineScope.GetInstanceID())
		machineScope.SetReady()
		return reconcile.Result{}, nil
	default:
		machineScope.SetFailureReason(capierrors.UpdateMachineError)
		machineScope.SetFailureMessage(errors.Errorf("Bare metal server status %q is unexpected", server.Status))
		return reconcile.Result{}, nil
	}
}

func (r *VultrBareMetalMachineReconciler) reconcileDelete(ctx context.Context, machineScope *scope.BareMetalMachineScope, clusterScope *scope.ClusterScope) (reconcile.Result, error) { //nolint: unparam
	machineScope.Info("Reconciling delete VultrBareMetalMachine")
	vultrBareMetalMachine := machineScope.VultrBareMetalMachine

	bmSvc := services.NewService(ctx, clusterScope)
	server, err := bmSvc.GetBareMetalServer(machineScope.GetInstanceID())
	if err != nil {
		return reconcile.Result{}, err
	}

	if server != nil {
		if err := bmSvc.DeleteBareMetalServer(machineScope.GetInstanceID()); err != nil {
			return reconcile.Result{}, err
		}
	} else {
		clusterScope.V(2).Info("Unable to locate bare metal server")
		r.Recorder.Eventf(vultrBareMetalMachine, corev1.EventTypeWarning, "NoBareMetalServerFound", "Skip deleting")
	}

	r.Recorder.Eventf(vultrBareMetalMachine, corev1.EventTypeNormal, "BareMetalServerDeleted", "Deleted a bare metal server - %s", machineScope.Name())
	controllerutil.RemoveFinalizer(vultrBareMetalMachine, infrav1.BareMetalMachineFinalizer)
	return reconcile.Result{}, nil
}
func (r *VultrBareMetalMachineReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager, _ controller.Options) error {
	clusterToObjectFunc, err := util.ClusterToTypedObjectsMapper(r.Client, &infrav1.VultrBareMetalMachineList{}, mgr.GetScheme())
	if err != nil {
		return errors.Wrapf(err, "failed to create mapper for Cluster to VultrBareMetalMachines")
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1.VultrBareMetalMachine{}).
		WithEventFilter(predicates.ResourceNotPaused(mgr.GetScheme(), ctrl.LoggerFrom(ctx))).
		Watches(
			&clusterv1.Machine{},
			handler.EnqueueRequestsFromMapFunc(util.MachineToInfrastructureMapFunc(infrav1.GroupVersion.WithKind("VultrBareMetalMachine"))),
		).
		Watches(
			&infrav1.VultrCluster{},
			handler.EnqueueRequestsFromMapFunc(r.VultrClusterToVultrBareMetalMachines(ctx)),
		).
		Watches(
			&clusterv1.Cluster{},
			handler.EnqueueRequestsFromMapFunc(clusterToObjectFunc),
			builder.WithPredicates(predicates.ClusterPausedTransitionsOrInfrastructureReady(mgr.GetScheme(), ctrl.LoggerFrom(ctx))),
		).
		Complete(r)
}

// VultrClusterToVultrBareMetalMachines convert the cluster to bare metal machines spec.
func (r *VultrBareMetalMachineReconciler) VultrClusterToVultrBareMetalMachines(ctx context.Context) handler.MapFunc {
	log := ctrl.LoggerFrom(ctx)
	return func(ctx context.Context, o client.Object) []ctrl.Request {
		result := []ctrl.Request{}

		c, ok := o.(*infrav1.VultrCluster)
		if !ok {
			log.Error(errors.Errorf("expected a VultrCluster but got a %T", o), "failed to get VultrBareMetalMachine for VultrCluster")
			return nil
		}

		cluster, err := util.GetOwnerCluster(ctx, r.Client, c.ObjectMeta)
		switch {
		case apierrors.IsNotFound(err) || cluster == nil:
			return result
		case err != nil:
			log.Error(err, "failed to get owning cluster")
			return result
		}

		labels := map[string]string{clusterv1.ClusterNameLabel: cluster.Name}
		machineList := &clusterv1.MachineList{}
		if err := r.List(ctx, machineList, client.InNamespace(c.Namespace), client.MatchingLabels(labels)); err != nil {
			log.Error(err, "failed to list Machines")
			return nil
		}
		for _, m := range machineList.Items {
			if m.Spec.InfrastructureRef.Name == "" {
				continue
			}
			name := client.ObjectKey{Namespace: m.Namespace, Name: m.Spec.InfrastructureRef.Name}
			result = append(result, ctrl.Request{NamespacedName: name})
		}

		return result
	}
}
