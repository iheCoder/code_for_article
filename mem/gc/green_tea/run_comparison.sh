#!/bin/bash
# Green Tea GC 对比测试脚本
# 用于对比传统 GC 和 Green Tea GC 的性能差异

set -e

echo "======================================"
echo "Go 1.25 GC 性能对比测试"
echo "======================================"
echo ""

# 获取 Go 版本
GO_VERSION=$(go version)
echo "Go 版本: $GO_VERSION"
echo ""

# 测试输出目录
OUTPUT_DIR="./gc_test_results"
mkdir -p $OUTPUT_DIR

echo "======================================"
echo "第一部分：传统 GC 测试"
echo "======================================"
echo ""

# 运行传统 GC 测试
echo ">>> 运行传统 GC - GC CPU 占比测试..."
go test -v -run=TestGCCPUUsage ./mem/gc/green_tea/ -timeout 30s > $OUTPUT_DIR/traditional_gc_cpu.log 2>&1
echo "✓ 完成"

echo ">>> 运行传统 GC - 延迟测试..."
go test -v -run=TestLatencyUnderGCPressure ./mem/gc/green_tea/ -timeout 40s > $OUTPUT_DIR/traditional_latency.log 2>&1
echo "✓ 完成"

echo ">>> 运行传统 GC - 基准测试..."
go test -bench=BenchmarkGCCPUOverhead -benchmem -run=^$ ./mem/gc/green_tea/ > $OUTPUT_DIR/traditional_bench_cpu.log 2>&1
go test -bench=BenchmarkThroughput -benchmem -run=^$ ./mem/gc/green_tea/ > $OUTPUT_DIR/traditional_bench_throughput.log 2>&1
echo "✓ 完成"

echo ""
echo "======================================"
echo "第二部分：Green Tea GC 测试"
echo "======================================"
echo ""

# Green Tea GC 可能的环境变量名称
# 注意：实际的环境变量名称可能不同，需要根据 Go 1.25 文档确认
export GOEXPERIMENT=greentea

echo "已设置环境变量: GOEXPERIMENT=$GOEXPERIMENT"
echo ""

echo ">>> 运行 Green Tea GC - GC CPU 占比测试..."
go test -v -run=TestGCCPUUsage ./mem/gc/green_tea/ -timeout 30s > $OUTPUT_DIR/greentea_gc_cpu.log 2>&1
echo "✓ 完成"

echo ">>> 运行 Green Tea GC - 延迟测试..."
go test -v -run=TestLatencyUnderGCPressure ./mem/gc/green_tea/ -timeout 40s > $OUTPUT_DIR/greentea_latency.log 2>&1
echo "✓ 完成"

echo ">>> 运行 Green Tea GC - 基准测试..."
go test -bench=BenchmarkGCCPUOverhead -benchmem -run=^$ ./mem/gc/green_tea/ > $OUTPUT_DIR/greentea_bench_cpu.log 2>&1
go test -bench=BenchmarkThroughput -benchmem -run=^$ ./mem/gc/green_tea/ > $OUTPUT_DIR/greentea_bench_throughput.log 2>&1
echo "✓ 完成"

echo ""
echo "======================================"
echo "测试完成！生成对比报告..."
echo "======================================"
echo ""

# 提取关键指标
echo "## 传统 GC vs Green Tea GC 性能对比" > $OUTPUT_DIR/comparison_report.md
echo "" >> $OUTPUT_DIR/comparison_report.md

echo "### GC CPU 占比对比" >> $OUTPUT_DIR/comparison_report.md
echo "" >> $OUTPUT_DIR/comparison_report.md
echo "**传统 GC:**" >> $OUTPUT_DIR/comparison_report.md
grep "GC CPU 占比:" $OUTPUT_DIR/traditional_gc_cpu.log >> $OUTPUT_DIR/comparison_report.md || echo "无数据" >> $OUTPUT_DIR/comparison_report.md
echo "" >> $OUTPUT_DIR/comparison_report.md
echo "**Green Tea GC:**" >> $OUTPUT_DIR/comparison_report.md
grep "GC CPU 占比:" $OUTPUT_DIR/greentea_gc_cpu.log >> $OUTPUT_DIR/comparison_report.md || echo "无数据" >> $OUTPUT_DIR/comparison_report.md
echo "" >> $OUTPUT_DIR/comparison_report.md

echo "### 延迟对比" >> $OUTPUT_DIR/comparison_report.md
echo "" >> $OUTPUT_DIR/comparison_report.md
echo "**传统 GC:**" >> $OUTPUT_DIR/comparison_report.md
grep "P99 延迟:" $OUTPUT_DIR/traditional_latency.log >> $OUTPUT_DIR/comparison_report.md || echo "无数据" >> $OUTPUT_DIR/comparison_report.md
echo "" >> $OUTPUT_DIR/comparison_report.md
echo "**Green Tea GC:**" >> $OUTPUT_DIR/comparison_report.md
grep "P99 延迟:" $OUTPUT_DIR/greentea_latency.log >> $OUTPUT_DIR/comparison_report.md || echo "无数据" >> $OUTPUT_DIR/comparison_report.md
echo "" >> $OUTPUT_DIR/comparison_report.md

echo "详细报告已保存到: $OUTPUT_DIR/"
echo "- traditional_*.log: 传统 GC 测试结果"
echo "- greentea_*.log: Green Tea GC 测试结果"
echo "- comparison_report.md: 对比报告"
echo ""
echo "查看完整对比报告："
echo "cat $OUTPUT_DIR/comparison_report.md"

