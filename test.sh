#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

PORT=$((30000 + RANDOM % 10000))
BASE_URL="http://localhost:$PORT"
PASS=0
FAIL=0

echo "========================================="
echo "  梦幻西游物价追踪 - 自动测试"
echo "========================================="

# Cleanup
pkill -f "xyq_product --port $PORT" 2>/dev/null || true
sleep 1

# Remove old test db
rm -f data/test.db

# Start server
echo ""
echo ">>> 启动测试服务器 (port $PORT)..."
./xyq_product --port $PORT --db data/test.db --init-db --xlsx "$(pwd)/梦幻将军物价表.xlsx" > /tmp/test_server.log 2>&1 &
SERVER_PID=$!
sleep 2

# Check server is running
if ! kill -0 $SERVER_PID 2>/dev/null; then
    echo "FATAL: Server failed to start!"
    cat /tmp/test_server.log
    exit 1
fi

check() {
    local name="$1"
    local expected="$2"
    local actual="$3"
    if echo "$actual" | grep -q "$expected"; then
        echo "  ✅ $name"
        PASS=$((PASS + 1))
    else
        echo "  ❌ $name"
        echo "     Expected: $expected"
        echo "     Got: $actual"
        FAIL=$((FAIL + 1))
    fi
}

echo ""
echo ">>> API 测试..."

# 1. Login - correct
RESP=$(curl -s -X POST $BASE_URL/api/login -H "Content-Type: application/json" -d '{"username":"admin","password":"123456"}')
check "登录 - 正确密码" "token" "$RESP"
TOKEN=$(echo "$RESP" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

# 2. Login - wrong password
RESP=$(curl -s -X POST $BASE_URL/api/login -H "Content-Type: application/json" -d '{"username":"admin","password":"wrong"}')
check "登录 - 错误密码" "invalid credentials" "$RESP"

# 3. Get categories
RESP=$(curl -s $BASE_URL/api/categories)
check "获取分类列表" "消耗品" "$RESP"

# 4. Get products
RESP=$(curl -s $BASE_URL/api/products)
check "获取商品列表" "梦幻精品粽子" "$RESP"

# 5. Unauthorized create
RESP=$(curl -s -o /dev/null -w "%{http_code}" -X POST $BASE_URL/api/products -H "Content-Type: application/json" -d '{"name":"test","category_id":1}')
check "未授权创建商品 (401)" "401" "$RESP"

# 6. Authorized create
RESP=$(curl -s -X POST $BASE_URL/api/products -H "Content-Type: application/json" -H "Authorization: Bearer $TOKEN" -d '{"name":"测试商品","category_id":1}')
check "授权创建商品" "id" "$RESP"
NEW_ID=$(echo "$RESP" | grep -o '"id":[0-9]*' | cut -d':' -f2)

# 7. Get prices for product 1
RESP=$(curl -s "$BASE_URL/api/products/1/prices")
check "获取商品价格记录" "price_date" "$RESP"

# 8. Trend data
RESP=$(curl -s "$BASE_URL/api/trend?product_ids=1,3")
check "获取走势数据" "product_name" "$RESP"

# 9. Update product
RESP=$(curl -s -X PUT "$BASE_URL/api/products/$NEW_ID" -H "Content-Type: application/json" -H "Authorization: Bearer $TOKEN" -d '{"name":"改名测试","category_id":2,"remark":"测试备注"}')
check "更新商品" "ok" "$RESP"

# 10. Delete product
RESP=$(curl -s -X DELETE "$BASE_URL/api/products/$NEW_ID" -H "Authorization: Bearer $TOKEN")
check "删除商品" "ok" "$RESP"

# 11. Add price
RESP=$(curl -s -X POST $BASE_URL/api/prices -H "Content-Type: application/json" -H "Authorization: Bearer $TOKEN" -d '{"product_id":1,"price_date":"2026-01-01","price":99.9}')
check "添加价格记录" "id" "$RESP"
PRICE_ID=$(echo "$RESP" | grep -o '"id":[0-9]*' | cut -d':' -f2)

# 12. Delete price
RESP=$(curl -s -X DELETE "$BASE_URL/api/prices/$PRICE_ID" -H "Authorization: Bearer $TOKEN")
check "删除价格记录" "ok" "$RESP"

# 13. Frontend
RESP=$(curl -s -o /dev/null -w "%{http_code}" $BASE_URL/)
check "前端页面加载 (200)" "200" "$RESP"

# 14. Filter products
RESP=$(curl -s "$BASE_URL/api/products?category_id=1")
check "按分类筛选商品" "消耗品" "$RESP"

# Cleanup
kill $SERVER_PID 2>/dev/null || true
rm -f data/test.db

echo ""
echo "========================================="
echo "  测试结果: ✅ $PASS 通过, ❌ $FAIL 失败"
echo "========================================="

if [ $FAIL -gt 0 ]; then
    echo ""
    echo "服务器日志:"
    cat /tmp/test_server.log
    exit 1
fi

exit 0
