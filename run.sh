#!/bin/sh
# 构建并启动 startAllSs-admin Web 管理台（前台运行，Ctrl-C 退出）
# 用法: ./run.sh [--port 8787] [--host 127.0.0.1] [--token 你的令牌]
set -e
DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$DIR"
mkdir -p bin
go build -o bin/startAllSs-admin .
exec ./bin/startAllSs-admin "$@"
