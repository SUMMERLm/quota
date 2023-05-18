/*
Copyright 2017 The Kubernetes Authors.

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

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// QuotaStatus defines the observed state of SubscriberRule
type QuotaStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// Quota is a specification for a Serverless Quotaresource
type Quota struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Status QuotaStatus `json:"status,omitempty"`
	Spec   QuotaSpec   `json:"spec,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// QuotaSpec is the spec for a Foo resource
type QuotaSpec struct {
	// +optional
	SupervisorName string `json:"supervisorName,omitempty"`
	// +optional
	LocalName string `json:"localName,omitempty"`
	// +optional
	// +mapType=atomic
	NetworkRegister map[string]string `json:"networkRegister,omitempty"`
	ChildName       []string          `json:"childName,omitempty"`
	// +optional
	// +mapType=atomic
	ChildAlert      map[string]bool `json:"childAlert,omitempty"`
	ClusterAreaType string          `json:"clusterAreaType,omitempty"`
	// +optional
	// +mapType=atomic
	PodQpsQuota map[string]int `json:"podQpsQuota,omitempty"`
	// +optional
	// +mapType=atomic
	PodQpsReal map[string]int `json:"podQpsReal,omitempty"`
	// +optional
	// +mapType=atomic
	PodQpsIncreaseOrDecrease map[string]int `json:"podQpsIncreaseOrDecrease,omitempty"`
}

//+kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// QuotaList is a list of Quota resources
type QuotaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Quota `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Quota{}, &QuotaList{})
}
