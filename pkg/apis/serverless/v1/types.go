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

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Quota is the Schema for the quotas API
type Quota struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   QuotaSpec   `json:"spec,omitempty"`
	Status QuotaStatus `json:"status,omitempty"`
}

// QuotaStatus defines the observed state of Quota
type QuotaStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// QuotaSpec defines the desired state of Quota
type QuotaSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Quota is an example field of Quota. Edit quota_types.go to remove/update
	// +optional
	SupervisorName string `json:"supervisorName,omitempty"`
	// +required
	LocalName string `json:"localName,omitempty"`
	// +optional
	NetworkRegister map[string]string `json:"networkRegister,omitempty"`
	// +optional
	ChildName []string `json:"childName,omitempty"`
	// +optional
	ChildAlert map[string]bool `json:"childAlert,omitempty"`
	// +optional
	ClusterAreaType string `json:"clusterAreaType,omitempty"`
	// +optional
	PodQpsQuota map[string]int `json:"podQpsQuota,omitempty"`
	// +optional
	PodQpsReal map[string]int `json:"podQpsReal,omitempty"`
	// +optional
	PodQpsIncreaseOrDecrease map[string]int `json:"podQpsIncreaseOrDecrease,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

//+kubebuilder:object:root=true

// QuotaList contains a list of Quota
type QuotaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Quota `json:"items"`
}
