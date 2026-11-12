#!/bin/bash

# Warden 复杂示例测试脚本
# 使用方式: ./test.sh

set -e

if [ -z "${API_KEY:-}" ]; then
    echo "请先设置 API_KEY，使其与 .env 中的值一致 (set API_KEY to the value in .env)。" >&2
    exit 1
fi
BASE_URL="${BASE_URL:-http://localhost:${PORT:-8081}}"
MOCK_API_BASE_URL="${MOCK_API_BASE_URL:-http://localhost:${MOCK_API_PORT:-8080}}"

echo "🧪 Warden 复杂示例测试"
echo "===================="
echo ""

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 测试函数
test_endpoint() {
    local name=$1
    local method=$2
    local url=$3
    local expected_status=$4
    shift 4

    echo -n "测试 $name... "
    
    response=$(curl -s -w "\n%{http_code}" -X "$method" "$@" "$url")
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')
    
    if [ "$http_code" = "$expected_status" ]; then
        echo -e "${GREEN}✓${NC} (状态码: $http_code)"
        if [ -n "$body" ] && [ "$body" != "null" ]; then
            echo "$body" | jq . 2>/dev/null || echo "$body"
        fi
        return 0
    else
        echo -e "${RED}✗${NC} (期望: $expected_status, 实际: $http_code)"
        echo "响应: $body"
        return 1
    fi
}

# 检查服务是否运行
echo "检查服务状态..."
if ! curl -s "$BASE_URL/health" > /dev/null; then
    echo -e "${RED}错误: Warden 服务未运行${NC}"
    echo "请先启动服务: docker-compose up -d"
    exit 1
fi
echo -e "${GREEN}✓ 服务运行中${NC}"
echo ""

# 测试 1: 健康检查
echo "1. 健康检查端点"
test_endpoint "健康检查" "GET" "$BASE_URL/health" "200"
echo ""

# 测试 2: 获取用户列表（需要认证）
echo "2. 用户列表端点"
test_endpoint "获取用户列表" "GET" "$BASE_URL/" \
    "200" -H "X-API-Key: $API_KEY"
echo ""

# 测试 3: 分页查询
echo "3. 分页查询"
test_endpoint "分页查询 (page=1, page_size=2)" "GET" \
    "$BASE_URL/?page=1&page_size=2" \
    "200" -H "X-API-Key: $API_KEY"
echo ""

# 测试 4: 未授权访问
echo "4. 安全测试"
test_endpoint "未授权访问" "GET" "$BASE_URL/" "401"
echo ""

# 测试 5: Prometheus 指标
echo "5. 监控指标"
test_endpoint "Prometheus 指标" "GET" "$BASE_URL/metrics" "200"
echo ""

# 测试 6: 日志级别管理
echo "6. 日志级别管理"
test_endpoint "获取日志级别" "GET" "$BASE_URL/log/level" \
    "200" -H "X-API-Key: $API_KEY"

test_endpoint "设置日志级别" "POST" "$BASE_URL/log/level" \
    "200" -H "X-API-Key: $API_KEY" -H "Content-Type: application/json" -d '{"level":"debug"}'
echo ""

# 测试 7: Mock API
echo "7. Mock API 测试"
if curl -s "$MOCK_API_BASE_URL/health" > /dev/null; then
    test_endpoint "Mock API 健康检查" "GET" "$MOCK_API_BASE_URL/health" "200"
    test_endpoint "Mock API 用户列表" "GET" "$MOCK_API_BASE_URL/api/users" \
        "200" -H "Authorization: Bearer mock-token"
else
    echo -e "${YELLOW}⚠ Mock API 未运行，跳过测试${NC}"
fi
echo ""

echo -e "${GREEN}✅ 所有测试完成！${NC}"

