#!/bin/bash

# MINIMAX_API_KEY=sk-api-lA4di60txraSE-hBLQ7vqGwIra2mSiqmu2qfye3CL62qOuBn8F2BABYvfTBCJ885GfcOn-74wexZKahRfQLmZ2lmrrIq2WnK2UUAX4x4YkUQzHYdC4ob0Io ./restart.sh

if [ -z "$MINIMAX_API_KEY" ]; then
  echo "错误：请先设置 MINIMAX_API_KEY"
  echo "用法：MINIMAX_API_KEY=your_key ./restart.sh"
  exit 1
fi

PID=$(lsof -ti :3000)
if [ -n "$PID" ]; then
  echo "停止旧进程 (PID: $PID)..."
  echo "$PID" | xargs kill
  sleep 1
fi

echo "重启服务..."
go run main.go
