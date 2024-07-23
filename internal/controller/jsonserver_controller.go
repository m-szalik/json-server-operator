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
	"fmt"
	examplecomv1 "github.com/m-szalik/json-server-operator/api/v1"
	"github.com/pkg/errors"
	v1 "k8s.io/api/apps/v1"
	corevV1 "k8s.io/api/core/v1"
	k8errors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"strings"
	"time"
)

// JsonServerReconciler reconciles a JsonServer object
type JsonServerReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=example.com,resources=jsonservers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=example.com,resources=jsonservers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=example.com,resources=jsonservers/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// the JsonServer object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.15.0/pkg/reconcile
func (r *JsonServerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (rr ctrl.Result, rErr error) {
	logger := log.FromContext(ctx)
	jsonServerResource := &examplecomv1.JsonServer{}
	err := r.Get(ctx, req.NamespacedName, jsonServerResource)
	if k8errors.IsNotFound(err) {
		logger.Info("resource " + req.Namespace + "@" + req.Name + " deleted, sub resources will be removed automatically")
		return ctrl.Result{}, nil
	}
	if err != nil {
		return ctrl.Result{}, errors.Wrapf(err, "cannot get resource %s", req)
	}
	fixActions, criticalErrors, err := r.validateResources(ctx, jsonServerResource)
	defer func() {
		if rErr != nil {
			criticalErrors = append(criticalErrors, rErr.Error())
		}
		refreshRequested, err := r.updateStatus(ctx, jsonServerResource, criticalErrors, len(fixActions) > 0)
		if err != nil {
			rr = ctrl.Result{Requeue: true}
			rErr = errors.Wrapf(err, "cannot update status")
		} else {
			if refreshRequested {
				rr = ctrl.Result{RequeueAfter: 15 * time.Second}
			}
		}
	}()
	if err != nil {
		return ctrl.Result{}, errors.Wrapf(err, "cannot validate current status")
	}
	err = validateJson(jsonServerResource.Spec.JsonConfig) // An extra check. This json is validated also via webHook
	if err != nil {
		return ctrl.Result{}, errors.Wrapf(err, "not valid jsonConfig")
	}
	// apply fix actions to bring current state to desired state.
	for _, fixAction := range fixActions {
		err = fixAction.Fix(ctx, r)
		if err != nil {
			logger.Error(err, "Action for: "+fixAction.String()+" error "+err.Error())
			criticalErrors = append(criticalErrors, fmt.Sprintf("internal - %s: %s", fixAction.String(), err.Error()))
		} else {
			logger.Info("Action for: " + fixAction.String() + " scheduled")
		}
		r.emmitEvent(jsonServerResource, fixAction, err)
	}
	if len(fixActions) > 0 {
		// actions have been taken, check the status again in a few seconds
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	} else {
		return ctrl.Result{}, nil
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *JsonServerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&examplecomv1.JsonServer{}).
		Owns(&v1.Deployment{}).
		Owns(&corevV1.ConfigMap{}).
		Owns(&corevV1.Service{}).
		Complete(r)
}

func (r *JsonServerReconciler) validateResources(ctx context.Context, jsonServer *examplecomv1.JsonServer) ([]FixAction, []string, error) {
	fixActions := make([]FixAction, 0)
	criticalErrors := make([]string, 0)
	resourceObjectsFunc := []func(*examplecomv1.JsonServer) client.Object{
		createJsonServerConfigMapResource,
		createJsonServerDeploymentResource,
		createJsonServerServiceResource,
	}
	for _, resourceObjectFactoryFunc := range resourceObjectsFunc {
		to := resourceObjectFactoryFunc(jsonServer)
		err := r.Get(ctx, client.ObjectKeyFromObject(jsonServer), to)
		desired := resourceObjectFactoryFunc(jsonServer)
		if k8errors.IsNotFound(err) {
			fixActions = append(fixActions, CreateResourceFixAction(jsonServer, desired))
		} else {
			if diffs := findResourceDifferences(desired, to); len(diffs) > 0 {
				fixActions = append(fixActions, UpdateResourceFixAction(jsonServer, desired, fmt.Sprintf("differences: [%s]", strings.Join(diffs, ", "))))
			}
		}
	}
	return fixActions, criticalErrors, nil
}

func (r *JsonServerReconciler) emmitEvent(jsonServer *examplecomv1.JsonServer, action FixAction, err error) {
	eventType := "Normal"
	message := action.String()
	if err != nil {
		eventType = "Warning"
		message = fmt.Sprintf("%s -> %s", message, err.Error())
	}
	r.Recorder.Event(jsonServer, eventType, action.Reason(), message)
}

func (r *JsonServerReconciler) getRunningPods(ctx context.Context, jsonServer *examplecomv1.JsonServer) (int, error) {
	deployment := &v1.Deployment{}
	err := r.Get(ctx, client.ObjectKeyFromObject(jsonServer), deployment)
	if err != nil {
		return 0, err
	}
	return int(deployment.Status.AvailableReplicas), nil
}

func findResourceDifferences(desired client.Object, current client.Object) []string {
	diffs := make([]string, 0)
	switch do := desired.(type) {
	case *corevV1.ConfigMap:
		co := current.(*corevV1.ConfigMap)
		if len(co.Data) != len(do.Data) {
			diffs = append(diffs, "data field len")
		}
		for key, doVal := range do.Data {
			coVal, ok := co.Data[key]
			if !ok {
				diffs = append(diffs, "missing field "+key)
			} else {
				if coVal != doVal {
					diffs = append(diffs, "data field "+key+" changed")
				}
			}
		}
	case *v1.Deployment:
		co := current.(*v1.Deployment)
		if do.Spec.Replicas != nil && *co.Spec.Replicas != *do.Spec.Replicas {
			diffs = append(diffs, "replicas")
		}
	}
	return diffs
}

func (r *JsonServerReconciler) updateStatus(ctx context.Context, jsonServerResource *examplecomv1.JsonServer, criticalErrors []string, fixActionExecuted bool) (bool, error) {
	refreshRequired := false
	logger := log.FromContext(ctx)
	status := examplecomv1.JsonServerStatus{SyncState: examplecomv1.SyncStateSynced, SyncMessage: "Synced successfully!"}
	if len(criticalErrors) > 0 {
		status = examplecomv1.JsonServerStatus{SyncState: examplecomv1.SyncStateError, SyncMessage: strings.Join(criticalErrors, "; ")}
	}
	if fixActionExecuted {
		status = examplecomv1.JsonServerStatus{SyncState: examplecomv1.SyncStateNotSynced, SyncMessage: "Updating"}
	}
	if runningPods, err := r.getRunningPods(ctx, jsonServerResource); err != nil {
		logger.Error(err, "cannot check running pods")
	} else {
		if jsonServerResource.Spec.Replicas != nil && runningPods != int(*jsonServerResource.Spec.Replicas) {
			status.SyncState = examplecomv1.SyncStateNotSynced
			status.SyncMessage = fmt.Sprintf("AvailableReplicas %d of %d", runningPods, *jsonServerResource.Spec.Replicas)
		}
		status.Replicas = int32(runningPods)
		refreshRequired = true
	}
	logger.Info("updating status of " + jsonServerResource.Namespace + "@" + jsonServerResource.Name)
	jsonServerResource.Status = status
	err := r.Status().Update(ctx, jsonServerResource)
	if err != nil && !k8errors.IsNotFound(err) {
		return true, errors.Wrapf(err, "cannot update status of %v", jsonServerResource)
	}
	return refreshRequired, nil
}
