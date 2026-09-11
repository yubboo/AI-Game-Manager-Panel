#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "$0")/../lib/init.sh"

fail() {
  echo "[ERROR] $*" >&2
  exit 1
}

need() {
  command -v "$1" >/dev/null 2>&1 || fail "缺少命令：$1"
}

need go
need node
need pnpm
need tar
need dpkg-deb
need cargo
need rustc

# Wails Linux 桌面端依赖 CGO + GTK + WebKitGTK。这里只检测，不擅自修改用户系统。
if ! pkg-config --exists gtk+-3.0 2>/dev/null; then
  fail "缺少 GTK3 开发库。Debian/Ubuntu 通常需要安装 libgtk-3-dev。"
fi
if pkg-config --exists webkit2gtk-4.1 2>/dev/null; then
  AGMP_WEBKIT_TAG="webkit2_41"
  AGMP_WEBKIT_DEP="libwebkit2gtk-4.1-0"
elif pkg-config --exists webkit2gtk-4.0 2>/dev/null; then
  AGMP_WEBKIT_TAG="webkit2_40"
  AGMP_WEBKIT_DEP="libwebkit2gtk-4.0-37"
else
  fail "缺少 WebKitGTK 开发库。请安装当前发行版提供的 libwebkit2gtk 开发包。"
fi
export AGMP_WEBKIT_TAG AGMP_WEBKIT_DEP

rm -rf "$AGMP_RELEASE_DIR" "$AGMP_BIN_DIR" "$AGMP_ROOT/build/work/linux-amd64"
mkdir -p "$AGMP_RELEASE_DIR" "$AGMP_BIN_DIR"

echo "==============================================="
echo " AI-Game-Manager-Panel $AGMP_VERSION - LINUX RELEASE"
echo "==============================================="

echo "[1/11] 项目骨架与多端 XiaoYu 检查..."
node scripts/common/check-github-safety.mjs
node scripts/common/check-project-layout.mjs
node scripts/common/check-release-key.mjs
[[ ! -f scripts/common/check-multi-client-ai.mjs ]] || node scripts/common/check-multi-client-ai.mjs
[[ ! -f scripts/common/check-headless-xiaoyu.mjs ]] || node scripts/common/check-headless-xiaoyu.mjs

echo "[2/11] 安装前端依赖..."
(
  cd frontend
  pnpm install
)

echo "[3/11] 前端类型检查与生产构建..."
(
  cd frontend
  pnpm run build
)

echo "[4/11] 同步 Web 静态资源..."
node scripts/common/sync-web-assets.mjs

echo "[5/11] Go 测试与 vet..."
go test ./...
go vet ./...

echo "[6/11] 构建 Linux XiaoYu Intelligence Core..."
XIAOYU_TARGET_DIR="$AGMP_ROOT/build/work/rust-linux-desktop"
rm -rf "$XIAOYU_TARGET_DIR"
CARGO_TARGET_DIR="$XIAOYU_TARGET_DIR" cargo build --manifest-path "$AGMP_ROOT/rust/Cargo.toml" -p xiaoyu-core --release
cp "$XIAOYU_TARGET_DIR/release/xiaoyu" "$AGMP_BIN_DIR/AI-Game-Manager-XiaoYu"
chmod +x "$AGMP_BIN_DIR/AI-Game-Manager-XiaoYu"
[[ -x "$AGMP_BIN_DIR/AI-Game-Manager-XiaoYu" ]] || fail "Linux XiaoYu Runtime 未生成"

echo "[7/11] 构建 Linux Web 管理程序..."
go build -trimpath -o "$AGMP_BIN_DIR/AI-Game-Manager-Panel-Web" ./cmd/aigame-manager-web

echo "[8/11] 构建 Linux Wails 桌面程序..."
if command -v wails >/dev/null 2>&1; then
  wails build -platform linux/amd64 -tags "$AGMP_WEBKIT_TAG" -s -o AI-Game-Manager-Panel
else
  echo "[INFO] PATH 中没有 wails，使用 Go 直接运行固定版本 Wails CLI。"
  go run github.com/wailsapp/wails/v2/cmd/wails@v2.15.0 build -platform linux/amd64 -tags "$AGMP_WEBKIT_TAG" -s -o AI-Game-Manager-Panel
fi

# Wails 默认输出 build/bin/AI-Game-Manager-Panel；移动到平台专属目录，避免与 Windows 产物互相覆盖。
if [[ -f "$AGMP_ROOT/build/bin/AI-Game-Manager-Panel" ]]; then
  mv "$AGMP_ROOT/build/bin/AI-Game-Manager-Panel" "$AGMP_BIN_DIR/AI-Game-Manager-Panel"
fi
[[ -x "$AGMP_BIN_DIR/AI-Game-Manager-Panel" ]] || fail "Linux 桌面程序未生成：build/work/linux/bin/AI-Game-Manager-Panel"
[[ -x "$AGMP_BIN_DIR/AI-Game-Manager-Panel-Web" ]] || fail "Linux Web 程序未生成：build/work/linux/bin/AI-Game-Manager-Panel-Web"

echo "[9/11] 生成 Linux 便携包..."
PORTABLE="$AGMP_ROOT/build/work/linux-amd64/AI-Game-Manager-Panel"
mkdir -p "$PORTABLE"
cp "$AGMP_BIN_DIR/AI-Game-Manager-Panel" "$PORTABLE/AI-Game-Manager-Panel"
cp "$AGMP_BIN_DIR/AI-Game-Manager-Panel-Web" "$PORTABLE/AI-Game-Manager-Panel-Web"
mkdir -p "$PORTABLE/internal/xiaoyu"
cp "$AGMP_BIN_DIR/AI-Game-Manager-XiaoYu" "$PORTABLE/internal/xiaoyu/AI-Game-Manager-XiaoYu"
chmod +x "$PORTABLE/internal/xiaoyu/AI-Game-Manager-XiaoYu"
cp -R configs "$PORTABLE/configs"
cp -R distribution/licenses "$PORTABLE/licenses"
cp README.md "$PORTABLE/README.md"
mkdir -p "$PORTABLE/runtime"/{data,log,backups,instances,temp,exports,plugins,cache}
cat > "$PORTABLE/run-ai-game-manager-panel.sh" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")" && pwd)"
export AGMP_ROOT="$ROOT"
export AGMP_XIAOYU_RUNTIME="$ROOT/internal/xiaoyu/AI-Game-Manager-XiaoYu"
exec "$ROOT/AI-Game-Manager-Panel" "$@"
SH
cat > "$PORTABLE/run-ai-game-manager-panel-web.sh" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")" && pwd)"
export AGMP_ROOT="$ROOT"
export AGMP_XIAOYU_RUNTIME="$ROOT/internal/xiaoyu/AI-Game-Manager-XiaoYu"
exec "$ROOT/AI-Game-Manager-Panel-Web" "$@"
SH
chmod +x "$PORTABLE/run-ai-game-manager-panel.sh" "$PORTABLE/run-ai-game-manager-panel-web.sh"
tar -C "$(dirname "$PORTABLE")" -czf "$AGMP_RELEASE_DIR/AI-Game-Manager-Panel-Linux-x64.tar.gz" AI-Game-Manager-Panel

echo "[10/11] 生成 Debian/Ubuntu 安装包..."
bash scripts/linux/actions/build-deb.sh

echo "[11/11] 验证 Linux 发布产物..."
[[ -s "$AGMP_RELEASE_DIR/AI-Game-Manager-Panel-Linux-x64.tar.gz" ]] || fail "缺少 Linux 便携包"
tar -tzf "$AGMP_RELEASE_DIR/AI-Game-Manager-Panel-Linux-x64.tar.gz" | grep -q 'internal/xiaoyu/AI-Game-Manager-XiaoYu$' || fail "Linux 便携包缺少 XiaoYu Runtime"
[[ -s "$AGMP_RELEASE_DIR/AI-Game-Manager-Panel_${AGMP_VERSION}_amd64.deb" ]] || fail "缺少 Linux .deb 安装包"

mkdir -p "$AGMP_ROOT/build/work"
cat > "$AGMP_ROOT/build/work/LINUX-RELEASE-STATUS.txt" <<STATUS
SUCCESS AI-Game-Manager-Panel $AGMP_VERSION
Desktop=$AGMP_BIN_DIR/AI-Game-Manager-Panel
Web=$AGMP_BIN_DIR/AI-Game-Manager-Panel-Web
Portable=$AGMP_RELEASE_DIR/AI-Game-Manager-Panel-Linux-x64.tar.gz
Deb=$AGMP_RELEASE_DIR/AI-Game-Manager-Panel_${AGMP_VERSION}_amd64.deb
STATUS

echo
echo "==============================================="
echo "LINUX RELEASE SUCCESS"
echo "Desktop:  build/work/linux/bin/AI-Game-Manager-Panel"
echo "Web:      build/work/linux/bin/AI-Game-Manager-Panel-Web"
echo "Portable: build/release/linux/AI-Game-Manager-Panel-Linux-x64.tar.gz"
echo "Deb:      build/release/linux/AI-Game-Manager-Panel_${AGMP_VERSION}_amd64.deb"
echo "==============================================="
