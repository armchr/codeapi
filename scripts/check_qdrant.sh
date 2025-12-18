#!/bin/bash

# Quick script to check Qdrant data

echo "=== Collections ==="
curl -s http://localhost:6333/collections | jq -r '.result.collections[] | .name'

echo -e "\n=== Collection Stats ==="
for collection in $(curl -s http://localhost:6333/collections | jq -r '.result.collections[] | .name'); do
    echo -e "\n📊 $collection:"
    curl -s http://localhost:6333/collections/$collection | jq '{
        points: .result.points_count,
        vectors: .result.indexed_vectors_count,
        status: .result.status,
        dimension: .result.config.params.vectors.size
    }'
done

echo -e "\n=== Sample Points from bot-go ==="
curl -s -X POST http://localhost:6333/collections/bot-go/points/scroll \
  -H 'Content-Type: application/json' \
  -d '{"limit": 10, "with_payload": true, "with_vector": false}' | \
  jq -r '.result.points[]? | "ID: \(.id)\n  Type: \(.payload.chunk_type)\n  Name: \(.payload.name)\n  File: \(.payload.file_path // "N/A")\n  Lines: \(.payload.start_line // 0)-\(.payload.end_line // 0)\n"'
