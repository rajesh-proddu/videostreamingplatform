# Metadata Service Stress Test

Manual load/stress runner for the metadata service.

## Goal

Drive a mixed workload near **10k QPS** with the default profile:

- **60%** `GET /videos/{id}`
- **20%** `GET /videos`
- **15%** `POST /videos`
- **5%** `PUT /videos/{id}`

This matches the approved **80% reads / 20% writes** mix.

## Run

```bash
go run ./tests/stress/metadataservice \
  --base-url http://localhost:8080 \
  --target-qps 10000 \
  --duration 1m \
  --workers 256 \
  --seed-videos 2000
```

Against AWS or another remote environment:

```bash
go run ./tests/stress/metadataservice \
  --base-url https://meta.example.com \
  --target-qps 10000 \
  --duration 2m
```

## Environment Variables

Each flag also has an environment variable equivalent:

| Flag | Environment variable |
|------|----------------------|
| `--base-url` | `METADATA_STRESS_BASE_URL` |
| `--target-qps` | `METADATA_STRESS_TARGET_QPS` |
| `--duration` | `METADATA_STRESS_DURATION` |
| `--workers` | `METADATA_STRESS_WORKERS` |
| `--seed-videos` | `METADATA_STRESS_SEED_VIDEOS` |
| `--seed-workers` | `METADATA_STRESS_SEED_WORKERS` |
| `--list-limit` | `METADATA_STRESS_LIST_LIMIT` |
| `--request-timeout` | `METADATA_STRESS_REQUEST_TIMEOUT` |
| `--read-by-id-pct` | `METADATA_STRESS_READ_BY_ID_PCT` |
| `--list-pct` | `METADATA_STRESS_LIST_PCT` |
| `--create-pct` | `METADATA_STRESS_CREATE_PCT` |
| `--update-pct` | `METADATA_STRESS_UPDATE_PCT` |
| `--run-tag` | `METADATA_STRESS_RUN_TAG` |

## Output

The runner prints:

- target vs achieved QPS
- total completed requests
- success/error counts
- status-code distribution
- p50 / p95 / p99 latency
- per-operation breakdown

## Notes

- This is a **manual performance tool**, not part of the default CI suite.
- The runner seeds metadata rows before load generation so reads and updates hit existing records.
- Created rows are not cleaned up automatically; use the run tag to identify records from a specific run.
- Whether the system reaches 10k QPS depends on the target environment, MySQL capacity, CPU, and network limits.
