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

// VultrStartupScriptReconciler reconciles a VultrStartupScript object.
type VultrStartupScriptReconciler struct {
	client.Client
	ReconcileTimeout time.Duration
	Recorder         record.EventRecorder
}

// SetupWithManager sets up the controller with the Manager.
func (r *VultrStartupScriptReconciler) SetupWithManager(_ context.Context, mgr ctrl.Manager, _ controller.Options) error {
	err := ctrl.NewControllerManagedBy(mgr).
		For(&infrav1.VultrStartupScript{}).
		Complete(r)
	if err != nil {
		return errors.Wrapf(err, "failed to build controller")
	}
	return nil
}

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=vultrstartupscripts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=vultrstartupscripts/status,verbs=get;update;patch

func (r *VultrStartupScriptReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	ctx, cancel := context.WithTimeout(ctx, reconciler.DefaultedLoopTimeout(r.ReconcileTimeout))
	defer cancel()

	log := ctrl.LoggerFrom(ctx)

	vultrScript := &infrav1.VultrStartupScript{}
	if err := r.Get(ctx, req.NamespacedName, vultrScript); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	vultrClient, err := scope.CreateVultrClient()
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "failed to create Vultr client")
	}

	if !vultrScript.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, log, vultrClient, vultrScript)
	}

	return r.reconcileNormal(ctx, log, vultrClient, vultrScript)
}

func (r *VultrStartupScriptReconciler) reconcileNormal(ctx context.Context, log logr.Logger, vultrClient *govultr.Client, vultrScript *infrav1.VultrStartupScript) (ctrl.Result, error) {
	log.Info("Reconciling VultrStartupScript")

	if controllerutil.AddFinalizer(vultrScript, infrav1.StartupScriptFinalizer) {
		if err := r.Update(ctx, vultrScript); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	if vultrScript.Status.ID != "" {
		script, _, err := vultrClient.StartupScript.Get(ctx, vultrScript.Status.ID)
		if err != nil {
			return ctrl.Result{}, errors.Wrap(err, "failed to get startup script")
		}

		vultrScript.Status.Ready = true
		vultrScript.Status.ID = script.ID
		if err := r.Status().Update(ctx, vultrScript); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	script, _, err := vultrClient.StartupScript.Create(ctx, &govultr.StartupScriptReq{
		Name:   vultrScript.Spec.Name,
		Script: vultrScript.Spec.Script,
		Type:   vultrScript.Spec.Type,
	})
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "failed to create startup script")
	}

	vultrScript.Status.Ready = true
	vultrScript.Status.ID = script.ID
	if err := r.Status().Update(ctx, vultrScript); err != nil {
		return ctrl.Result{}, err
	}

	r.Recorder.Eventf(vultrScript, "Normal", "StartupScriptCreated", "Created startup script %s", script.ID)
	return ctrl.Result{}, nil
}

func (r *VultrStartupScriptReconciler) reconcileDelete(ctx context.Context, log logr.Logger, vultrClient *govultr.Client, vultrScript *infrav1.VultrStartupScript) (ctrl.Result, error) {
	log.Info("Reconciling delete VultrStartupScript")

	if vultrScript.Status.ID != "" {
		if err := vultrClient.StartupScript.Delete(ctx, vultrScript.Status.ID); err != nil {
			return ctrl.Result{}, errors.Wrap(err, "failed to delete startup script")
		}
		r.Recorder.Eventf(vultrScript, "Normal", "StartupScriptDeleted", "Deleted startup script %s", vultrScript.Status.ID)
	}

	controllerutil.RemoveFinalizer(vultrScript, infrav1.StartupScriptFinalizer)
	if err := r.Update(ctx, vultrScript); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}
