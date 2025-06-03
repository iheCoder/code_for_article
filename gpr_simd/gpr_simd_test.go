package gpr_simd

import (
	"fmt"
	"math/rand"
	"testing"
)

// 版本 A：常规 GPR 点积
func dotProductGPR(a, b []float64) float64 {
	sum := 0.0
	for i := 0; i < len(a); i++ {
		sum += a[i] * b[i]
	}
	return sum
}

// 版本 B：Go + 编译器 auto-vectorization
func dotProductSIMD(a, b []float64) float64 {
	var sum float64
	n := len(a)
	for i := 0; i < n-4; i += 4 {
		sum += a[i]*b[i] + a[i+1]*b[i+1] + a[i+2]*b[i+2] + a[i+3]*b[i+3]
	}
	for i := n - n%4; i < n; i++ {
		sum += a[i] * b[i]
	}
	return sum
}

// 版本 C：混用 GPR + SIMD 临时变量，模拟寄存器压力
func dotProductGPR_SIMD_Mix(a, b []float64, scalar float64) float64 {
	var sum float64
	n := len(a)
	temp := make([]float64, 4)

	for i := 0; i < n-4; i += 4 {
		temp[0] = a[i] * b[i]
		temp[1] = a[i+1] * b[i+1]
		temp[2] = a[i+2] * b[i+2]
		temp[3] = a[i+3] * b[i+3]

		for j := 0; j < 4; j++ {
			sum += temp[j] * scalar // scalar 是通过 GPR 储存传入的
		}
	}
	return sum
}

// Benchmark additions:

// Global variable to store benchmark results to prevent compiler optimizations.
var globalResult float64

// Helper to generate float64 slices for benchmark data.
func generateFloat64Slice(size int) []float64 {
	s := make([]float64, size)
	for i := 0; i < size; i++ {
		s[i] = rand.Float64() // Values don't strictly matter, just need data
	}
	return s
}

// benchmarkTestSizes defines various input slice sizes to test for finding the performance tipping point.
var benchmarkTestSizes = []int{
	1, 2, 3, 4, 5, 6, 7, 8, // Small sizes, including those less than SIMD unroll factor
	12, 16, 24, 32, 48, 64, // Medium sizes
	96, 128, 192, 256, 384, 512, // Larger sizes
	768, 1024, 1536, 2048, 3072, 4096, // Even larger
	8192, // A fairly large size
}

// init seeds the random number generator to ensure reproducible benchmark data.
func init() {
	rand.Seed(42) // Using a fixed seed
}

// BenchmarkDotProductGPR_BySize benchmarks the dotProductGPR function across various input sizes.
func BenchmarkDotProductGPR_BySize(b *testing.B) {
	var r float64 // Local variable to store result within this benchmark suite
	for _, size := range benchmarkTestSizes {
		// Slices are generated once per size to ensure fairness for b.N iterations
		aVec := generateFloat64Slice(size)
		bVec := generateFloat64Slice(size)

		b.Run(fmt.Sprintf("GPR_size_%d", size), func(b *testing.B) {
			b.ReportAllocs() // Report memory allocations
			b.ResetTimer()   // Reset timer after setup for this sub-benchmark
			for i := 0; i < b.N; i++ {
				// Assign to r to ensure the call isn't optimized away by the compiler
				r = dotProductGPR(aVec, bVec)
			}
		})
	}
	globalResult = r // Assign to globalResult to ensure r (and thus benchmarked function calls) is used
}

// BenchmarkDotProductSIMD_BySize benchmarks the dotProductSIMD function across various input sizes.
func BenchmarkDotProductSIMD_BySize(b *testing.B) {
	var r float64 // Local variable to store result
	for _, size := range benchmarkTestSizes {
		aVec := generateFloat64Slice(size)
		bVec := generateFloat64Slice(size)

		b.Run(fmt.Sprintf("SIMD_size_%d", size), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				r = dotProductSIMD(aVec, bVec)
			}
		})
	}
	globalResult = r // Assign to globalResult
}
