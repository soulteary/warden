#!/bin/bash

# Warden 简单示例启动脚本
# 使用方式: ./run.sh

set -e

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
DATA_FILE="${DATA_FILE:-$SCRIPT_DIR/data.json}"
case "$DATA_FILE" in
  /*) ;;
  *) DATA_FILE="$PWD/$DATA_FILE" ;;
esac

if [ -z "${API_KEY:-}" ]; then
    echo "请先设置 API_KEY，并在客户端使用同一密钥。" >&2
    exit 1
fi
export API_KEY

echo "🚀 启动 Warden 简单示例..."

# 检查 Redis 是否运行
if ! redis-cli -h localhost -p 6379 ping > /dev/null 2>&1; then
    echo "⚠️  Redis 未运行，请先启动 Redis："
    echo "   docker run -d --name redis -p 6379:6379 redis:7.4-alpine"
    echo "   或: redis-server"
    exit 1
fi

# 检查数据文件是否存在
if [ ! -f "$DATA_FILE" ]; then
    echo "⚠️  数据文件 data.json 不存在，正在创建示例文件..."
    cat > "$DATA_FILE" << 'EOF'
[
    {
        "phone": "13800138000",
        "mail": "admin@example.com"
    },
    {
        "phone": "13900139000",
        "mail": "user@example.com"
    }
]
EOF
    echo "✅ 已创建 data.json 文件"
fi

# 切换到项目根目录
cd "$SCRIPT_DIR/../.."

# 运行服务
echo "📦 启动 Warden 服务..."
go run . \
  --data-file "$DATA_FILE" \
  --port 8081 \
  --redis localhost:6379 \
  --mode ONLY_LOCAL

