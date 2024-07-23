package controller

import (
	"fmt"
	examplecomv1 "github.com/m-szalik/json-server-operator/api/v1"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func TestJsonServerReconciler_emmitEvent(t *testing.T) {
	fakeClient := fake.NewClientBuilder().Build()
	fakeRecorder := record.NewFakeRecorder(10)
	jsonServer := &examplecomv1.JsonServer{}
	r := &JsonServerReconciler{
		Client:   fakeClient,
		Recorder: fakeRecorder,
	}
	type args struct {
		action FixAction
		err    error
	}
	tests := []struct {
		name          string
		args          args
		expectedEvent string
	}{
		{name: "action without error", args: args{action: CreateResourceFixAction(jsonServer, createJsonServerServiceResource(jsonServer)), err: nil}, expectedEvent: "Normal Create-Service Service / was missing"},
		{name: "action with error", args: args{action: CreateResourceFixAction(jsonServer, createJsonServerServiceResource(jsonServer)), err: fmt.Errorf("some error")}, expectedEvent: "Warning Create-Service Service / was missing -> some error"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r.emmitEvent(jsonServer, tt.args.action, tt.args.err)
			eventContent := <-fakeRecorder.Events
			assert.Equal(t, tt.expectedEvent, eventContent)
		})
	}
}

func Test_findResourceDifferences(t *testing.T) {
	type args struct {
		desired client.Object
		current client.Object
	}
	tests := []struct {
		args args
		want []string
	}{
		{
			args: args{
				desired: &v1.ConfigMap{Data: map[string]string{"k": "value"}},
				current: &v1.ConfigMap{Data: map[string]string{"k": "value", "q": ""}},
			}, want: []string{"data field len"},
		},
		{
			args: args{
				desired: &v1.ConfigMap{Data: map[string]string{"k": "value"}},
				current: &v1.ConfigMap{Data: map[string]string{"k": "value"}},
			}, want: []string{},
		},
		{
			args: args{
				desired: &v1.ConfigMap{Data: map[string]string{"k": "_VALUE_"}},
				current: &v1.ConfigMap{Data: map[string]string{"k": "value"}},
			}, want: []string{"data field k changed"},
		},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("findResourceDifferences(%v vs %v)", tt.args.current, tt.args.desired), func(t *testing.T) {
			assert.Equalf(t, tt.want, findResourceDifferences(tt.args.desired, tt.args.current), "findResourceDifferences(%v, %v)", tt.args.desired, tt.args.current)
		})
	}
}
