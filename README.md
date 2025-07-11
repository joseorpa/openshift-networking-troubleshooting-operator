# OpenShift Networking Troubleshooting Operator

A Kubernetes operator for streaming and troubleshooting OpenShift networking component logs. This operator has been built using the operator-sdk framework and provides a declarative way to manage log streaming from OpenShift networking pods.

## Overview

The OpenShift Networking Troubleshooting Operator allows you to:
- Stream logs from OpenShift networking pods (like ovn-kubernetes components)
- Configure log streaming parameters declaratively
- Track streaming status and health
- Prepare for future Kafka integration for log aggregation

## Features

- **Declarative Configuration**: Define log streaming using Kubernetes custom resources
- **Pod/Container Validation**: Automatically validates target pods and containers exist
- **Status Tracking**: Monitor streaming status and health through Kubernetes status conditions
- **Error Handling**: Graceful handling of pod lifecycle events and failures
- **Kafka Ready**: Prepared for future Kafka integration for log aggregation

## Custom Resource Definition

The operator uses the `NetworkingTroubleshooter` custom resource:

```yaml
apiVersion: networking.troubleshooting.openshift.io/v1alpha1
kind: NetworkingTroubleshooter
metadata:
  name: networkingtroubleshooter-sample
spec:
  enabled: true
  logStream:
    namespace: "openshift-ovn-kubernetes"
    podName: "ovn-master-5vj2s"
    containerName: "ovnkube-master"
    follow: true
    includeTimestamps: true
    tailLines: 100
  kafka:
    enabled: false
    brokers:
      - "kafka-broker-1:9092"
      - "kafka-broker-2:9092"
    topic: "openshift-networking-logs"
```

### Spec Fields

- `enabled`: Enable/disable the troubleshooter
- `logStream.namespace`: Kubernetes namespace of the target pod
- `logStream.podName`: Name of the pod to stream logs from
- `logStream.containerName`: Container name within the pod
- `logStream.follow`: Whether to follow log stream for new entries
- `logStream.includeTimestamps`: Include timestamps in log entries
- `logStream.tailLines`: Number of lines from the end of logs to show
- `kafka.enabled`: Enable Kafka output (future feature)
- `kafka.brokers`: List of Kafka broker addresses
- `kafka.topic`: Kafka topic for log streaming

## Getting Started

### Prerequisites

- Go 1.24+
- operator-sdk v1.41+
- Access to a Kubernetes cluster (OpenShift recommended)
- kubectl/oc CLI tools

### Installation

1. **Clone the repository**:
```bash
git clone https://github.com/joseorpa/openshift-networking-troubleshooting-operator.git
cd openshift-networking-troubleshooting-operator
```

2. **Build the operator**:
```bash
make build
```

3. **Install the CRDs**:
```bash
make install
```

4. **Deploy the operator**:
```bash
make deploy
```

### Usage

1. **Create a NetworkingTroubleshooter resource**:
```bash
kubectl apply -f config/samples/networking_v1alpha1_networkingtroubleshooter.yaml
```

2. **Monitor the status**:
```bash
kubectl get networkingtroubleshooter networkingtroubleshooter-sample -o yaml
```

3. **Check operator logs**:
```bash
kubectl logs -f deployment/openshift-networking-troubleshooting-operator-controller-manager -n openshift-networking-troubleshooting-operator-system
```

## Development

### Running Locally

```bash
# Run the operator locally against your configured cluster
make run
```

### Testing

```bash
# Run tests
make test

# Run tests with coverage
make test-coverage
```

### Building Images

```bash
# Build and push the operator image
make docker-build docker-push IMG=<your-registry>/openshift-networking-troubleshooting-operator:tag
```

## Architecture

The operator consists of:

1. **NetworkingTroubleshooter CRD**: Defines the desired state for log streaming
2. **Controller**: Reconciles the actual state with the desired state
3. **Log Streaming Logic**: Handles the actual streaming of logs from pods

### Controller Logic

The controller:
1. Validates the target pod and container exist
2. Starts/stops log streaming based on the spec
3. Updates status conditions to reflect current state
4. Handles pod lifecycle events (creation, deletion, restart)
5. Manages concurrent log streams for multiple resources

## Migration from Standalone Script

This operator evolved from a standalone Go script (`streamer.go`) that performed basic log streaming. The operator provides:

- **Declarative Management**: Configuration through Kubernetes resources
- **Status Tracking**: Real-time status and health monitoring
- **Error Handling**: Robust error handling and recovery
- **Scalability**: Support for multiple concurrent log streams
- **Kubernetes Integration**: Full integration with Kubernetes RBAC and lifecycle

## Troubleshooting

### Common Issues

1. **Pod Not Found**: Ensure the specified pod exists in the target namespace
2. **Container Not Found**: Verify the container name exists in the pod
3. **Permission Errors**: Check RBAC permissions for the operator service account
4. **Resource Not Found**: Ensure CRDs are installed correctly

### Checking Logs

```bash
# Operator logs
kubectl logs -f deployment/openshift-networking-troubleshooting-operator-controller-manager -n openshift-networking-troubleshooting-operator-system

# Resource status
kubectl describe networkingtroubleshooter <resource-name>
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests: `make test`
5. Submit a pull request

## License

Licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE) for details.

## Future Enhancements

- [ ] Kafka integration for log aggregation
- [ ] Multi-pod log streaming
- [ ] Log filtering and transformation
- [ ] Metrics collection and monitoring
- [ ] Alert integration
- [ ] Log storage and retention policies