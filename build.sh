#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

echo "========================================="
echo "  梦幻西游物价追踪 - 一键构建脚本"
echo "========================================="

# Step 1: Install frontend dependencies and build
echo ""
echo ">>> [1/2] 构建前端..."
cd web
if [ ! -d "node_modules" ]; then
    echo "安装前端依赖..."
    npm install
fi
npm run build
cd ..

# Step 2: Build Go backend
echo ""
echo ">>> [2/2] 构建后端..."
cd server
go build -o ../xyq_product .
cd ..

echo ""
echo "========================================="
echo "  构建完成！"
echo "  运行: ./xyq_product"
echo "  访问: http://localhost:8081"
echo "  管理员: admin / 123456"
echo "========================================="
