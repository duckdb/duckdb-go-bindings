#!/bin/bash
# This script compares the list of Go static libraries with the downloaded static libraries
# And outputs the differences to separate files

set -e  # Exit on any error

show_help() {
  cat << EOF
Usage: $0 -h <hash> [-v <version>] | -l <local_path>

Compare Go static libraries with DuckDB static libraries.

OPTIONS:
  -h <hash>        Git commit hash (required for download, e.g., 431ad092c9)
  -v <version>     Optional version tag (e.g., 1.4.0 or v1.4.0)
  -l <local_path>  Use local directory instead of downloading

EXAMPLES:
  $0 -h 431ad092c9              # Download from hash
  $0 -h 431ad092c9 -v 1.4.0     # Download from hash with version tag
  $0 -l ./static-libs           # Use local files

EOF
}

# Parse command line arguments
VERSION=""
HASH=""
LOCAL_PATH=""

while getopts "v:h:l:" opt; do
  case $opt in
    v)
      VERSION="${OPTARG#v}"  # Remove 'v' prefix if present
      ;;
    h)
      HASH="$OPTARG"
      ;;
    l)
      LOCAL_PATH=${PWD}/"$OPTARG"
      ;;
    \?)
      show_help
      exit 1
      ;;
  esac
done

# Show help if no arguments provided
if [ $# -eq 0 ]; then
  show_help
  exit 1
fi

# Check that both -h and -l are not provided
if [ -n "$HASH" ] && [ -n "$LOCAL_PATH" ]; then
  echo "Error: Cannot specify both -h (hash) and -l (local path)"
  echo ""
  show_help
  exit 1
fi

# Validate arguments based on mode
if [ -n "$LOCAL_PATH" ]; then
  # Local mode - validate path
  if [ ! -d "$LOCAL_PATH" ]; then
    echo "Error: Local path '$LOCAL_PATH' does not exist or is not a directory"
    exit 1
  fi
  echo "Using local static libraries from: $LOCAL_PATH"
else
  # Download mode - hash is required
  if [ -z "$HASH" ]; then
    echo "Error: Hash (-h) is required when downloading"
    echo ""
    show_help
    exit 1
  fi

  # Build DOWNLOAD_PATH for remote download
  if [ -n "$VERSION" ]; then
    DOWNLOAD_PATH="${HASH}/v${VERSION}"
  else
    DOWNLOAD_PATH="${HASH}"
  fi
fi

rm -rf list_of_libs
mkdir list_of_libs

cd lib || exit 1

GO_LIBS=("darwin-amd64" "darwin-arm64" "linux-amd64" "linux-arm64" "windows-amd64")
IDX=0
for lib in "${GO_LIBS[@]}"; do
  cd "${lib}" || exit 1
  OUTPUT="go-${lib}.txt"
  ls *.a > ../../list_of_libs/"${OUTPUT}"
  GO_FILES[${IDX}]=$OUTPUT
  IDX=$((IDX + 1))
  cd .. || exit 1
done

cd ../list_of_libs || exit 1

mkdir -p downloaded-static-libs
cd downloaded-static-libs || exit 1

# download or copy the static libs and output the list of files
STATIC_LIBS=("osx-amd64" "osx-arm64" "linux-amd64" "linux-arm64" "windows-mingw")

IDX=0
for lib in "${STATIC_LIBS[@]}"; do
  echo "Processing static library for: ${lib}"
  STATIC_LIB="static-libs-${lib}"

  if [ -n "$LOCAL_PATH" ]; then
    # Check if unzipped directory exists
    LOCAL_DIR="${LOCAL_PATH}/${STATIC_LIB}"
    LOCAL_ZIP="${LOCAL_PATH}/${STATIC_LIB}.zip"

    if [ -d "$LOCAL_DIR" ]; then
      # Copy or symlink the directory
      cp -r "$LOCAL_DIR" "${STATIC_LIB}"
    elif [ -f "$LOCAL_ZIP" ]; then
      # Extract zip file
      unzip -o "$LOCAL_ZIP" -d "${STATIC_LIB}"
    else
      echo "Warning: Neither directory nor zip found for ${lib}"
      echo "  Looked for: $LOCAL_DIR or $LOCAL_ZIP"
      echo "Skipping ${lib}..."
      IDX=$((IDX + 1))
      continue
    fi
  else
    # Download files
    ZIP="${STATIC_LIB}.zip"
    URL="https://duckdb-staging.duckdb.org/${DOWNLOAD_PATH}/duckdb/duckdb/github_release/${ZIP}"

    curl -s -L -o "$ZIP" "$URL"
    unzip -q -o "$ZIP" -d "${STATIC_LIB}"
  fi

  cd "${STATIC_LIB}" || exit 1
  ls *.a > ../../"${STATIC_LIB}.txt"
  STATIC_FILES[${IDX}]="${STATIC_LIB}.txt"
  IDX=$((IDX + 1))
  cd .. || exit 1
done

cd .. || exit 1

echo ""
echo "Comparing Go static libraries with downloaded static libraries..."

for i in "${!GO_FILES[@]}"; do
  echo ""
  echo "Comparing ${STATIC_LIBS[$i]}:"

  diff "${GO_FILES[$i]}" "${STATIC_FILES[$i]}" > "diff_${STATIC_LIBS[$i]}.txt" || true

  if [ -s "diff_${STATIC_LIBS[$i]}.txt" ]; then
    echo "Differences found:"
    cat "diff_${STATIC_LIBS[$i]}.txt"
  else
    echo "✓ No differences (files are identical)"
  fi
done