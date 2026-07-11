#!/usr/bin/env bash
# =============================================================================
# 后端服务管理脚本 — 停止 → 编译 → 启动
# 用法: bash scripts/restart.sh
# =============================================================================

set -euo pipefail

# 配置
PROJECT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
BINARY_NAME="emergency-msystem-backend"
BINARY_PATH="${PROJECT_DIR}/bin/${BINARY_NAME}"
LOG_DIR="${PROJECT_DIR}/logs"
PID_FILE="${PROJECT_DIR}/bin/.pid"
LOG_FILE="${LOG_DIR}/server.log"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info()  { echo -e "${GREEN}[INFO]${NC}  $*"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC}  $*"; }
log_error() { echo -e "${RED}[ERROR]${NC} $*"; }
log_step()  { echo -e "${BLUE}[STEP]${NC}  $*"; }

# =============================================================================
# Step 1: 停止服务
# =============================================================================
stop_service() {
  log_step "正在停止服务..."

  if [ -f "${PID_FILE}" ]; then
    local pid
    pid=$(cat "${PID_FILE}" 2>/dev/null || true)
    if [ -n "${pid}" ] && kill -0 "${pid}" 2>/dev/null; then
      log_info "发送 SIGTERM 信号到 PID=${pid} ..."
      kill -TERM "${pid}" 2>/dev/null || true

      # 等待进程退出（最多 10 秒）
      for i in $(seq 1 10); do
        if ! kill -0 "${pid}" 2>/dev/null; then
          log_info "服务已停止 (PID=${pid})"
          rm -f "${PID_FILE}"
          return 0
        fi
        sleep 1
      done

      # 超时，强制杀死
      log_warn "优雅退出超时，强制终止 PID=${pid}"
      kill -KILL "${pid}" 2>/dev/null || true
      sleep 1
      rm -f "${PID_FILE}"
      log_info "服务已强制停止"
      return 0
    else
      log_warn "PID 文件存在但进程已不存在，清理 PID 文件"
      rm -f "${PID_FILE}"
    fi
  fi

  # 兜底：通过进程名匹配杀掉
  local running_pids
  running_pids=$(pgrep -f "${BINARY_NAME}" 2>/dev/null || true)
  if [ -n "${running_pids}" ]; then
    log_warn "发现残留进程: ${running_pids}，正在终止..."
    echo "${running_pids}" | xargs kill -TERM 2>/dev/null || true
    sleep 2
    # 二次确认
    local left
    left=$(pgrep -f "${BINARY_NAME}" 2>/dev/null || true)
    if [ -n "${left}" ]; then
      echo "${left}" | xargs kill -KILL 2>/dev/null || true
    fi
  fi

  log_info "停止完成"
}

# =============================================================================
# Step 2: 编译服务
# =============================================================================
build_service() {
  log_step "正在编译服务..."

  cd "${PROJECT_DIR}"

  # 创建 bin 目录
  mkdir -p "${PROJECT_DIR}/bin"

  log_info "Go 版本: $(go version 2>/dev/null || echo 'unknown')"

  # GOWORK=off 避免 go.work 导致的编译缓存污染
  # -a 强制重新编译所有依赖包
  if GOWORK=off go build -a -o "${BINARY_PATH}" main.go; then
    log_info "编译成功 → ${BINARY_PATH}"
  else
    log_error "编译失败，请检查代码错误"
    exit 1
  fi
}

# =============================================================================
# Step 3: 启动服务
# =============================================================================
start_service() {
  log_step "正在启动服务..."

  # 创建日志目录
  mkdir -p "${LOG_DIR}"

  # 切换到项目目录（确保能找到 config/.env 等相对路径）
  cd "${PROJECT_DIR}"

  # 后台启动
  nohup "${BINARY_PATH}" >> "${LOG_FILE}" 2>&1 &
  local pid=$!

  echo "${pid}" > "${PID_FILE}"

  # 等待服务就绪（最多 15 秒）
  log_info "等待服务启动..."
  for i in $(seq 1 15); do
    if ! kill -0 "${pid}" 2>/dev/null; then
      log_error "进程启动后立即退出，请检查日志: tail -f ${LOG_FILE}"
      rm -f "${PID_FILE}"
      exit 1
    fi

    # 健康检查
    if curl -s -o /dev/null -w "%{http_code}" "http://localhost:8080/health" 2>/dev/null | grep -q "200"; then
      log_info "服务启动成功 (PID=${pid})"
      log_info "健康检查: http://localhost:8080/health"
      log_info "日志文件: ${LOG_FILE}"
      return 0
    fi
    sleep 1
  done

  # 超时但仍存活，给出警告
  if kill -0 "${pid}" 2>/dev/null; then
    log_warn "服务已启动 (PID=${pid})，但健康检查未就绪，请手动确认"
    log_info "查看日志: tail -f ${LOG_FILE}"
  else
    log_error "服务启动失败，请查看日志: tail -f ${LOG_FILE}"
    rm -f "${PID_FILE}"
    exit 1
  fi
}

# =============================================================================
# 主流程
# =============================================================================
main() {
  echo ""
  echo "════════════════════════════════════════"
  echo "  应急管理系统 — 后端重启脚本"
  echo "════════════════════════════════════════"
  echo "  项目目录: ${PROJECT_DIR}"
  echo "  可执行文件: ${BINARY_PATH}"
  echo ""

  stop_service
  echo ""
  build_service
  echo ""
  start_service

  echo ""
  echo "════════════════════════════════════════"
  echo -e "  ${GREEN}重启完成${NC}"
  echo "════════════════════════════════════════"
  echo ""
  log_info "查看实时日志: tail -f ${LOG_FILE}"
  log_info "停止服务:     kill \$(cat ${PID_FILE})"
  log_info "仅重启:       bash ${PROJECT_DIR}/scripts/restart.sh"
}

main "$@"
