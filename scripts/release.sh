#!/usr/bin/env bash

set -euo pipefail

usage() {
  cat <<'USAGE'
Usage: ./scripts/release.sh vX.Y.Z [remote]

Re-entrant release script. Safe to run multiple times.
USAGE
}

if [ "$#" -lt 1 ] || [ "$#" -gt 2 ]; then
  usage
  exit 2
fi

version="$1"
remote="${2:-origin}"

if [[ "${version}" != v* ]]; then
  echo "error: version must start with 'v' (example: v0.3.2)" >&2
  exit 2
fi

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${root_dir}"

current_branch="$(git rev-parse --abbrev-ref HEAD)"
if [ "${current_branch}" != "main" ] && [[ "${current_branch}" != v1.* ]]; then
  echo "error: release must be run on main or a v1.* branch" >&2
  exit 1
fi

lib_dirs=(
  lib/darwin-amd64
  lib/darwin-arm64
  lib/linux-amd64
  lib/linux-arm64
  lib/windows-amd64
)
lib_mods=()
lib_tags=()
push_args=()

for dir in "${lib_dirs[@]}"; do
  lib_mods+=("github.com/duckdb/duckdb-go-bindings/${dir}")
  tag="${dir}/${version}"
  lib_tags+=("${tag}")
  push_args+=("refs/tags/${tag}")
done

echo "==> Tagging lib modules"
for tag in "${lib_tags[@]}"; do
  if git rev-parse -q --verify "refs/tags/${tag}" >/dev/null; then
    echo "- skip existing tag ${tag}"
  else
    echo "- create tag ${tag}"
    git tag "${tag}"
  fi
done

echo "==> Pushing lib module tags to ${remote}"
git push "${remote}" "${push_args[@]}"

echo "==> Updating root module dependencies"
export GOWORK=off # avoid local workspaces overriding module resolution
export GOPRIVATE="github.com/duckdb/duckdb-go-bindings" # bypass proxy/sumdb for these modules
for mod in "${lib_mods[@]}"; do
  go mod edit -require="${mod}@${version}"
done

go mod tidy

if ! git diff --quiet -- go.mod go.sum; then
  cat <<'NOTE'
==> Root module deps changed

Open a PR with the updated go.mod/go.sum, merge it into main,
then run this script again to tag the root module.
NOTE
  exit 1
fi

root_tag="${version}"
if git rev-parse -q --verify "refs/tags/${root_tag}" >/dev/null; then
  echo "==> Root tag ${root_tag} already exists; nothing to do"
  exit 0
fi

echo "==> Tagging root module ${root_tag}"
git tag "${root_tag}"

echo "==> Pushing root tag to ${remote}"
git push "${remote}" "refs/tags/${root_tag}"

echo "==> Release complete"
