#!/bin/bash
set -e

cd "$(dirname "$0")/../.."

echo "🔬 Running TypeScript baseline benchmarks..."
echo ""

# Ensure TypeScript server is ready
echo "Checking TypeScript server..."
deno check src/server/server.ts

echo ""

# Run LSP operation benchmarks
echo "Running LSP operation benchmarks..."
./tools/lsp-bench/lsp-bench \
    --server "deno run --allow-all src/server/server.ts" \
    --testdata ./test/testdata \
    --iterations 100 \
    --output baseline-results.json

echo ""
echo "✅ Baseline results saved to baseline-results.json"
