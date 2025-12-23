# Benchmark 测试文档

## 概述

本目录包含用于测试 AI 数据脱敏插件性能和内存使用的 benchmark 测试文件。

## 测试文件

### 1. `lib/check_bench_test.go`
测试敏感词检测相关的函数：
- `CheckMessage`: 敏感词检测
- `FindSensitiveWordMatches`: 敏感词位置查找

### 2. `lib/handler_bench_test.go`
测试流式处理相关的函数：
- `ProcessOpenAIStreamResponse`: 流式响应处理
- 缓冲区管理
- 滑动窗口机制

## 运行测试

### 快速运行

```bash
# 运行所有测试
./run_benchmark.sh

# 或手动运行
go test -v ./lib/...
```

### 运行准确率测试

```bash
# 运行准确率测试
go test -v -run "TestAccuracy" ./lib/...

# 运行内存使用测试
go test -v -run "TestMemoryUsage" ./lib/...
```

### 运行性能基准测试

```bash
# 运行所有基准测试
go test -bench=. -benchmem ./lib/...

# 运行特定基准测试
go test -bench=BenchmarkCheckMessage -benchmem ./lib/...

# 运行并发测试
go test -bench=BenchmarkConcurrent -benchmem ./lib/...
```

### 生成性能分析报告

```bash
# 生成内存分析报告
go test -bench=BenchmarkMemory -benchmem -memprofile=mem.prof ./lib/...
go tool pprof mem.prof

# 生成 CPU 分析报告
go test -bench=BenchmarkCheckMessage -cpuprofile=cpu.prof ./lib/...
go tool pprof cpu.prof
```

## 测试指标

### 准确率指标

- **检测准确率**: 正确检测敏感词的比例
- **误报率**: 将正常文本误判为敏感词的比例
- **漏报率**: 未检测到敏感词的比例
- **位置准确性**: 敏感词位置查找的准确性

### 性能指标

- **吞吐量**: 每秒处理的文本数量
- **延迟**: 单次检测的平均时间
- **并发性能**: 并发场景下的性能表现

### 内存指标

- **内存分配**: 每次操作分配的内存大小
- **分配次数**: 每次操作的内存分配次数
- **内存峰值**: 处理过程中的最大内存使用

## 查看结果

### 基准测试结果格式

```
BenchmarkCheckMessage_NonStream-8    1000000    1200 ns/op    512 B/op    2 allocs/op
```

说明：
- `1000000`: 执行的迭代次数
- `1200 ns/op`: 每次操作的平均时间（纳秒）
- `512 B/op`: 每次操作分配的内存（字节）
- `2 allocs/op`: 每次操作的内存分配次数

### 内存分析

使用 `go tool pprof` 查看详细的内存使用情况：

```bash
go tool pprof mem.prof
(pprof) top          # 查看内存使用最多的函数
(pprof) list CheckMessage  # 查看特定函数的详细内存使用
(pprof) web          # 生成可视化图表（需要安装 graphviz）
```

## 性能优化建议

根据 benchmark 结果，可以关注以下方面：

1. **内存分配优化**
   - 减少不必要的内存分配
   - 使用对象池复用内存
   - 预分配切片和 map 容量

2. **算法优化**
   - 优化字符串搜索算法
   - 使用更高效的数据结构
   - 减少重复计算

3. **并发优化**
   - 减少锁竞争
   - 使用无锁数据结构
   - 优化缓存策略

## 预期性能指标

基于当前实现，预期性能指标：

- **检测速度**: 
  - 短文本（<100 字符）: < 1μs
  - 中等文本（100-1000 字符）: < 10μs
  - 长文本（>1000 字符）: < 100μs

- **内存使用**:
  - 单次检测: < 1KB
  - 流式处理缓冲区: 10KB（可配置）

- **准确率**:
  - 检测准确率: > 99.9%
  - 误报率: < 0.1%

## 持续集成

可以将 benchmark 测试集成到 CI/CD 流程中：

```yaml
# .github/workflows/benchmark.yml
name: Benchmark
on: [push, pull_request]
jobs:
  benchmark:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - uses: actions/setup-go@v2
      - run: go test -bench=. -benchmem ./lib/...
```

## 注意事项

1. **测试环境**: 确保测试环境稳定，避免其他进程影响结果
2. **预热**: 首次运行可能较慢，建议先运行一次预热
3. **多次运行**: 建议多次运行取平均值，以获得更准确的结果
4. **内存分析**: 内存分析会生成较大的文件，注意清理

