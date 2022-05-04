/*
Copyright 2022.

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

package controllers

import (
	"context"
	"fmt"

	"github.com/gardener/dependency-watchdog/internal/prober"
	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	gardencorev1beta1helper "github.com/gardener/gardener/pkg/apis/core/v1beta1/helper"
	gardenerv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/scale"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ClusterReconciler reconciles a Cluster object
type ClusterReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	ProberMgr   prober.Manager
	ScaleGetter scale.ScalesGetter
	ProbeConfig *prober.Config
}

//+kubebuilder:rbac:groups=gardener.cloud,resources=clusters,verbs=get;list;watch
//+kubebuilder:rbac:groups=gardener.cloud,resources=clusters/status,verbs=get

// Reconcile listens to create/update/delete events for `Cluster` resources and
// manages probes for the shoot control namespace for these clusters by looking at the cluster state.
func (r *ClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	fmt.Printf("Request is: %v\n", req)
	logger := log.FromContext(ctx)
	cluster, notFound, err := r.getCluster(ctx, req.Namespace, req.Name)
	if err != nil {
		fmt.Printf("Unable to get cluster resource Error is %v", err)
		logger.Error(err, "Unable to get the cluster resource, requeing for reconciliation", "namespace", req.Namespace, "name", req.Name)
		return ctrl.Result{}, err
	}
	// If the cluster is not found then any existing probes if present will be unregistered
	if notFound {
		fmt.Println("Cluster not found")
		logger.V(4).Info("Cluster not found, any existing probes will be removed if present", "namespace", req.Namespace, "name", req.Name)
		r.ProberMgr.Unregister(req.Name)
		return ctrl.Result{}, nil
	}

	// If cluster is marked for deletion then any existing probes will be unregistered
	if cluster.DeletionTimestamp != nil {
		fmt.Println("deletion time stamp is set")
		logger.V(4).Info("Cluster has been marked for deletion, any existing probes will be removed if present", "namespace", req.Namespace, "name", req.Name)
		r.ProberMgr.Unregister(req.Name)
		return ctrl.Result{}, nil
	}

	shoot, err := extensionscontroller.ShootFromCluster(cluster)
	if err != nil {
		fmt.Println("Error extracting shoot from cluster")
		logger.Error(err, "Error extracting shoot from cluster.", "namespace", req.Namespace, "name", req.Name)
		return ctrl.Result{}, err
	}

	// if hibernation is enabled then we will remove any existing prober
	if gardencorev1beta1helper.HibernationIsEnabled(shoot) {
		fmt.Println("Hibernation enabled")
		logger.V(4).Info("Cluster hibernation is enabled, prober will be removed if present", "namespace", req.Namespace, "name", req.Name)
		r.ProberMgr.Unregister(req.Name)
	} else {
		if shoot.Status.IsHibernated {
			fmt.Println("shoot is hibernated")
			logger.V(4).Info("Cluster is waking up and is not yet ready, it is too early to start probing for this shoot. Any existing probes will be removed if present", "namespace", req.Namespace, "name", req.Name)
			r.ProberMgr.Unregister(req.Name)
		} else {
			fmt.Println("new shoot encountered, create corresponding prober")
			logger.V(4).Info("Starting a new probe for cluster if not present", "namespace", req.Namespace, "name", req.Name)
			r.startProber(req.Name)
		}
	}
	return ctrl.Result{}, nil
}

// getCluster will retrieve the cluster object given the namespace and name Not found is not treated as an error and is handled differently in the caller
func (r *ClusterReconciler) getCluster(ctx context.Context, namespace string, name string) (cluster *gardenerv1alpha1.Cluster, notFound bool, err error) {
	cluster = &gardenerv1alpha1.Cluster{}
	if err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, cluster); err != nil {
		if errors.IsNotFound(err) {
			return nil, true, nil
		}
		return nil, false, err
	}
	return cluster, false, nil
}

// startProber sets up a new probe against a given key which uniquely identifies the probe.
// Typically, the key in case of a shoot cluster is the shoot namespace
func (r *ClusterReconciler) startProber(key string) {
	_, ok := r.ProberMgr.GetProber(key)
	if !ok {
		deploymentScaler := prober.NewDeploymentScaler(key, r.ProbeConfig, r.Client, r.ScaleGetter)
		shootClientCreator := prober.NewShootClientCreator(r.Client)
		p := prober.NewProber(key, r.ProbeConfig, r.Client, deploymentScaler, shootClientCreator)
		r.ProberMgr.Register(*p)
		go p.Run()
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gardenerv1alpha1.Cluster{}).
		Complete(r)
}
