#!/usr/bin/env bash
set -euo pipefail

# AI Game Manager Panel Linux Server Edition 安装器。
# 安装阶段需要 root 用于创建系统用户和 systemd 单元；服务进程默认以独立 ai-game-manager-panel 用户运行，不长期使用 root。

APP_NAME="AI Game Manager Panel Server"
SERVICE_NAME="${AGMP_SERVICE_NAME:-ai-game-manager-panel}"
SERVICE_USER="${AGMP_SERVICE_USER:-ai-game-manager-panel}"
INSTALL_ROOT="${AGMP_INSTALL_ROOT:-/opt/ai-game-manager-panel}"
CONFIG_ROOT="${AGMP_CONFIG_ROOT:-/etc/ai-game-manager-panel}"
RUNTIME_ROOT="${AGMP_RUNTIME_ROOT:-/var/lib/ai-game-manager-panel}"
LOG_ROOT="${AGMP_LOG_ROOT:-/var/log/ai-game-manager-panel}"
LISTEN="${AGMP_LISTEN:-127.0.0.1:17890}"
ALLOW_REMOTE_WEB="${AGMP_ALLOW_REMOTE_WEB:-0}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

red(){ printf '\033[31m%s\033[0m\n' "$*"; }
green(){ printf '\033[32m%s\033[0m\n' "$*"; }
cyan(){ printf '\033[36m%s\033[0m\n' "$*"; }
yellow(){ printf '\033[33m%s\033[0m\n' "$*"; }

require_root() {
  if [[ "$(id -u)" -ne 0 ]]; then
    red "安装系统服务需要 root。请使用 sudo bash install.sh 或切换 root 后执行。"
    exit 1
  fi
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64) echo amd64 ;;
    aarch64|arm64) echo arm64 ;;
    *) red "当前 CPU 架构暂不支持：$(uname -m)"; exit 1 ;;
  esac
}

detect_distro() {
  if [[ -r /etc/os-release ]]; then
    . /etc/os-release
    echo "${ID:-unknown}"
  else
    echo unknown
  fi
}

install_local() {
  require_root
  local arch distro binary xiaoyu remote_flag
  arch="$(detect_arch)"
  distro="$(detect_distro)"
  binary="$SCRIPT_DIR/ai-game-manager-panel-server"
  xiaoyu="$SCRIPT_DIR/internal/xiaoyu/AI-Game-Manager-XiaoYu"
  [[ -x "$binary" ]] || { red "未找到同目录 ai-game-manager-panel-server 可执行文件。"; exit 1; }
  [[ -x "$xiaoyu" ]] || { red "当前 Server Edition 缺少 XiaoYu Runtime；为防止产生无 AI 的 Web 版本，安装已终止。"; exit 1; }
  remote_flag=""
  case "${ALLOW_REMOTE_WEB,,}" in 1|true|yes|on) remote_flag=" -allow-remote-web" ;; esac

  cyan "检测到系统：$distro / $arch"
  if ! id "$SERVICE_USER" >/dev/null 2>&1; then
    useradd --system --home "$RUNTIME_ROOT" --shell /usr/sbin/nologin "$SERVICE_USER"
  fi

  install -d -m 0755 "$INSTALL_ROOT" "$CONFIG_ROOT" "$RUNTIME_ROOT" "$LOG_ROOT"
  install -d -m 0755 "$RUNTIME_ROOT/runtime" "$RUNTIME_ROOT/runtime/data" "$RUNTIME_ROOT/runtime/log" "$RUNTIME_ROOT/runtime/backups" "$RUNTIME_ROOT/runtime/instances" "$RUNTIME_ROOT/runtime/temp" "$RUNTIME_ROOT/runtime/exports" "$RUNTIME_ROOT/runtime/plugins" "$RUNTIME_ROOT/runtime/cache"
  install -m 0755 "$binary" "$INSTALL_ROOT/ai-game-manager-panel-server"
  install -d -m 0755 "$INSTALL_ROOT/internal/xiaoyu"
  install -m 0755 "$xiaoyu" "$INSTALL_ROOT/internal/xiaoyu/AI-Game-Manager-XiaoYu"
  if [[ -d "$SCRIPT_DIR/configs-default" && ! -e "$CONFIG_ROOT/app.json" ]]; then
    cp -R "$SCRIPT_DIR/configs-default/." "$CONFIG_ROOT/"
  fi
  chown -R "$SERVICE_USER:$SERVICE_USER" "$RUNTIME_ROOT" "$LOG_ROOT"

  cat > "/etc/systemd/system/$SERVICE_NAME.service" <<UNIT
[Unit]
Description=AI Game Manager Panel Game Server Management Platform
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=$SERVICE_USER
Group=$SERVICE_USER
WorkingDirectory=$RUNTIME_ROOT
Environment=AGMP_ROOT=$RUNTIME_ROOT
Environment=AGMP_XIAOYU_RUNTIME=$INSTALL_ROOT/internal/xiaoyu/AI-Game-Manager-XiaoYu
ExecStart=$INSTALL_ROOT/ai-game-manager-panel-server -listen $LISTEN$remote_flag
Restart=on-failure
RestartSec=3
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=full
ProtectHome=true
ReadWritePaths=$RUNTIME_ROOT $LOG_ROOT
LimitNOFILE=1048576

[Install]
WantedBy=multi-user.target
UNIT

  # 首次运行配置复制到运行根目录。程序统一从 AGMP_ROOT/configs 读取。
  install -d -o "$SERVICE_USER" -g "$SERVICE_USER" "$RUNTIME_ROOT/configs"
  if [[ -d "$CONFIG_ROOT" ]]; then
    cp -an "$CONFIG_ROOT/." "$RUNTIME_ROOT/configs/" || true
    chown -R "$SERVICE_USER:$SERVICE_USER" "$RUNTIME_ROOT/configs"
  fi

  systemctl daemon-reload
  systemctl enable --now "$SERVICE_NAME"
  green "$APP_NAME 安装完成。"
  green "XiaoYu Runtime：$INSTALL_ROOT/internal/xiaoyu/AI-Game-Manager-XiaoYu"
  green "Web 管理监听：http://$LISTEN"
  if [[ -z "$remote_flag" ]]; then
    yellow "安全默认：仅允许 loopback。公网域名部署建议由 Caddy/Nginx 提供 HTTPS，再反向代理到 127.0.0.1:${LISTEN##*:}。"
  else
    yellow "已显式允许非 loopback 监听。请务必在前方配置 HTTPS、防火墙和可信反向代理，不要裸露 HTTP 管理端。"
  fi
}

status(){ systemctl status "$SERVICE_NAME" --no-pager || true; }
logs(){ journalctl -u "$SERVICE_NAME" -n 200 -f; }

menu() {
  clear || true
  green "AI游戏管理器面板 Linux Server Edition"
  echo "-----------------------------------------------"
  echo "[0] 安装并启动 AI Game Manager Panel"
  echo "[1] 启动 AI Game Manager Panel"
  echo "[2] 停止 AI Game Manager Panel"
  echo "[3] 重启 AI Game Manager Panel"
  echo "[4] 查看状态"
  echo "[5] 查看实时日志"
  echo "[6] 设置开机自启"
  echo "[7] 取消开机自启"
  echo "[8] 环境诊断"
  echo "[9] 退出"
  echo "-----------------------------------------------"
  read -r -p "请输入要执行的操作 [0-9]: " choice
  case "$choice" in
    0) install_local ;;
    1) require_root; systemctl start "$SERVICE_NAME" ;;
    2) require_root; systemctl stop "$SERVICE_NAME" ;;
    3) require_root; systemctl restart "$SERVICE_NAME" ;;
    4) status ;;
    5) logs ;;
    6) require_root; systemctl enable "$SERVICE_NAME" ;;
    7) require_root; systemctl disable "$SERVICE_NAME" ;;
    8) echo "系统=$(detect_distro) 架构=$(detect_arch)"; command -v systemctl; command -v tar; command -v curl || true; if [[ -x "$INSTALL_ROOT/internal/xiaoyu/AI-Game-Manager-XiaoYu" ]]; then AGMP_ROOT="$RUNTIME_ROOT" "$INSTALL_ROOT/internal/xiaoyu/AI-Game-Manager-XiaoYu" --root "$RUNTIME_ROOT" doctor --json || true; else yellow "XiaoYu Runtime 尚未安装"; fi ;;
    9) exit 0 ;;
    *) red "无效选项"; exit 2 ;;
  esac
}

if [[ "${1:-}" == "--install" ]]; then
  install_local
else
  menu
fi
