/*
Copyright 2025.

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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// LogStreamConfig defines the configuration for log streaming
type LogStreamConfig struct {
	// Namespace is the Kubernetes namespace to stream logs from
	Namespace string `json:"namespace"`

	// PodName is the name of the pod to stream logs from
	PodName string `json:"podName"`

	// ContainerName is the name of the container within the pod
	ContainerName string `json:"containerName"`

	// Follow indicates whether to follow the log stream for new entries
	// +optional
	Follow *bool `json:"follow,omitempty"`

	// IncludeTimestamps indicates whether to include timestamps in log entries
	// +optional
	IncludeTimestamps *bool `json:"includeTimestamps,omitempty"`

	// TailLines indicates how many lines from the end of the logs to show
	// +optional
	TailLines *int64 `json:"tailLines,omitempty"`
}

// KafkaConfig defines the configuration for Kafka output
type KafkaConfig struct {
	// Brokers is a list of Kafka broker addresses
	Brokers []string `json:"brokers"`

	// Topic is the Kafka topic to send logs to
	Topic string `json:"topic"`

	// Enabled indicates whether Kafka output is enabled
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
}

// NetworkingTroubleshooterSpec defines the desired state of NetworkingTroubleshooter.
type NetworkingTroubleshooterSpec struct {
	// LogStream defines the configuration for log streaming
	LogStream LogStreamConfig `json:"logStream"`

	// Kafka defines the configuration for Kafka output (optional)
	// +optional
	Kafka *KafkaConfig `json:"kafka,omitempty"`

	// Enabled indicates whether the troubleshooter is enabled
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
}

// NetworkingTroubleshooterStatus defines the observed state of NetworkingTroubleshooter.
type NetworkingTroubleshooterStatus struct {
	// Phase represents the current phase of the troubleshooter
	Phase string `json:"phase,omitempty"`

	// Message provides additional information about the current state
	Message string `json:"message,omitempty"`

	// LastStreamTime is the timestamp of the last successful log stream
	LastStreamTime *metav1.Time `json:"lastStreamTime,omitempty"`

	// StreamingActive indicates whether log streaming is currently active
	StreamingActive bool `json:"streamingActive"`

	// Conditions represent the latest available observations of the troubleshooter's state
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// NetworkingTroubleshooter is the Schema for the networkingtroubleshooters API.
type NetworkingTroubleshooter struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NetworkingTroubleshooterSpec   `json:"spec,omitempty"`
	Status NetworkingTroubleshooterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NetworkingTroubleshooterList contains a list of NetworkingTroubleshooter.
type NetworkingTroubleshooterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NetworkingTroubleshooter `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NetworkingTroubleshooter{}, &NetworkingTroubleshooterList{})
}
