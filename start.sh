#!/bin/bash
set -e

if [ -z "$MINIMAX_API_KEY" ]; then
  echo "错误：请先设置 MINIMAX_API_KEY"
  echo "用法：MINIMAX_API_KEY=your_key ./start.sh"
  exit 1
fi

echo "启动服务..."
go run main.go
