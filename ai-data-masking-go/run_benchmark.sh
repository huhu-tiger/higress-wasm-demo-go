#!/bin/bash

# Benchmark 测试脚本
# 用于运行性能测试和内存分析

set -e

echo "=========================================="
echo "AI Data Masking - Benchmark 测试"
echo "=========================================="
echo ""

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 1. 运行准确率测试
echo -e "${GREEN}[1/4] 运行准确率测试...${NC}"
go test -v -run "TestAccuracy" ./lib/... 2>&1 | tee test_accuracy.log
echo ""

# 2. 运行内存使用测试
echo -e "${GREEN}[2/4] 运行内存使用测试...${NC}"
go test -v -run "TestMemoryUsage" ./lib/... 2>&1 | tee test_memory.log
echo ""

# 3. 运行性能基准测试
echo -e "${GREEN}[3/4] 运行性能基准测试...${NC}"
go test -bench=. -benchmem -benchtime=3s ./lib/... 2>&1 | tee benchmark.log
echo ""

# 4. 生成详细的内存分析报告
echo -e "${GREEN}[4/4] 生成内存分析报告...${NC}"
go test -bench=BenchmarkMemory -benchmem -memprofile=mem.prof ./lib/... > /dev/null 2>&1
if [ -f mem.prof ]; then
    echo "内存分析文件已生成: mem.prof"
    echo "使用以下命令查看详细报告:"
    echo "  go tool pprof mem.prof"
    echo "  (pprof) top"
    echo "  (pprof) web"
fi
echo ""

# 5. 生成 CPU 分析报告（可选）
echo -e "${YELLOW}[可选] 生成 CPU 分析报告...${NC}"
go test -bench=BenchmarkCheckMessage -cpuprofile=cpu.prof ./lib/... > /dev/null 2>&1
if [ -f cpu.prof ]; then
    echo "CPU 分析文件已生成: cpu.prof"
    echo "使用以下命令查看详细报告:"
    echo "  go tool pprof cpu.prof"
fi
echo ""

# 6. 汇总结果
echo "=========================================="
echo "测试完成！结果文件："
echo "  - test_accuracy.log: 准确率测试结果"
echo "  - test_memory.log: 内存使用测试结果"
echo "  - benchmark.log: 性能基准测试结果"
echo "  - mem.prof: 内存分析文件（如生成）"
echo "  - cpu.prof: CPU 分析文件（如生成）"
echo "=========================================="

# 显示关键指标
if [ -f benchmark.log ]; then
    echo ""
    echo -e "${GREEN}关键性能指标：${NC}"
    echo "----------------------------------------"
    grep -E "Benchmark|B/op|allocs/op" benchmark.log | head -20
fi

