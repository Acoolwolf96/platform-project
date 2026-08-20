# Lightweight Internal Developer Platform

A resource-efficient Internal Developer Platform (IDP) built on Kubernetes, GitOps, Infrastructure as Code, and configuration automation.

Application teams provision and inspect workloads through a developer portal without interacting directly with Kubernetes, Argo CD, Terraform, or Ansible. The platform demonstrates core platform engineering patterns: self-service provisioning, tenant isolation, GitOps reconciliation, resource governance, and developer abstraction.

---

## Overview

The platform serves three application teams :— Billing, Checkout, and Search. Each team gets an isolated Kubernetes namespace with resource quotas, network policies, and Argo CD project boundaries.

Instead of applying Kubernetes manifests directly, the IDP writes the desired application configuration to GitHub. Argo CD detects the change and reconciles it into the cluster.

```text
Developer
   |
   v
Internal Developer Platform
   |
   |  GitHub API
   v
Git Repository
   |
   |  desired state
   v
Argo CD
   |
   v
K3s Cluster
   |
   +-- team-billing
   +-- team-checkout
   +-- team-search
```

### Cluster

- One K3s control-plane node, one K3s agent node
- Not a highly-available control plane, a single-node control plane by design
- Argo CD, Traefik, and Kubernetes Metrics Server run alongside the team workloads and the IDP

---

## Internal Developer Platform (IDP)

The IDP is a custom Go application. It runs as a single container containing the compiled Go backend, an embedded frontend, and a CA certificate bundle, no Node.js runtime, no separate frontend service, no database, no Redis, no persistent volumes.

Resource footprint:

```
CPU request:     25m
CPU limit:       150m
Memory request:  32Mi
Memory limit:    96Mi

Observed usage:
CPU:     ~1m
Memory:  ~4Mi
```

---

## Developer Portal Features

### Service Catalog

The dashboard discovers workloads from Kubernetes automatically and displays service name, team, container image, desired/available replicas, and health status.

### Self-Service Application Provisioning

Developers provision applications through the portal by providing team, service name, container image, container port, and a service size:

```
Team: Billing
Service: payments-api
Image: nginx:alpine
Port: 80
Size: Small
```

The developer does not create Kubernetes Deployments, Services, namespaces, Argo CD Applications, or resource definitions — the platform generates these.

### Provisioning Workflow

```
Developer
   |
   v
POST /api/provision
   |
   v
IDP validates request
   |
   v
Generate Kubernetes manifest
   |
   v
GitHub Contents API
   |
   v
gitops/team-<team>/<service>.yaml
   |
   v
Argo CD detects Git change
   |
   v
Kubernetes Deployment + Service
```

The IDP holds read-only Kubernetes permissions. Application creation happens through Git rather than direct Kubernetes writes, so Git remains the platform's desired state.

### Provisioning Status

The portal tracks the asynchronous GitOps deployment:

```
Submitting request → Request submitted → Waiting for GitOps reconciliation → Deploying → Healthy
```

Once a workload is healthy, the developer is redirected to its service details page.

### Service Details

Each discovered service has a details view showing health, team, namespace, container image, replicas, and CPU/memory requests and limits.

```
payments-api
Status            Healthy
Team              billing
Namespace         team-billing
Image             nginx:alpine
Replicas          1/1
CPU request       25m
CPU limit         100m
Memory request    32Mi
Memory limit      64Mi
```

---

## Resource Profiles

Two predefined application sizes prevent developers from choosing arbitrary CPU/memory values, giving a simple platform-controlled golden path.

**Small**
```
CPU request:     25m
CPU limit:       100m
Memory request:  32Mi
Memory limit:    64Mi
```

**Medium**
```
CPU request:     50m
CPU limit:       200m
Memory request:  64Mi
Memory limit:    128Mi
```

---

## Team Isolation

Each team has its own namespace (`team-billing`, `team-checkout`, `team-search`) containing a ResourceQuota, a NetworkPolicy, and an Argo CD AppProject/Application. Argo CD projects restrict each team to its own namespace and Git path; network policies block unrestricted cross-namespace traffic.

---

## CI/CD for the IDP

```
Push IDP source
      |
      v
GitHub Actions
      |
      v
Build Go container
      |
      v
Push immutable image to GHCR
      |
      v
Update GitOps deployment manifest
      |
      v
Argo CD
      |
      v
Roll out updated IDP
```

Images are tagged with the Git commit SHA.

---

## Accessing the IDP

Currently exposed via NodePort. This is a temporary measure until Traefik-based routing (see Planned Improvements) is in place.

---

## Current Capabilities

- Infrastructure provisioning with Terraform
- K3s installation with Ansible
- Multi-node Kubernetes cluster
- Argo CD GitOps with the App-of-Apps pattern
- Team namespace isolation, resource quotas, network policies, Argo CD AppProjects
- GitHub Actions container builds with immutable IDP releases
- Developer-facing service catalog
- Self-service application provisioning
- Git-based desired-state updates with live provisioning status
- Service details view with CPU/memory visibility and workload health discovery

---

## Planned Improvements

**Service Exposure** — use the existing Traefik installation to give developers application endpoints and an "Open Service" action.

**Team Resource Visibility** — expose namespace quota, current usage, and remaining capacity.

**Developer Actions** — controlled self-service operations (scaling, restarts, image/version updates), all still routed through GitOps rather than direct Kubernetes mutation.

**Platform Maintenance Automation** — extend Ansible to perform rolling node maintenance:

```
cordon → drain → maintenance → wait for node → verify Ready → uncordon
```

Replica counts and placement policies will be added first, so workload availability during node maintenance can actually be demonstrated.

---

## Design Principles

**Developers interact with the platform, not Kubernetes** — the IDP hides infrastructure detail behind simple workflows.

**Git is the desired state** — provisioning modifies Git, not Kubernetes resources directly.

**Platform-controlled defaults** — predefined application sizes replace raw CPU/memory configuration.

**Tenant isolation** — isolated namespaces, quotas, policies, and Argo CD boundaries per team.

**Lightweight architecture** — components are chosen for resource efficiency rather than added complexity.

---

## Goal

Demonstrate how an internal platform can give developers a simple self-service experience while the platform layer handles infrastructure automation, Kubernetes configuration, GitOps reconciliation, resource governance, and workload isolation.

```
Developer Experience
        |
        v
Platform Abstraction
        |
        v
GitOps & Kubernetes
        |
        v
Infrastructure
```
