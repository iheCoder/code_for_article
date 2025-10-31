package green_tea

import (
	"runtime"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// 延迟和吞吐量测试
// Green Tea GC 应该改善尾延迟和整体吞吐量

// TestLatencyUnderGCPressure 测试 GC 压力下的延迟表现
func TestLatencyUnderGCPressure(t *testing.T) {
	testCases := []struct {
		name       string
		gcPercent  int
		allocRate  int // 每秒分配次数
		measureOps int
	}{
		{"LowGCPressure_50", 200, 1000, 10000},
		{"NormalGCPressure_100", 100, 5000, 10000},
		{"HighGCPressure_50", 50, 10000, 10000},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			debug.SetGCPercent(tc.gcPercent)
			runtime.GC()

			// 启动后台分配压力
			stopCh := make(chan struct{})
			var wg sync.WaitGroup
			wg.Add(1)

			go func() {
				defer wg.Done()
				ticker := time.NewTicker(time.Second / time.Duration(tc.allocRate))
				defer ticker.Stop()

				allocations := make([][]byte, 0, 100)
				for {
					select {
					case <-stopCh:
						return
					case <-ticker.C:
						data := make([]byte, 8192)
						data[0] = 1
						allocations = append(allocations, data)
						if len(allocations) > 50 {
							allocations = allocations[25:]
						}
					}
				}
			}()

			// 等待系统稳定
			time.Sleep(100 * time.Millisecond)

			// 测量操作延迟
			latencies := make([]time.Duration, tc.measureOps)

			for i := 0; i < tc.measureOps; i++ {
				start := time.Now()

				// 模拟一个需要低延迟的操作
				performLatencySensitiveOp()

				latencies[i] = time.Since(start)

				// 适当间隔
				time.Sleep(time.Microsecond * 50)
			}

			close(stopCh)
			wg.Wait()

			// 分析延迟分布
			stats := calculateLatencyStats(latencies)

			t.Logf("========== 延迟分析 (GC Percent: %d) ==========", tc.gcPercent)
			t.Logf("样本数: %d", len(latencies))
			t.Logf("平均延迟: %.2f μs", stats.mean)
			t.Logf("P50 延迟: %.2f μs", stats.p50)
			t.Logf("P95 延迟: %.2f μs", stats.p95)
			t.Logf("P99 延迟: %.2f μs", stats.p99)
			t.Logf("P999 延迟: %.2f μs", stats.p999)
			t.Logf("最大延迟: %.2f μs", stats.max)
			t.Logf("标准差: %.2f μs", stats.stddev)
			t.Logf("=========================================")

			// Green Tea GC 应该显著降低 P99 和 P999 延迟
			if stats.p99 > 1000 { // 1ms
				t.Logf("警告: P99 延迟较高 (%.2f μs)", stats.p99)
			} else {
				t.Logf("✓ P99 延迟良好 (%.2f μs)", stats.p99)
			}
		})
	}
}

// BenchmarkThroughputUnderGC 测试 GC 对吞吐量的影响
func BenchmarkThroughputUnderGC(b *testing.B) {
	gcSettings := []int{50, 100, 200}

	for _, gcPercent := range gcSettings {
		b.Run(formatGCPercent(gcPercent), func(b *testing.B) {
			debug.SetGCPercent(gcPercent)
			runtime.GC()

			var memStats1, memStats2 runtime.MemStats
			runtime.ReadMemStats(&memStats1)

			// 启动后台分配压力
			stopCh := make(chan struct{})
			var wg sync.WaitGroup
			wg.Add(1)

			go func() {
				defer wg.Done()
				allocations := make([][]byte, 0, 100)
				ticker := time.NewTicker(time.Microsecond * 100)
				defer ticker.Stop()

				for {
					select {
					case <-stopCh:
						return
					case <-ticker.C:
						data := make([]byte, 4096)
						data[0] = 1
						allocations = append(allocations, data)
						if len(allocations) > 50 {
							allocations = allocations[25:]
						}
					}
				}
			}()

			b.ResetTimer()

			// 测量吞吐量
			var opsCompleted atomic.Int64
			for i := 0; i < b.N; i++ {
				performWorkUnit()
				opsCompleted.Add(1)
			}

			b.StopTimer()
			close(stopCh)
			wg.Wait()

			runtime.ReadMemStats(&memStats2)

			// 计算指标
			gcCPU := memStats2.GCCPUFraction * 100
			numGC := memStats2.NumGC - memStats1.NumGC
			pauseTotal := float64(memStats2.PauseTotalNs-memStats1.PauseTotalNs) / 1e6

			b.ReportMetric(gcCPU, "gc_cpu_%")
			b.ReportMetric(float64(numGC), "gc_count")
			b.ReportMetric(pauseTotal, "pause_ms")
			b.ReportMetric(float64(opsCompleted.Load())/b.Elapsed().Seconds(), "ops/sec")
		})
	}
}

// TestConcurrentThroughput 测试并发场景下的吞吐量
func TestConcurrentThroughput(t *testing.T) {
	workerCounts := []int{1, 4, 8, 16}

	for _, workers := range workerCounts {
		t.Run(formatWorkers(workers), func(t *testing.T) {
			debug.SetGCPercent(100)
			runtime.GC()

			var memStats1, memStats2 runtime.MemStats
			runtime.ReadMemStats(&memStats1)

			duration := 5 * time.Second
			stopCh := make(chan struct{})
			var wg sync.WaitGroup
			var totalOps atomic.Int64

			// 启动工作协程
			for i := 0; i < workers; i++ {
				wg.Add(1)
				go func(id int) {
					defer wg.Done()
					localOps := 0
					allocations := make([][]byte, 0, 10)

					for {
						select {
						case <-stopCh:
							totalOps.Add(int64(localOps))
							return
						default:
							// 执行工作单元
							performWorkUnit()

							// 分配一些内存
							data := make([]byte, 2048)
							data[0] = byte(id)
							allocations = append(allocations, data)

							if len(allocations) > 5 {
								allocations = allocations[3:]
							}

							localOps++
						}
					}
				}(i)
			}

			// 运行指定时长
			time.Sleep(duration)
			close(stopCh)
			wg.Wait()

			runtime.ReadMemStats(&memStats2)

			ops := totalOps.Load()
			throughput := float64(ops) / duration.Seconds()
			gcCPU := memStats2.GCCPUFraction * 100
			numGC := memStats2.NumGC - memStats1.NumGC

			t.Logf("========== 并发吞吐量测试 ==========")
			t.Logf("Worker 数量: %d", workers)
			t.Logf("测试时长: %v", duration)
			t.Logf("总操作数: %d", ops)
			t.Logf("吞吐量: %.0f ops/sec", throughput)
			t.Logf("GC CPU 占比: %.2f%%", gcCPU)
			t.Logf("GC 次数: %d", numGC)
			t.Logf("平均每秒 GC: %.1f", float64(numGC)/duration.Seconds())
			t.Logf("=====================================")
		})
	}
}

// performLatencySensitiveOp 模拟延迟敏感操作
func performLatencySensitiveOp() {
	// 创建一个小对象并进行一些计算
	data := make([]int, 100)
	sum := 0
	for i := range data {
		data[i] = i
		sum += data[i]
	}
	_ = sum
}

// performWorkUnit 执行一个工作单元
func performWorkUnit() {
	// 模拟一些计算和内存操作
	data := make([]byte, 512)
	for i := 0; i < len(data); i += 8 {
		data[i] = byte(i & 0xFF)
	}

	// 简单计算
	sum := 0
	for i := range data {
		sum += int(data[i])
	}
	_ = sum
}

// LatencyStats 延迟统计
type LatencyStats struct {
	mean   float64
	p50    float64
	p95    float64
	p99    float64
	p999   float64
	max    float64
	stddev float64
}

// calculateLatencyStats 计算延迟统计信息
func calculateLatencyStats(latencies []time.Duration) LatencyStats {
	if len(latencies) == 0 {
		return LatencyStats{}
	}

	// 转换为微秒并排序
	values := make([]float64, len(latencies))
	sum := 0.0
	maxVal := 0.0

	for i, lat := range latencies {
		us := float64(lat.Nanoseconds()) / 1000.0
		values[i] = us
		sum += us
		if us > maxVal {
			maxVal = us
		}
	}

	// 排序以计算百分位数
	sortFloat64(values)

	mean := sum / float64(len(values))

	// 计算标准差
	variance := 0.0
	for _, v := range values {
		diff := v - mean
		variance += diff * diff
	}
	stddev := 0.0
	if len(values) > 1 {
		stddev = (variance / float64(len(values)))
		stddev = sqrt(stddev)
	}

	return LatencyStats{
		mean:   mean,
		p50:    percentile(values, 0.50),
		p95:    percentile(values, 0.95),
		p99:    percentile(values, 0.99),
		p999:   percentile(values, 0.999),
		max:    maxVal,
		stddev: stddev,
	}
}

// percentile 计算百分位数
func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(float64(len(sorted)) * p)
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

// sortFloat64 简单排序
func sortFloat64(arr []float64) {
	// 使用简单的快速排序
	if len(arr) < 2 {
		return
	}
	quickSort(arr, 0, len(arr)-1)
}

func quickSort(arr []float64, low, high int) {
	if low < high {
		pi := partition(arr, low, high)
		quickSort(arr, low, pi-1)
		quickSort(arr, pi+1, high)
	}
}

func partition(arr []float64, low, high int) int {
	pivot := arr[high]
	i := low - 1
	for j := low; j < high; j++ {
		if arr[j] < pivot {
			i++
			arr[i], arr[j] = arr[j], arr[i]
		}
	}
	arr[i+1], arr[high] = arr[high], arr[i+1]
	return i + 1
}

// sqrt 简单平方根实现
func sqrt(x float64) float64 {
	if x == 0 {
		return 0
	}
	z := x
	for i := 0; i < 10; i++ {
		z = (z + x/z) / 2
	}
	return z
}

func formatGCPercent(pct int) string {
	return "GCPercent_" + itoa(pct)
}

func formatWorkers(n int) string {
	return "Workers_" + itoa(n)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}

	negative := n < 0
	if negative {
		n = -n
	}

	digits := make([]byte, 0, 10)
	for n > 0 {
		digits = append(digits, byte('0'+n%10))
		n /= 10
	}

	if negative {
		digits = append(digits, '-')
	}

	// 反转
	for i, j := 0, len(digits)-1; i < j; i, j = i+1, j-1 {
		digits[i], digits[j] = digits[j], digits[i]
	}

	return string(digits)
}
