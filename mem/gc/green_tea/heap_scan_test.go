package green_tea

import (
	"runtime"
	"runtime/debug"
	"testing"
	"time"
)

// 堆扫描效率测试
// Green Tea GC 的一个关键改进是提升堆扫描效率，减少扫描时间

// TestHeapScanEfficiency 测试堆扫描效率
func TestHeapScanEfficiency(t *testing.T) {
	testCases := []struct {
		name           string
		objectCount    int
		objectSize     int
		pointerDensity float64 // 指针密度（0-1）
	}{
		{
			name:           "大量小对象_低指针密度",
			objectCount:    100000,
			objectSize:     128,
			pointerDensity: 0.1,
		},
		{
			name:           "大量小对象_高指针密度",
			objectCount:    100000,
			objectSize:     128,
			pointerDensity: 0.8,
		},
		{
			name:           "中等数量中等对象",
			objectCount:    10000,
			objectSize:     4096,
			pointerDensity: 0.5,
		},
		{
			name:           "少量大对象",
			objectCount:    1000,
			objectSize:     65536,
			pointerDensity: 0.3,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			debug.SetGCPercent(100)
			runtime.GC()

			// 创建堆结构
			t.Logf("创建堆结构: %d 个对象, 每个 %d 字节, 指针密度 %.1f%%",
				tc.objectCount, tc.objectSize, tc.pointerDensity*100)

			var memStats1 runtime.MemStats
			runtime.ReadMemStats(&memStats1)

			heap := createHeapStructure(tc.objectCount, tc.objectSize, tc.pointerDensity)

			var memStats2 runtime.MemStats
			runtime.ReadMemStats(&memStats2)

			heapSize := memStats2.HeapAlloc - memStats1.HeapAlloc

			t.Logf("堆大小: %.2f MB", float64(heapSize)/(1024*1024))

			// 强制触发 GC 并测量扫描时间
			scanResults := measureGCScanPerformance(t, 5)

			t.Logf("========== 堆扫描效率分析 ==========")
			t.Logf("平均 GC 周期时长: %.2f ms", scanResults.avgCycleTime)
			t.Logf("平均标记时间: %.2f ms", scanResults.avgMarkTime)
			t.Logf("平均扫描速率: %.2f MB/ms", scanResults.scanRate)
			t.Logf("平均 STW 时间: %.2f ms", scanResults.avgSTWTime)
			t.Logf("堆利用率: %.1f%%", scanResults.heapUtilization)
			t.Logf("对象存活率: %.1f%%", scanResults.survivalRate*100)
			t.Logf("=====================================")

			// Green Tea GC 应该提供更高的扫描速率
			if scanResults.scanRate < 100 {
				t.Logf("警告: 扫描速率较低 (%.2f MB/ms)", scanResults.scanRate)
			} else {
				t.Logf("✓ 扫描速率良好 (%.2f MB/ms)", scanResults.scanRate)
			}

			// 保持引用避免被优化
			_ = heap
		})
	}
}

// BenchmarkHeapScanning 基准测试不同堆结构的扫描性能
func BenchmarkHeapScanning(b *testing.B) {
	scenarios := []struct {
		name    string
		objects int
		size    int
		density float64
	}{
		{"SmallObjects_LowPointers", 50000, 64, 0.1},
		{"SmallObjects_HighPointers", 50000, 64, 0.8},
		{"MediumObjects_MixedPointers", 5000, 4096, 0.5},
		{"LargeObjects_LowPointers", 500, 65536, 0.2},
	}

	for _, sc := range scenarios {
		b.Run(sc.name, func(b *testing.B) {
			debug.SetGCPercent(100)

			// 创建堆结构
			heap := createHeapStructure(sc.objects, sc.size, sc.density)

			var memStats1, memStats2 runtime.MemStats
			runtime.ReadMemStats(&memStats1)

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				runtime.GC()
			}

			b.StopTimer()

			runtime.ReadMemStats(&memStats2)

			avgPause := float64(memStats2.PauseTotalNs-memStats1.PauseTotalNs) / float64(b.N) / 1e6
			numGC := memStats2.NumGC - memStats1.NumGC

			b.ReportMetric(avgPause, "pause_ms/gc")
			b.ReportMetric(float64(numGC), "gc_count")
			b.ReportMetric(float64(memStats2.HeapAlloc)/(1024*1024), "heap_mb")

			_ = heap
		})
	}
}

// TestScanWorkDistribution 测试扫描工作分配
func TestScanWorkDistribution(t *testing.T) {
	debug.SetGCPercent(100)
	runtime.GC()

	// 创建复杂的对象图
	objectGraph := createComplexObjectGraph(10000)

	t.Logf("创建了复杂对象图，开始测量 GC 扫描性能...")

	// 测量多次 GC 的性能
	iterations := 10
	var totalPauseNs uint64
	var totalMarkTime float64

	var memStats1, memStats2 runtime.MemStats

	for i := 0; i < iterations; i++ {
		runtime.ReadMemStats(&memStats1)
		startGC := time.Now()

		runtime.GC()

		gcDuration := time.Since(startGC)
		runtime.ReadMemStats(&memStats2)

		pauseNs := memStats2.PauseTotalNs - memStats1.PauseTotalNs
		totalPauseNs += pauseNs
		totalMarkTime += gcDuration.Seconds() * 1000 // ms

		if i%3 == 0 {
			t.Logf("GC #%d: 周期=%0.2f ms, STW=%0.2f ms",
				i+1, gcDuration.Seconds()*1000, float64(pauseNs)/1e6)
		}
	}

	avgPause := float64(totalPauseNs) / float64(iterations) / 1e6
	avgMarkTime := totalMarkTime / float64(iterations)

	t.Logf("========== 扫描工作分配分析 ==========")
	t.Logf("迭代次数: %d", iterations)
	t.Logf("平均 STW 暂停: %.2f ms", avgPause)
	t.Logf("平均标记时间: %.2f ms", avgMarkTime)
	t.Logf("并发标记占比: %.1f%%", (1-avgPause/avgMarkTime)*100)
	t.Logf("======================================")

	_ = objectGraph
}

// HeapObject 堆对象
type HeapObject struct {
	data     []byte
	pointers []*HeapObject
	id       int
}

// createHeapStructure 创建堆结构
func createHeapStructure(objectCount, objectSize int, pointerDensity float64) []*HeapObject {
	objects := make([]*HeapObject, objectCount)

	// 计算每个对象应该有多少指针
	// 假设每个指针占 8 字节
	maxPointers := objectSize / 64 // 保守估计
	if maxPointers < 1 {
		maxPointers = 1
	}

	numPointers := int(float64(maxPointers) * pointerDensity)
	if numPointers < 0 {
		numPointers = 0
	}

	// 创建对象
	for i := 0; i < objectCount; i++ {
		obj := &HeapObject{
			data:     make([]byte, objectSize),
			pointers: make([]*HeapObject, numPointers),
			id:       i,
		}

		// 填充数据
		for j := 0; j < len(obj.data); j += 8 {
			obj.data[j] = byte(i & 0xFF)
		}

		objects[i] = obj
	}

	// 建立指针关系
	for i := 0; i < objectCount; i++ {
		for j := 0; j < numPointers && j < len(objects[i].pointers); j++ {
			// 创建随机引用（使用简单的伪随机）
			targetIdx := (i*7 + j*13) % objectCount
			objects[i].pointers[j] = objects[targetIdx]
		}
	}

	return objects
}

// createComplexObjectGraph 创建复杂对象图
func createComplexObjectGraph(nodeCount int) []*GraphNode {
	nodes := make([]*GraphNode, nodeCount)

	// 创建节点
	for i := 0; i < nodeCount; i++ {
		nodes[i] = &GraphNode{
			id:       i,
			data:     make([]byte, 256),
			children: make([]*GraphNode, 0, 5),
		}
	}

	// 建立树状和交叉引用
	for i := 1; i < nodeCount; i++ {
		// 树状引用
		parent := (i - 1) / 3
		if parent < len(nodes) {
			nodes[parent].children = append(nodes[parent].children, nodes[i])
		}

		// 交叉引用
		if i%7 == 0 {
			cross := (i * 13) % nodeCount
			nodes[i].children = append(nodes[i].children, nodes[cross])
		}
	}

	return nodes
}

// GraphNode 图节点
type GraphNode struct {
	id       int
	data     []byte
	children []*GraphNode
}

// ScanResults GC 扫描结果
type ScanResults struct {
	avgCycleTime    float64
	avgMarkTime     float64
	avgSTWTime      float64
	scanRate        float64
	heapUtilization float64
	survivalRate    float64
}

// measureGCScanPerformance 测量 GC 扫描性能
func measureGCScanPerformance(_ *testing.T, iterations int) ScanResults {
	var totalCycleTime float64
	var totalMarkTime float64
	var totalSTWTime float64
	var totalHeapScanned uint64

	var memStatsBefore, memStatsAfter runtime.MemStats

	for i := 0; i < iterations; i++ {
		runtime.ReadMemStats(&memStatsBefore)

		startTime := time.Now()
		runtime.GC()
		cycleTime := time.Since(startTime)

		runtime.ReadMemStats(&memStatsAfter)

		// 累计指标
		pauseNs := memStatsAfter.PauseTotalNs - memStatsBefore.PauseTotalNs
		totalCycleTime += cycleTime.Seconds() * 1000 // ms
		totalSTWTime += float64(pauseNs) / 1e6       // ms
		totalMarkTime += cycleTime.Seconds() * 1000  // ms
		totalHeapScanned += memStatsAfter.HeapAlloc

		// 短暂休息
		time.Sleep(10 * time.Millisecond)
	}

	avgCycleTime := totalCycleTime / float64(iterations)
	avgMarkTime := totalMarkTime / float64(iterations)
	avgSTWTime := totalSTWTime / float64(iterations)

	// 计算扫描速率 (MB/ms)
	avgHeapMB := float64(totalHeapScanned/uint64(iterations)) / (1024 * 1024)
	scanRate := avgHeapMB / avgMarkTime

	// 读取最终统计
	var finalStats runtime.MemStats
	runtime.ReadMemStats(&finalStats)

	heapUtilization := float64(finalStats.HeapAlloc) / float64(finalStats.HeapSys) * 100
	survivalRate := 0.8 // 简化估算

	return ScanResults{
		avgCycleTime:    avgCycleTime,
		avgMarkTime:     avgMarkTime,
		avgSTWTime:      avgSTWTime,
		scanRate:        scanRate,
		heapUtilization: heapUtilization,
		survivalRate:    survivalRate,
	}
}

// TestGCScannerOptimizations 测试扫描器优化
func TestGCScannerOptimizations(t *testing.T) {
	scenarios := []struct {
		name        string
		setupFn     func() interface{}
		description string
	}{
		{
			name: "连续内存布局",
			setupFn: func() interface{} {
				// 创建连续的数组，对缓存友好
				return make([]int64, 1000000)
			},
			description: "测试连续内存的扫描效率",
		},
		{
			name: "分散指针结构",
			setupFn: func() interface{} {
				// 创建指针密集的结构
				nodes := make([]*GraphNode, 10000)
				for i := range nodes {
					nodes[i] = &GraphNode{
						id:   i,
						data: make([]byte, 128),
					}
				}
				return nodes
			},
			description: "测试分散指针的扫描效率",
		},
		{
			name: "混合结构",
			setupFn: func() interface{} {
				type MixedStruct struct {
					numbers []int64
					objects []*HeapObject
					data    []byte
				}
				return &MixedStruct{
					numbers: make([]int64, 10000),
					objects: createHeapStructure(1000, 256, 0.5),
					data:    make([]byte, 100000),
				}
			},
			description: "测试混合数据结构的扫描效率",
		},
	}

	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			debug.SetGCPercent(100)
			runtime.GC()

			t.Logf("场景: %s", sc.description)

			// 创建数据结构
			data := sc.setupFn()

			var memStats1 runtime.MemStats
			runtime.ReadMemStats(&memStats1)

			// 执行多次 GC 测量
			iterations := 5
			startTime := time.Now()

			for i := 0; i < iterations; i++ {
				runtime.GC()
			}

			elapsed := time.Since(startTime)

			var memStats2 runtime.MemStats
			runtime.ReadMemStats(&memStats2)

			avgGCTime := elapsed.Milliseconds() / int64(iterations)
			totalPause := (memStats2.PauseTotalNs - memStats1.PauseTotalNs) / 1e6
			avgPause := totalPause / uint64(iterations)

			t.Logf("平均 GC 时间: %d ms", avgGCTime)
			t.Logf("平均 STW 暂停: %d ms", avgPause)
			t.Logf("堆大小: %.2f MB", float64(memStats2.HeapAlloc)/(1024*1024))
			t.Logf("并发标记效率: %.1f%%", (1-float64(avgPause)/float64(avgGCTime))*100)

			_ = data
		})
	}
}
