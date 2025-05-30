package perf_optimize_but_lower

import (
	"math"
	"math/rand"
	"testing"
	"time"
)

// 方式 A：查找表 LUT（sqrt(x) 预先算好）
type SqrtLUT struct {
	table []float64
}

func NewSqrtLUT(n int) *SqrtLUT {
	t := make([]float64, n)
	for i := 0; i < n; i++ {
		t[i] = math.Sqrt(float64(i))
	}
	return &SqrtLUT{table: t}
}

func (lut *SqrtLUT) Sqrt(x int) float64 {
	if x < 0 || x >= len(lut.table) {
		// Handle out-of-bounds access, though for benchmark purposes,
		// we'll ensure x is within bounds.
		return math.NaN()
	}
	return lut.table[x]
}

// 方式 B：直接计算 sqrt(x)
func DirectSqrt(x int) float64 {
	return math.Sqrt(float64(x))
}

const (
	smallLutSize = 1024      // For L1/L2 cache hit scenario
	largeLutSize = 100000000 // For L3 cache miss scenario (approx 800MB for float64)
	numLookups   = 1000      // Number of lookups per benchmark iteration
)

var (
	smallLut *SqrtLUT
	largeLut *SqrtLUT
	rng      *rand.Rand
)

func init() {
	smallLut = NewSqrtLUT(smallLutSize)
	// Initialize largeLut lazily in the benchmark to avoid long init times for all tests
	// largeLut = NewSqrtLUT(largeLutSize)

	source := rand.NewSource(time.Now().UnixNano())
	rng = rand.New(source)
}

// Scenario 1: Method A - L1/L2 Cache Hit
func BenchmarkSqrtLUT_CacheHit(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for j := 0; j < numLookups; j++ {
			_ = smallLut.Sqrt(rng.Intn(smallLutSize))
		}
	}
}

// Scenario 2: Method A - L3 Cache Miss
func BenchmarkSqrtLUT_CacheMiss(b *testing.B) {
	if largeLut == nil { // Lazy initialization
		largeLut = NewSqrtLUT(largeLutSize)
	}
	b.ResetTimer() // Reset timer after large LUT initialization
	for i := 0; i < b.N; i++ {
		for j := 0; j < numLookups; j++ {
			_ = largeLut.Sqrt(rng.Intn(largeLutSize))
		}
	}
}

// Scenario 3: Method B - Direct Calculation
func BenchmarkDirectSqrt(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for j := 0; j < numLookups; j++ {
			_ = DirectSqrt(rng.Intn(smallLutSize)) // Use smallLutSize for comparable input range
		}
	}
}
