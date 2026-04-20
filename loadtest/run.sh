#!/usr/bin/env bash
set -euo pipefail

VEGETA=~/go/bin/vegeta
METADATA_URL="${METADATA_SERVICE_URL:-http://localhost:8080}"
DATA_URL="${DATA_SERVICE_URL:-http://localhost:8081}"
RATE="${RATE:-10000}"
DURATION="${DURATION:-30s}"
RESULTS_DIR="$(dirname "$0")/results"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

mkdir -p "$RESULTS_DIR"

log() { echo -e "${GREEN}[LOAD]${NC} $*"; }
header() { echo -e "\n${CYAN}═══════════════════════════════════════════════════${NC}"; echo -e "${CYAN}  $*${NC}"; echo -e "${CYAN}═══════════════════════════════════════════════════${NC}"; }

# Seed some data for read tests
log "Seeding test data..."
VIDEO_IDS=()
for i in $(seq 1 20); do
  ID=$(curl -s -X POST "$METADATA_URL/videos" \
    -H "Content-Type: application/json" \
    -d "{\"title\":\"loadtest-$i-$RANDOM\",\"size_bytes\":1024}" | python3 -c "import sys,json; print(json.load(sys.stdin).get('id',''))" 2>/dev/null)
  if [ -n "$ID" ]; then
    VIDEO_IDS+=("$ID")
  fi
done
log "Seeded ${#VIDEO_IDS[@]} videos"

cleanup() {
  log "Cleaning up test videos..."
  for id in "${VIDEO_IDS[@]}"; do
    curl -s -X DELETE "$METADATA_URL/videos/$id" > /dev/null 2>&1 || true
  done
}
trap cleanup EXIT

# ── Test 1: GET /health (baseline) ──────────────────────────────────
header "Test 1: GET /health @ ${RATE}/s for ${DURATION}"
echo "GET $METADATA_URL/health" | \
  $VEGETA attack -rate="$RATE" -duration="$DURATION" -timeout=5s | \
  tee "$RESULTS_DIR/health.bin" | \
  $VEGETA report
echo ""
$VEGETA report -type=hist[0,1ms,5ms,10ms,25ms,50ms,100ms,250ms,500ms,1s] < "$RESULTS_DIR/health.bin"

# ── Test 2: GET /videos (list) ──────────────────────────────────────
header "Test 2: GET /videos?limit=10 @ ${RATE}/s for ${DURATION}"
echo "GET $METADATA_URL/videos?limit=10&offset=0" | \
  $VEGETA attack -rate="$RATE" -duration="$DURATION" -timeout=5s | \
  tee "$RESULTS_DIR/list-videos.bin" | \
  $VEGETA report
echo ""
$VEGETA report -type=hist[0,1ms,5ms,10ms,25ms,50ms,100ms,250ms,500ms,1s] < "$RESULTS_DIR/list-videos.bin"

# ── Test 3: GET /videos/:id (single read, round-robin) ─────────────
if [ ${#VIDEO_IDS[@]} -gt 0 ]; then
  header "Test 3: GET /videos/:id (round-robin ${#VIDEO_IDS[@]} videos) @ ${RATE}/s for ${DURATION}"
  TARGETS_FILE="$RESULTS_DIR/get-video-targets.txt"
  > "$TARGETS_FILE"
  for id in "${VIDEO_IDS[@]}"; do
    echo "GET $METADATA_URL/videos/$id" >> "$TARGETS_FILE"
  done

  $VEGETA attack -targets="$TARGETS_FILE" -rate="$RATE" -duration="$DURATION" -timeout=5s | \
    tee "$RESULTS_DIR/get-video.bin" | \
    $VEGETA report
  echo ""
  $VEGETA report -type=hist[0,1ms,5ms,10ms,25ms,50ms,100ms,250ms,500ms,1s] < "$RESULTS_DIR/get-video.bin"
fi

# ── Test 4: POST /videos (create) ──────────────────────────────────
# Lower rate for writes — they hit DB + Kafka
WRITE_RATE=$(( RATE / 10 ))
[ "$WRITE_RATE" -lt 100 ] && WRITE_RATE=100

header "Test 4: POST /videos (create) @ ${WRITE_RATE}/s for ${DURATION}"
cat > "$RESULTS_DIR/create-target.txt" <<EOF
POST $METADATA_URL/videos
Content-Type: application/json
@$RESULTS_DIR/create-body.json
EOF
echo '{"title":"loadtest-create","description":"vegeta load test","size_bytes":2048}' > "$RESULTS_DIR/create-body.json"

$VEGETA attack -targets="$RESULTS_DIR/create-target.txt" -rate="$WRITE_RATE" -duration="$DURATION" -timeout=5s | \
  tee "$RESULTS_DIR/create-video.bin" | \
  $VEGETA report
echo ""
$VEGETA report -type=hist[0,1ms,5ms,10ms,25ms,50ms,100ms,250ms,500ms,1s] < "$RESULTS_DIR/create-video.bin"

# ── Test 5: Mixed workload (80% reads, 20% writes) ─────────────────
header "Test 5: Mixed workload (reads + writes) @ ${RATE}/s for ${DURATION}"
MIXED_FILE="$RESULTS_DIR/mixed-targets.txt"
> "$MIXED_FILE"
# 80% reads
for i in $(seq 1 8); do
  echo "GET $METADATA_URL/videos?limit=10&offset=0" >> "$MIXED_FILE"
  if [ ${#VIDEO_IDS[@]} -gt 0 ]; then
    idx=$(( i % ${#VIDEO_IDS[@]} ))
    echo "GET $METADATA_URL/videos/${VIDEO_IDS[$idx]}" >> "$MIXED_FILE"
  fi
done
# 20% writes
for i in $(seq 1 4); do
  echo "POST $METADATA_URL/videos" >> "$MIXED_FILE"
  echo "Content-Type: application/json" >> "$MIXED_FILE"
  echo "@$RESULTS_DIR/create-body.json" >> "$MIXED_FILE"
  echo "" >> "$MIXED_FILE"
done

$VEGETA attack -targets="$MIXED_FILE" -rate="$RATE" -duration="$DURATION" -timeout=5s | \
  tee "$RESULTS_DIR/mixed.bin" | \
  $VEGETA report
echo ""
$VEGETA report -type=hist[0,1ms,5ms,10ms,25ms,50ms,100ms,250ms,500ms,1s] < "$RESULTS_DIR/mixed.bin"

# ── Test 6: Data service health @ same rate ─────────────────────────
header "Test 6: GET /health (data-service) @ ${RATE}/s for ${DURATION}"
echo "GET $DATA_URL/health" | \
  $VEGETA attack -rate="$RATE" -duration="$DURATION" -timeout=5s | \
  tee "$RESULTS_DIR/data-health.bin" | \
  $VEGETA report
echo ""
$VEGETA report -type=hist[0,1ms,5ms,10ms,25ms,50ms,100ms,250ms,500ms,1s] < "$RESULTS_DIR/data-health.bin"

# ── Summary ─────────────────────────────────────────────────────────
header "SUMMARY"
echo -e "${YELLOW}Target rate: ${RATE} req/s | Duration: ${DURATION}${NC}"
echo ""
for f in health list-videos get-video create-video mixed data-health; do
  if [ -f "$RESULTS_DIR/$f.bin" ]; then
    STATS=$($VEGETA report -type=text < "$RESULTS_DIR/$f.bin" 2>/dev/null | head -6)
    SUCCESS=$(echo "$STATS" | grep "Success" | awk '{print $NF}')
    P99=$(echo "$STATS" | grep "99th" | awk '{print $NF}')
    THROUGHPUT=$(echo "$STATS" | grep "Throughput" | awk '{print $NF}')
    printf "  %-20s success=%-8s p99=%-12s throughput=%s\n" "$f" "$SUCCESS" "$P99" "$THROUGHPUT"
  fi
done
echo ""
log "Raw results in $RESULTS_DIR/"
