# Kubernetes Node Resiliency Scenarios

Concrete resiliency scenarios for both the **local Kind setup** and **AWS EKS deployment**, focused on **node down / node up** behavior while metadata-service traffic is active.

## Goal

Validate that the platform:

- stays available through single-node disruption
- recovers automatically when pods are rescheduled
- preserves metadata correctness during recovery
- exposes the disruption clearly in logs, metrics, and traces

## Preconditions

Before running any scenario:

1. The cluster is healthy and at desired node count.
2. `metadata-service` has at least 2 replicas.
3. The metadata stress runner is available:

   ```bash
   go run ./tests/stress/metadataservice \
     --base-url http://<metadata-base-url> \
     --target-qps 10000 \
     --duration 3m
   ```

4. Access details match your environment:

   - **Local Kind**: metadata URL is typically `http://localhost:8080`
   - **AWS EKS**: metadata URL is the service LoadBalancer URL, e.g. `http://<metadata-lb>:8080`
5. You can access the cluster with `kubectl`.
6. For AWS hard-failure scenarios, you can also access the account with `aws`.

## Common Checks

Capture these in every scenario:

| Check | What to measure |
|------|------------------|
| **Availability** | 2xx rate, 5xx rate, timeout count |
| **Recovery** | time until desired pods are Ready again |
| **Correctness** | created and updated metadata remains readable |
| **Kubernetes** | eviction, rescheduling, node conditions, pod restart count |
| **Observability** | latency spike, error spike, trace gaps, recovery trend |

## Scenario Matrix

| ID | Scenario | Local Kind action | AWS EKS action | Expected outcome |
|------|----------|----------------|------------------|
| **N1** | Single node drain during steady read traffic | `kubectl drain` one worker node | `kubectl drain` one worker node | short latency spike, pods evicted cleanly, service remains available |
| **N2** | Single hard node failure during 80/20 mixed load | stop one Kind worker container | terminate EC2 instance behind one worker node | brief disruption, replacement or restarted node returns, metadata-service recovers |
| **N3** | Targeted metadata node drain | drain the Kind node hosting one metadata pod | drain the EKS node hosting one metadata pod | replacement pod becomes Ready on another node, no prolonged outage |
| **N4** | Targeted metadata node hard failure | stop the Kind node hosting a metadata pod, then start it again | terminate the EKS node hosting a metadata pod | no data corruption, pod recreated after node loss/recovery |
| **N5** | Observability node loss | drain or stop the Kind node running Prometheus/Grafana/Jaeger | drain or terminate the EKS node running Prometheus/Grafana/Jaeger | application remains available, observability degrades only temporarily |
| **N6** | Sequential node failures | disrupt one node, wait partial recovery, disrupt a second node | disrupt one node, wait partial recovery, disrupt a second node | validates minimum safe capacity; may expose capacity limits |
| **N7** | Graceful node return path | drain a node, validate recovery, then uncordon it | drain a node, validate recovery, then uncordon it | cluster returns to schedulable healthy state |
| **N8** | Hard failure plus node return validation | stop a Kind node and bring it back with `docker start` | terminate one node and wait for managed node group replacement | node availability returns and workloads recover |

## Detailed Scenarios

### N1: Single node drain during steady read traffic

**Why**: safest first resiliency check; validates pod eviction and rescheduling without simulating infra loss.

**Steps**

```bash
kubectl get nodes
go run ./tests/stress/metadataservice --base-url http://<metadata-base-url> --target-qps 10000 --duration 3m &
scripts/aws/resiliency-node-cycle.sh snapshot
scripts/aws/resiliency-node-cycle.sh drain --node <node-name>
scripts/aws/resiliency-node-cycle.sh wait-pods --selector app=metadata-service --expected-ready 2
scripts/aws/resiliency-node-cycle.sh uncordon --node <node-name>
```

**Pass**

- metadata-service regains 2 Ready pods
- stress run continues without extended outage
- no persistent 5xx spike after recovery window

### N2: Single node termination during mixed load

**Why**: validates actual node-down behavior, managed node group replacement, and pod recovery.

**Steps**

**Local Kind steps**

```bash
go run ./tests/stress/metadataservice --base-url http://localhost:8080 --target-qps 10000 --duration 5m &
scripts/aws/resiliency-node-cycle.sh snapshot
scripts/aws/resiliency-node-cycle.sh kind-stop --node <kind-worker-node>
scripts/aws/resiliency-node-cycle.sh wait-pods --selector app=metadata-service --expected-ready 2 --timeout 900
scripts/aws/resiliency-node-cycle.sh kind-start --node <kind-worker-node>
scripts/aws/resiliency-node-cycle.sh snapshot
```

**AWS EKS steps**

```bash
go run ./tests/stress/metadataservice --base-url http://<metadata-lb>:8080 --target-qps 10000 --duration 5m &
scripts/aws/resiliency-node-cycle.sh snapshot
scripts/aws/resiliency-node-cycle.sh terminate --node <node-name> --region us-east-1
scripts/aws/resiliency-node-cycle.sh wait-pods --selector app=metadata-service --expected-ready 2 --timeout 15m
scripts/aws/resiliency-node-cycle.sh snapshot
```

**Pass**

- replacement node joins the cluster
- metadata-service returns to desired replica count
- metadata reads and writes succeed after recovery

### N3: Targeted metadata node drain

**Why**: ensures the service survives loss of an active metadata-service node specifically.

**Steps**

```bash
kubectl get pods -n videostreamingplatform -l app=metadata-service -o wide
scripts/aws/resiliency-node-cycle.sh drain --node <node-hosting-metadata-pod>
scripts/aws/resiliency-node-cycle.sh wait-pods --selector app=metadata-service --expected-ready 2
scripts/aws/resiliency-node-cycle.sh uncordon --node <node-hosting-metadata-pod>
```

**Pass**

- the replacement pod becomes Ready
- `GET /videos` and `POST /videos` both succeed after recovery

### N4: Targeted metadata node hard failure

**Why**: same as N3, but with real node loss instead of graceful eviction.

**Checks**

- metadata-service reconnects cleanly to MySQL after pod restart
- no stuck crash loops due to stale DB connections
- for local Kind, the stopped node can be started again after recovery validation

### N5: Observability node loss

**Why**: verifies observability is not a hard dependency for serving traffic.

**Checks**

- metadata-service stays healthy
- Prometheus / Grafana / Jaeger recover after reschedule or replacement
- application data path remains healthy even if dashboards disappear briefly

### N6: Sequential node failures

**Why**: exposes whether the cluster has enough spare capacity or if it is effectively single-fault-tolerant only.

**Checks**

- whether metadata-service stays partially available
- whether PDB/anti-affinity settings are sufficient
- whether resource requests are too tight for current node sizing

### N7: Graceful node return path

**Why**: validates operator workflow for maintenance windows.

**Steps**

```bash
scripts/aws/resiliency-node-cycle.sh drain --node <node-name>
scripts/aws/resiliency-node-cycle.sh wait-pods --selector app=metadata-service --expected-ready 2
scripts/aws/resiliency-node-cycle.sh uncordon --node <node-name>
kubectl get nodes
```

### N8: Hard failure plus node return validation

**Why**: validates real node-down behavior in both environments, not just Kubernetes scheduling.

**Checks**

- **Local Kind**: stopped worker comes back with `docker start <node-container>`
- **AWS EKS**: terminated instance disappears and a replacement worker joins
- workloads return to desired state

## Evidence to Collect

```bash
scripts/aws/resiliency-node-cycle.sh snapshot
kubectl get events -A --sort-by=.lastTimestamp | tail -50
kubectl logs -l app=metadata-service -n videostreamingplatform --tail=100
kubectl get pods -n videostreamingplatform -o wide
kubectl get nodes -o wide
```

## Notes for This Project

- Start with **single-node** scenarios first; both local Kind and AWS have limited spare capacity.
- Run mixed-load validation with the metadata stress runner, not the correctness-focused e2e suite.
- Multi-node failure is useful, but may fail by design on a 2-node cluster with limited headroom.
- In local Kind, use `http://localhost:8080` for metadata-service traffic because the local service is exposed via NodePort mappings.
