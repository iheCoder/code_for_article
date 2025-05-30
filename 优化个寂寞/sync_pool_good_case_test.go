package perf_optimize_but_lower

import (
	"sync"
	"testing"
)

const expensiveObjectSize = 4096 // 4KB
const numExpensiveObjectsToProcess = 5000

type ExpensiveObject struct {
	Data []byte
	// ExampleField int
}

// globalSinkForExpensiveObject is used to ensure the compiler doesn't optimize away
// the object allocations and processing.
var globalSinkForExpensiveObject *ExpensiveObject
var globalCounterForExpensiveObjectTest int // Used to simulate work

func processExpensiveObject(obj *ExpensiveObject) {
	if len(obj.Data) > 0 {
		obj.Data[0] = byte(globalCounterForExpensiveObjectTest % 255) // Simulate some modification
		globalCounterForExpensiveObjectTest++
	}
	globalSinkForExpensiveObject = obj // Make sure obj is "used"
}

// --- Benchmark: Direct Allocation of Expensive Objects ---
func BenchmarkDirectAllocationExpensiveObjects(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer() // Ensure setup like initial allocations for b.N isn't counted
	for n := 0; n < b.N; n++ {
		for i := 0; i < numExpensiveObjectsToProcess; i++ {
			obj := &ExpensiveObject{
				Data: make([]byte, expensiveObjectSize),
			}
			// obj.ExampleField = i // Simulate more initialization
			processExpensiveObject(obj)
		}
	}
}

// --- Benchmark: Using sync.Pool for Expensive Objects ---
var expensiveObjectPool = sync.Pool{
	New: func() interface{} {
		// This allocation happens when Get() finds no reusable objects.
		return &ExpensiveObject{
			Data: make([]byte, expensiveObjectSize),
		}
	},
}

func BenchmarkSyncPoolExpensiveObjects(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		for i := 0; i < numExpensiveObjectsToProcess; i++ {
			obj := expensiveObjectPool.Get().(*ExpensiveObject)
			// obj.ExampleField = i // Simulate re-initialization

			// Crucial: Ensure the object is in a clean state.
			// For this simple struct, overwriting fields might be enough.
			// If Data could be resliced smaller by processExpensiveObject,
			// we might need to ensure its length/capacity here.
			// e.g., obj.Data = obj.Data[:expensiveObjectSize]
			// A Reset method is best practice for complex objects:
			// obj.Reset()

			processExpensiveObject(obj)
			expensiveObjectPool.Put(obj)
		}
	}
}
