# Lightweight Internal Developer Platform


A resource-efficient Internal Developer Platform (IDP) built on Kubernetes, GitOps, Infrastructure as Code, and configuration automation.


The project simulates a platform engineering environment where application teams can provision and inspect workloads through a developer-facing portal without interacting directly with Kubernetes, Argo CD, Terraform, or Ansible.


The platform is intentionally designed for a small homelab environment while still demonstrating real platform engineering patterns such as self-service provisioning, tenant isolation, GitOps reconciliation, resource governance, and developer abstraction.


---


## Overview


The platform provides three simulated application teams:


- Billing
- Checkout
- Search


Each team receives an isolated Kubernetes namespace with resource quotas, network policies, and Argo CD project boundaries.


Developers interact with the platform through a lightweight custom IDP.


Instead of applying Kubernetes manifests directly, the IDP writes the desired application configuration to GitHub. Argo CD then detects the change and reconciles it into the Kubernetes cluster.


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
Platform Stack
Infrastructure
Proxmox VE
Terraform
Ubuntu Server 24.04
Configuration & Cluster Automation
Ansible
K3s
Kubernetes
GitOps
GitHub
Argo CD
GitHub Actions
GitHub Container Registry
Developer Platform
Go
Embedded HTML/CSS/JavaScript
Kubernetes API
GitHub Contents API
Networking
Traefik
Kubernetes Services
NetworkPolicy
Architecture

The current K3s cluster contains two virtual machines:

Proxmox Host
|
+-- k3s-server
|   IP: 10.0.0.210
|   2 vCPU
|   3 GB RAM
|
+-- k3s-agent
    IP: 10.0.0.211
    1 vCPU
    1.5 GB RAM

The cluster currently uses:

one K3s server/control-plane node
one K3s agent node
Argo CD
Traefik
Kubernetes Metrics Server
lightweight team workloads
the custom IDP

The cluster is intentionally small and is not a highly available control plane.

Internal Developer Platform

The IDP is a custom Go application designed specifically for the homelab resource constraints.

The entire application runs as a single container containing:

compiled Go backend
embedded frontend
CA certificate bundle

It does not require:

Node.js runtime
React runtime
database
Redis
persistent volumes
separate frontend service

Current resource configuration:

CPU request:     25m
CPU limit:       150m
Memory request:  32Mi
Memory limit:    96Mi

Observed usage:

CPU:     ~1m
Memory:  ~4Mi
Developer Portal Features
Service Catalog

The dashboard automatically discovers workloads from Kubernetes and displays:

service name
team
container image
desired replicas
available replicas
health status
Self-Service Application Provisioning

Developers can provision applications directly through the portal.

The developer provides:

team
service name
container image
container port
service size

Example:

Team: Billing
Service: payments-api
Image: nginx:alpine
Port: 80
Size: Small

The developer does not need to create:

Kubernetes Deployments
Kubernetes Services
namespaces
Argo CD Applications
resource definitions

The platform handles these details.

Provisioning Workflow

When a developer creates a service:

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

The IDP has read-only Kubernetes permissions.

Application creation is performed through Git rather than direct Kubernetes write access.

This keeps Git as the desired state of the platform.

Provisioning Status

The portal tracks the asynchronous GitOps deployment process.

Developers see:

Submitting request
       |
       v
Request submitted
       |
       v
Waiting for GitOps reconciliation
       |
       v
Deploying
       |
       v
Healthy

Once the workload becomes healthy, the developer is redirected to its service details page.

Service Details

Each discovered service has a dedicated details view showing:

health
team
namespace
container image
replicas
CPU request
CPU limit
memory request
memory limit

Example:

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
Resource Profiles

The platform currently exposes two predefined application sizes.

Small
CPU request:     25m
CPU limit:       100m
Memory request:  32Mi
Memory limit:    64Mi
Medium
CPU request:     50m
CPU limit:       200m
Memory request:  64Mi
Memory limit:    128Mi

This prevents developers from manually selecting arbitrary resource values and provides a simple platform-controlled golden path.

Team Isolation

Each team receives its own namespace:

team-billing
team-checkout
team-search

The namespaces contain:

ResourceQuota
NetworkPolicy
Argo CD AppProject
Argo CD Application

Argo CD projects restrict teams to their corresponding namespaces and Git paths.

Network policies prevent unrestricted cross-namespace communication.

GitOps Repository Structure
platform-project/
|
+-- idp/
|   +-- main.go
|   +-- provision.go
|   +-- Dockerfile
|   +-- web/
|       +-- index.html
|       +-- create.html
|       +-- service.html
|
+-- gitops/
|   +-- apps/
|   |
|   +-- idp/
|   |
|   +-- team-billing/
|   |
|   +-- team-checkout/
|   |
|   +-- team-search/
|
+-- terraform/
|
+-- ansible/
|
+-- .github/
    +-- workflows/
        +-- idp-image.yml
IDP Container Delivery

Changes to the IDP source trigger GitHub Actions.

The workflow:

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

Images are tagged using the Git commit SHA.

Accessing the IDP

The IDP is currently exposed using NodePort:

http://10.0.0.211:30080

This exposure method is temporary while platform networking is developed further.

Current Resource Usage

Example observed node usage:

k3s-agent
CPU:     ~3%
Memory:  ~52%


k3s-server
CPU:     ~5%
Memory:  ~65%

The IDP itself consumes approximately:

1m CPU
4Mi memory

The platform intentionally prioritizes lightweight components because the Proxmox host has limited memory.

Current Capabilities

Implemented:

Infrastructure provisioning with Terraform
K3s installation with Ansible
Multi-node Kubernetes cluster
Argo CD GitOps
App-of-Apps pattern
Team namespace isolation
Resource quotas
Network policies
Argo CD AppProjects
GitHub Actions container builds
Immutable IDP container releases
Developer-facing service catalog
Self-service application provisioning
Git-based desired-state updates
Live provisioning status
Service details view
CPU and memory visibility
Kubernetes workload health discovery
Planned Improvements

The next platform capabilities include:

Service Exposure

Use the existing Traefik installation to provide developers with application endpoints and an Open Service action.

Team Resource Visibility

Expose:

namespace quota
current resource usage
remaining capacity
Developer Actions

Potential controlled self-service operations:

scaling
application restart
image/version update

These operations should continue to use GitOps rather than direct Kubernetes mutation.

Platform Maintenance Automation

Extend Ansible to perform rolling node maintenance:

cordon
  |
drain
  |
maintenance
  |
wait for node
  |
verify Kubernetes Ready
  |
uncordon

Application replicas and placement policies will be added before demonstrating workload availability during node maintenance.

Design Principles

The project follows several platform engineering principles.

Developers interact with the platform, not Kubernetes

The IDP hides infrastructure implementation details behind simple workflows.

Git is the desired state

Application provisioning modifies Git rather than directly changing Kubernetes resources.

Platform-controlled defaults

Developers select predefined application sizes rather than managing raw CPU and memory configuration.

Tenant isolation

Teams receive isolated namespaces, quotas, policies, and Argo CD boundaries.

Lightweight architecture

Components are selected based on the constraints of the homelab rather than adding infrastructure simply for architectural complexity.

Project Goal

The goal of this project is to demonstrate how an internal platform can provide developers with a simple self-service experience while the platform layer handles infrastructure automation, Kubernetes configuration, GitOps reconciliation, resource governance, and workload isolation.

The project focuses on the boundary between:

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

rather than requiring application developers to understand every layer underneath the platform.
