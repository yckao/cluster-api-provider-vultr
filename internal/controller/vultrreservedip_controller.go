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

// VultrReservedIPReconciler reconciles a VultrReservedIP object.
type VultrReservedIPReconciler struct {
	client.Client
	ReconcileTimeout time.Duration
	Recorder         record.EventRecorder
}

// SetupWithManager sets up the controller with the Manager.
func (r *VultrReservedIPReconciler) SetupWithManager(_ context.Context, mgr ctrl.Manager, _ controller.Options) error {
	err := ctrl.NewControllerManagedBy(mgr).
		For(&infrav1.VultrReservedIP{}).
		Complete(r)
	if err != nil {
		return errors.Wrapf(err, "failed to build controller")
	}
	return nil
}

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=vultrreservedips,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=vultrreservedips/status,verbs=get;update;patch

func (r *VultrReservedIPReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	ctx, cancel := context.WithTimeout(ctx, reconciler.DefaultedLoopTimeout(r.ReconcileTimeout))
	defer cancel()

	log := ctrl.LoggerFrom(ctx)

	vultrRIP := &infrav1.VultrReservedIP{}
	if err := r.Get(ctx, req.NamespacedName, vultrRIP); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	vultrClient, err := scope.CreateVultrClient()
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "failed to create Vultr client")
	}

	if !vultrRIP.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, log, vultrClient, vultrRIP)
	}

	return r.reconcileNormal(ctx, log, vultrClient, vultrRIP)
}

func (r *VultrReservedIPReconciler) reconcileNormal(ctx context.Context, log logr.Logger, vultrClient *govultr.Client, vultrRIP *infrav1.VultrReservedIP) (ctrl.Result, error) {
	log.Info("Reconciling VultrReservedIP")

	if controllerutil.AddFinalizer(vultrRIP, infrav1.ReservedIPFinalizer) {
		if err := r.Update(ctx, vultrRIP); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	if vultrRIP.Status.ID != "" {
		rip, _, err := vultrClient.ReservedIP.Get(ctx, vultrRIP.Status.ID)
		if err != nil {
			return ctrl.Result{}, errors.Wrap(err, "failed to get reserved IP")
		}

		vultrRIP.Status.Ready = true
		vultrRIP.Status.ID = rip.ID
		vultrRIP.Status.Subnet = rip.Subnet
		vultrRIP.Status.SubnetSize = rip.SubnetSize
		vultrRIP.Status.InstanceID = rip.InstanceID
		if err := r.Status().Update(ctx, vultrRIP); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	rip, _, err := vultrClient.ReservedIP.Create(ctx, &govultr.ReservedIPReq{
		Region: vultrRIP.Spec.Region,
		IPType: vultrRIP.Spec.IPType,
		Label:  vultrRIP.Spec.Label,
	})
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "failed to create reserved IP")
	}

	vultrRIP.Status.Ready = true
	vultrRIP.Status.ID = rip.ID
	vultrRIP.Status.Subnet = rip.Subnet
	vultrRIP.Status.SubnetSize = rip.SubnetSize
	vultrRIP.Status.InstanceID = rip.InstanceID
	if err := r.Status().Update(ctx, vultrRIP); err != nil {
		return ctrl.Result{}, err
	}

	r.Recorder.Eventf(vultrRIP, "Normal", "ReservedIPCreated", "Created reserved IP %s (%s)", rip.ID, rip.Subnet)
	return ctrl.Result{}, nil
}

func (r *VultrReservedIPReconciler) reconcileDelete(ctx context.Context, log logr.Logger, vultrClient *govultr.Client, vultrRIP *infrav1.VultrReservedIP) (ctrl.Result, error) {
	log.Info("Reconciling delete VultrReservedIP")

	if vultrRIP.Status.ID != "" {
		if err := vultrClient.ReservedIP.Delete(ctx, vultrRIP.Status.ID); err != nil {
			return ctrl.Result{}, errors.Wrap(err, "failed to delete reserved IP")
		}
		r.Recorder.Eventf(vultrRIP, "Normal", "ReservedIPDeleted", "Deleted reserved IP %s", vultrRIP.Status.ID)
	}

	controllerutil.RemoveFinalizer(vultrRIP, infrav1.ReservedIPFinalizer)
	if err := r.Update(ctx, vultrRIP); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}
