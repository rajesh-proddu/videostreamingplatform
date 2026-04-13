# Kafka → Elasticsearch Consumer Contract

This document defines the contract for the Python Kafka-to-Elasticsearch consumer service that indexes video metadata for search.

## Overview

The consumer reads from the `video-events` Kafka topic and maintains the `videos` Elasticsearch index in sync with the metadataservice database.

## Kafka Topic

- **Topic name**: `video-events`
- **Key**: Event type string (e.g., `video.created`)
- **Value**: JSON-encoded `VideoEvent` envelope

## Event Schema

```json
{
  "version": "1.0",
  "type": "video.created | video.updated | video.deleted",
  "timestamp": "2025-01-15T10:30:00Z",
  "payload": {
    "id": "uuid",
    "title": "string",
    "description": "string",
    "duration": 120,
    "size_bytes": 1048576,
    "format": "mp4",
    "upload_status": "PENDING",
    "created_at": "2025-01-15T10:30:00Z",
    "updated_at": "2025-01-15T10:30:00Z"
  }
}
```

For `video.deleted`, the payload contains only `{"id": "uuid"}`.

## Expected Consumer Behavior

| Event Type      | ES Action                                     |
|-----------------|-----------------------------------------------|
| `video.created` | Index document into `videos` index            |
| `video.updated` | Update document in `videos` index (upsert)    |
| `video.deleted` | Delete document from `videos` index           |

## Elasticsearch Index

- **Index name**: `videos`
- **Index template**: See `scripts/es-video-template.json`
- **Document ID**: Use the video `id` from the payload as the ES document `_id`

## Consumer Requirements

1. Use a consumer group (e.g., `video-es-indexer`) for exactly-once delivery with idempotent writes
2. Handle schema version mismatches gracefully (log and skip unknown versions)
3. Use bulk indexing for throughput when processing backlogs
4. Include health check endpoint for Kubernetes readiness probe
5. Expose Prometheus metrics (messages processed, errors, lag)

## Configuration

| Variable          | Default           | Description                    |
|-------------------|-------------------|--------------------------------|
| `KAFKA_BROKERS`   | `localhost:9092`  | Kafka bootstrap servers        |
| `KAFKA_TOPIC`     | `video-events`    | Topic to consume               |
| `KAFKA_GROUP`     | `video-es-indexer`| Consumer group ID              |
| `ES_URL`          | `http://localhost:9200` | Elasticsearch URL        |
| `ES_INDEX`        | `videos`          | Target index name              |

## Reference Implementation

The consumer should be implemented in Python (per PRD data pipeline language choice). Suggested libraries:
- `confluent-kafka` or `kafka-python` for Kafka consumption
- `elasticsearch-py` for ES indexing
- `prometheus_client` for metrics

The consumer service lives in a separate repository. This document serves as the contract between the Go metadataservice (producer) and the Python consumer.
