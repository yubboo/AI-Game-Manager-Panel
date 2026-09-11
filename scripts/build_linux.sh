#!/usr/bin/env bash
set -euo pipefail
# AGMP Linux Server Edition 统一构建入口。
# 未指定架构时由 build-server.sh 使用当前原生架构，避免误产出缺少 XiaoYu 的跨架构包。
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
exec bash scripts/linux/actions/build-server.sh "$@"
