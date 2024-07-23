package controller

import (
	"context"
	"fmt"
	v1 "github.com/m-szalik/json-server-operator/api/v1"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

type FixAction interface {
	Fix(ctx context.Context, r *JsonServerReconciler) error
	Reason() string
	String() string
}

type baseFixAction struct {
	JsonServer *v1.JsonServer
}

type createResourceFixAction struct {
	baseFixAction
	resource client.Object
}

func (c *createResourceFixAction) Reason() string {
	return fmt.Sprintf("Create-%s", objectType(c.resource))
}

func (c *createResourceFixAction) Fix(ctx context.Context, r *JsonServerReconciler) error {
	if err := setControllerReference(c.JsonServer, c.resource, r); err != nil {
		return err
	}
	return r.Create(ctx, c.resource)
}

func (c *createResourceFixAction) String() string {
	return fmt.Sprintf("%s %s was missing", objectType(c.resource), client.ObjectKeyFromObject(c.resource))
}

func CreateResourceFixAction(jsonServer *v1.JsonServer, resource client.Object) FixAction {
	return &createResourceFixAction{
		baseFixAction: baseFixAction{jsonServer},
		resource:      resource,
	}
}

type updateResourceFixAction struct {
	baseFixAction
	resource client.Object
	reason   string
}

func (u *updateResourceFixAction) Reason() string {
	return fmt.Sprintf("Update-%s", objectType(u.resource))
}

func (u *updateResourceFixAction) Fix(ctx context.Context, r *JsonServerReconciler) error {
	if err := setControllerReference(u.JsonServer, u.resource, r); err != nil {
		return err
	}
	return r.Update(ctx, u.resource)
}

func (u *updateResourceFixAction) String() string {
	return fmt.Sprintf("%s %s was out of sync - %s", objectType(u.resource), client.ObjectKeyFromObject(u.resource), u.reason)
}

func UpdateResourceFixAction(jsonServer *v1.JsonServer, resource client.Object, reason string) FixAction {
	return &updateResourceFixAction{
		baseFixAction: baseFixAction{jsonServer},
		resource:      resource,
		reason:        reason,
	}
}

func objectType(o client.Object) string {
	str := fmt.Sprintf("%T", o)
	parts := strings.Split(str, ".")
	return parts[len(parts)-1]
}

func setControllerReference(jsonServer *v1.JsonServer, resource client.Object, r *JsonServerReconciler) error {
	err := ctrl.SetControllerReference(jsonServer, resource, r.Scheme)
	return errors.Wrapf(err, "cannot set ControllerReference for %s owned by %s", objectType(resource), client.ObjectKeyFromObject(jsonServer))
}
