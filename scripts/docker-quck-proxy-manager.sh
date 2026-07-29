#!/usr/bin/env bash
#==============================================
# docker-quck-proxy Linux 管理脚本
# 支持：安装、卸载、配置、启停管理
#==============================================
set -euo pipefail

# ---------- 全局常量 ----------
APP_NAME="docker-quck-proxy"
GITHUB_REPO="lingoox/docker-quck-proxy"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/${APP_NAME}"
SERVICE_FILE="/etc/systemd/system/${APP_NAME}.service"
ENV_FILE="${CONFIG_DIR}/.env"
LOG_DIR="/var/log/${APP_NAME}"
BINARY="${INSTALL_DIR}/${APP_NAME}"
SERVICE_NAME="${APP_NAME}"
VERSION_FILE="${CONFIG_DIR}/version"

# ---------- 颜色输出 ----------
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

info()  { echo -e "${GREEN}[INFO]${NC} $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC} $*"; }
error() { echo -e "${RED}[ERROR]${NC} $*" >&2; }
title() { echo -e "\n${CYAN}══════════════════════════════════════════${NC}"; }
header(){ echo -e "${BLUE}$*${NC}"; }

# ---------- 依赖检查 ----------
check_root() {
    if [ "$(id -u)" -ne 0 ]; then
        error "请使用 root 权限运行（sudo）"
        exit 1
    fi
}

check_deps() {
    local missing=false
    for cmd in curl tar systemctl wget; do
        if ! command -v "$cmd" &>/dev/null; then
            error "缺少依赖: $cmd"
            missing=true
        fi
    done
    if [ "$missing" = true ]; then
        exit 1
    fi
}

# ---------- 平台检测 ----------
detect_platform() {
    local arch
    arch=$(uname -m)
    case "$arch" in
        x86_64|amd64)     echo "linux-amd64" ;;
        aarch64|arm64)    echo "linux-arm64" ;;
        armv7l|armv7hf)   echo "armv7hf-linux" ;;
        *)
            error "不支持的架构: $arch"
            exit 1
            ;;
    esac
}

# ---------- 版本获取 ----------
get_latest_version() {
    local version
    version=$(curl -s "https://api.github.com/repos/${GITHUB_REPO}/releases/latest" \
        | grep '"tag_name":' | sed 's/.*"v\(.*\)",/\1/' 2>/dev/null)
    if [ -z "$version" ]; then
        error "无法获取最新版本，请检查网络或 GitHub API 限制"
        return 1
    fi
    echo "$version"
}

get_installed_version() {
    if [ -f "$VERSION_FILE" ]; then
        cat "$VERSION_FILE"
    else
        echo "未安装"
    fi
}

# ---------- 配置文件读写 ----------
load_config() {
    if [ -f "$ENV_FILE" ]; then
        # shellcheck source=/dev/null
        source "$ENV_FILE"
    fi
    UPSTREAM="${UPSTREAM:-https://registry-1.docker.io}"
    LISTEN_ADDR="${LISTEN_ADDR:-:5000}"
    LOG_DIR="${LOG_DIR:-/var/log/${APP_NAME}}"
    LOG_ENABLED="${LOG_ENABLED:-false}"
    HTTP_PROXY="${HTTP_PROXY:-}"
    HTTPS_PROXY="${HTTPS_PROXY:-}"
}

save_config() {
    cat > "$ENV_FILE" <<-EOF
# docker-quck-proxy 配置文件
# 修改后执行: sudo systemctl restart ${APP_NAME}

UPSTREAM=${UPSTREAM}
LISTEN_ADDR=${LISTEN_ADDR}
LOG_DIR=${LOG_DIR}
LOG_ENABLED=${LOG_ENABLED}
HTTP_PROXY=${HTTP_PROXY}
HTTPS_PROXY=${HTTPS_PROXY}
EOF
    chmod 600 "$ENV_FILE"
    info "配置已保存到 $ENV_FILE"
}

show_config() {
    load_config
    echo ""
    header "══════════ ${APP_NAME} 当前配置 ══════════"
    printf "%-20s : %s\n" "UPSTREAM"      "${UPSTREAM}"
    printf "%-20s : %s\n" "LISTEN_ADDR"   "${LISTEN_ADDR}"
    printf "%-20s : %s\n" "LOG_DIR"       "${LOG_DIR}"
    printf "%-20s : %s\n" "LOG_ENABLED"   "${LOG_ENABLED}"
    printf "%-20s : %s\n" "HTTP_PROXY"    "${HTTP_PROXY:-<未设置>}"
    printf "%-20s : %s\n" "HTTPS_PROXY"   "${HTTPS_PROXY:-<未设置>}"
    header "═══════════════════════════════════════"
    echo ""
}

edit_config() {
    load_config

    title
    header "修改配置参数（留空则保持原值）"
    echo ""

    read -rp "上游地址 [${UPSTREAM}]: " input
    UPSTREAM="${input:-${UPSTREAM}}"

    read -rp "监听地址 [${LISTEN_ADDR}]: " input
    LISTEN_ADDR="${input:-${LISTEN_ADDR}}"

    read -rp "日志目录 [${LOG_DIR}]: " input
    LOG_DIR="${input:-${LOG_DIR}}"

    read -rp "启用日志 (true/false) [${LOG_ENABLED}]: " input
    LOG_ENABLED="${input:-${LOG_ENABLED}}"

    read -rp "HTTP 代理 [${HTTP_PROXY:-<空>}]: " input
    HTTP_PROXY="${input:-${HTTP_PROXY}}"

    read -rp "HTTPS 代理 [${HTTPS_PROXY:-<空>}]: " input
    HTTPS_PROXY="${input:-${HTTPS_PROXY}}"

    save_config

    if systemctl is-active --quiet "${SERVICE_NAME}" 2>/dev/null; then
        echo ""
        warn "配置已更改，建议重启服务以生效"
    fi
}

# ---------- 安装功能 ----------
download_and_install() {
    local version=$1
    local platform=$2
    local url
    local pkg_file

    if [ "$platform" = "linux-amd64" ]; then
        pkg_file="proxy-linux-amd64.tar.gz"
    elif [ "$platform" = "linux-arm64" ]; then
        pkg_file="proxy-linux-arm64.tar.gz"
    elif [ "$platform" = "armv7hf-linux" ]; then
        pkg_file="proxy-armv7hf-linux.tar.gz"
    else
        error "未知平台: ${platform}"
        return 1
    fi

    url="https://github.com/${GITHUB_REPO}/releases/download/v${version}/${pkg_file}"

    info "下载: ${url}"
    wget -q --show-progress -O "/tmp/${pkg_file}" "$url" || {
        error "下载失败，请检查版本号或网络"
        return 1
    }

    info "解压中..."
    tar xzf "/tmp/${pkg_file}" -C /tmp/ || {
        error "解压失败"
        return 1
    }

    # 查找二进制文件
    local binary_src
    binary_src=$(find /tmp -maxdepth 2 -name "proxy-*" -type f ! -name "*.tar.gz" 2>/dev/null | head -1)
    if [ -z "$binary_src" ]; then
        error "未找到二进制文件"
        return 1
    fi

    mkdir -p "$INSTALL_DIR"
    cp "$binary_src" "${BINARY}"
    chmod +x "${BINARY}"
    info "二进制已安装到: ${BINARY}"

    # 清理临时文件
    rm -rf "/tmp/${pkg_file}" "/tmp/proxy-"*

    echo "$version" > "$VERSION_FILE"
}

install_service() {
    mkdir -p "$CONFIG_DIR" "$LOG_DIR"

    cat > "$SERVICE_FILE" <<-SERVICEEOF
[Unit]
Description=docker-quck-proxy - Docker Hub mirror proxy
Documentation=https://github.com/${GITHUB_REPO}
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=${CONFIG_DIR}
EnvironmentFile=${ENV_FILE}
ExecStart=${BINARY} --upstream \${UPSTREAM} --port \${LISTEN_ADDR} --logdir \${LOG_DIR} --log-enabled \${LOG_ENABLED}
Restart=always
RestartSec=5
StandardOutput=append:${LOG_DIR}/proxy.log
StandardError=append:${LOG_DIR}/proxy-error.log

[Install]
WantedBy=multi-user.target
SERVICEEOF

    systemctl daemon-reload
    info "systemd 服务已安装: ${SERVICE_NAME}"
}

do_install() {
    check_root

    if [ -f "${BINARY}" ]; then
        local installed_ver
        installed_ver=$(get_installed_version)
        warn "${APP_NAME} 已安装（版本: ${installed_ver}）"
        read -rp "是否重新安装? [y/N]: " confirm
        if [[ ! "$confirm" =~ ^[Yy]$ ]]; then
            info "已取消安装"
            return
        fi
    fi

    title
    header "${APP_NAME} 安装程序"
    echo ""

    local version platform
    version=$(get_latest_version || echo "")

    if [ -z "$version" ]; then
        read -rp "请输入版本号（如 0.2.0）: " version
    else
        info "检测到最新版本: v${version}"
        read -rp "是否使用此版本? [Y/n]: " confirm
        if [[ "$confirm" =~ ^[Nn]$ ]]; then
            read -rp "请输入版本号（如 0.2.0）: " version
        fi
    fi

    platform=$(detect_platform)
    info "检测到平台: ${platform}"

    echo ""
    info "开始安装 ${APP_NAME} v${version}..."

    download_and_install "$version" "$platform" || return 1

    load_config
    save_config

    install_service

    echo ""
    info "✅ 安装完成!"
    info "版本: v${version}"
    info "二进制: ${BINARY}"
    info "配置: ${ENV_FILE}"
    info "日志: ${LOG_DIR}"
    echo ""
    info "启动服务: sudo systemctl start ${SERVICE_NAME}"
    info "查看状态: sudo systemctl status ${SERVICE_NAME}"
    echo ""
}

# ---------- 卸载功能 ----------
do_uninstall() {
    check_root

    title
    warn "警告: 这将完全卸载 ${APP_NAME}！"
    echo ""
    echo "以下文件/目录将被删除:"
    echo "  - ${BINARY}"
    echo "  - ${SERVICE_FILE}"
    echo "  - ${CONFIG_DIR}"
    echo "  - ${LOG_DIR}"
    echo "  - /tmp/proxy-*"
    echo ""

    read -rp "确认卸载? [y/N]: " confirm
    if [[ ! "$confirm" =~ ^[Yy]$ ]]; then
        info "已取消卸载"
        return
    fi

    if systemctl is-active --quiet "${SERVICE_NAME}" 2>/dev/null; then
        info "停止服务..."
        systemctl stop "${SERVICE_NAME}"
    fi

    if systemctl is-enabled --quiet "${SERVICE_NAME}" 2>/dev/null; then
        systemctl disable "${SERVICE_NAME}"
    fi

    rm -f "${BINARY}" "${SERVICE_FILE}"
    rm -rf "${CONFIG_DIR}" "${LOG_DIR}"
    systemctl daemon-reload

    info "✅ ${APP_NAME} 已完全卸载"
}

# ---------- 服务管理 ----------
do_status() {
    title
    header "${APP_NAME} 运行状态"
    echo ""

    if [ ! -f "${BINARY}" ]; then
        error "${APP_NAME} 未安装，请先安装"
        return
    fi

    local installed_ver
    installed_ver=$(get_installed_version)

    printf "%-18s: %s\n" "版本" "${installed_ver}"
    printf "%-18s: %s\n" "二进制路径" "${BINARY}"
    echo ""

    if systemctl is-active --quiet "${SERVICE_NAME}" 2>/dev/null; then
        echo -e "服务状态: ${GREEN}● 运行中${NC}"
    else
        echo -e "服务状态: ${RED}○ 已停止${NC}"
    fi

    systemctl status "${SERVICE_NAME}" --no-pager 2>&1 || true

    echo ""
    show_config

    # 显示最近的日志
    local logfile="${LOG_DIR}/proxy.log"
    if [ -f "$logfile" ]; then
        header "最近 5 条日志:"
        tail -5 "$logfile" 2>/dev/null || echo "(无日志)"
    fi
    echo ""
}

do_start() {
    check_root
    title
    header "启动 ${APP_NAME}..."
    systemctl start "${SERVICE_NAME}"
    sleep 1
    if systemctl is-active --quiet "${SERVICE_NAME}" 2>/dev/null; then
        info "✅ ${APP_NAME} 已启动"
    else
        error "启动失败，请检查: sudo journalctl -u ${SERVICE_NAME} -n 50"
    fi
}

do_stop() {
    check_root
    title
    header "停止 ${APP_NAME}..."
    if systemctl is-active --quiet "${SERVICE_NAME}" 2>/dev/null; then
        systemctl stop "${SERVICE_NAME}"
        info "✅ ${APP_NAME} 已停止"
    else
        warn "${APP_NAME} 未在运行"
    fi
}

do_restart() {
    check_root
    title
    header "重启 ${APP_NAME}..."
    if systemctl is-active --quiet "${SERVICE_NAME}" 2>/dev/null; then
        systemctl restart "${SERVICE_NAME}"
        sleep 1
        if systemctl is-active --quiet "${SERVICE_NAME}" 2>/dev/null; then
            info "✅ ${APP_NAME} 已重启"
        else
            error "重启失败，请检查: sudo journalctl -u ${SERVICE_NAME} -n 50"
        fi
    else
        warn "${APP_NAME} 未在运行，将直接启动..."
        systemctl start "${SERVICE_NAME}"
        info "✅ ${APP_NAME} 已启动"
    fi
}

# ---------- 主菜单 ----------
show_menu() {
    clear
    echo ""
    echo -e "${CYAN}█████████████████████████████████████████████████████${NC}"
    echo -e "${CYAN}██                                                   ██${NC}"
    echo -e "${CYAN}██          ${NC}docker-quck-proxy 管理面板${CYAN}             ██${NC}"
    echo -e "${CYAN}██                                                   ██${NC}"
    echo -e "${CYAN}█████████████████████████████████████████████████████${NC}"
    echo ""

    local installed_ver="(未安装)"
    if [ -f "${BINARY}" ]; then
        installed_ver=$(get_installed_version)
    fi
    echo -e "  版本: ${GREEN}${installed_ver}${NC}"

    if systemctl is-active --quiet "${SERVICE_NAME}" 2>/dev/null; then
        echo -e "  状态: ${GREEN}● 运行中${NC}"
    else
        echo -e "  状态: ${RED}○ 已停止${NC}"
    fi

    local platform
    platform=$(detect_platform 2>/dev/null || echo "unknown")
    echo -e "  平台: ${YELLOW}${platform}${NC}"
    echo ""

    header "═════════════════ 菜单 ═════════════════"
    echo ""
    echo -e "  ${GREEN}1${NC})  📥 安装"
    echo -e "  ${GREEN}2${NC})  🔄 重启"
    echo -e "  ${GREEN}3${NC})  📊 查看状态"
    echo -e "  ${GREEN}4${NC})  ⏹ 停止"
    echo -e "  ${GREEN}5${NC})  ⚙️ 切换配置参数"
    echo -e "  ${GREEN}6${NC})  📋 查看配置"
    echo -e "  ${GREEN}7${NC})  🗑 卸载"
    echo -e "  ${GREEN}0${NC})  🚪 退出"
    echo ""
    header "════════════════════════════════════════"
    echo ""
}

main() {
    # 非 root 时，有些操作需要 sudo
    if [ "$(id -u)" -ne 0 ]; then
        # 尝试用 sudo 重新执行自己
        exec sudo "$0" "$@"
    fi

    check_deps

    while true; do
        show_menu
        read -rp "请输入选项 [0-7]: " choice

        case "$choice" in
            1)
                do_install
                echo ""
                read -rp "按 Enter 返回菜单..."
                ;;
            2)
                do_restart
                echo ""
                read -rp "按 Enter 返回菜单..."
                ;;
            3)
                do_status
                read -rp "按 Enter 返回菜单..."
                ;;
            4)
                do_stop
                echo ""
                read -rp "按 Enter 返回菜单..."
                ;;
            5)
                edit_config
                echo ""
                read -rp "按 Enter 返回菜单..."
                ;;
            6)
                show_config
                read -rp "按 Enter 返回菜单..."
                ;;
            7)
                do_uninstall
                echo ""
                read -rp "按 Enter 返回菜单..."
                ;;
            0)
                info "感谢使用 ${APP_NAME}，再见!"
                exit 0
                ;;
            *)
                warn "无效选项，请重新选择"
                sleep 1
                ;;
        esac
    done
}

main "$@"
