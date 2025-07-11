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

package controller

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	networkingv1alpha1 "github.com/joseorpa/openshift-networking-troubleshooting-operator/api/v1alpha1"
)

// NetworkingTroubleshooterReconciler reconciles a NetworkingTroubleshooter object
type NetworkingTroubleshooterReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	// Kubernetes clientset for log streaming
	Clientset kubernetes.Interface

	// Map to track active log streams
	activeStreams map[string]context.CancelFunc
	streamMutex   sync.RWMutex
}

// +kubebuilder:rbac:groups=networking.troubleshooting.openshift.io,resources=networkingtroubleshooters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.troubleshooting.openshift.io,resources=networkingtroubleshooters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=networking.troubleshooting.openshift.io,resources=networkingtroubleshooters/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=pods/log,verbs=get

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *NetworkingTroubleshooterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// Fetch the NetworkingTroubleshooter instance
	var troubleshooter networkingv1alpha1.NetworkingTroubleshooter
	if err := r.Get(ctx, req.NamespacedName, &troubleshooter); err != nil {
		if errors.IsNotFound(err) {
			// Resource not found, could have been deleted
			log.Info("NetworkingTroubleshooter resource not found. Ignoring since object must be deleted")
			r.stopLogStream(req.NamespacedName.String())
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get NetworkingTroubleshooter")
		return ctrl.Result{}, err
	}

	// Check if the troubleshooter is enabled
	if troubleshooter.Spec.Enabled != nil && !*troubleshooter.Spec.Enabled {
		log.Info("NetworkingTroubleshooter is disabled, stopping log stream")
		r.stopLogStream(req.NamespacedName.String())
		return r.updateStatus(ctx, &troubleshooter, "Disabled", "NetworkingTroubleshooter is disabled", false)
	}

	// Validate the pod exists
	pod := &corev1.Pod{}
	podKey := client.ObjectKey{
		Namespace: troubleshooter.Spec.LogStream.Namespace,
		Name:      troubleshooter.Spec.LogStream.PodName,
	}

	if err := r.Get(ctx, podKey, pod); err != nil {
		if errors.IsNotFound(err) {
			log.Info("Target pod not found", "pod", troubleshooter.Spec.LogStream.PodName, "namespace", troubleshooter.Spec.LogStream.Namespace)
			r.stopLogStream(req.NamespacedName.String())
			return r.updateStatus(ctx, &troubleshooter, "PodNotFound", fmt.Sprintf("Pod %s not found in namespace %s", troubleshooter.Spec.LogStream.PodName, troubleshooter.Spec.LogStream.Namespace), false)
		}
		log.Error(err, "Failed to get pod")
		return ctrl.Result{}, err
	}

	// Check if the container exists in the pod
	containerExists := false
	for _, container := range pod.Spec.Containers {
		if container.Name == troubleshooter.Spec.LogStream.ContainerName {
			containerExists = true
			break
		}
	}

	if !containerExists {
		log.Info("Container not found in pod", "container", troubleshooter.Spec.LogStream.ContainerName, "pod", troubleshooter.Spec.LogStream.PodName)
		r.stopLogStream(req.NamespacedName.String())
		return r.updateStatus(ctx, &troubleshooter, "ContainerNotFound", fmt.Sprintf("Container %s not found in pod %s", troubleshooter.Spec.LogStream.ContainerName, troubleshooter.Spec.LogStream.PodName), false)
	}

	// Start log streaming if not already active
	streamKey := req.NamespacedName.String()
	if !r.isStreamActive(streamKey) {
		log.Info("Starting log stream", "pod", troubleshooter.Spec.LogStream.PodName, "container", troubleshooter.Spec.LogStream.ContainerName)

		// Create a context for the log stream
		streamCtx, cancel := context.WithCancel(context.Background())
		r.setStreamActive(streamKey, cancel)

		// Start the log streaming in a goroutine
		go r.streamLogs(streamCtx, &troubleshooter)

		// Update status to indicate streaming has started
		if _, err := r.updateStatus(ctx, &troubleshooter, "Streaming", "Log streaming is active", true); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Requeue to check status periodically
	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

// streamLogs handles the actual log streaming logic
func (r *NetworkingTroubleshooterReconciler) streamLogs(ctx context.Context, troubleshooter *networkingv1alpha1.NetworkingTroubleshooter) {
	log := logf.FromContext(ctx)

	// Set up log options
	follow := true
	if troubleshooter.Spec.LogStream.Follow != nil {
		follow = *troubleshooter.Spec.LogStream.Follow
	}

	timestamps := true
	if troubleshooter.Spec.LogStream.IncludeTimestamps != nil {
		timestamps = *troubleshooter.Spec.LogStream.IncludeTimestamps
	}

	logOptions := &corev1.PodLogOptions{
		Container:  troubleshooter.Spec.LogStream.ContainerName,
		Follow:     follow,
		Timestamps: timestamps,
	}

	if troubleshooter.Spec.LogStream.TailLines != nil {
		logOptions.TailLines = troubleshooter.Spec.LogStream.TailLines
	}

	// Request the log stream
	req := r.Clientset.CoreV1().Pods(troubleshooter.Spec.LogStream.Namespace).GetLogs(troubleshooter.Spec.LogStream.PodName, logOptions)
	podLogs, err := req.Stream(ctx)
	if err != nil {
		log.Error(err, "Error opening log stream")
		return
	}
	defer podLogs.Close()

	log.Info("--- Streaming Logs ---", "pod", troubleshooter.Spec.LogStream.PodName, "container", troubleshooter.Spec.LogStream.ContainerName)

	// Stream logs to stdout (in a real implementation, this would go to Kafka)
	// TODO: Implement Kafka producer integration here
	_, err = io.Copy(os.Stdout, podLogs)
	if err != nil && err != io.EOF {
		log.Error(err, "Error copying log stream")
	}

	log.Info("Log streaming finished")
}

// updateStatus updates the status of the NetworkingTroubleshooter
func (r *NetworkingTroubleshooterReconciler) updateStatus(ctx context.Context, troubleshooter *networkingv1alpha1.NetworkingTroubleshooter, phase, message string, streaming bool) (ctrl.Result, error) {
	troubleshooter.Status.Phase = phase
	troubleshooter.Status.Message = message
	troubleshooter.Status.StreamingActive = streaming

	if streaming {
		now := metav1.Now()
		troubleshooter.Status.LastStreamTime = &now
	}

	// Update conditions
	condition := metav1.Condition{
		Type:    "Ready",
		Status:  metav1.ConditionTrue,
		Reason:  phase,
		Message: message,
	}

	if !streaming && phase != "Disabled" {
		condition.Status = metav1.ConditionFalse
	}

	// Update or add the condition
	found := false
	for i, cond := range troubleshooter.Status.Conditions {
		if cond.Type == "Ready" {
			troubleshooter.Status.Conditions[i] = condition
			found = true
			break
		}
	}
	if !found {
		troubleshooter.Status.Conditions = append(troubleshooter.Status.Conditions, condition)
	}

	if err := r.Status().Update(ctx, troubleshooter); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// Helper methods for managing active streams
func (r *NetworkingTroubleshooterReconciler) isStreamActive(key string) bool {
	r.streamMutex.RLock()
	defer r.streamMutex.RUnlock()
	_, exists := r.activeStreams[key]
	return exists
}

func (r *NetworkingTroubleshooterReconciler) setStreamActive(key string, cancel context.CancelFunc) {
	r.streamMutex.Lock()
	defer r.streamMutex.Unlock()
	if r.activeStreams == nil {
		r.activeStreams = make(map[string]context.CancelFunc)
	}
	r.activeStreams[key] = cancel
}

func (r *NetworkingTroubleshooterReconciler) stopLogStream(key string) {
	r.streamMutex.Lock()
	defer r.streamMutex.Unlock()
	if cancel, exists := r.activeStreams[key]; exists {
		cancel()
		delete(r.activeStreams, key)
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *NetworkingTroubleshooterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&networkingv1alpha1.NetworkingTroubleshooter{}).
		Named("networkingtroubleshooter").
		Complete(r)
}
