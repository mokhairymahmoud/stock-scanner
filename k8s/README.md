# Kubernetes Deployment Manifests

This directory contains Kubernetes manifests for deploying the Stock Scanner application.

## Structure

```
k8s/
├── base/                    # Base Kubernetes manifests
│   ├── namespace.yaml       # Namespace definition
│   ├── configmap.yaml       # Configuration values
│   ├── secrets.yaml.example # Secrets template (copy and customize)
│   ├── *-deployment.yaml    # Deployment manifests for each service
│   ├── services.yaml        # Service definitions
│   ├── hpa.yaml            # Horizontal Pod Autoscalers
│   ├── ingress.yaml        # Ingress configuration
│   └── kustomization.yaml  # Kustomize configuration
└── README.md               # This file
```

## Prerequisites

1. **Kubernetes cluster** (v1.24+)
2. **kubectl** configured to access your cluster
3. **kustomize** (optional, for advanced customization)
4. **Docker image** built and pushed to a registry accessible by your cluster

## Quick Start

### 1. Build and Push Docker Image

```bash
# Build the image
docker build -t stock-scanner:latest .

# Tag for your registry (replace with your registry)
docker tag stock-scanner:latest your-registry/stock-scanner:latest

# Push to registry
docker push your-registry/stock-scanner:latest
```

### 2. Create Secrets

```bash
# Copy the example secrets file
cp k8s/base/secrets.yaml.example k8s/base/secrets.yaml

# Edit secrets.yaml with your actual values
# IMPORTANT: Never commit secrets.yaml to version control!

# Create the secret in Kubernetes
kubectl create namespace stock-scanner
kubectl apply -f k8s/base/secrets.yaml
```

### 3. Update Image Reference

If your image is in a different registry, update the image references in the deployment files:

```bash
# Option 1: Use sed to replace image references
find k8s/base -name "*-deployment.yaml" -exec sed -i '' 's|stock-scanner:latest|your-registry/stock-scanner:latest|g' {} \;

# Option 2: Use kustomize (recommended)
# Create k8s/overlays/production/kustomization.yaml with image transformers
```

### 4. Deploy

```bash
# Apply all manifests
kubectl apply -k k8s/base

# Or apply individually
kubectl apply -f k8s/base/namespace.yaml
kubectl apply -f k8s/base/configmap.yaml
kubectl apply -f k8s/base/secrets.yaml
kubectl apply -f k8s/base/
```

### 5. Verify Deployment

```bash
# Check all pods are running
kubectl get pods -n stock-scanner

# Check services
kubectl get svc -n stock-scanner

# Check deployments
kubectl get deployments -n stock-scanner

# View logs for a service
kubectl logs -f deployment/ingest -n stock-scanner
```

## Services

The application consists of 7 microservices:

1. **ingest** - Market data ingestion service
2. **bars** - Bar aggregation service
3. **indicator** - Technical indicator computation service
4. **scanner** - Rule scanning service (scales horizontally)
5. **alert** - Alert processing service (scales horizontally)
6. **ws-gateway** - WebSocket gateway for real-time alerts (scales horizontally)
7. **api** - REST API service (scales horizontally)

## Configuration

### ConfigMap

Most configuration is stored in the `stock-scanner-config` ConfigMap. To update:

```bash
# Edit the configmap
kubectl edit configmap stock-scanner-config -n stock-scanner

# Or update from file
kubectl apply -f k8s/base/configmap.yaml

# Restart pods to pick up changes
kubectl rollout restart deployment -n stock-scanner
```

### Secrets

Sensitive data (passwords, API keys, JWT secrets) is stored in the `stock-scanner-secrets` Secret.

**Important:** Never commit actual secrets to version control. Use the `.example` file as a template.

## Scaling

### Horizontal Pod Autoscaling (HPA)

The following services have HPA configured:

- **scanner**: 3-10 replicas (CPU: 70%, Memory: 80%)
- **alert**: 2-5 replicas (CPU: 70%, Memory: 80%)
- **ws-gateway**: 2-10 replicas (CPU: 70%, Memory: 80%)
- **api**: 2-10 replicas (CPU: 70%, Memory: 80%)

To manually scale:

```bash
kubectl scale deployment scanner --replicas=5 -n stock-scanner
```

## Ingress

The Ingress resource exposes:
- **API**: `api.stock-scanner.local` → api service
- **WebSocket**: `ws.stock-scanner.local` → ws-gateway service

### Setup Ingress Controller

If you don't have an ingress controller:

```bash
# For NGINX Ingress Controller
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.8.1/deploy/static/provider/cloud/deploy.yaml

# For local development, add to /etc/hosts:
# 127.0.0.1 api.stock-scanner.local
# 127.0.0.1 ws.stock-scanner.local
```

### TLS/HTTPS

To enable TLS, uncomment and configure the TLS section in `ingress.yaml`:

1. Create a TLS secret:
```bash
kubectl create secret tls stock-scanner-tls \
  --cert=path/to/cert.pem \
  --key=path/to/key.pem \
  -n stock-scanner
```

2. Update `ingress.yaml` to reference the secret

## Dependencies

The application requires:

1. **Redis** - For pub/sub and streams
2. **TimescaleDB** - For time-series data storage

These can be deployed separately or using Helm charts:

```bash
# Redis (using Bitnami Helm chart)
helm repo add bitnami https://charts.bitnami.com/bitnami
helm install redis bitnami/redis -n stock-scanner

# TimescaleDB (using TimescaleDB Helm chart)
helm repo add timescaledb https://charts.timescale.com
helm install timescaledb timescaledb/timescaledb-single -n stock-scanner
```

Or use the existing docker-compose setup for local development.

## Monitoring

### Prometheus

The services expose metrics on their health ports. Configure Prometheus to scrape:

- `ingest:8081/metrics`
- `bars:8083/metrics`
- `indicator:8085/metrics`
- `scanner:8087/metrics`
- `alert:8093/metrics`
- `ws-gateway:8089/metrics`
- `api:8091/metrics`

### Health Checks

All services expose health endpoints:
- Liveness: `/health`
- Readiness: `/health`
- Metrics: `/metrics`

## Troubleshooting

### Pods Not Starting

```bash
# Check pod status
kubectl describe pod <pod-name> -n stock-scanner

# Check logs
kubectl logs <pod-name> -n stock-scanner

# Check events
kubectl get events -n stock-scanner --sort-by='.lastTimestamp'
```

### Common Issues

1. **Image pull errors**: Ensure image is accessible from cluster
2. **ConfigMap/Secret not found**: Ensure namespace exists and resources are created
3. **Database connection errors**: Check DB_HOST and DB_PASSWORD in secrets
4. **Redis connection errors**: Check REDIS_HOST and REDIS_PASSWORD in secrets

### Port Forwarding (for local testing)

```bash
# Forward API service
kubectl port-forward svc/api 8090:8090 -n stock-scanner

# Forward WebSocket gateway
kubectl port-forward svc/ws-gateway 8088:8088 -n stock-scanner
```

## Production Considerations

1. **Resource Limits**: Adjust CPU/memory limits based on your workload
2. **Replica Counts**: Start with minimum replicas and scale based on metrics
3. **Secrets Management**: Use a secrets management system (Vault, Sealed Secrets, etc.)
4. **Image Tags**: Use specific version tags instead of `latest`
5. **Network Policies**: Implement network policies for security
6. **Pod Disruption Budgets**: Add PDBs for high availability
7. **Backup Strategy**: Implement backups for TimescaleDB
8. **Monitoring**: Set up comprehensive monitoring and alerting
9. **Logging**: Configure centralized logging (Loki, ELK, etc.)
10. **Tracing**: Set up distributed tracing (Jaeger, Zipkin, etc.)

## Cleanup

To remove all resources:

```bash
kubectl delete namespace stock-scanner
```

Or delete individual resources:

```bash
kubectl delete -k k8s/base
```

