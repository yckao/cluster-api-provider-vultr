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
	"time"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/pkg/errors"
	"github.com/vultr/govultr/v3"

	infrav1 "github.com/vultr/cluster-api-provider-vultr/api/v1beta1"
	"github.com/vultr/cluster-api-provider-vultr/cloud/scope"
	"github.com/vultr/cluster-api-provider-vultr/util/reconciler"
)

// VultrVPCReconciler reconciles a VultrVPC object.
type VultrVPCReconciler struct {
	client.Client
	ReconcileTimeout time.Duration
	Recorder         record.EventRecorder
}

// SetupWithManager sets up the controller with the Manager.
func (r *VultrVPCReconciler) SetupWithManager(_ context.Context, mgr ctrl.Manager, _ controller.Options) error {
	err := ctrl.NewControllerManagedBy(mgr).
		For(&infrav1.VultrVPC{}).
		Complete(r)
	if err != nil {
		return errors.Wrapf(err, "failed to build controller")
	}
	return nil
}

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=vultrvpcs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=vultrvpcs/status,verbs=get;update;patch

func (r *VultrVPCReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	ctx, cancel := context.WithTimeout(ctx, reconciler.DefaultedLoopTimeout(r.ReconcileTimeout))
	defer cancel()

	log := ctrl.LoggerFrom(ctx)

	vultrVPC := &infrav1.VultrVPC{}
	if err := r.Get(ctx, req.NamespacedName, vultrVPC); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	vultrClient, err := scope.CreateVultrClient()
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "failed to create Vultr client")
	}

	// Handle deleted resources
	if !vultrVPC.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, log, vultrClient, vultrVPC)
	}

	return r.reconcileNormal(ctx, log, vultrClient, vultrVPC)
}

func (r *VultrVPCReconciler) reconcileNormal(ctx context.Context, log logr.Logger, vultrClient *govultr.Client, vultrVPC *infrav1.VultrVPC) (ctrl.Result, error) {
	log.Info("Reconciling VultrVPC")

	if controllerutil.AddFinalizer(vultrVPC, infrav1.VPCFinalizer) {
		if err := r.Update(ctx, vultrVPC); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	if vultrVPC.Status.ID != "" {
		vpc, _, err := vultrClient.VPC.Get(ctx, vultrVPC.Status.ID)
		if err != nil {
			return ctrl.Result{}, errors.Wrap(err, "failed to get VPC")
		}

		vultrVPC.Status.Ready = true
		vultrVPC.Status.ID = vpc.ID
		if err := r.Status().Update(ctx, vultrVPC); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	vpc, _, err := vultrClient.VPC.Create(ctx, &govultr.VPCReq{
		Region:       vultrVPC.Spec.Region,
		Description:  vultrVPC.Spec.Description,
		V4Subnet:     vultrVPC.Spec.V4Subnet,
		V4SubnetMask: vultrVPC.Spec.V4SubnetMask,
	})
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "failed to create VPC")
	}

	vultrVPC.Status.Ready = true
	vultrVPC.Status.ID = vpc.ID
	if err := r.Status().Update(ctx, vultrVPC); err != nil {
		return ctrl.Result{}, err
	}

	r.Recorder.Eventf(vultrVPC, "Normal", "VPCCreated", "Created VPC %s", vpc.ID)
	return ctrl.Result{}, nil
}

func (r *VultrVPCReconciler) reconcileDelete(ctx context.Context, log logr.Logger, vultrClient *govultr.Client, vultrVPC *infrav1.VultrVPC) (ctrl.Result, error) {
	log.Info("Reconciling delete VultrVPC")

	if vultrVPC.Status.ID != "" {
		if err := vultrClient.VPC.Delete(ctx, vultrVPC.Status.ID); err != nil {
			return ctrl.Result{}, errors.Wrap(err, "failed to delete VPC")
		}
		r.Recorder.Eventf(vultrVPC, "Normal", "VPCDeleted", "Deleted VPC %s", vultrVPC.Status.ID)
	}

	controllerutil.RemoveFinalizer(vultrVPC, infrav1.VPCFinalizer)
	if err := r.Update(ctx, vultrVPC); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}
