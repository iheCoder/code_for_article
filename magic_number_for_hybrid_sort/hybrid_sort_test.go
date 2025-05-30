package magic_number_for_hybrid_sort

import (
	"math/rand"
	"sort"
	"testing"
	"time"
)

// InsertionSort 对 data[left...right] 进行插入排序
func InsertionSort(data []int, left, right int) {
	for i := left + 1; i <= right; i++ {
		key := data[i]
		j := i - 1
		for j >= left && data[j] > key {
			data[j+1] = data[j]
			j--
		}
		data[j+1] = key
	}
}

// partition 是快速排序的核心分区函数
// 这里使用 Hoare 分区方案的一个变体
func partition(data []int, left, right int) int {
	pivot := data[left+(right-left)/2] // 选择中间元素作为基准
	i := left - 1
	j := right + 1
	for {
		for {
			i++
			if data[i] >= pivot {
				break
			}
		}
		for {
			j--
			if data[j] <= pivot {
				break
			}
		}
		if i >= j {
			return j
		}
		data[i], data[j] = data[j], data[i]
	}
}

// quickSortRecursive 是快速排序的递归部分
func quickSortRecursive(data []int, left, right int) {
	if left < right {
		pivotIndex := partition(data, left, right)
		quickSortRecursive(data, left, pivotIndex)    // 注意：Hoare 分区产生的 pivotIndex 可能需要调整递归边界
		quickSortRecursive(data, pivotIndex+1, right) // 对于 Hoare，通常是 left, p 和 p+1, right
	}
}

// QuickSort 对整个切片进行快速排序
func QuickSort(data []int) {
	if len(data) < 2 {
		return
	}
	quickSortRecursive(data, 0, len(data)-1)
}

// HybridSortRecursive 是混合排序的递归实现
func HybridSortRecursive(data []int, left, right, threshold int) {
	if right-left+1 < threshold {
		InsertionSort(data, left, right)
		return
	}
	if left < right { // 添加这个检查以避免无限递归或越界
		pivotIndex := partition(data, left, right)
		HybridSortRecursive(data, left, pivotIndex, threshold)
		HybridSortRecursive(data, pivotIndex+1, right, threshold)
	}
}

// HybridSort 是混合排序的入口函数
func HybridSort(data []int, threshold int) {
	if len(data) == 0 {
		return
	}
	HybridSortRecursive(data, 0, len(data)-1, threshold)
}

// --- 基准测试相关 ---

// generateRandomData 生成指定大小的随机整数切片
func generateRandomData(size int) []int {
	data := make([]int, size)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < size; i++ {
		data[i] = r.Intn(size * 10) // 值的范围可以调整
	}
	return data
}

// generateNearlySortedData 生成大部分有序的切片
func generateNearlySortedData(size int, swaps int) []int {
	data := make([]int, size)
	for i := 0; i < size; i++ {
		data[i] = i
	}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < swaps; i++ {
		idx1 := r.Intn(size)
		idx2 := r.Intn(size)
		data[idx1], data[idx2] = data[idx2], data[idx1]
	}
	return data
}

// 基准测试函数模板
func benchmarkHybridSort(b *testing.B, threshold int, dataSize int, dataType string) {
	var data []int
	for n := 0; n < b.N; n++ {
		b.StopTimer() // 停止计时器，准备数据
		switch dataType {
		case "random":
			data = generateRandomData(dataSize)
		case "nearlySorted":
			data = generateNearlySortedData(dataSize, dataSize/20) // 5% 元素乱序
		default:
			data = generateRandomData(dataSize)
		}
		b.StartTimer() // 重新启动计时器
		HybridSort(data, threshold)
	}
}

// 针对不同阈值和数据类型的具体基准测试

// --- 随机数据 ---
func BenchmarkHybridSort_Rand_T4_N1000(b *testing.B)   { benchmarkHybridSort(b, 4, 1000, "random") }
func BenchmarkHybridSort_Rand_T8_N1000(b *testing.B)   { benchmarkHybridSort(b, 8, 1000, "random") }
func BenchmarkHybridSort_Rand_T16_N1000(b *testing.B)  { benchmarkHybridSort(b, 16, 1000, "random") }
func BenchmarkHybridSort_Rand_T24_N1000(b *testing.B)  { benchmarkHybridSort(b, 24, 1000, "random") }
func BenchmarkHybridSort_Rand_T32_N1000(b *testing.B)  { benchmarkHybridSort(b, 32, 1000, "random") }
func BenchmarkHybridSort_Rand_T48_N1000(b *testing.B)  { benchmarkHybridSort(b, 48, 1000, "random") }
func BenchmarkHybridSort_Rand_T64_N1000(b *testing.B)  { benchmarkHybridSort(b, 64, 1000, "random") }
func BenchmarkHybridSort_Rand_T128_N1000(b *testing.B) { benchmarkHybridSort(b, 128, 1000, "random") }
func BenchmarkHybridSort_Rand_T256_N1000(b *testing.B) { benchmarkHybridSort(b, 256, 1000, "random") }
func BenchmarkHybridSort_Rand_T96_N1000(b *testing.B)  { benchmarkHybridSort(b, 96, 1000, "random") }
func BenchmarkHybridSort_Rand_T192_N1000(b *testing.B) { benchmarkHybridSort(b, 192, 1000, "random") }

// --- 近乎有序数据 ---
func BenchmarkHybridSort_NearlySorted_T4_N1000(b *testing.B) {
	benchmarkHybridSort(b, 4, 1000, "nearlySorted")
}
func BenchmarkHybridSort_NearlySorted_T8_N1000(b *testing.B) {
	benchmarkHybridSort(b, 8, 1000, "nearlySorted")
}
func BenchmarkHybridSort_NearlySorted_T16_N1000(b *testing.B) {
	benchmarkHybridSort(b, 16, 1000, "nearlySorted")
}
func BenchmarkHybridSort_NearlySorted_T32_N1000(b *testing.B) {
	benchmarkHybridSort(b, 32, 1000, "nearlySorted")
}
func BenchmarkHybridSort_NearlySorted_T64_N1000(b *testing.B) {
	benchmarkHybridSort(b, 64, 1000, "nearlySorted")
}

// --- 近乎有序数据 (更大阈值) ---
func BenchmarkHybridSort_NearlySorted_T96_N1000(b *testing.B) {
	benchmarkHybridSort(b, 96, 1000, "nearlySorted")
}
func BenchmarkHybridSort_NearlySorted_T128_N1000(b *testing.B) {
	benchmarkHybridSort(b, 128, 1000, "nearlySorted")
}
func BenchmarkHybridSort_NearlySorted_T192_N1000(b *testing.B) {
	benchmarkHybridSort(b, 192, 1000, "nearlySorted")
}
func BenchmarkHybridSort_NearlySorted_T256_N1000(b *testing.B) {
	benchmarkHybridSort(b, 256, 1000, "nearlySorted")
}

// 为了比较，我们也可以加入纯粹的快速排序和插入排序的基准测试（针对小数组）
func BenchmarkQuickSort_N32(b *testing.B) {
	data := generateRandomData(32)
	for n := 0; n < b.N; n++ {
		b.StopTimer()
		copyData := make([]int, len(data))
		copy(copyData, data)
		b.StartTimer()
		QuickSort(copyData)
	}
}

func BenchmarkInsertionSort_N32(b *testing.B) {
	data := generateRandomData(32)
	for n := 0; n < b.N; n++ {
		b.StopTimer()
		copyData := make([]int, len(data))
		copy(copyData, data)
		b.StartTimer()
		InsertionSort(copyData, 0, len(copyData)-1)
	}
}

// 还可以添加更多不同大小 (N) 和不同数据分布的基准测试
// 例如： BenchmarkHybridSort_Rand_T16_N10000, BenchmarkHybridSort_Reversed_T16_N1000 等

// 为了确保排序正确性，可以添加一个辅助的测试函数（非基准测试）
func TestHybridSort_Correctness(t *testing.T) {
	thresholds := []int{4, 8, 16, 32, 64}
	sizes := []int{0, 1, 10, 50, 100, 500}
	for _, threshold := range thresholds {
		for _, size := range sizes {
			data := generateRandomData(size)
			expected := make([]int, size)
			copy(expected, data)
			sort.Ints(expected) // 使用标准库排序作为参照

			HybridSort(data, threshold)

			if !equal(data, expected) {
				t.Errorf("HybridSort (threshold %d, size %d) failed. Expected %v, got %v", threshold, size, expected, data)
			}
		}
	}
}

func TestInsertionSort_Correctness(t *testing.T) {
	sizes := []int{0, 1, 10, 50}
	for _, size := range sizes {
		data := generateRandomData(size)
		expected := make([]int, size)
		copy(expected, data)
		sort.Ints(expected)

		InsertionSort(data, 0, len(data)-1)
		if !equal(data, expected) {
			t.Errorf("InsertionSort (size %d) failed. Expected %v, got %v", size, expected, data)
		}
	}
}

func TestQuickSort_Correctness(t *testing.T) {
	sizes := []int{0, 1, 10, 50, 100, 500}
	for _, size := range sizes {
		data := generateRandomData(size)
		expected := make([]int, size)
		copy(expected, data)
		sort.Ints(expected)

		QuickSort(data)
		if !equal(data, expected) {
			t.Errorf("QuickSort (size %d) failed. Expected %v, got %v", size, expected, data)
		}
	}
}

func equal(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
