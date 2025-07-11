package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)
// Add some more comments
// main function to run the log streamer example.
func main() {
	// --- Kubernetes Client Setup ---
	// Build Kubernetes config from kubeconfig file.
	// It tries to find the kubeconfig in the user's home directory.
	var kubeconfig string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = filepath.Join(home, ".kube", "config")
	} else {
		log.Fatal("Could not find home directory for kubeconfig.")
	}

	// Use the current context in kubeconfig.
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		log.Fatalf("Error building kubeconfig: %v", err)
	}

	// Create a Kubernetes clientset.
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Error creating Kubernetes clientset: %v", err)
	}

	// --- Log Streaming Parameters ---
	// Define the namespace, pod name, and container name for which to stream logs.
	// You will need to replace these with actual values from your openshift-ovn-kubernetes environment.
	// Example: Find a pod in the 'openshift-ovn-kubernetes' namespace, e.g., 'ovn-master-5vj2s'.
	// Then inspect its containers, e.g., 'ovnkube-master'.
	namespace := "openshift-ovn-kubernetes" // The target namespace
	podName := "ovn-master-5vj2s"           // Replace with an actual pod name from the namespace
	containerName := "ovnkube-master"       // Replace with an actual container name in the pod

	log.Printf("Attempting to stream logs from pod '%s' (container '%s') in namespace '%s'...",
		podName, containerName, namespace)

	// --- Stream Logs ---
	// Create a context with a timeout for the log stream.
	// This ensures the stream doesn't run indefinitely in this example.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel() // Ensure the context is cancelled when main exits.

	// Define log options.
	// 'Follow: true' means it will continuously stream new logs.
	// 'TailLines: &one' could be used to get the last 'n' lines, but 'Follow' is better for streaming.
	// 'Container: containerName' specifies which container's logs to get.
	// 'Timestamps: true' adds timestamps to log lines.
	logOptions := &corev1.PodLogOptions{
		Container: containerName,
		Follow:    true,
		Timestamps: true,
	}

	// Request the log stream from the Kubernetes API.
	// This returns a ReadCloser which can be read like a file.
	req := clientset.CoreV1().Pods(namespace).GetLogs(podName, logOptions)
	podLogs, err := req.Stream(ctx)
	if err != nil {
		log.Fatalf("Error opening log stream: %v", err)
	}
	defer podLogs.Close() // Ensure the log stream is closed.

	// Copy the log stream content to standard output.
	// In a real operator, you would process these lines and send them to Kafka.
	log.Println("--- Streaming Logs ---")
	_, err = io.Copy(os.Stdout, podLogs)
	if err != nil && err != io.EOF { // io.EOF is expected when the stream ends (e.g., context timeout)
		log.Fatalf("Error copying log stream to stdout: %v", err)
	}

	log.Println("--- Log streaming finished. ---")
	log.Println("To use this in your operator, you would replace os.Stdout with your Kafka producer logic.")
	log.Println("Remember to handle pod lifecycle events and error conditions for continuous streaming.")
}

