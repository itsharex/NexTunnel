#!/usr/bin/env bash
set -Eeuo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LOCAL_ENV_FILE="${SCRIPT_DIR}/.env"

# 允许用户复制 .env.example 为 .env 后用文件覆盖默认部署参数。
if [[ -f "${LOCAL_ENV_FILE}" ]]; then
  set -a
  # shellcheck disable=SC1090
  source "${LOCAL_ENV_FILE}"
  set +a
fi

ACTION="install"
if [[ $# -gt 0 && "${1}" != --* ]]; then
  ACTION="$1"
  shift
fi

DEFAULT_REPOSITORY="Lee-zg/NexTunnel"
REPOSITORY="${NEXTUNNEL_REPOSITORY:-${DEFAULT_REPOSITORY}}"
VERSION="${NEXTUNNEL_VERSION:-latest}"
RELEASE_BASE_URL="${NEXTUNNEL_RELEASE_BASE_URL:-}"
GITHUB_PROXY="${NEXTUNNEL_GITHUB_PROXY:-}"
PACKAGE_URL="${NEXTUNNEL_PACKAGE_URL:-}"
PACKAGE_SHA256="${NEXTUNNEL_PACKAGE_SHA256:-}"
ARCH="${NEXTUNNEL_ARCH:-}"
INSTALL_DIR="${NEXTUNNEL_INSTALL_DIR:-/opt/nextunnel}"
CONFIG_DIR="${NEXTUNNEL_CONFIG_DIR:-/etc/nextunnel}"
DATA_DIR="${NEXTUNNEL_DATA_DIR:-/var/lib/nextunnel}"
SERVICE_USER="${NEXTUNNEL_SERVICE_USER:-nextunnel}"
SERVICE_GROUP="${NEXTUNNEL_SERVICE_GROUP:-${SERVICE_USER}}"
CLI_LINK_PATH="${NEXTUNNEL_CLI_LINK_PATH:-/usr/local/bin/nextunnel}"
NON_INTERACTIVE="${NON_INTERACTIVE:-false}"
FORCE="${FORCE:-false}"
PURGE="${PURGE:-false}"

PUBLIC_HOST="${NEXTUNNEL_PUBLIC_HOST:-127.0.0.1}"
RELAY_BIND="${RELAY_BIND:-0.0.0.0}"
RELAY_CONTROL_PORT="${RELAY_CONTROL_PORT:-7000}"
RELAY_QUIC_PORT="${RELAY_QUIC_PORT:-7443}"
RELAY_AUTH_TOKEN="${RELAY_AUTH_TOKEN:-}"
RELAY_REQUIRE_AUTH="${RELAY_REQUIRE_AUTH:-true}"
RELAY_STATS_INTERVAL="${RELAY_STATS_INTERVAL:-30s}"
CONTROL_PLANE_PORT="${CONTROL_PLANE_PORT:-9090}"
CONTROL_PLANE_API_TOKEN="${CONTROL_PLANE_API_TOKEN:-}"
DASHBOARD_ENABLED="${DASHBOARD_ENABLED:-true}"
DASHBOARD_PORT="${DASHBOARD_PORT:-8080}"
DASHBOARD_SECRET_KEY="${DASHBOARD_SECRET_KEY:-}"
DASHBOARD_ADMIN_USER="${DASHBOARD_ADMIN_USER:-admin}"
DASHBOARD_ADMIN_PASSWORD="${DASHBOARD_ADMIN_PASSWORD:-}"
DASHBOARD_ALLOWED_ORIGINS="${DASHBOARD_ALLOWED_ORIGINS:-http://127.0.0.1:8080,http://localhost:8080}"
NAT_PRIMARY_ADDR="${NAT_PRIMARY_ADDR:-0.0.0.0}"
NAT_ALT_ADDR="${NAT_ALT_ADDR:-127.0.0.1}"
NAT_PORT="${NAT_PORT:-3478}"
NAT_REALM="${NAT_REALM:-nextunnel.local}"

ENV_FILE="${CONFIG_DIR%/}/server.env"
BIN_DIR="${INSTALL_DIR%/}/bin"
WEB_DIR="${INSTALL_DIR%/}/web/dashboard"
DEPLOY_DIR="${INSTALL_DIR%/}/deploy/server"
SERVICE_NAMES=(nextunnel-control-plane.service nextunnel-relay.service nextunnel-nat-detector.service)
DASHBOARD_SERVICE_NAME=nextunnel-dashboard.service

log() {
  printf '[NexTunnel] %s\n' "$1"
}

warn() {
  printf '[NexTunnel] %s\n' "$1" >&2
}

usage() {
  cat <<'EOF'
用法：
  ./install.sh [install|up|down|restart|status|logs|health|update|uninstall|config] [选项]

常用选项：
  --package-url URL        指定服务端 Release 包地址，支持 https://、file:// 或本地文件路径
  --sha256 HASH            可选，校验服务端 Release 包 SHA256
  --version VERSION        指定 GitHub Release 版本，例如 v0.3.1-alpha；默认 latest
  --release-base-url URL   指定 Release 下载基址；默认使用 GitHub Releases
  --github-proxy URL       可选，仅代理脚本自动生成的 GitHub 下载地址；生产环境建议使用可信自建代理
  --arch ARCH              指定架构 amd64/arm64；默认自动识别
  --public-host HOST       指定客户端访问的公网 IP 或域名
  --relay-token TOKEN      指定 Relay 认证 Token
  --control-token TOKEN    指定 Control Plane API Token
  --dashboard-port PORT    指定 Dashboard HTTP 端口，默认 8080
  --dashboard-password PWD 指定 Dashboard 默认管理员密码，首次初始化必需
  --dashboard-disabled     仅部署核心服务，不启动 Dashboard
  --service-user USER      指定 systemd 服务运行用户，默认 nextunnel
  --service-group GROUP    指定 systemd 服务运行用户组，默认与用户同名
  --cli-link PATH          指定 nextunnel CLI 软链接路径，默认 /usr/local/bin/nextunnel；设置 none 可跳过
  --non-interactive        使用环境变量/默认值，不进入交互输入
  --force                  重新生成配置文件
  --purge                  uninstall 时同时删除安装目录、配置和数据

配置方式优先级：
  命令行选项 > 环境变量 > deploy/server/.env > 交互输入/默认值
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --package-url)
      PACKAGE_URL="${2:?--package-url 需要参数}"
      shift 2
      ;;
    --version)
      VERSION="${2:?--version 需要参数}"
      shift 2
      ;;
    --sha256)
      PACKAGE_SHA256="${2:?--sha256 需要参数}"
      shift 2
      ;;
    --release-base-url)
      RELEASE_BASE_URL="${2:?--release-base-url 需要参数}"
      shift 2
      ;;
    --github-proxy)
      GITHUB_PROXY="${2:?--github-proxy 需要参数}"
      shift 2
      ;;
    --repository)
      REPOSITORY="${2:?--repository 需要参数}"
      shift 2
      ;;
    --arch)
      ARCH="${2:?--arch 需要参数}"
      shift 2
      ;;
    --install-dir)
      INSTALL_DIR="${2:?--install-dir 需要参数}"
      BIN_DIR="${INSTALL_DIR%/}/bin"
      WEB_DIR="${INSTALL_DIR%/}/web/dashboard"
      shift 2
      ;;
    --config-dir)
      CONFIG_DIR="${2:?--config-dir 需要参数}"
      ENV_FILE="${CONFIG_DIR%/}/server.env"
      shift 2
      ;;
    --data-dir)
      DATA_DIR="${2:?--data-dir 需要参数}"
      shift 2
      ;;
    --public-host)
      PUBLIC_HOST="${2:?--public-host 需要参数}"
      shift 2
      ;;
    --relay-port)
      RELAY_CONTROL_PORT="${2:?--relay-port 需要参数}"
      shift 2
      ;;
    --relay-quic-port)
      RELAY_QUIC_PORT="${2:?--relay-quic-port 需要参数}"
      shift 2
      ;;
    --control-plane-port)
      CONTROL_PLANE_PORT="${2:?--control-plane-port 需要参数}"
      shift 2
      ;;
    --dashboard-port)
      DASHBOARD_PORT="${2:?--dashboard-port 需要参数}"
      shift 2
      ;;
    --nat-port)
      NAT_PORT="${2:?--nat-port 需要参数}"
      shift 2
      ;;
    --relay-token)
      RELAY_AUTH_TOKEN="${2:?--relay-token 需要参数}"
      shift 2
      ;;
    --control-token)
      CONTROL_PLANE_API_TOKEN="${2:?--control-token 需要参数}"
      shift 2
      ;;
    --dashboard-secret)
      DASHBOARD_SECRET_KEY="${2:?--dashboard-secret 需要参数}"
      shift 2
      ;;
    --dashboard-admin)
      DASHBOARD_ADMIN_USER="${2:?--dashboard-admin 需要参数}"
      shift 2
      ;;
    --dashboard-password)
      DASHBOARD_ADMIN_PASSWORD="${2:?--dashboard-password 需要参数}"
      shift 2
      ;;
    --dashboard-origins)
      DASHBOARD_ALLOWED_ORIGINS="${2:?--dashboard-origins 需要参数}"
      shift 2
      ;;
    --dashboard-disabled)
      DASHBOARD_ENABLED="false"
      shift
      ;;
    --service-user)
      SERVICE_USER="${2:?--service-user 需要参数}"
      if [[ -z "${NEXTUNNEL_SERVICE_GROUP:-}" ]]; then
        SERVICE_GROUP="${SERVICE_USER}"
      fi
      shift 2
      ;;
    --service-group)
      SERVICE_GROUP="${2:?--service-group 需要参数}"
      shift 2
      ;;
    --cli-link)
      CLI_LINK_PATH="${2:?--cli-link 需要参数}"
      shift 2
      ;;
    --non-interactive)
      NON_INTERACTIVE="true"
      shift
      ;;
    --force)
      FORCE="true"
      shift
      ;;
    --purge)
      PURGE="true"
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      printf '未知选项：%s\n' "$1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

require_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    printf '缺少命令：%s\n' "$1" >&2
    exit 1
  fi
}

require_root() {
  if [[ "$(id -u)" -ne 0 ]]; then
    printf '该操作需要 root 权限，请使用 sudo 重新执行。\n' >&2
    exit 1
  fi
}

random_secret() {
  if command -v openssl >/dev/null 2>&1; then
    openssl rand -base64 32 | tr '+/' '-_' | tr -d '='
    return
  fi
  head -c 32 /dev/urandom | base64 | tr '+/' '-_' | tr -d '='
}

read_setting() {
  local prompt="$1"
  local default_value="$2"
  local value=""
  if [[ "${NON_INTERACTIVE}" == "true" ]]; then
    printf '%s' "${default_value}"
    return
  fi
  read -r -p "${prompt} [${default_value}]: " value
  if [[ -z "${value}" ]]; then
    printf '%s' "${default_value}"
  else
    printf '%s' "${value}"
  fi
}

validate_env_value() {
  local name="$1"
  local value="$2"
  if [[ "${value}" == *$'\n'* || "${value}" == *$'\r'* ]]; then
    printf '配置项 %s 不能包含换行符。\n' "${name}" >&2
    exit 1
  fi
}

validate_port() {
  local name="$1"
  local value="$2"
  if ! [[ "${value}" =~ ^[0-9]+$ ]] || (( value < 1 || value > 65535 )); then
    printf '配置项 %s 必须是 1-65535 之间的端口号，当前值：%s\n' "${name}" "${value}" >&2
    exit 1
  fi
}

write_env_line() {
  local name="$1"
  local value="$2"
  validate_env_value "${name}" "${value}"
  printf '%s=%s\n' "${name}" "${value}"
}

normalize_arch() {
  local raw_arch="${1:-$(uname -m)}"
  case "${raw_arch}" in
    x86_64|amd64)
      printf 'amd64'
      ;;
    aarch64|arm64)
      printf 'arm64'
      ;;
    *)
      printf '不支持的 CPU 架构：%s，仅支持 amd64/arm64。\n' "${raw_arch}" >&2
      exit 1
      ;;
  esac
}

resolve_release_base_url() {
  if [[ -n "${RELEASE_BASE_URL}" ]]; then
    printf '%s' "${RELEASE_BASE_URL%/}"
    return
  fi
  local github_release_url
  if [[ "${VERSION}" == "latest" ]]; then
    github_release_url="$(printf 'https://github.com/%s/releases/latest/download' "${REPOSITORY}")"
  else
    github_release_url="$(printf 'https://github.com/%s/releases/download/%s' "${REPOSITORY}" "${VERSION}")"
  fi
  if [[ -z "${GITHUB_PROXY}" ]]; then
    printf '%s' "${github_release_url}"
    return
  fi
  # 仅在用户显式配置时改写 GitHub URL，避免默认使用不可信第三方代理。
  if [[ "${GITHUB_PROXY}" == *"{url}"* ]]; then
    printf '%s' "${GITHUB_PROXY/\{url\}/${github_release_url}}"
    return
  fi
  printf '%s/%s' "${GITHUB_PROXY%/}" "${github_release_url}"
}

resolve_package_url() {
  if [[ -n "${PACKAGE_URL}" ]]; then
    printf '%s' "${PACKAGE_URL}"
    return
  fi
  local resolved_arch
  resolved_arch="$(normalize_arch "${ARCH}")"
  printf '%s/nextunnel-server-linux-%s.tar.gz' "$(resolve_release_base_url)" "${resolved_arch}"
}

download_file() {
  local source="$1"
  local target="$2"
  case "${source}" in
    http://*|https://*)
      if command -v curl >/dev/null 2>&1; then
        curl -fL --retry 3 --connect-timeout 15 -o "${target}" "${source}"
      elif command -v wget >/dev/null 2>&1; then
        wget -O "${target}" "${source}"
      else
        printf '缺少 curl 或 wget，无法下载 Release 包。\n' >&2
        exit 1
      fi
      ;;
    file://*)
      cp "${source#file://}" "${target}"
      ;;
    *)
      if [[ -f "${source}" ]]; then
        cp "${source}" "${target}"
      else
        printf '无法识别的包地址或本地文件不存在：%s\n' "${source}" >&2
        exit 1
      fi
      ;;
  esac
}

verify_file_checksum() {
  local file_path="$1"
  if [[ -z "${PACKAGE_SHA256}" ]]; then
    return
  fi
  require_command sha256sum
  local actual_checksum
  actual_checksum="$(sha256sum "${file_path}" | awk '{print $1}')"
  if [[ "${actual_checksum,,}" != "${PACKAGE_SHA256,,}" ]]; then
    printf 'Release 包 SHA256 校验失败，期望 %s，实际 %s。\n' "${PACKAGE_SHA256}" "${actual_checksum}" >&2
    exit 1
  fi
}

find_binary() {
  local root_dir="$1"
  local binary_name="$2"
  local found_path
  found_path="$(find "${root_dir}" -type f \( -path "*/bin/${binary_name}" -o -name "${binary_name}" \) -print -quit)"
  if [[ -z "${found_path}" ]]; then
    printf 'Release 包中未找到二进制：%s\n' "${binary_name}" >&2
    exit 1
  fi
  printf '%s' "${found_path}"
}

optional_find_binary() {
  local root_dir="$1"
  local binary_name="$2"
  find "${root_dir}" -type f \( -path "*/bin/${binary_name}" -o -name "${binary_name}" \) -print -quit
}

find_dashboard_web_dir() {
  local root_dir="$1"
  local found_dir
  found_dir="$(find "${root_dir}" -type f -path "*/web/dashboard/index.html" -print -quit)"
  if [[ -n "${found_dir}" ]]; then
    dirname "${found_dir}"
  fi
}

find_deploy_script_dir() {
  local root_dir="$1"
  local found_file
  found_file="$(find "${root_dir}" -type f -path "*/deploy/server/install.sh" -print -quit)"
  if [[ -n "${found_file}" ]]; then
    dirname "${found_file}"
  fi
}

runtime_service_names() {
  printf '%s\n' "${SERVICE_NAMES[@]}"
  if [[ "${DASHBOARD_ENABLED}" == "true" && -x "${BIN_DIR}/dashboard" ]]; then
    printf '%s\n' "${DASHBOARD_SERVICE_NAME}"
  fi
}

collect_runtime_services() {
  if [[ -f "${ENV_FILE}" ]]; then
    set -a
    # shellcheck disable=SC1090
    source "${ENV_FILE}"
    set +a
  fi
  mapfile -t RUNTIME_SERVICE_NAMES < <(runtime_service_names)
}

create_system_user() {
  if [[ -z "${SERVICE_USER}" || "${SERVICE_USER}" == "root" ]]; then
    SERVICE_USER="root"
    SERVICE_GROUP="root"
    return
  fi
  if id "${SERVICE_USER}" >/dev/null 2>&1; then
    if ! getent group "${SERVICE_GROUP}" >/dev/null 2>&1; then
      SERVICE_GROUP="$(id -gn "${SERVICE_USER}")"
    fi
    return
  fi

  local nologin_shell="/usr/sbin/nologin"
  [[ -x "${nologin_shell}" ]] || nologin_shell="/sbin/nologin"

  if command -v useradd >/dev/null 2>&1; then
    if ! getent group "${SERVICE_GROUP}" >/dev/null 2>&1; then
      groupadd --system "${SERVICE_GROUP}"
    fi
    useradd --system --no-create-home --home-dir "${DATA_DIR}" --shell "${nologin_shell}" --gid "${SERVICE_GROUP}" "${SERVICE_USER}"
    return
  fi
  if command -v adduser >/dev/null 2>&1; then
    if ! getent group "${SERVICE_GROUP}" >/dev/null 2>&1 && command -v addgroup >/dev/null 2>&1; then
      addgroup -S "${SERVICE_GROUP}"
    fi
    adduser -S -D -h "${DATA_DIR}" -s "${nologin_shell}" -G "${SERVICE_GROUP}" "${SERVICE_USER}"
    return
  fi

  warn "未找到 useradd/adduser，服务将以 root 运行。"
  SERVICE_USER="root"
  SERVICE_GROUP="root"
}

generate_env() {
  if [[ -f "${ENV_FILE}" && "${FORCE}" != "true" ]]; then
    warn "配置文件已存在，保留当前配置：${ENV_FILE}。使用 --force 可重新生成。"
    return
  fi

  local public_host="${PUBLIC_HOST}"
  local relay_port="${RELAY_CONTROL_PORT}"
  local relay_quic_port="${RELAY_QUIC_PORT}"
  local control_port="${CONTROL_PLANE_PORT}"
  local dashboard_port="${DASHBOARD_PORT}"
  local nat_port="${NAT_PORT}"
  local relay_token="${RELAY_AUTH_TOKEN:-$(random_secret)}"
  local control_token="${CONTROL_PLANE_API_TOKEN:-$(random_secret)}"
  local dashboard_secret="${DASHBOARD_SECRET_KEY:-$(random_secret)}"
  local dashboard_password="${DASHBOARD_ADMIN_PASSWORD:-$(random_secret)}"

  public_host="$(read_setting '公网 IP 或域名' "${public_host}")"
  relay_port="$(read_setting 'Relay TCP 端口' "${relay_port}")"
  relay_quic_port="$(read_setting 'Relay QUIC UDP 端口' "${relay_quic_port}")"
  control_port="$(read_setting 'Control Plane HTTP 端口' "${control_port}")"
  if [[ "${DASHBOARD_ENABLED}" == "true" ]]; then
    dashboard_port="$(read_setting 'Dashboard HTTP 端口' "${dashboard_port}")"
  fi
  nat_port="$(read_setting 'NAT Detector UDP 端口' "${nat_port}")"

  if [[ "${NON_INTERACTIVE}" != "true" ]]; then
    relay_token="$(read_setting 'Relay 共享认证 Token' "${relay_token}")"
    control_token="$(read_setting 'Control Plane Bearer Token' "${control_token}")"
    if [[ "${DASHBOARD_ENABLED}" == "true" ]]; then
      dashboard_password="$(read_setting 'Dashboard 管理员密码' "${dashboard_password}")"
    fi
  fi

  if [[ -z "${relay_token}" || -z "${control_token}" ]]; then
    printf 'RelayToken 和 ControlToken 不能为空。\n' >&2
    exit 1
  fi
  if [[ "${DASHBOARD_ENABLED}" == "true" && ( -z "${dashboard_secret}" || -z "${dashboard_password}" ) ]]; then
    printf 'DashboardSecret 和 DashboardAdminPassword 不能为空。\n' >&2
    exit 1
  fi

  validate_port RELAY_CONTROL_PORT "${relay_port}"
  validate_port RELAY_QUIC_PORT "${relay_quic_port}"
  validate_port CONTROL_PLANE_PORT "${control_port}"
  validate_port DASHBOARD_PORT "${dashboard_port}"
  validate_port NAT_PORT "${nat_port}"

  mkdir -p "${CONFIG_DIR}" "${DATA_DIR}"
  {
    printf '# NexTunnel 服务端运行配置，由 deploy/server/install.sh 生成\n'
    write_env_line NEXTUNNEL_PUBLIC_HOST "${public_host}"
    write_env_line RELAY_BIND "${RELAY_BIND}"
    write_env_line RELAY_CONTROL_PORT "${relay_port}"
    write_env_line RELAY_QUIC_PORT "${relay_quic_port}"
    write_env_line RELAY_AUTH_TOKEN "${relay_token}"
    write_env_line RELAY_REQUIRE_AUTH "${RELAY_REQUIRE_AUTH}"
    write_env_line RELAY_STATS_INTERVAL "${RELAY_STATS_INTERVAL}"
    write_env_line CONTROL_PLANE_LISTEN "0.0.0.0:${control_port}"
    write_env_line CONTROL_PLANE_PORT "${control_port}"
    write_env_line CONTROL_PLANE_API_TOKEN "${control_token}"
    write_env_line CONTROL_PLANE_STORE_PATH "${DATA_DIR%/}/control-plane.db"
    write_env_line DASHBOARD_ENABLED "${DASHBOARD_ENABLED}"
    write_env_line DASHBOARD_LISTEN "0.0.0.0:${dashboard_port}"
    write_env_line DASHBOARD_PORT "${dashboard_port}"
    write_env_line DASHBOARD_SECRET_KEY "${dashboard_secret}"
    write_env_line DASHBOARD_ADMIN_USER "${DASHBOARD_ADMIN_USER}"
    write_env_line DASHBOARD_ADMIN_PASSWORD "${dashboard_password}"
    write_env_line DASHBOARD_ALLOWED_ORIGINS "${DASHBOARD_ALLOWED_ORIGINS},http://${public_host}:${dashboard_port}"
    write_env_line DASHBOARD_STORE_PATH "${DATA_DIR%/}/dashboard.db"
    write_env_line DASHBOARD_STATIC_DIR "${WEB_DIR}"
    write_env_line NAT_PRIMARY_ADDR "${NAT_PRIMARY_ADDR}"
    write_env_line NAT_ALT_ADDR "${NAT_ALT_ADDR}"
    write_env_line NAT_PORT "${nat_port}"
    write_env_line NAT_REALM "${NAT_REALM}"
    write_env_line NEXTUNNEL_DATA_DIR "${DATA_DIR}"
  } >"${ENV_FILE}"
  chmod 600 "${ENV_FILE}"
  log "已生成配置：${ENV_FILE}"
}

load_runtime_env() {
  if [[ ! -f "${ENV_FILE}" ]]; then
    printf '未找到配置文件：%s，请先执行 install 或 config。\n' "${ENV_FILE}" >&2
    exit 1
  fi
  set -a
  # shellcheck disable=SC1090
  source "${ENV_FILE}"
  set +a
}

install_release_package() {
  require_command tar

  local resolved_package_url
  local tmp_dir
  local archive_path
  local extract_dir
  local relay_binary
  local control_binary
  local nat_binary
  local cli_binary
  local dashboard_binary
  local dashboard_web_dir
  local deploy_script_dir

  resolved_package_url="$(resolve_package_url)"
  tmp_dir="$(mktemp -d)"
  archive_path="${tmp_dir}/nextunnel-server.tar.gz"
  extract_dir="${tmp_dir}/extract"
  mkdir -p "${extract_dir}"
  trap "rm -rf '${tmp_dir}'" EXIT

  log "下载服务端 Release 包：${resolved_package_url}"
  download_file "${resolved_package_url}" "${archive_path}"
  verify_file_checksum "${archive_path}"

  log "解压并校验服务端二进制"
  tar -xzf "${archive_path}" -C "${extract_dir}"
  relay_binary="$(find_binary "${extract_dir}" relay-server)"
  control_binary="$(find_binary "${extract_dir}" control-plane)"
  nat_binary="$(find_binary "${extract_dir}" nat-detector)"
  cli_binary="$(optional_find_binary "${extract_dir}" nextunnel)"
  dashboard_binary="$(optional_find_binary "${extract_dir}" dashboard)"
  dashboard_web_dir="$(find_dashboard_web_dir "${extract_dir}")"
  deploy_script_dir="$(find_deploy_script_dir "${extract_dir}")"

  mkdir -p "${BIN_DIR}" "${WEB_DIR}" "${DEPLOY_DIR}"
  install -m 0755 "${relay_binary}" "${BIN_DIR}/relay-server"
  install -m 0755 "${control_binary}" "${BIN_DIR}/control-plane"
  install -m 0755 "${nat_binary}" "${BIN_DIR}/nat-detector"
  if [[ -n "${cli_binary}" ]]; then
    install -m 0755 "${cli_binary}" "${BIN_DIR}/nextunnel"
  else
    warn "Release 包未包含 nextunnel CLI，系统命令链接将不会创建。"
  fi
  if [[ -n "${dashboard_binary}" ]]; then
    install -m 0755 "${dashboard_binary}" "${BIN_DIR}/dashboard"
  else
    warn "Release 包未包含 dashboard 二进制，本次仅安装核心三服务。"
  fi
  if [[ -n "${dashboard_web_dir}" ]]; then
    rm -rf "${WEB_DIR:?}/"*
    cp -a "${dashboard_web_dir}/." "${WEB_DIR}/"
  else
    warn "Release 包未包含 Dashboard Web 静态资源。"
  fi
  if [[ -n "${deploy_script_dir}" ]]; then
    cp -a "${deploy_script_dir}/." "${DEPLOY_DIR}/"
    chmod +x "${DEPLOY_DIR}/install.sh" || true
  else
    warn "Release 包未包含 deploy/server 安装脚本。"
  fi
  rm -rf "${tmp_dir}"
  trap - EXIT
  log "服务端二进制已安装到：${BIN_DIR}"
}

install_cli_link() {
  if [[ "${CLI_LINK_PATH}" == "none" || "${CLI_LINK_PATH}" == "false" || "${CLI_LINK_PATH}" == "0" ]]; then
    warn "已跳过 nextunnel CLI 系统链接。可直接执行：${BIN_DIR}/nextunnel"
    return
  fi
  if [[ ! -x "${BIN_DIR}/nextunnel" ]]; then
    warn "未找到 ${BIN_DIR}/nextunnel，无法创建系统命令链接。"
    return
  fi
  if [[ -e "${CLI_LINK_PATH}" && ! -L "${CLI_LINK_PATH}" ]]; then
    warn "${CLI_LINK_PATH} 已存在且不是软链接，已跳过创建。可直接执行：${BIN_DIR}/nextunnel"
    return
  fi
  local link_parent
  link_parent="$(dirname "${CLI_LINK_PATH}")"
  mkdir -p "${link_parent}"
  ln -sfn "${BIN_DIR}/nextunnel" "${CLI_LINK_PATH}"
  log "nextunnel CLI 已链接到：${CLI_LINK_PATH}"
}

write_systemd_units() {
  require_command systemctl
  mkdir -p /etc/systemd/system

  cat >/etc/systemd/system/nextunnel-relay.service <<EOF
[Unit]
Description=NexTunnel Relay Server
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=${SERVICE_USER}
Group=${SERVICE_GROUP}
EnvironmentFile=${ENV_FILE}
ExecStart=${BIN_DIR}/relay-server --bind \${RELAY_BIND} --control-port \${RELAY_CONTROL_PORT} --quic-port \${RELAY_QUIC_PORT} --auth-token \${RELAY_AUTH_TOKEN} --require-auth --stats-interval \${RELAY_STATS_INTERVAL}
Restart=on-failure
RestartSec=5s
NoNewPrivileges=true
PrivateTmp=true
ProtectHome=true
ProtectSystem=full

[Install]
WantedBy=multi-user.target
EOF

  cat >/etc/systemd/system/nextunnel-control-plane.service <<EOF
[Unit]
Description=NexTunnel Control Plane
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=${SERVICE_USER}
Group=${SERVICE_GROUP}
EnvironmentFile=${ENV_FILE}
ExecStart=${BIN_DIR}/control-plane --listen \${CONTROL_PLANE_LISTEN} --api-token \${CONTROL_PLANE_API_TOKEN} --store-path \${CONTROL_PLANE_STORE_PATH}
Restart=on-failure
RestartSec=5s
NoNewPrivileges=true
PrivateTmp=true
ProtectHome=true
ProtectSystem=full
ReadWritePaths=${DATA_DIR}

[Install]
WantedBy=multi-user.target
EOF

  cat >/etc/systemd/system/nextunnel-nat-detector.service <<EOF
[Unit]
Description=NexTunnel NAT Detector
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=${SERVICE_USER}
Group=${SERVICE_GROUP}
EnvironmentFile=${ENV_FILE}
ExecStart=${BIN_DIR}/nat-detector --primary-addr \${NAT_PRIMARY_ADDR} --alt-addr \${NAT_ALT_ADDR} --port \${NAT_PORT} --realm \${NAT_REALM}
Restart=on-failure
RestartSec=5s
NoNewPrivileges=true
PrivateTmp=true
ProtectHome=true
ProtectSystem=full

[Install]
WantedBy=multi-user.target
EOF

  if [[ "${DASHBOARD_ENABLED}" == "true" && -x "${BIN_DIR}/dashboard" ]]; then
    cat >/etc/systemd/system/nextunnel-dashboard.service <<EOF
[Unit]
Description=NexTunnel Dashboard
After=network-online.target nextunnel-control-plane.service
Wants=network-online.target

[Service]
Type=simple
User=${SERVICE_USER}
Group=${SERVICE_GROUP}
EnvironmentFile=${ENV_FILE}
ExecStart=${BIN_DIR}/dashboard --listen \${DASHBOARD_LISTEN} --secret-key \${DASHBOARD_SECRET_KEY} --admin-user \${DASHBOARD_ADMIN_USER} --admin-password \${DASHBOARD_ADMIN_PASSWORD} --allowed-origins \${DASHBOARD_ALLOWED_ORIGINS} --store-path \${DASHBOARD_STORE_PATH} --static-dir \${DASHBOARD_STATIC_DIR}
Restart=on-failure
RestartSec=5s
NoNewPrivileges=true
PrivateTmp=true
ProtectHome=true
ProtectSystem=full
ReadWritePaths=${DATA_DIR}

[Install]
WantedBy=multi-user.target
EOF
  else
    rm -f /etc/systemd/system/nextunnel-dashboard.service
  fi

  systemctl daemon-reload
  log "systemd 服务文件已写入 /etc/systemd/system"
}

prepare_permissions() {
  create_system_user
  mkdir -p "${INSTALL_DIR}" "${BIN_DIR}" "${WEB_DIR}" "${CONFIG_DIR}" "${DATA_DIR}"
  chmod 750 "${CONFIG_DIR}"
  chmod 750 "${DATA_DIR}"
  if [[ "${SERVICE_USER}" != "root" ]]; then
    chown -R root:root "${INSTALL_DIR}" "${CONFIG_DIR}"
    chown -R "${SERVICE_USER}:${SERVICE_GROUP}" "${DATA_DIR}"
  fi
}

assert_safe_remove_path() {
  local path_to_remove="$1"
  if [[ -z "${path_to_remove}" || "${path_to_remove}" == "/" || "${#path_to_remove}" -lt 8 ]]; then
    printf '拒绝删除高风险路径：%s\n' "${path_to_remove}" >&2
    exit 1
  fi
}

print_connection_info() {
  load_runtime_env
  printf '\n连接信息：\n'
  printf '  Relay TCP:       %s:%s\n' "${NEXTUNNEL_PUBLIC_HOST}" "${RELAY_CONTROL_PORT}"
  printf '  Relay QUIC UDP:  %s:%s\n' "${NEXTUNNEL_PUBLIC_HOST}" "${RELAY_QUIC_PORT}"
  printf '  NAT UDP:         %s:%s\n' "${NEXTUNNEL_PUBLIC_HOST}" "${NAT_PORT}"
  printf '  Control Plane:   http://%s:%s\n' "${NEXTUNNEL_PUBLIC_HOST}" "${CONTROL_PLANE_PORT}"
  if [[ "${DASHBOARD_ENABLED:-false}" == "true" ]]; then
    printf '  Dashboard:       http://%s:%s\n' "${NEXTUNNEL_PUBLIC_HOST}" "${DASHBOARD_PORT}"
    printf '  Dashboard User:  %s\n' "${DASHBOARD_ADMIN_USER}"
  fi
  printf '  Relay Token:     %s\n' "${RELAY_AUTH_TOKEN}"
  printf '  Control Token:   %s\n' "${CONTROL_PLANE_API_TOKEN}"
  if [[ -x "${CLI_LINK_PATH}" ]]; then
    printf '  CLI:             %s\n' "${CLI_LINK_PATH}"
  elif [[ -x "${BIN_DIR}/nextunnel" ]]; then
    printf '  CLI:             %s\n' "${BIN_DIR}/nextunnel"
  fi
  printf '\n外部客户端无法连接时，请确认：\n'
  printf '  1. 安装时已设置公网 IP 或域名：--public-host <公网IP或域名>，当前为 %s。\n' "${NEXTUNNEL_PUBLIC_HOST}"
  printf '  2. 腾讯云安全组和系统防火墙已放行 %s/tcp、%s/udp、%s/udp。\n' "${RELAY_CONTROL_PORT}" "${RELAY_QUIC_PORT}" "${NAT_PORT}"
  if [[ "${DASHBOARD_ENABLED:-false}" == "true" ]]; then
    printf '  3. 如需访问 Dashboard，还需放行 %s/tcp。\n' "${DASHBOARD_PORT}"
  fi
  printf '  4. 本机检查命令：sudo %s health；日志命令：sudo %s logs。\n' "${DEPLOY_DIR}/install.sh" "${DEPLOY_DIR}/install.sh"
}

health_check() {
  load_runtime_env
  require_command curl
  require_command systemctl
  collect_runtime_services
  log "检查 systemd 服务状态"
  local failed_services=()
  local service_name
  for service_name in "${RUNTIME_SERVICE_NAMES[@]}"; do
    if ! systemctl is-active --quiet "${service_name}"; then
      failed_services+=("${service_name}")
    fi
  done
  if (( ${#failed_services[@]} > 0 )); then
    systemctl --no-pager --full status "${failed_services[@]}" >&2 || true
    printf '服务未处于 active 状态：%s\n' "${failed_services[*]}" >&2
    exit 1
  fi
  log "检查 Control Plane 健康状态"
  curl -fsS "http://127.0.0.1:${CONTROL_PLANE_PORT}/healthz" >/dev/null
  log "检查 Relay TCP 端口"
  if command -v nc >/dev/null 2>&1; then
    nc -z 127.0.0.1 "${RELAY_CONTROL_PORT}"
  else
    timeout 3 bash -c "</dev/tcp/127.0.0.1/${RELAY_CONTROL_PORT}"
  fi
  if [[ "${DASHBOARD_ENABLED:-false}" == "true" && -x "${BIN_DIR}/dashboard" ]]; then
    log "检查 Dashboard 健康状态"
    curl -fsS "http://127.0.0.1:${DASHBOARD_PORT}/api/v1/health" >/dev/null
  fi
  log "健康检查通过"
}

install_stack() {
  require_root
  require_command systemctl
  prepare_permissions
  generate_env
  install_release_package
  install_cli_link
  write_systemd_units
  collect_runtime_services
  systemctl enable --now "${RUNTIME_SERVICE_NAMES[@]}"
  health_check
  print_connection_info
}

update_stack() {
  require_root
  require_command systemctl
  prepare_permissions
  [[ -f "${ENV_FILE}" ]] || generate_env
  install_release_package
  install_cli_link
  write_systemd_units
  collect_runtime_services
  systemctl restart "${RUNTIME_SERVICE_NAMES[@]}"
  health_check
  print_connection_info
}

uninstall_stack() {
  require_root
  require_command systemctl
  collect_runtime_services
  systemctl disable --now "${RUNTIME_SERVICE_NAMES[@]}" >/dev/null 2>&1 || true
  rm -f /etc/systemd/system/nextunnel-relay.service \
        /etc/systemd/system/nextunnel-control-plane.service \
        /etc/systemd/system/nextunnel-nat-detector.service \
        /etc/systemd/system/nextunnel-dashboard.service
  systemctl daemon-reload
  if [[ "${PURGE}" == "true" ]]; then
    assert_safe_remove_path "${INSTALL_DIR}"
    assert_safe_remove_path "${CONFIG_DIR}"
    assert_safe_remove_path "${DATA_DIR}"
    rm -rf "${INSTALL_DIR}" "${CONFIG_DIR}" "${DATA_DIR}"
    warn "已卸载并删除安装目录、配置和数据。"
  else
    warn "已停止并移除 systemd 服务，默认保留 ${INSTALL_DIR}、${CONFIG_DIR}、${DATA_DIR}。使用 --purge 可彻底删除。"
  fi
  if [[ "${CLI_LINK_PATH}" != "none" && -L "${CLI_LINK_PATH}" && "$(readlink "${CLI_LINK_PATH}")" == "${BIN_DIR}/nextunnel" ]]; then
    rm -f "${CLI_LINK_PATH}"
    warn "已移除 CLI 系统链接：${CLI_LINK_PATH}"
  fi
}

case "${ACTION}" in
  install)
    install_stack
    ;;
  up)
    require_root
    require_command systemctl
    collect_runtime_services
    systemctl start "${RUNTIME_SERVICE_NAMES[@]}"
    ;;
  down)
    require_root
    require_command systemctl
    collect_runtime_services
    systemctl stop "${RUNTIME_SERVICE_NAMES[@]}"
    ;;
  restart)
    require_root
    require_command systemctl
    collect_runtime_services
    systemctl restart "${RUNTIME_SERVICE_NAMES[@]}"
    ;;
  status)
    require_command systemctl
    collect_runtime_services
    systemctl --no-pager status "${RUNTIME_SERVICE_NAMES[@]}"
    ;;
  logs)
    require_command journalctl
    collect_runtime_services
    journalctl_args=()
    for service_name in "${RUNTIME_SERVICE_NAMES[@]}"; do
      journalctl_args+=(-u "${service_name}")
    done
    journalctl -f -n 200 "${journalctl_args[@]}"
    ;;
  health)
    health_check
    ;;
  update)
    update_stack
    ;;
  uninstall)
    uninstall_stack
    ;;
  config)
    require_root
    prepare_permissions
    generate_env
    print_connection_info
    ;;
  *)
    printf '未知操作：%s\n' "${ACTION}" >&2
    usage >&2
    exit 2
    ;;
esac
