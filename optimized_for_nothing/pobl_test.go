package optimized_for_nothing

import (
	"math/rand"
	"testing"
	"time"
)

func addIfEven(arr []int) []int {
	out := make([]int, len(arr))
	for i, v := range arr {
		if v%2 == 0 {
			out[i] = v + 1
		} else {
			out[i] = v
		}
	}
	return out
}

func addIfEvenBranchless(arr []int) []int {
	out := make([]int, len(arr))
	for i, v := range arr {
		isEven := int(^v & 1) // 1 if even, 0 if odd
		out[i] = v + isEven
	}
	return out
}

const size = 10000

func generateMostlyEvenSlice(n int) []int {
	s := make([]int, n)
	for i := 0; i < n; i++ {
		if rand.Float32() < 0.9 { // 80% chance of being even
			s[i] = rand.Intn(n/2) * 2
		} else {
			s[i] = rand.Intn(n/2)*2 + 1
		}
	}
	return s
}

func generateAlternatingSlice(n int) []int {
	s := make([]int, n)
	for i := 0; i < n; i++ {
		if i%2 == 0 {
			s[i] = i // Even
		} else {
			s[i] = i // Odd
		}
	}
	return s
}

func generateRandomSlice(n int) []int {
	s := make([]int, n)
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < n; i++ {
		s[i] = rand.Intn(n)
	}
	return s
}

var (
	mostlyEvenSlice  = generateMostlyEvenSlice(size)
	alternatingSlice = generateAlternatingSlice(size)
	randomSlice      = generateRandomSlice(size)
)

func BenchmarkAddIfEven_MostlyEven(b *testing.B) {
	for i := 0; i < b.N; i++ {
		addIfEven(mostlyEvenSlice)
	}
}

func BenchmarkAddIfEvenBranchless_MostlyEven(b *testing.B) {
	for i := 0; i < b.N; i++ {
		addIfEvenBranchless(mostlyEvenSlice)
	}
}

func BenchmarkAddIfEven_Alternating(b *testing.B) {
	for i := 0; i < b.N; i++ {
		addIfEven(alternatingSlice)
	}
}

func BenchmarkAddIfEvenBranchless_Alternating(b *testing.B) {
	for i := 0; i < b.N; i++ {
		addIfEvenBranchless(alternatingSlice)
	}
}

func BenchmarkAddIfEven_Random(b *testing.B) {
	for i := 0; i < b.N; i++ {
		addIfEven(randomSlice)
	}
}

func BenchmarkAddIfEvenBranchless_Random(b *testing.B) {
	for i := 0; i < b.N; i++ {
		addIfEvenBranchless(randomSlice)
	}
}
