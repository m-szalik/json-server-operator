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

package v1

import (
	"fmt"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"strings"
)

const requiredPrefix = "app-"

// log is for logging in this package.
var jsonserverlog = logf.Log.WithName("jsonserver-resource")

func (r *JsonServer) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-example-com-v1-jsonserver,mutating=true,failurePolicy=fail,sideEffects=None,groups=example.com,resources=jsonservers,verbs=create;update,versions=v1,name=mjsonserver.kb.io,admissionReviewVersions=v1
// +kubebuilder:webhook:path=/validate-example-com-v1-jsonserver,mutating=false,failurePolicy=fail,sideEffects=None,groups=example.com,resources=jsonservers,verbs=create;update,versions=v1,name=vjsonserver.kb.io,admissionReviewVersions=v1
var defaultReplicas int32 = 2
var _ webhook.Defaulter = &JsonServer{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *JsonServer) Default() {
	jsonserverlog.Info("default", "name", r.Name)
	if r.Spec.Replicas == nil {
		r.Spec.Replicas = &defaultReplicas
	}
}

// +kubebuilder:webhook:path=/validate-example-com-v1-jsonserver,mutating=false,failurePolicy=fail,sideEffects=None,groups=example.com,resources=jsonservers,verbs=create;update,versions=v1,name=vjsonserver.kb.io,admissionReviewVersions=v1
var _ webhook.Validator = &JsonServer{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *JsonServer) ValidateCreate() (admission.Warnings, error) {
	jsonserverlog.Info("webhook validate create", "name", r.Name)
	return r.validate()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *JsonServer) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	jsonserverlog.Info("webhook validate update", "name", r.Name)
	return r.validate()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *JsonServer) ValidateDelete() (admission.Warnings, error) {
	// nothing here
	return nil, nil
}
func (r *JsonServer) validate() (warnings admission.Warnings, err error) {
	warnings = admission.Warnings{}
	validationErrors := make([]string, 0)
	if r.Spec.Replicas != nil && *r.Spec.Replicas < 0 {
		validationErrors = append(validationErrors, "replicas must be gather or equal 0")
	}
	if !strings.HasPrefix(r.Name, requiredPrefix) {
		validationErrors = append(validationErrors, fmt.Sprintf("resource name must start with '%s'", requiredPrefix))
	}
	if jsonErr := validateJson(r.Spec.JsonConfig); jsonErr != nil {
		validationErrors = append(validationErrors, fmt.Sprintf("invalid jsonConfig - %s", jsonErr))
	}
	if len(validationErrors) > 0 {
		jsonserverlog.Info("validation issues", "name", r.Name, "issues", strings.Join(validationErrors, ";"))
		return warnings, fmt.Errorf("validation issues: %s", strings.Join(validationErrors, "; "))
	} else {
		jsonserverlog.Info("validation OK", "name", r.Name)
		return warnings, nil
	}
}
