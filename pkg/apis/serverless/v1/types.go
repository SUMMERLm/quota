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

// Quota is a specification for a Serverless Quotaresource
type Quota struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec QuotaSpec `json:"spec"`
}

// QuotaSpec is the spec for a Foo resource
type QuotaSpec struct {
	SupervisorName string `json:"supervisorName,omitempty"`
	LocalName string `json:"localName,omitempty"`
	NetworkRegister map[string]string `json:"networkRegister,omitempty"`
	ChildName []string `json:"childName,omitempty"`
	ChildAlert map[string]bool `json:"childAlert,omitempty"`
	ClusterAreaType string `json:"clusterAreaType,omitempty"`
	PodQpsQuota map[string]int `json:"podQpsQuota,omitempty"`
	PodQpsReal map[string]int `json:"podQpsReal,omitempty"`
	PodQpsIncreaseOrDecrease map[string]int `json:"podQpsIncreaseOrDecrease,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// QuotaList is a list of Foo resources
type QuotaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Quota `json:"items"`
}
