# Connect Kubernetes clusters to PMM

!!! caution alert alert-warning "Important"
    Kubernetes cluster monitoring with PMM is still in [Technical Preview](../../../reference/glossary.md#technical-preview) and is subject to change. We recommend that early adopters use this feature for testing purposes only.

PMM Server can act as the centralized observability backend for both your databases *and* the Kubernetes clusters they run on. The integration uses the [Victoria Metrics Kubernetes monitoring stack](https://github.com/VictoriaMetrics/helm-charts/tree/master/charts/victoria-metrics-k8s-stack) Helm chart to scrape cluster-level metrics and push them into PMM Server.

This is useful when:

- You run Percona DBs on Kubernetes and want one place to correlate DB-level and cluster-level signals (pod restarts, node pressure, scheduler events) against query latency, replication lag, etc.
- You want pod/container resource metrics (cAdvisor) for the same DB instances PMM already monitors via `pmm-admin add`.
- You want to surface custom resource state (e.g. `PerconaXtraDBCluster`, `PerconaServerMongoDB`, `PerconaPGCluster`) inside PMM's dashboards.

## What gets captured

The Victoria Metrics Kubernetes monitoring stack captures:

| Component | What it reports |
|-----------|-----------------|
| **cAdvisor** (embedded in kubelet) | Per-container CPU, memory, network, filesystem usage |
| **kubelet** | Node-level pod lifecycle, volume mounts, container runtime status |
| **CoreDNS** | Cluster DNS resolution metrics |
| **kube-state-metrics** | Object state — Deployments, Pods, PVCs, custom resources (incl. Percona operator CRs) |
| **node-exporter** (optional) | Host-level hardware and OS metrics from `/proc` and `/sys` |
| **kube-apiserver** | API request volume, admission, authentication, etcd interactions |
| **Control plane** (`kube-controller-manager`, `kube-scheduler`, `etcd`) | Control-plane workload and reconciliation timings |

Once the stack is running and pushing to PMM, query these metrics from **PMM → Explore** (Code mode), or visualize them on the [Kubernetes overview dashboard](../../../reference/dashboards/kubernetes_monitor_operators.md).

## Set up Kubernetes monitoring

The Helm-based setup procedure is identical across Percona operators. Pick the page that matches the operator you've deployed (or any of them if you don't run a Percona operator — the integration with PMM is the same):

- [Percona Operator for MySQL based on Percona Server for MySQL](https://docs.percona.com/percona-operator-for-mysql/ps/monitor-kubernetes.html)
- [Percona Operator for MySQL based on Percona XtraDB Cluster](https://docs.percona.com/percona-operator-for-mysql/pxc/monitor-kubernetes.html)
- [Percona Operator for MongoDB](https://docs.percona.com/percona-operator-for-mongodb/monitor-kubernetes.html)
- [Percona Operator for PostgreSQL](https://docs.percona.com/percona-operator-for-postgresql/2.0/monitor-kubernetes.html)

Each page covers:

- A **Quick install** script — single-line `curl | bash` with flags for your PMM Server URL, service-account token, and cluster identifier.
- A **Manual install** walkthrough — namespace setup, Secrets for the PMM token, ConfigMap for `kube-state-metrics`, Helm install of the `victoria-metrics-k8s-stack` chart.
- **Verification** — sample `kubectl get pods` output and metrics-browser queries.
- **Uninstall** — cleanup script with CRD-removal options.

## Prerequisites

Before you start, make sure you have:

- A running PMM Server reachable from inside the Kubernetes cluster you want to monitor. See [Install PMM Server](../../install-pmm-server/index.md).
- A PMM Server [service-account token](../../../api/authentication.md#generate-a-service-account-and-token) with permission to write metrics.
- A unique identifier for the Kubernetes cluster. If you push metrics from multiple clusters to the same PMM Server, each cluster needs a distinct identifier or labels will collide.
- [Helm](https://helm.sh/docs/intro/install/) on the workstation you'll install the chart from.

## Related topics

- [Kubernetes overview dashboard](../../../reference/dashboards/kubernetes_monitor_operators.md)
- [Install PMM Server with Helm on Kubernetes](../../install-pmm-server/deployment-options/helm/index.md)
- [Service account authentication](../../../api/authentication.md)
- [VictoriaMetrics reference](../../../reference/third-party/victoria.md)
