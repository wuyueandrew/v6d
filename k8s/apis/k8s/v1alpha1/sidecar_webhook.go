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

// Package v1alpha1 contains API Schema definitions for the k8s v1alpha1 API group
package v1alpha1

import (
	"github.com/v6d-io/v6d/k8s/pkg/log"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var sidecarlog = log.WithName("sidecar-resource")

// SetupWebhookWithManager sets up the webhook with the manager
func (r *Sidecar) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// nolint: lll
//+kubebuilder:webhook:path=/mutate-k8s-v6d-io-v1alpha1-sidecar,mutating=true,failurePolicy=fail,sideEffects=None,groups=k8s.v6d.io,resources=sidecars,verbs=create;update,versions=v1alpha1,name=msidecar.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &Sidecar{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *Sidecar) Default() {
	sidecarlog.Info("default", "name", r.Name)
}

// nolint: lll
//+kubebuilder:webhook:path=/validate-k8s-v6d-io-v1alpha1-sidecar,mutating=false,failurePolicy=fail,sideEffects=None,groups=k8s.v6d.io,resources=sidecars,verbs=create;update,versions=v1alpha1,name=vsidecar.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &Sidecar{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Sidecar) ValidateCreate() error {
	sidecarlog.Info("validate create", "name", r.Name)

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Sidecar) ValidateUpdate(old runtime.Object) error {
	sidecarlog.Info("validate update", "name", r.Name)

	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Sidecar) ValidateDelete() error {
	sidecarlog.Info("validate delete", "name", r.Name)

	return nil
}
