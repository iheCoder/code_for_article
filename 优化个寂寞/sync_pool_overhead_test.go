package perf_optimize_but_lower

import (
	"sync"
	"testing"
)

// Point is a simple struct we might consider pooling.
type Point struct {
	X, Y int
	// data [16]byte // Optionally add some payload to see if it changes dynamics
}

var globalSinkPoint Point // To prevent optimizations from removing the work

// Work done with the point, kept minimal
func processPoint(p *Point) {
	globalSinkPoint.X += p.X
	globalSinkPoint.Y += p.Y
}

const numPointsToProcess = 10000

// --- Benchmark: Direct Allocation ---
func BenchmarkDirectAllocationPoints(b *testing.B) {
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		for i := 0; i < numPointsToProcess; i++ {
			p := &Point{X: i, Y: i + 1}
			processPoint(p)
		}
	}
}

// --- Benchmark: Using sync.Pool ---
var pointPool = sync.Pool{
	New: func() interface{} {
		// The b.ReportAllocs() in the benchmark won't see this New allocation
		// directly for each Get, but it contributes to overall memory if not reused.
		return &Point{}
	},
}

func BenchmarkSyncPoolPoints(b *testing.B) {
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		for i := 0; i < numPointsToProcess; i++ {
			p := pointPool.Get().(*Point)
			p.X = i
			p.Y = i + 1
			// No complex reset needed for this simple Point struct if fields are always overwritten.
			// If Point had more complex state or slices/maps, a Reset method would be crucial.
			processPoint(p)
			pointPool.Put(p)
		}
	}
}

// --- Benchmark: Using sync.Pool with a slightly larger object to see if it changes ---
type LargePoint struct {
	X, Y int
	Data [128]byte // Larger payload
}

var globalSinkLargePoint LargePoint
var largePointPool = sync.Pool{
	New: func() interface{} {
		return &LargePoint{}
	},
}

func processLargePoint(p *LargePoint) {
	globalSinkLargePoint.X += p.X
	globalSinkLargePoint.Y += p.Y
}

func BenchmarkDirectAllocationLargePoints(b *testing.B) {
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		for i := 0; i < numPointsToProcess; i++ {
			p := &LargePoint{X: i, Y: i + 1}
			processLargePoint(p)
		}
	}
}

func BenchmarkSyncPoolLargePoints(b *testing.B) {
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		for i := 0; i < numPointsToProcess; i++ {
			p := largePointPool.Get().(*LargePoint)
			p.X = i
			p.Y = i + 1
			processLargePoint(p)
			largePointPool.Put(p)
		}
	}
}
