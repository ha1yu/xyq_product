#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

echo "========================================="
echo "  梦幻西游物价追踪 - 一键构建脚本"
echo "========================================="

# Step 1: Install frontend dependencies and build
echo ""
echo ">>> [1/3] 构建前端..."
cd web
if [ ! -d "node_modules" ]; then
    echo "安装前端依赖..."
    npm install
fi
npm run build
cd ..

# Step 2: Build Go backend
echo ""
echo ">>> [2/3] 构建后端..."
cd server
go build -o ../xyq_product .
cd ..

# Step 3: Initialize database if not exists
echo ""
echo ">>> [3/3] 初始化数据库..."
if [ ! -f "data/mhxy.db" ]; then
    ./xyq_product --db data/mhxy.db
    echo "数据库初始化完成"
else
    echo "数据库已存在，跳过初始化"
fi

echo ""
echo "========================================="
echo "  构建完成！"
echo "  运行: ./xyq_product"
echo "  访问: http://localhost:8081"
echo "  管理员: admin / 123456"
echo "========================================="
