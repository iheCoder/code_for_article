package green_tea

import (
	"runtime"
	"runtime/debug"
	"sync/atomic"
	"testing"
	"time"
)

// GC CPU 占比测试
// Green Tea GC 的主要改进之一是降低 GC CPU 占用

//	go test -v -run=TestGCCPUUsage ./mem/gc/green_tea/ -timeout 30s 2>&1 | head -100
//
// === RUN   TestGCCPUUsage
// === RUN   TestGCCPUUsage/小对象高频分配
//
//	gc_cpu_test.go:81: ========== GC CPU 占用分析 ==========
//	gc_cpu_test.go:82: 工作负载类型: high_frequency
//	gc_cpu_test.go:83: 测试时长: 5.000024375s
//	gc_cpu_test.go:84: GC 次数: 163
//	gc_cpu_test.go:85: GC CPU 占比: 0.09%
//	gc_cpu_test.go:86: STW 暂停占比: 0.10%
//	gc_cpu_test.go:87: 总暂停时间: 4.85 ms
//	gc_cpu_test.go:89: 平均每次 GC 暂停: 0.03 ms
//	gc_cpu_test.go:91: 分配字节数: 488 MB
//	gc_cpu_test.go:92: ========================================
//	gc_cpu_test.go:99: ✓ GC CPU 占比良好 (0.09%)
//
// === RUN   TestGCCPUUsage/中等对象中频分配
//
//	gc_cpu_test.go:81: ========== GC CPU 占用分析 ==========
//	gc_cpu_test.go:82: 工作负载类型: medium_frequency
//	gc_cpu_test.go:83: 测试时长: 5.000023709s
//	gc_cpu_test.go:84: GC 次数: 104
//	gc_cpu_test.go:85: GC CPU 占比: 0.07%
//	gc_cpu_test.go:86: STW 暂停占比: 0.06%
//	gc_cpu_test.go:87: 总暂停时间: 3.16 ms
//	gc_cpu_test.go:89: 平均每次 GC 暂停: 0.03 ms
//	gc_cpu_test.go:91: 分配字节数: 3121 MB
//	gc_cpu_test.go:92: ========================================
//	gc_cpu_test.go:99: ✓ GC CPU 占比良好 (0.07%)
//
// === RUN   TestGCCPUUsage/大对象低频分配
//
//	gc_cpu_test.go:81: ========== GC CPU 占用分析 ==========
//	gc_cpu_test.go:82: 工作负载类型: low_frequency
//	gc_cpu_test.go:83: 测试时长: 5.000425166s
//	gc_cpu_test.go:84: GC 次数: 14
//	gc_cpu_test.go:85: GC CPU 占比: 0.06%
//	gc_cpu_test.go:86: STW 暂停占比: 0.01%
//	gc_cpu_test.go:87: 总暂停时间: 0.57 ms
//	gc_cpu_test.go:89: 平均每次 GC 暂停: 0.04 ms
//	gc_cpu_test.go:91: 分配字节数: 4740 MB
//	gc_cpu_test.go:92: ========================================
//	gc_cpu_test.go:99: ✓ GC CPU 占比良好 (0.06%)
func TestGCCPUUsage(t *testing.T) {
	// 禁用 GC 百分比限制，让我们能观察到实际的 GC 开销
	debug.SetGCPercent(100)

	testCases := []struct {
		name         string
		allocSize    int
		allocRate    time.Duration
		duration     time.Duration
		workloadType string
	}{
		{
			name:         "小对象高频分配",
			allocSize:    1024, // 1KB
			allocRate:    time.Microsecond * 10,
			duration:     time.Second * 5,
			workloadType: "high_frequency",
		},
		{
			name:         "中等对象中频分配",
			allocSize:    1024 * 64, // 64KB
			allocRate:    time.Microsecond * 100,
			duration:     time.Second * 5,
			workloadType: "medium_frequency",
		},
		{
			name:         "大对象低频分配",
			allocSize:    1024 * 1024, // 1MB
			allocRate:    time.Millisecond,
			duration:     time.Second * 5,
			workloadType: "low_frequency",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			runtime.GC() // 清理初始状态

			var stats1, stats2 debug.GCStats
			debug.ReadGCStats(&stats1)

			// 记录开始时的运行时统计
			var memStats1 runtime.MemStats
			runtime.ReadMemStats(&memStats1)

			startTime := time.Now()

			// 执行工作负载
			runAllocationWorkload(tc.allocSize, tc.allocRate, tc.duration)

			elapsed := time.Since(startTime)

			// 读取结束时的统计
			var memStats2 runtime.MemStats
			runtime.ReadMemStats(&memStats2)
			debug.ReadGCStats(&stats2)

			// 计算 GC 指标
			numGC := memStats2.NumGC - memStats1.NumGC
			totalPauseNs := memStats2.PauseTotalNs - memStats1.PauseTotalNs
			gcCPUFraction := memStats2.GCCPUFraction

			// 估算 GC CPU 占比（包括 STW 和并发标记）
			gcCPUPercent := gcCPUFraction * 100
			pausePercent := float64(totalPauseNs) / float64(elapsed.Nanoseconds()) * 100

			t.Logf("========== GC CPU 占用分析 ==========")
			t.Logf("工作负载类型: %s", tc.workloadType)
			t.Logf("测试时长: %v", elapsed)
			t.Logf("GC 次数: %d", numGC)
			t.Logf("GC CPU 占比: %.2f%%", gcCPUPercent)
			t.Logf("STW 暂停占比: %.2f%%", pausePercent)
			t.Logf("总暂停时间: %.2f ms", float64(totalPauseNs)/1e6)
			if numGC > 0 {
				t.Logf("平均每次 GC 暂停: %.2f ms", float64(totalPauseNs)/float64(numGC)/1e6)
			}
			t.Logf("分配字节数: %d MB", (memStats2.TotalAlloc-memStats1.TotalAlloc)/(1024*1024))
			t.Logf("========================================")

			// Green Tea GC 的目标：GC CPU 占比应该显著低于传统实现
			// 根据 Go 1.25 的设计目标，GC CPU 占比应该在 5% 以下
			if gcCPUPercent > 10 {
				t.Logf("警告: GC CPU 占比较高 (%.2f%%), Green Tea GC 预期应更低", gcCPUPercent)
			} else {
				t.Logf("✓ GC CPU 占比良好 (%.2f%%)", gcCPUPercent)
			}
		})
	}
}

// BenchmarkGCCPUOverhead 基准测试 GC CPU 开销
//
//	go test -v -run=TestGCCPUUsage ./mem/gc/green_tea/ -timeout 30s 2>&1 | head -100
//
// === RUN   TestGCCPUUsage
// === RUN   TestGCCPUUsage/小对象高频分配
//
//	gc_cpu_test.go:81: ========== GC CPU 占用分析 ==========
//	gc_cpu_test.go:82: 工作负载类型: high_frequency
//	gc_cpu_test.go:83: 测试时长: 5.000024375s
//	gc_cpu_test.go:84: GC 次数: 163
//	gc_cpu_test.go:85: GC CPU 占比: 0.09%
//	gc_cpu_test.go:86: STW 暂停占比: 0.10%
//	gc_cpu_test.go:87: 总暂停时间: 4.85 ms
//	gc_cpu_test.go:89: 平均每次 GC 暂停: 0.03 ms
//	gc_cpu_test.go:91: 分配字节数: 488 MB
//	gc_cpu_test.go:92: ========================================
//	gc_cpu_test.go:99: ✓ GC CPU 占比良好 (0.09%)
//
// === RUN   TestGCCPUUsage/中等对象中频分配
//
//	gc_cpu_test.go:81: ========== GC CPU 占用分析 ==========
//	gc_cpu_test.go:82: 工作负载类型: medium_frequency
//	gc_cpu_test.go:83: 测试时长: 5.000023709s
//	gc_cpu_test.go:84: GC 次数: 104
//	gc_cpu_test.go:85: GC CPU 占比: 0.07%
//	gc_cpu_test.go:86: STW 暂停占比: 0.06%
//	gc_cpu_test.go:87: 总暂停时间: 3.16 ms
//	gc_cpu_test.go:89: 平均每次 GC 暂停: 0.03 ms
//	gc_cpu_test.go:91: 分配字节数: 3121 MB
//	gc_cpu_test.go:92: ========================================
//	gc_cpu_test.go:99: ✓ GC CPU 占比良好 (0.07%)
//
// === RUN   TestGCCPUUsage/大对象低频分配
//
//	gc_cpu_test.go:81: ========== GC CPU 占用分析 ==========
//	gc_cpu_test.go:82: 工作负载类型: low_frequency
//	gc_cpu_test.go:83: 测试时长: 5.000425166s
//	gc_cpu_test.go:84: GC 次数: 14
//	gc_cpu_test.go:85: GC CPU 占比: 0.06%
//	gc_cpu_test.go:86: STW 暂停占比: 0.01%
//	gc_cpu_test.go:87: 总暂停时间: 0.57 ms
//	gc_cpu_test.go:89: 平均每次 GC 暂停: 0.04 ms
//	gc_cpu_test.go:91: 分配字节数: 4740 MB
//	gc_cpu_test.go:92: ========================================
//	gc_cpu_test.go:99: ✓ GC CPU 占比良好 (0.06%)
func BenchmarkGCCPUOverhead(b *testing.B) {
	benchmarks := []struct {
		name      string
		allocSize int
	}{
		{"SmallAlloc_128B", 128},
		{"MediumAlloc_8KB", 8 * 1024},
		{"LargeAlloc_1MB", 1024 * 1024},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			debug.SetGCPercent(100)
			runtime.GC()

			var memStats1, memStats2 runtime.MemStats
			runtime.ReadMemStats(&memStats1)

			b.ResetTimer()

			// 执行分配
			allocations := make([][]byte, 0, b.N)
			for i := 0; i < b.N; i++ {
				data := make([]byte, bm.allocSize)
				allocations = append(allocations, data)

				// 周期性清理，模拟真实场景
				if i%1000 == 0 && i > 0 {
					allocations = allocations[:0]
				}
			}

			b.StopTimer()

			runtime.ReadMemStats(&memStats2)
			gcCPUPercent := memStats2.GCCPUFraction * 100
			numGC := memStats2.NumGC - memStats1.NumGC

			b.ReportMetric(gcCPUPercent, "gc_cpu_%")
			b.ReportMetric(float64(numGC), "gc_count")
			b.ReportMetric(float64(memStats2.PauseTotalNs-memStats1.PauseTotalNs)/1e6, "pause_ms")

			// 防止编译器优化
			_ = allocations
		})
	}
}

// runAllocationWorkload 运行分配工作负载
func runAllocationWorkload(allocSize int, allocRate time.Duration, duration time.Duration) {
	deadline := time.Now().Add(duration)
	ticker := time.NewTicker(allocRate)
	defer ticker.Stop()

	allocations := make([][]byte, 0, 1000)

	for time.Now().Before(deadline) {
		select {
		case <-ticker.C:
			// 分配内存
			data := make([]byte, allocSize)
			// 写入数据防止优化
			for i := 0; i < len(data); i += 64 {
				data[i] = byte(i)
			}
			allocations = append(allocations, data)

			// 周期性清理，模拟对象生命周期
			if len(allocations) > 500 {
				allocations = allocations[250:]
			}
		}
	}

	// 防止编译器优化
	_ = allocations
}

// getCPUTime 获取当前进程的 CPU 时间（纳秒）
func getCPUTime() int64 {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	// 注意：这是一个简化实现，实际 CPU 时间需要通过系统调用获取
	return time.Now().UnixNano()
}

// TestGCCPUFractionTracking 跟踪 GC CPU 占比随时间的变化
func TestGCCPUFractionTracking(t *testing.T) {
	debug.SetGCPercent(100)
	runtime.GC()

	duration := 10 * time.Second
	sampleInterval := 500 * time.Millisecond

	// 启动分配工作负载
	stopCh := make(chan struct{})
	var allocCount atomic.Int64

	go func() {
		ticker := time.NewTicker(time.Microsecond * 50)
		defer ticker.Stop()

		allocations := make([][]byte, 0, 100)
		for {
			select {
			case <-stopCh:
				return
			case <-ticker.C:
				data := make([]byte, 4096)
				data[0] = byte(allocCount.Add(1))
				allocations = append(allocations, data)

				if len(allocations) > 50 {
					allocations = allocations[25:]
				}
			}
		}
	}()

	// 采样 GC 指标
	t.Logf("开始跟踪 GC CPU 占比 (持续 %v)...", duration)
	t.Logf("时间(s)\tGC_CPU%%\tNumGC\tHeapMB\tPauseMs")

	startTime := time.Now()
	ticker := time.NewTicker(sampleInterval)
	defer ticker.Stop()

	var lastNumGC uint32
	var lastPauseNs uint64

	for {
		select {
		case <-ticker.C:
			var m runtime.MemStats
			runtime.ReadMemStats(&m)

			elapsed := time.Since(startTime).Seconds()
			gcCPUPercent := m.GCCPUFraction * 100
			heapMB := float64(m.HeapAlloc) / (1024 * 1024)

			deltaGC := m.NumGC - lastNumGC
			deltaPauseNs := m.PauseTotalNs - lastPauseNs
			pauseMs := float64(deltaPauseNs) / 1e6

			t.Logf("%.1f\t%.2f\t%d\t%.1f\t%.2f",
				elapsed, gcCPUPercent, deltaGC, heapMB, pauseMs)

			lastNumGC = m.NumGC
			lastPauseNs = m.PauseTotalNs

			if time.Since(startTime) >= duration {
				close(stopCh)
				return
			}
		}
	}
}
