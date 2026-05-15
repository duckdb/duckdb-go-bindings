#!/usr/bin/env bash
# Build both bench binaries (upstream/fork) and run the comparator.
#
# Usage:
#   ./run.sh                   # default benchmark
#   ./run.sh -scale 0.1        # 10% iter counts (quick smoke)
#   ./run.sh -runs 7           # 7 measured runs per scenario
#   ./run.sh -only bind_varchar,bind_blob
#
# All flags after a `--` (or otherwise unknown) are forwarded to compare.
set -euo pipefail

cd "$(dirname "$0")"

CGO_FLAG=${CGO_ENABLED:-1}
export CGO_ENABLED="$CGO_FLAG"

echo "==> building upstream bench..."
( cd upstream && go mod tidy && go build -o ../bench_upstream . )

echo "==> building fork bench..."
( cd fork && go mod tidy && go build -o ../bench_fork . )

echo "==> building compare orchestrator..."
( cd compare && go build -o ../compare_bin . )

echo "==> running comparison..."
exec ./compare_bin -fork ./bench_fork -upstream ./bench_upstream "$@"
