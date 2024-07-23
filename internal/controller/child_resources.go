package controller

import (
	"fmt"
	examplecomv1 "github.com/m-szalik/json-server-operator/api/v1"
	v1 "k8s.io/api/apps/v1"
	corevV1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	configMapField = "db.json"
	md5sumLabel    = "md5sum"
	port           = 3000
)

func createOwnerReferences(jsonServer *examplecomv1.JsonServer, blockOwnerDeletion bool) []metav1.OwnerReference {
	ownerReferences := make([]metav1.OwnerReference, 0)
	ownerRef := metav1.OwnerReference{
		APIVersion:         jsonServer.APIVersion,
		Kind:               jsonServer.Kind,
		Name:               jsonServer.Name,
		UID:                jsonServer.UID,
		BlockOwnerDeletion: &blockOwnerDeletion,
	}
	ownerReferences = append(ownerReferences, ownerRef)
	return ownerReferences
}

func createJsonServerConfigMapResource(jsonServer *examplecomv1.JsonServer) client.Object {
	jsonContent := jsonServer.Spec.JsonConfig
	return &corevV1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:            jsonServer.Name,
			Namespace:       jsonServer.Namespace,
			OwnerReferences: createOwnerReferences(jsonServer, false),
			Labels: map[string]string{
				md5sumLabel: md5hash(jsonContent),
			},
		},
		Data: map[string]string{
			configMapField: jsonContent,
		},
	}
}

func createJsonServerDeploymentResource(jsonServer *examplecomv1.JsonServer) client.Object {
	labels := map[string]string{"app": jsonServer.Name}
	for key, val := range jsonServer.Labels {
		labels[key] = val
	}
	deployment := &v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:            jsonServer.Name,
			Namespace:       jsonServer.Namespace,
			OwnerReferences: createOwnerReferences(jsonServer, false),
		},
		Spec: v1.DeploymentSpec{
			Replicas: jsonServer.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corevV1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corevV1.PodSpec{
					Containers: []corevV1.Container{{
						Image: "backplane/json-server",
						Name:  "json-server",
						Args:  []string{fmt.Sprintf("/data/%s", configMapField)},
						Ports: []corevV1.ContainerPort{{
							Name:          "http",
							ContainerPort: port,
							Protocol:      "TCP",
						}},
						VolumeMounts: []corevV1.VolumeMount{{
							Name:      "json-config",
							ReadOnly:  true,
							MountPath: "/data",
						}},
					}},
					Volumes: []corevV1.Volume{{
						Name: "json-config",
						VolumeSource: corevV1.VolumeSource{
							ConfigMap: &corevV1.ConfigMapVolumeSource{
								LocalObjectReference: corevV1.LocalObjectReference{Name: jsonServer.Name},
							},
						},
					}},
				},
			},
		},
	}
	return deployment
}

func createJsonServerServiceResource(jsonServer *examplecomv1.JsonServer) client.Object {
	service := &corevV1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:            jsonServer.Name,
			Namespace:       jsonServer.Namespace,
			OwnerReferences: createOwnerReferences(jsonServer, false),
		},
		Spec: corevV1.ServiceSpec{
			Ports: []corevV1.ServicePort{{
				Name:     "http",
				Protocol: "TCP",
				Port:     port,
				TargetPort: intstr.IntOrString{
					Type:   0,
					IntVal: port,
					StrVal: "http",
				},
			}},
			Selector: map[string]string{
				"app": jsonServer.Name,
			},
		},
	}
	return service
}
