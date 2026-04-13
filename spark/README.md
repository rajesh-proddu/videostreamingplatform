# Watch History Ingestion — Spark Job

Spark Structured Streaming job that reads watch events from the `watch-events` Kafka topic and persists them to an Apache Iceberg `watch_history` table.

## Prerequisites

- Python 3.10+
- Apache Spark 3.5+
- A running Kafka broker with the `watch-events` topic
- Iceberg-compatible catalog (Hadoop catalog for local dev)

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `KAFKA_BROKERS` | `localhost:9092` | Kafka bootstrap servers |
| `KAFKA_WATCH_TOPIC` | `watch-events` | Kafka topic to consume |
| `ICEBERG_CATALOG` | `local` | Spark catalog name |
| `ICEBERG_WAREHOUSE` | `/tmp/iceberg-warehouse` | Iceberg warehouse path |
| `CHECKPOINT_DIR` | `/tmp/watch-history-checkpoint` | Streaming checkpoint dir |

## Iceberg Table Schema

```sql
CREATE TABLE watch_history (
    event_type    STRING,    -- "watch.started" or "watch.completed"
    video_id      STRING,
    user_id       STRING,
    session_id    STRING,    -- correlates started/completed pairs
    bytes_read    BIGINT,    -- 0 for started, total for completed
    event_ts      TIMESTAMP, -- when the event occurred
    ingested_at   TIMESTAMP  -- when the row was written
) PARTITIONED BY (days(event_ts))
```

## Local Development

```bash
# Install dependencies
pip install -r requirements.txt

# Run the job (requires local Kafka + Iceberg catalog)
spark-submit \
  --packages org.apache.iceberg:iceberg-spark-runtime-3.5_2.12:1.7.1,org.apache.spark:spark-sql-kafka-0-10_2.12:3.5.3 \
  src/watch_history_ingest.py
```

## Docker

```bash
docker build -t watch-history-ingest .
docker run --rm \
  -e KAFKA_BROKERS=kafka:9092 \
  -e ICEBERG_WAREHOUSE=s3://bucket/warehouse \
  watch-history-ingest
```

## Running Tests

```bash
pip install -e ".[dev]"
pytest tests/ -v
```

## Future: Trino Query Layer

The Iceberg table can be queried via Trino for analytics (e.g., most-watched videos, user watch patterns). Trino integration is planned for Phase III.
