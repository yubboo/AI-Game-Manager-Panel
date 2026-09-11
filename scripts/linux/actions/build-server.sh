#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "$0")/../lib/init.sh"
fail() { echo "[ERROR] $*" >&2; exit 1; }
need() { command -v "$1" >/dev/null 2>&1 || fail "缺少命令：$1"; }
need go; need node; need tar; need pnpm; need cargo; need rustc

host_arch() {
  case "$(uname -m)" in
    x86_64|amd64) echo amd64 ;;
    aarch64|arm64) echo arm64 ;;
    *) fail "当前构建主机架构不受支持：$(uname -m)" ;;
  esac
}

TARGET="${1:-$(host_arch)}"
case "$TARGET" in amd64|arm64|all) ;; *) fail "用法：bash scripts/build_linux.sh [amd64|arm64|all]" ;; esac
BUILD_CHANNEL="${AGMP_BUILD_CHANNEL:-candidate}"
case "$BUILD_CHANNEL" in candidate|release) ;; *) fail "AGMP_BUILD_CHANNEL 只允许 candidate 或 release" ;; esac
if [[ "$TARGET" == "all" && "${AGMP_ALLOW_CROSS_RUST:-0}" != "1" ]]; then
  fail "XiaoYu 是 Linux Server 必选组件。为避免生成缺 AI 的跨架构包，all 必须由 amd64/arm64 原生 CI Matrix 分别构建；如已配置 Rust 交叉工具链，可显式设置 AGMP_ALLOW_CROSS_RUST=1。"
fi

SERVER_RELEASE_DIR="$AGMP_ROOT/build/$BUILD_CHANNEL/linux-server"
SERVER_BIN_DIR="$AGMP_ROOT/build/work/linux-server/bin"
RUST_TARGET_DIR="$AGMP_ROOT/build/work/rust-linux-server"
rm -rf "$SERVER_BIN_DIR" "$RUST_TARGET_DIR"
mkdir -p "$SERVER_RELEASE_DIR" "$SERVER_BIN_DIR" "$RUST_TARGET_DIR"

build_xiaoyu_arch() {
  local arch="$1" triple output
  output="$SERVER_BIN_DIR/AI-Game-Manager-XiaoYu-linux-$arch"
  case "$arch" in
    amd64) triple="x86_64-unknown-linux-gnu" ;;
    arm64) triple="aarch64-unknown-linux-gnu" ;;
  esac
  if [[ "$arch" != "$(host_arch)" ]]; then
    command -v rustup >/dev/null 2>&1 || fail "跨架构 XiaoYu 构建需要 rustup 与对应 linker：$triple"
    rustup target add "$triple" >/dev/null
  fi
  echo "[BUILD] XiaoYu linux/$arch -> $output"
  if ! CARGO_TARGET_DIR="$RUST_TARGET_DIR/$arch" cargo build --manifest-path "$AGMP_ROOT/rust/Cargo.toml" -p xiaoyu-core --release --target "$triple"; then
    fail "XiaoYu linux/$arch 构建失败。正式多架构发行建议在对应 amd64/arm64 原生 Runner 构建，禁止跳过 XiaoYu 后继续出包。"
  fi
  cp "$RUST_TARGET_DIR/$arch/$triple/release/xiaoyu" "$output"
  chmod +x "$output"
}

echo "[1/8] 构建 Vue 前端并同步到 Server Edition..."
( cd frontend; pnpm install; pnpm run build )
node scripts/common/sync-web-assets.mjs

echo "[2/8] 项目与多端 XiaoYu 架构检查..."
node scripts/common/check-github-safety.mjs
node scripts/common/check-project-layout.mjs
if [[ "$BUILD_CHANNEL" == "release" ]]; then
  node scripts/common/check-release-key.mjs
else
  node scripts/common/check-release-key.mjs --allow-unconfigured
fi
[[ ! -f scripts/common/check-multi-client-ai.mjs ]] || node scripts/common/check-multi-client-ai.mjs
[[ ! -f scripts/common/check-headless-xiaoyu.mjs ]] || node scripts/common/check-headless-xiaoyu.mjs

echo "[3/8] Go 测试..."
go test ./internal/...

build_arch() {
  local arch="$1"
  local output="$SERVER_BIN_DIR/ai-game-manager-panel-server-linux-$arch"
  local xiaoyu="$SERVER_BIN_DIR/AI-Game-Manager-XiaoYu-linux-$arch"
  build_xiaoyu_arch "$arch"
  echo "[BUILD] AGMP linux/$arch -> $output"
  CGO_ENABLED=0 GOOS=linux GOARCH="$arch" go build -trimpath -ldflags="-s -w" -o "$output" ./cmd/aigame-manager-web
  chmod +x "$output"
  local stage="$AGMP_ROOT/build/work/linux-server-$arch/AI-Game-Manager-Panel-Server"
  rm -rf "$stage"; mkdir -p "$stage/internal/xiaoyu"
  cp "$output" "$stage/ai-game-manager-panel-server"
  cp "$xiaoyu" "$stage/internal/xiaoyu/AI-Game-Manager-XiaoYu"
  cp -R configs "$stage/configs-default"
  cp -R distribution/licenses "$stage/licenses"
  cp README.md "$stage/README.md"
  cp distribution/installer/linux/install.sh "$stage/install.sh"
  cp distribution/installer/linux/ai-game-manager-panel-ctl "$stage/ai-game-manager-panel-ctl"
  chmod +x "$stage/ai-game-manager-panel-server" "$stage/internal/xiaoyu/AI-Game-Manager-XiaoYu" "$stage/install.sh" "$stage/ai-game-manager-panel-ctl"
  [[ -x "$stage/internal/xiaoyu/AI-Game-Manager-XiaoYu" ]] || fail "Linux Server 包缺少 XiaoYu Runtime"
  tar -C "$(dirname "$stage")" -czf "$SERVER_RELEASE_DIR/AI-Game-Manager-Panel-Server-Linux-$arch.tar.gz" AI-Game-Manager-Panel-Server
}

echo "[4/8] 构建 AGMP + XiaoYu 多架构 Server Edition..."
if [[ "$TARGET" == "all" || "$TARGET" == "amd64" ]]; then build_arch amd64; fi
if [[ "$TARGET" == "all" || "$TARGET" == "arm64" ]]; then build_arch arm64; fi

echo "[5/8] 生成校验文件..."
( cd "$SERVER_RELEASE_DIR"; sha256sum *.tar.gz > SHA256SUMS )

echo "[6/8] 验证 AGMP Server 与 XiaoYu 产物..."
if [[ "$TARGET" == "all" || "$TARGET" == "amd64" ]]; then
  [[ -s "$SERVER_BIN_DIR/ai-game-manager-panel-server-linux-amd64" ]] || fail "amd64 AGMP 未生成"
  [[ -s "$SERVER_BIN_DIR/AI-Game-Manager-XiaoYu-linux-amd64" ]] || fail "amd64 XiaoYu 未生成"
fi
if [[ "$TARGET" == "all" || "$TARGET" == "arm64" ]]; then
  [[ -s "$SERVER_BIN_DIR/ai-game-manager-panel-server-linux-arm64" ]] || fail "arm64 AGMP 未生成"
  [[ -s "$SERVER_BIN_DIR/AI-Game-Manager-XiaoYu-linux-arm64" ]] || fail "arm64 XiaoYu 未生成"
fi

echo "[7/8] 验证源码不允许 AI-less Linux Server 出包..."
for archive in "$SERVER_RELEASE_DIR"/*.tar.gz; do
  tar -tzf "$archive" | grep -q 'internal/xiaoyu/AI-Game-Manager-XiaoYu$' || fail "产物缺少 XiaoYu：$archive"
done

echo "[8/8] 完成。"
echo "最终产物：build/$BUILD_CHANNEL/linux-server/（AGMP Core + XiaoYu Runtime + Web UI）"
