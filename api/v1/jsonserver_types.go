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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type SyncState string

const (
	SyncStateNotSynced = "NotSynced"
	SyncStateSynced    = "Synced"
	SyncStateError     = "Error"
)

// EDIT THIS FILE! THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// JsonServerSpec defines the desired state of JsonServer
type JsonServerSpec struct {
	// Number of replicas
	Replicas *int32 `json:"replicas,omitempty"`
	// valid json
	JsonConfig string `json:"jsonConfig"`
}

// JsonServerStatus defines the observed state of JsonServer
type JsonServerStatus struct {
	SyncState   SyncState `json:"state"`
	SyncMessage string    `json:"message,omitempty"`
	Replicas    int32     `json:"replicas"`
	Selector    string    `json:"selector,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:subresource:scale:specpath=.spec.replicas,statuspath=.status.replicas,selectorpath=.status.selector
// JsonServer is the Schema for the jsonservers API
type JsonServer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   JsonServerSpec   `json:"spec,omitempty"`
	Status JsonServerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// JsonServerList contains a list of JsonServer
type JsonServerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []JsonServer `json:"items"`
}

func init() {
	SchemeBuilder.Register(&JsonServer{}, &JsonServerList{})
}
