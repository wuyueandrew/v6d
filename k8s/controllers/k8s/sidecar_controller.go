/** Copyright 2020-2023 Alibaba Group Holding Limited.

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

// Package k8s contains k8s API versions.
package k8s

import (
	"context"
	"strings"
	"time"

	"github.com/pkg/errors"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/apache/skywalking-swck/operator/pkg/kubernetes"

	k8sv1alpha1 "github.com/v6d-io/v6d/k8s/apis/k8s/v1alpha1"
	"github.com/v6d-io/v6d/k8s/pkg/log"
	"github.com/v6d-io/v6d/k8s/pkg/templates"
)

// SidecarEtcd contains the configuration about etcd of sidecar container
var SidecarEtcd EtcdConfig

// getSidecarEtcdConfig get etcd configuratiin from Etcd
func getSidecarEtcdConfig() EtcdConfig {
	return SidecarEtcd
}

// SidecarSvcLabelSelector contains the label selector of sidecar service
var SidecarSvcLabelSelector []ServiceLabelSelector

func getSidecarSvcLabelSelector() []ServiceLabelSelector {
	return SidecarSvcLabelSelector
}

// SidecarReconciler reconciles a Sidecar object
type SidecarReconciler struct {
	client.Client
	record.EventRecorder
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=k8s.v6d.io,resources=sidecars,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=k8s.v6d.io,resources=sidecars/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=k8s.v6d.io,resources=sidecars/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;create;delete
// +kubebuilder:rbac:groups="",resources=services,verbs=get;create;update;delete

// Reconcile the sidecar.
func (r *SidecarReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithName("controllers").WithName("Sidecar")

	sidecar := &k8sv1alpha1.Sidecar{}
	if err := r.Get(ctx, req.NamespacedName, sidecar); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	logger.V(1).Info("Reconciling Sidecar", "sidecar", *sidecar)

	// deploy the sidecar service
	sidecarApp := kubernetes.Application{
		Client:   r.Client,
		FileRepo: templates.Repo,
		CR:       sidecar,
		GVK:      k8sv1alpha1.GroupVersion.WithKind("Sidecar"),
		Recorder: r.EventRecorder,
		TmplFunc: map[string]interface{}{
			"getEtcdConfig":           getSidecarEtcdConfig,
			"getServiceLabelSelector": getSidecarSvcLabelSelector,
		},
	}
	// setup the etcd configuration
	replicas := sidecar.Spec.Replicas
	SidecarEtcd = BuildEtcdConfig(sidecar.Namespace, replicas, sidecar.Spec.VineyardConfig.Image)

	for i := 0; i < replicas; i++ {
		SidecarEtcd.Rank = i
		if _, err := sidecarApp.Apply(ctx, "etcd/etcd.yaml", logger, false); err != nil {
			logger.Error(err, "failed to apply etcd pod")
			return ctrl.Result{}, err
		}
		if _, err := sidecarApp.Apply(ctx, "etcd/service.yaml", logger, false); err != nil {
			logger.Error(err, "failed to apply etcd service")
			return ctrl.Result{}, err
		}
	}

	s := strings.Split(sidecar.Spec.Service.Selector, "=")

	SidecarSvcLabelSelector = make([]ServiceLabelSelector, 1)
	SidecarSvcLabelSelector[0].Key = s[0]
	SidecarSvcLabelSelector[0].Value = s[1]
	if _, err := sidecarApp.Apply(ctx, "vineyardd/etcd-service.yaml", logger, true); err != nil {
		logger.Error(err, "failed to apply etcd service")
		return ctrl.Result{}, err
	}
	if _, err := sidecarApp.Apply(ctx, "vineyardd/service.yaml", logger, true); err != nil {
		logger.Error(err, "failed to apply vineyard rpc service")
		return ctrl.Result{}, err
	}

	if err := r.UpdateStatus(ctx, sidecar); err != nil {
		logger.Error(err, "failed to update sidecar status")
		return ctrl.Result{}, err
	}

	// reconcile every minute
	duration, _ := time.ParseDuration("1m")
	return ctrl.Result{RequeueAfter: duration}, nil
}

// UpdateStatus updates the status of the Sidecar.
func (r *SidecarReconciler) UpdateStatus(ctx context.Context, sidecar *k8sv1alpha1.Sidecar) error {
	podList := &v1.PodList{}
	s := strings.Split(sidecar.Spec.Selector, "=")
	current := 0
	opts := []client.ListOption{
		client.MatchingLabels{
			s[0]: s[1],
		},
	}
	if err := r.List(ctx, podList, opts...); err != nil {
		log.V(1).Error(err, "failed to list pod")
		return err
	}

	for i := range podList.Items {
		pod := podList.Items[i]
		for _, c := range pod.Spec.Containers {
			if c.Name == "vineyard-sidecar" && pod.Status.Phase == v1.PodRunning {
				current++
				break
			}
		}
	}

	// get the injected vineyardd
	status := &k8sv1alpha1.SidecarStatus{
		Current: int32(current),
	}
	if err := ApplyStatueUpdate(ctx, r.Client, sidecar, r.Status(),
		func(sidecar *k8sv1alpha1.Sidecar) (error, *k8sv1alpha1.Sidecar) {
			sidecar.Status = *status
			sidecar.Kind = "Sidecar"
			if err := kubernetes.ApplyOverlay(sidecar, &k8sv1alpha1.Sidecar{Status: *status}); err != nil {
				return errors.Wrap(err, "failed to overlay sidecar's status"), nil
			}
			return nil, sidecar
		},
	); err != nil {
		return errors.Wrap(err, "failed to update status")
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SidecarReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8sv1alpha1.Sidecar{}).
		Complete(r)
}
