#!/bin/bash

# Warden 本地独立测试脚本
# 该脚本可以在不依赖 Stargate 和 Herald 的情况下测试 warden 项目的所有功能
# 使用方式: ./scripts/test-local.sh

set -e

# 配置
API_KEY="${API_KEY:-test-api-key-$(date +%s)}"
BASE_URL="${BASE_URL:-http://localhost:8081}"
REDIS_URL="${REDIS_URL:-localhost:6379}"

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 测试统计
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# 测试函数
test_endpoint() {
    local name=$1
    local method=$2
    local url=$3
    local headers=$4
    local expected_status=$5
    local expected_content=$6  # 可选：期望的响应内容（使用 grep 检查）

    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    echo -n "测试 $name... "
    
    if [ "$method" = "GET" ]; then
        response=$(curl -s -w "\n%{http_code}" $headers "$url" 2>/dev/null || echo -e "\n000")
    else
        response=$(curl -s -w "\n%{http_code}" -X "$method" $headers "$url" 2>/dev/null || echo -e "\n000")
    fi
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')
    
    if [ "$http_code" = "$expected_status" ]; then
        # 如果提供了期望内容，检查响应体
        if [ -n "$expected_content" ]; then
            if echo "$body" | grep -q "$expected_content" 2>/dev/null; then
                echo -e "${GREEN}✓${NC} (状态码: $http_code)"
                PASSED_TESTS=$((PASSED_TESTS + 1))
                if [ -n "$body" ] && [ "$body" != "null" ]; then
                    echo "$body" | jq . 2>/dev/null || echo "$body" | head -n 5
                fi
                return 0
            else
                echo -e "${RED}✗${NC} (状态码: $http_code, 但响应内容不匹配)"
                echo "期望包含: $expected_content"
                echo "实际响应: $body" | head -n 3
                FAILED_TESTS=$((FAILED_TESTS + 1))
                return 1
            fi
        else
            echo -e "${GREEN}✓${NC} (状态码: $http_code)"
            PASSED_TESTS=$((PASSED_TESTS + 1))
            if [ -n "$body" ] && [ "$body" != "null" ]; then
                echo "$body" | jq . 2>/dev/null || echo "$body" | head -n 5
            fi
            return 0
        fi
    else
        echo -e "${RED}✗${NC} (期望: $expected_status, 实际: $http_code)"
        echo "响应: $body" | head -n 3
        FAILED_TESTS=$((FAILED_TESTS + 1))
        return 1
    fi
}

# 检查依赖
echo -e "${BLUE}检查依赖...${NC}"

# 检查 Redis
if ! redis-cli -h ${REDIS_URL%%:*} -p ${REDIS_URL##*:} ping > /dev/null 2>&1; then
    echo -e "${YELLOW}⚠️  Redis 未运行，某些测试可能会失败${NC}"
    echo "   提示: docker run -d --name redis -p 6379:6379 redis:6.2.4"
else
    echo -e "${GREEN}✓ Redis 运行中${NC}"
fi

# 检查 warden 服务
if ! curl -s "$BASE_URL/health" > /dev/null 2>&1; then
    echo -e "${RED}错误: Warden 服务未运行${NC}"
    echo "请先启动服务:"
    echo "  go run main.go --port 8081 --redis $REDIS_URL --mode ONLY_LOCAL"
    echo "或使用 Docker Compose:"
    echo "  docker-compose up -d"
    exit 1
fi
echo -e "${GREEN}✓ Warden 服务运行中${NC}"
echo ""

# 准备测试数据
echo -e "${BLUE}准备测试数据...${NC}"
TEST_DATA_FILE="/tmp/warden-test-data.json"
cat > "$TEST_DATA_FILE" << 'EOF'
[
    {
        "phone": "13800138000",
        "mail": "admin@example.com",
        "user_id": "test-admin-001",
        "status": "active",
        "scope": ["read", "write", "admin"],
        "role": "admin"
    },
    {
        "phone": "13900139000",
        "mail": "user@example.com",
        "user_id": "test-user-002",
        "status": "active",
        "scope": ["read"],
        "role": "user"
    },
    {
        "phone": "13700137000",
        "mail": "guest@example.com",
        "status": "active"
    }
]
EOF
echo -e "${GREEN}✓ 测试数据已准备${NC}"
echo ""

# 开始测试
echo -e "${BLUE}开始测试...${NC}"
echo "===================="
echo ""

# 测试 1: 健康检查
echo -e "${YELLOW}1. 健康检查端点${NC}"
test_endpoint "健康检查" "GET" "$BASE_URL/health" "" "200"
echo ""

# 测试 2: 获取用户列表（需要认证）
echo -e "${YELLOW}2. 用户列表端点${NC}"
test_endpoint "获取用户列表（无认证）" "GET" "$BASE_URL/" "" "401"
test_endpoint "获取用户列表（有认证）" "GET" "$BASE_URL/" \
    "-H 'X-API-Key: $API_KEY'" "200"
echo ""

# 测试 3: 分页查询
echo -e "${YELLOW}3. 分页查询${NC}"
test_endpoint "分页查询 (page=1, page_size=2)" "GET" \
    "$BASE_URL/?page=1&page_size=2" \
    "-H 'X-API-Key: $API_KEY'" "200"
test_endpoint "分页查询 (无效页码)" "GET" \
    "$BASE_URL/?page=999&page_size=10" \
    "-H 'X-API-Key: $API_KEY'" "200"
echo ""

# 测试 4: 查询单个用户（通过 phone）
echo -e "${YELLOW}4. 查询单个用户（通过 phone）${NC}"
test_endpoint "查询用户（phone，无认证）" "GET" \
    "$BASE_URL/user?phone=13800138000" "" "401"
test_endpoint "查询用户（phone，有认证）" "GET" \
    "$BASE_URL/user?phone=13800138000" \
    "-H 'X-API-Key: $API_KEY'" "200" "13800138000"
echo ""

# 测试 5: 查询单个用户（通过 mail）
echo -e "${YELLOW}5. 查询单个用户（通过 mail）${NC}"
test_endpoint "查询用户（mail）" "GET" \
    "$BASE_URL/user?mail=admin@example.com" \
    "-H 'X-API-Key: $API_KEY'" "200" "admin@example.com"
echo ""

# 测试 6: 查询单个用户（通过 user_id）
echo -e "${YELLOW}6. 查询单个用户（通过 user_id）${NC}"
test_endpoint "查询用户（user_id）" "GET" \
    "$BASE_URL/user?user_id=test-admin-001" \
    "-H 'X-API-Key: $API_KEY'" "200" "test-admin-001"
echo ""

# 测试 7: 错误场景测试
echo -e "${YELLOW}7. 错误场景测试${NC}"
test_endpoint "查询用户（缺少参数）" "GET" \
    "$BASE_URL/user" \
    "-H 'X-API-Key: $API_KEY'" "400"
test_endpoint "查询用户（多个参数）" "GET" \
    "$BASE_URL/user?phone=13800138000&mail=admin@example.com" \
    "-H 'X-API-Key: $API_KEY'" "400"
test_endpoint "查询用户（不存在）" "GET" \
    "$BASE_URL/user?phone=99999999999" \
    "-H 'X-API-Key: $API_KEY'" "404"
echo ""

# 测试 8: Prometheus 指标
echo -e "${YELLOW}8. 监控指标${NC}"
test_endpoint "Prometheus 指标" "GET" "$BASE_URL/metrics" "" "200"
echo ""

# 测试 9: 日志级别管理
echo -e "${YELLOW}9. 日志级别管理${NC}"
test_endpoint "获取日志级别（无认证）" "GET" "$BASE_URL/log/level" "" "401"
test_endpoint "获取日志级别（有认证）" "GET" "$BASE_URL/log/level" \
    "-H 'X-API-Key: $API_KEY'" "200"

test_endpoint "设置日志级别" "POST" "$BASE_URL/log/level" \
    "-H 'X-API-Key: $API_KEY' -H 'Content-Type: application/json' -d '{\"level\":\"debug\"}'" "200"
echo ""

# 测试 10: 验证新字段
echo -e "${YELLOW}10. 验证新字段${NC}"
response=$(curl -s -H "X-API-Key: $API_KEY" "$BASE_URL/user?phone=13800138000" 2>/dev/null)
if echo "$response" | jq -e '.user_id, .status, .scope, .role' > /dev/null 2>&1; then
    echo -e "${GREEN}✓ 新字段存在（user_id, status, scope, role）${NC}"
    echo "$response" | jq .
    PASSED_TESTS=$((PASSED_TESTS + 1))
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
else
    echo -e "${RED}✗ 新字段缺失或格式错误${NC}"
    echo "响应: $response"
    FAILED_TESTS=$((FAILED_TESTS + 1))
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
fi
echo ""

# 清理测试数据
echo -e "${BLUE}清理测试数据...${NC}"
rm -f "$TEST_DATA_FILE"
echo -e "${GREEN}✓ 测试数据已清理${NC}"
echo ""

# 输出测试报告
echo "===================="
echo -e "${BLUE}测试报告${NC}"
echo "===================="
echo "总测试数: $TOTAL_TESTS"
echo -e "${GREEN}通过: $PASSED_TESTS${NC}"
if [ $FAILED_TESTS -gt 0 ]; then
    echo -e "${RED}失败: $FAILED_TESTS${NC}"
    exit 1
else
    echo -e "${GREEN}失败: $FAILED_TESTS${NC}"
    echo ""
    echo -e "${GREEN}✅ 所有测试通过！${NC}"
    exit 0
fi
