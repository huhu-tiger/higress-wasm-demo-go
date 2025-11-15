#!/bin/bash
# Go 环境设置脚本

# 设置 GOPATH
export GOPATH=/data/work/go

# 启用 Go Modules
export GO111MODULE=on

# 设置模块缓存目录到 pkg/mod
export GOMODCACHE=$GOPATH/pkg/mod

# 设置 Go 代理（使用国内镜像加速）
export GOPROXY=https://goproxy.cn,direct

# 显示当前设置
echo "=== Go 环境变量 ==="
echo "GOPATH: $GOPATH"
echo "GOMODCACHE: $GOMODCACHE"
echo "GO111MODULE: $GO111MODULE"
echo "GOPROXY: $GOPROXY"
echo "=================="

# 下载依赖
echo ""
echo "开始下载依赖..."
go mod download

echo ""
echo "依赖下载完成！"
echo "依赖包位置: $GOMODCACHE"
