#!/bin/bash

# Warden 简单示例启动脚本
# 使用方式: ./run.sh

set -e

echo "🚀 启动 Warden 简单示例..."

# 检查 Redis 是否运行
if ! redis-cli -h localhost -p 6379 ping > /dev/null 2>&1; then
    echo "⚠️  Redis 未运行，请先启动 Redis："
    echo "   docker run -d --name redis -p 6379:6379 redis:6.2.4"
    echo "   或: redis-server"
    exit 1
fi

# 检查数据文件是否存在
if [ ! -f "data.json" ]; then
    echo "⚠️  数据文件 data.json 不存在，正在创建示例文件..."
    cat > data.json << 'EOF'
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

# 设置默认 API Key（如果未设置）
if [ -z "$API_KEY" ]; then
    export API_KEY="demo-api-key-$(date +%s)"
    echo "ℹ️  未设置 API_KEY，使用临时密钥: $API_KEY"
fi

# 切换到项目根目录
cd "$(dirname "$0")/../.."

# 运行服务
echo "📦 启动 Warden 服务..."
go run . \
  --port 8081 \
  --redis localhost:6379 \
  --mode ONLY_LOCAL

