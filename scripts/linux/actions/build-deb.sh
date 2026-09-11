#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "$0")/../lib/init.sh"

STAGE="$AGMP_ROOT/build/work/deb-amd64"
APPDIR="$STAGE/usr/lib/ai-game-manager-panel"
BINDIR="$STAGE/usr/bin"
DESKTOPDIR="$STAGE/usr/share/applications"
ICONDIR="$STAGE/usr/share/icons/hicolor/256x256/apps"
rm -rf "$STAGE"
mkdir -p "$STAGE/DEBIAN" "$APPDIR/internal/xiaoyu" "$BINDIR" "$DESKTOPDIR" "$ICONDIR"

cp "$AGMP_BIN_DIR/AI-Game-Manager-Panel" "$APPDIR/AI-Game-Manager-Panel"
cp "$AGMP_BIN_DIR/AI-Game-Manager-Panel-Web" "$APPDIR/AI-Game-Manager-Panel-Web"
cp "$AGMP_BIN_DIR/AI-Game-Manager-XiaoYu" "$APPDIR/internal/xiaoyu/AI-Game-Manager-XiaoYu"
cp -R "$AGMP_ROOT/configs" "$APPDIR/configs-default"
cp -R "$AGMP_ROOT/distribution/licenses" "$APPDIR/licenses"
cp "$AGMP_ROOT/README.md" "$APPDIR/README.md"
if [[ -f "$AGMP_ROOT/build/appicon.png" ]]; then cp "$AGMP_ROOT/build/appicon.png" "$ICONDIR/ai-game-manager-panel.png"; fi

cat > "$STAGE/DEBIAN/control" <<EOF2
Package: ai-game-manager-panel
Version: $AGMP_VERSION
Section: games
Priority: optional
Architecture: amd64
Maintainer: AI-Game-Manager-Panel Project
Depends: libgtk-3-0, ${AGMP_WEBKIT_DEP}
Description: AI Game Manager Panel - 现代化智能 AI 一键游戏服务器部署与管理平台
EOF2

# .deb 中程序本体位于只读系统目录；启动器负责切换到当前用户可写的 XDG 数据目录。
cp "$AGMP_ROOT/distribution/installer/linux/ai-game-manager-panel" "$BINDIR/ai-game-manager-panel"
cp "$AGMP_ROOT/distribution/installer/linux/ai-game-manager-panel-web" "$BINDIR/ai-game-manager-panel-web"
cp "$AGMP_ROOT/distribution/installer/linux/ai-game-manager-panel.desktop" "$DESKTOPDIR/ai-game-manager-panel.desktop"
chmod +x "$BINDIR/ai-game-manager-panel" "$BINDIR/ai-game-manager-panel-web" "$APPDIR/AI-Game-Manager-Panel" "$APPDIR/AI-Game-Manager-Panel-Web" "$APPDIR/internal/xiaoyu/AI-Game-Manager-XiaoYu"

dpkg-deb --build --root-owner-group "$STAGE" "$AGMP_RELEASE_DIR/AI-Game-Manager-Panel_${AGMP_VERSION}_amd64.deb" >/dev/null
