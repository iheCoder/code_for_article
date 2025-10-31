package green_tea

import (
	"os"
	"runtime"
	"runtime/debug"
	"testing"
)

// GCMode GC 模式
type GCMode string

const (
	TraditionalGC GCMode = "Traditional"
	GreenTeaGC    GCMode = "GreenTea"
)

// GetCurrentGCMode 获取当前 GC 模式
func GetCurrentGCMode() GCMode {
	// 检查 GOEXPERIMENT 环境变量
	goexp := os.Getenv("GOEXPERIMENT")
	if goexp == "greentea" || goexp == "GreenTea" || goexp == "greenteagc" || goexp == "GreenTeaGC" {
		return GreenTeaGC
	}

	// 检查 Go 版本字符串中是否包含 greenteagc
	version := runtime.Version()
	if len(version) > 0 && (contains(version, "greenteagc") || contains(version, "GreenTeaGC")) {
		return GreenTeaGC
	}

	// 检查其他可能的环境变量
	if os.Getenv("GOGC_GREENTEA") == "1" {
		return GreenTeaGC
	}

	return TraditionalGC
}

// contains 检查字符串是否包含子串
func contains(s, substr string) bool {
	return len(s) >= len(substr) && indexOfSubstr(s, substr) >= 0
}

// indexOfSubstr 查找子串位置
func indexOfSubstr(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// TestGCModeDetection 测试 GC 模式检测
func TestGCModeDetection(t *testing.T) {
	mode := GetCurrentGCMode()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	var gcStats debug.GCStats
	debug.ReadGCStats(&gcStats)

	t.Logf("====================================")
	t.Logf("当前 GC 模式: %s", mode)
	t.Logf("====================================")
	t.Logf("Go 版本: %s", runtime.Version())
	t.Logf("GOEXPERIMENT: %s", os.Getenv("GOEXPERIMENT"))
	t.Logf("GOGC: %s", os.Getenv("GOGC"))
	t.Logf("NumCPU: %d", runtime.NumCPU())
	t.Logf("GOMAXPROCS: %d", runtime.GOMAXPROCS(0))
	t.Logf("====================================")

	if mode == TraditionalGC {
		t.Logf("⚠️  警告: 当前使用传统 GC")
		t.Logf("要启用 Green Tea GC，请使用以下命令之一：")
		t.Logf("  GOEXPERIMENT=greentea go test ...")
		t.Logf("  或在代码中通过 runtime API 启用（如果支持）")
	} else {
		t.Logf("✓ Green Tea GC 已启用")
	}
}

// BenchmarkGCModeComparison 对比不同 GC 模式
func BenchmarkGCModeComparison(b *testing.B) {
	mode := GetCurrentGCMode()
	b.Logf("当前 GC 模式: %s", mode)

	debug.SetGCPercent(100)
	runtime.GC()

	var memStats1, memStats2 runtime.MemStats
	runtime.ReadMemStats(&memStats1)

	b.ResetTimer()

	allocations := make([][]byte, 0, b.N)
	for i := 0; i < b.N; i++ {
		data := make([]byte, 1024)
		data[0] = byte(i)
		allocations = append(allocations, data)

		if i%1000 == 0 && i > 0 {
			allocations = allocations[:0]
		}
	}

	b.StopTimer()

	runtime.ReadMemStats(&memStats2)

	gcCPU := memStats2.GCCPUFraction * 100
	numGC := memStats2.NumGC - memStats1.NumGC
	pauseTotal := float64(memStats2.PauseTotalNs-memStats1.PauseTotalNs) / 1e6

	b.ReportMetric(gcCPU, "gc_cpu_%")
	b.ReportMetric(float64(numGC), "gc_count")
	b.ReportMetric(pauseTotal, "pause_ms")

	_ = allocations
}
