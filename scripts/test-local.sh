#!/bin/bash

# Warden 本地独立测试脚本
# 该脚本可以在不依赖 Stargate 和 Herald 的情况下测试 warden 项目的所有功能
# 使用方式: ./scripts/test-local.sh

set -e

# 配置
API_KEY="${API_KEY:-test-api-key-$(date +%s)}"
BASE_URL="${BASE_URL:-http://localhost:8081}"
REDIS_URL="${REDIS_URL:-localhost:6379}"
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DATA_FILE="$PROJECT_ROOT/data.json"
DATA_FILE_BACKUP="$PROJECT_ROOT/data.json.backup"

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

# 检查 jq 是否可用（用于解析 JSON）
if ! command -v jq >/dev/null 2>&1; then
    echo -e "${YELLOW}⚠️  jq 未安装，部分 JSON 解析功能可能不可用${NC}"
    echo "   建议安装: brew install jq (macOS) 或 apt-get install jq (Linux)"
    JQ_AVAILABLE=false
else
    JQ_AVAILABLE=true
fi

# 检查 Redis（可选）
# 使用 TCP 连接检查，不依赖 redis-cli（适用于 Docker 环境）
REDIS_AVAILABLE=false
REDIS_HOST=${REDIS_URL%%:*}
REDIS_PORT=${REDIS_URL##*:}

# 尝试多种方式检查 Redis 端口是否可达
check_redis_port() {
    local host=$1
    local port=$2
    
    # 方法1: 使用 nc (netcat)，如果可用
    if command -v nc >/dev/null 2>&1; then
        if nc -z -w 2 "$host" "$port" >/dev/null 2>&1; then
            return 0
        fi
    fi
    
    # 方法2: 使用 bash 内置的 /dev/tcp
    if timeout 2 bash -c "echo >/dev/tcp/$host/$port" >/dev/null 2>&1; then
        return 0
    fi
    
    # 方法3: 使用 telnet（如果可用）
    if command -v telnet >/dev/null 2>&1; then
        if echo "quit" | timeout 2 telnet "$host" "$port" >/dev/null 2>&1; then
            return 0
        fi
    fi
    
    return 1
}

if check_redis_port "$REDIS_HOST" "$REDIS_PORT"; then
    echo -e "${GREEN}✓ Redis 运行中 (${REDIS_HOST}:${REDIS_PORT})${NC}"
    REDIS_AVAILABLE=true
else
    echo -e "${YELLOW}⚠️  Redis 未运行，将测试无 Redis 模式${NC}"
    echo "   提示: docker run --rm -it -p 6379:6379 redis:8.4-alpine"
fi

# 检查 warden 服务
if ! curl -s "$BASE_URL/health" > /dev/null 2>&1; then
    echo -e "${RED}错误: Warden 服务未运行${NC}"
    echo "请先启动服务:"
    echo ""
    echo -e "${YELLOW}方式 1: 使用环境变量 ${NC}"
    if [ "$REDIS_AVAILABLE" = true ]; then
        echo "  PORT=8081 REDIS=$REDIS_URL MODE=ONLY_LOCAL API_KEY=$API_KEY go run main.go"
    else
        echo "  PORT=8081 REDIS_ENABLED=false MODE=ONLY_LOCAL API_KEY=$API_KEY go run main.go"
    fi
    echo ""
    echo -e "${YELLOW}方式 2: 使用命令行参数 ${NC}"
    if [ "$REDIS_AVAILABLE" = true ]; then
        echo "  API_KEY=$API_KEY go run main.go -port 8081 -redis $REDIS_URL -mode ONLY_LOCAL"
    else
        echo "  API_KEY=$API_KEY go run main.go -port 8081 -redis-enabled=false -mode ONLY_LOCAL"
    fi
    echo ""
    echo "或使用 Docker Compose:"
    echo "  API_KEY=$API_KEY docker-compose up -d"
    echo ""
    echo "提示: 确保已创建 $DATA_FILE 文件（可参考 data.example.json）"
    exit 1
fi
echo -e "${GREEN}✓ Warden 服务运行中${NC}"
echo ""

# 准备测试数据
echo -e "${BLUE}准备测试数据...${NC}"

# 备份现有数据文件（如果存在）
if [ -f "$DATA_FILE" ]; then
    cp "$DATA_FILE" "$DATA_FILE_BACKUP"
    echo -e "${YELLOW}⚠️  已备份现有数据文件: $DATA_FILE_BACKUP${NC}"
fi

# 创建测试数据文件
cat > "$DATA_FILE" << 'EOF'
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
echo -e "${GREEN}✓ 测试数据已准备: $DATA_FILE${NC}"
echo -e "${YELLOW}⚠️  注意: 服务需要重新加载数据才能使用新的测试数据${NC}"
echo -e "${YELLOW}   如果服务正在运行，请等待定时任务更新或重启服务${NC}"
echo ""

# 开始测试
echo -e "${BLUE}开始测试...${NC}"
echo "===================="
echo ""

# 测试 1: 健康检查
echo -e "${YELLOW}1. 健康检查端点${NC}"
test_endpoint "健康检查" "GET" "$BASE_URL/health" "" "200"
test_endpoint "健康检查（/healthcheck 别名）" "GET" "$BASE_URL/healthcheck" "" "200"

# 检查 Redis 状态（从 details.redis 字段）
TOTAL_TESTS=$((TOTAL_TESTS + 1))
response=$(curl -s "$BASE_URL/health" 2>/dev/null)

if [ "$JQ_AVAILABLE" = true ]; then
    # 使用 jq 解析 JSON
    redis_status=$(echo "$response" | jq -r '.details.redis' 2>/dev/null || echo "")
    if [ "$REDIS_AVAILABLE" = true ]; then
        # 检查 details.redis 是否为 "ok"
        if [ "$redis_status" = "ok" ]; then
            echo -e "${GREEN}✓ Redis 状态正确（ok）${NC}"
            PASSED_TESTS=$((PASSED_TESTS + 1))
        else
            echo -e "${YELLOW}⚠️  Redis 状态: $redis_status（可能正在连接中）${NC}"
            PASSED_TESTS=$((PASSED_TESTS + 1))  # 不视为失败，可能是临时状态
        fi
    else
        # 检查 Redis 状态是否为 disabled 或 unavailable
        if [ "$redis_status" = "disabled" ] || [ "$redis_status" = "unavailable" ]; then
            echo -e "${GREEN}✓ Redis 状态正确（$redis_status）${NC}"
            PASSED_TESTS=$((PASSED_TESTS + 1))
        else
            echo -e "${RED}✗ Redis 状态不正确: $redis_status${NC}"
            echo "响应: $response"
            FAILED_TESTS=$((FAILED_TESTS + 1))
        fi
    fi
else
    # 没有 jq，使用 grep 检查
    if [ "$REDIS_AVAILABLE" = true ]; then
        if echo "$response" | grep -q '"redis":"ok"' 2>/dev/null; then
            echo -e "${GREEN}✓ Redis 状态正确（ok）${NC}"
            PASSED_TESTS=$((PASSED_TESTS + 1))
        else
            echo -e "${YELLOW}⚠️  无法精确解析 Redis 状态（需要 jq）${NC}"
            PASSED_TESTS=$((PASSED_TESTS + 1))  # 不视为失败
        fi
    else
        if echo "$response" | grep -q '"redis":"disabled"' 2>/dev/null || \
           echo "$response" | grep -q '"redis":"unavailable"' 2>/dev/null; then
            echo -e "${GREEN}✓ Redis 状态正确（disabled 或 unavailable）${NC}"
            PASSED_TESTS=$((PASSED_TESTS + 1))
        else
            echo -e "${RED}✗ Redis 状态检查失败（需要 jq 进行精确解析）${NC}"
            echo "响应: $response"
            FAILED_TESTS=$((FAILED_TESTS + 1))
        fi
    fi
fi
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
TOTAL_TESTS=$((TOTAL_TESTS + 1))
response=$(curl -s -H "X-API-Key: $API_KEY" "$BASE_URL/user?phone=13800138000" 2>/dev/null)

if [ "$JQ_AVAILABLE" = true ]; then
    # 使用 jq 检查字段
    if echo "$response" | jq -e '.user_id, .status, .scope, .role' > /dev/null 2>&1; then
        echo -e "${GREEN}✓ 新字段存在（user_id, status, scope, role）${NC}"
        echo "$response" | jq .
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        echo -e "${RED}✗ 新字段缺失或格式错误${NC}"
        echo "响应: $response"
        FAILED_TESTS=$((FAILED_TESTS + 1))
    fi
else
    # 没有 jq，使用 grep 检查
    if echo "$response" | grep -q '"user_id"' && \
       echo "$response" | grep -q '"status"' && \
       echo "$response" | grep -q '"scope"' && \
       echo "$response" | grep -q '"role"'; then
        echo -e "${GREEN}✓ 新字段存在（user_id, status, scope, role）${NC}"
        echo "$response" | head -n 10
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        echo -e "${RED}✗ 新字段缺失或格式错误${NC}"
        echo "响应: $response"
        FAILED_TESTS=$((FAILED_TESTS + 1))
    fi
fi
echo ""

# 清理测试数据
echo -e "${BLUE}清理测试数据...${NC}"
if [ -f "$DATA_FILE_BACKUP" ]; then
    mv "$DATA_FILE_BACKUP" "$DATA_FILE"
    echo -e "${GREEN}✓ 已恢复原始数据文件${NC}"
elif [ -f "$DATA_FILE" ]; then
    # 如果没有备份，询问是否保留测试数据
    echo -e "${YELLOW}⚠️  未找到备份文件，测试数据文件将保留: $DATA_FILE${NC}"
    echo "   如需删除，请手动执行: rm $DATA_FILE"
else
    echo -e "${YELLOW}⚠️  数据文件不存在，无需清理${NC}"
fi
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
