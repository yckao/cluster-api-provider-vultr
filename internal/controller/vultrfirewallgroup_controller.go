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

// VultrFirewallGroupReconciler reconciles a VultrFirewallGroup object.
type VultrFirewallGroupReconciler struct {
	client.Client
	ReconcileTimeout time.Duration
	Recorder         record.EventRecorder
}

// SetupWithManager sets up the controller with the Manager.
func (r *VultrFirewallGroupReconciler) SetupWithManager(_ context.Context, mgr ctrl.Manager, _ controller.Options) error {
	err := ctrl.NewControllerManagedBy(mgr).
		For(&infrav1.VultrFirewallGroup{}).
		Complete(r)
	if err != nil {
		return errors.Wrapf(err, "failed to build controller")
	}
	return nil
}

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=vultrfirewallgroups,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=vultrfirewallgroups/status,verbs=get;update;patch

func (r *VultrFirewallGroupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	ctx, cancel := context.WithTimeout(ctx, reconciler.DefaultedLoopTimeout(r.ReconcileTimeout))
	defer cancel()

	log := ctrl.LoggerFrom(ctx)

	vultrFWGroup := &infrav1.VultrFirewallGroup{}
	if err := r.Get(ctx, req.NamespacedName, vultrFWGroup); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	vultrClient, err := scope.CreateVultrClient()
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "failed to create Vultr client")
	}

	if !vultrFWGroup.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, log, vultrClient, vultrFWGroup)
	}

	return r.reconcileNormal(ctx, log, vultrClient, vultrFWGroup)
}

func (r *VultrFirewallGroupReconciler) reconcileNormal(ctx context.Context, log logr.Logger, vultrClient *govultr.Client, vultrFWGroup *infrav1.VultrFirewallGroup) (ctrl.Result, error) {
	log.Info("Reconciling VultrFirewallGroup")

	if controllerutil.AddFinalizer(vultrFWGroup, infrav1.FirewallGroupFinalizer) {
		if err := r.Update(ctx, vultrFWGroup); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	var groupID string

	if vultrFWGroup.Status.ID != "" {
		fg, _, err := vultrClient.FirewallGroup.Get(ctx, vultrFWGroup.Status.ID)
		if err != nil {
			return ctrl.Result{}, errors.Wrap(err, "failed to get firewall group")
		}
		groupID = fg.ID
	} else {
		fg, _, err := vultrClient.FirewallGroup.Create(ctx, &govultr.FirewallGroupReq{
			Description: vultrFWGroup.Spec.Description,
		})
		if err != nil {
			return ctrl.Result{}, errors.Wrap(err, "failed to create firewall group")
		}
		groupID = fg.ID
		r.Recorder.Eventf(vultrFWGroup, "Normal", "FirewallGroupCreated", "Created firewall group %s", fg.ID)
	}

	// Reconcile firewall rules
	if err := r.reconcileRules(ctx, vultrClient, vultrFWGroup, groupID); err != nil {
		return ctrl.Result{}, errors.Wrap(err, "failed to reconcile firewall rules")
	}

	// Refresh group to get updated rule count
	fg, _, err := vultrClient.FirewallGroup.Get(ctx, groupID)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "failed to get firewall group")
	}

	vultrFWGroup.Status.Ready = true
	vultrFWGroup.Status.ID = groupID
	vultrFWGroup.Status.RuleCount = fg.RuleCount
	if err := r.Status().Update(ctx, vultrFWGroup); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *VultrFirewallGroupReconciler) reconcileRules(ctx context.Context, vultrClient *govultr.Client, vultrFWGroup *infrav1.VultrFirewallGroup, groupID string) error {
	// List existing rules
	existingRules, err := r.listAllFirewallRules(ctx, vultrClient, groupID)
	if err != nil {
		return errors.Wrap(err, "failed to list firewall rules")
	}

	// Build a set of desired rules for comparison
	desiredRules := vultrFWGroup.Spec.Rules

	// Delete rules not in spec
	for _, existing := range existingRules {
		found := false
		for _, desired := range desiredRules {
			if ruleMatches(existing, desired) {
				found = true
				break
			}
		}
		if !found {
			if err := vultrClient.FirewallRule.Delete(ctx, groupID, existing.ID); err != nil {
				return errors.Wrapf(err, "failed to delete firewall rule %d", existing.ID)
			}
		}
	}

	// Create rules missing from existing
	for _, desired := range desiredRules {
		found := false
		for _, existing := range existingRules {
			if ruleMatches(existing, desired) {
				found = true
				break
			}
		}
		if !found {
			_, _, err := vultrClient.FirewallRule.Create(ctx, groupID, &govultr.FirewallRuleReq{
				IPType:     desired.IPType,
				Protocol:   desired.Protocol,
				Subnet:     desired.Subnet,
				SubnetSize: desired.SubnetSize,
				Port:       desired.Port,
				Source:     desired.Source,
				Notes:      desired.Notes,
			})
			if err != nil {
				return errors.Wrap(err, "failed to create firewall rule")
			}
		}
	}

	return nil
}

func ruleMatches(existing govultr.FirewallRule, desired infrav1.VultrFirewallRuleSpec) bool {
	return existing.IPType == desired.IPType &&
		existing.Protocol == desired.Protocol &&
		existing.Subnet == desired.Subnet &&
		existing.SubnetSize == desired.SubnetSize &&
		existing.Port == desired.Port
}

func (r *VultrFirewallGroupReconciler) listAllFirewallRules(ctx context.Context, vultrClient *govultr.Client, groupID string) ([]govultr.FirewallRule, error) {
	var allRules []govultr.FirewallRule
	options := &govultr.ListOptions{PerPage: 100}

	for {
		rules, meta, _, err := vultrClient.FirewallRule.List(ctx, groupID, options)
		if err != nil {
			return nil, err
		}
		allRules = append(allRules, rules...)
		if meta.Links.Next == "" {
			break
		}
		options.Cursor = meta.Links.Next
	}

	return allRules, nil
}

func (r *VultrFirewallGroupReconciler) reconcileDelete(ctx context.Context, log logr.Logger, vultrClient *govultr.Client, vultrFWGroup *infrav1.VultrFirewallGroup) (ctrl.Result, error) {
	log.Info("Reconciling delete VultrFirewallGroup")

	if vultrFWGroup.Status.ID != "" {
		if err := vultrClient.FirewallGroup.Delete(ctx, vultrFWGroup.Status.ID); err != nil {
			return ctrl.Result{}, errors.Wrap(err, "failed to delete firewall group")
		}
		r.Recorder.Eventf(vultrFWGroup, "Normal", "FirewallGroupDeleted", "Deleted firewall group %s", vultrFWGroup.Status.ID)
	}

	controllerutil.RemoveFinalizer(vultrFWGroup, infrav1.FirewallGroupFinalizer)
	if err := r.Update(ctx, vultrFWGroup); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}
