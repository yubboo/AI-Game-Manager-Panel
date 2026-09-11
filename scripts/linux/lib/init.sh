#!/usr/bin/env bash
set -euo pipefail
AGMP_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
cd "$AGMP_ROOT"
AGMP_VERSION="$(node -p "require('./frontend/package.json').version")"
AGMP_RELEASE_DIR="$AGMP_ROOT/build/release/linux"
AGMP_WORK_BIN_DIR="$AGMP_ROOT/build/work/linux/bin"
AGMP_BIN_DIR="$AGMP_WORK_BIN_DIR"
export AGMP_ROOT AGMP_VERSION AGMP_RELEASE_DIR AGMP_WORK_BIN_DIR AGMP_BIN_DIR
