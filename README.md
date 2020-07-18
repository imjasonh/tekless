# Tekton without Kubernetes

## Goal

Provide an alternative Tekton implementation that doesn't require a Kubernetes
cluster running full-time. This has cost advantages, and security attack
surface area advantages. It's arguably not "Serverless", since there's still a
management cost to upgrading the API server running on Cloud Run. But it would
provide a billing model where costs are only incurred while work is being
performed.

## Design

[Tekton](https://tekton.dev) exposes K8s-native APIs for running
run-to-completion workloads. Its OSS reference implementation runs on K8s, but
that's not a hard requirement.

Instead of requiring a full K8s cluster to be running all the time, we can
provide an API service (on Cloud Run, for example) that accepts requests for
Tekton resources (i.e., not the full K8s API), converts them to Pods using
Tekton's own
[`MakePod`](https://github.com/tektoncd/pipeline/blob/master/pkg/pod/pod.go)
code, and runs that Pod on a freshly spun-up single-use COS VM, using kubelet
static pods. While the Pod is running, we can observe its state and update it
in Datastore storage or Cloud SQL, stream logs and metrics to Stackdriver using
`google-fluentd` on the VM, publish results to Tekton's forthcoming Results API
(also running in Cloud Run). When the pod is complete, delete the VM.

We can observe the pod either by periodically pinging the VM's exposed kubelet
HTTP API, or (better yet) run another agent on the VM that watches its local
kubelet API and POSTs updates to the API service. This agent could also be
responsible for shutting down the VM upon completion.

There's a time penalty of ~5-10s for spinning up COS VMs on-demand, but users
who want scale-to-zero for infrequent Tekton workloads might be okay with that.
VMs could be pooled in a Managed Instance Group and restarted to return back to
original state -- care would need to be taken to prevent a VM from running two
workloads. With the K8s API server out of the picture and single-tenant COS
VMs, this could be a route to a multitenant hosted Tekton.

Scheduling persistent volume-backed workspaces gets a little harder (less so
for single-tenant), and many of the `PodTemplate` features in Tekton (most
related to K8s scheduling) won't be supported. Pod resource requests would be
translated to VM size requests.

The API server could also expose endpoints to manage Secrets and ConfigMaps.
Secrets could be stored in [Secret
Manager](https://cloud.google.com/secret-manager), ConfigMaps just in
Datastore. Normally the kubelet resolves secrets for `envFrom`s at runtime by
asking the K8s API server, so the CR version would serve those to the kubelet
too. We can resolve `envFrom.configMap` before passing it to the VM.
Alternatively, we could accept Secret Manager URLs in the Tekton API's
`secretName` field, and not try to mimic the K8s Secrets endpoints (same for
ConfigMaps).

The API server on Cloud Run would have regional endpoints that could operate
VMs in that region, or users could state they don't care where the workload
runs, for global availability (modulo the API being unavailable in the region
-- a global LB would be necessary here).

The API server and COS VM config could be opensource with versioned releases,
and instructions for users to upgrade their own API servers. This isn't exactly
a "serverless" experience, but it does present much simpler administration than
a full K8s cluster.
