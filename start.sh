#!/bin/bash
# docker-quck-proxy 启动脚本
# 用法：./start.sh
#      ./start.sh <custom.env>

set -e

# 加载 .env 文件（可选）
ENV_FILE="${1:-.env}"
if [ -f "$ENV_FILE" ]; then
    echo "📁 加载配置: $ENV_FILE"
    # 加载环境变量到当前 shell
    export $(grep -v '^#' "$ENV_FILE" | xargs)
else
    echo "⚠️  未找到 $ENV_FILE，使用默认配置"
fi

# 启动代理
echo "🚀 启动 docker-quck-proxy..."
go run .
