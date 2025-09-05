package brc1

import (
	"math/bits"
	"math/rand"
	"sync/atomic"
	"testing"
)

// -----------------------------
// Test data generator (shared)
// -----------------------------

// makeStations returns N pseudo station names (mostly <= 12 bytes)
func makeStations(n int) [][]byte {
	out := make([][]byte, n)
	for i := 0; i < n; i++ {
		// lengths 4..12
		// e.g. s1234, st12345, sx123456, cycling a tiny alphabet so bytes vary
		switch i % 3 {
		case 0:
			out[i] = []byte("s" + itoa3(i%1000)) // 4 bytes like s042
		case 1:
			out[i] = []byte("st" + itoa4(i%10000)) // 6 bytes like st0042
		default:
			out[i] = []byte("sx" + itoa5(i%100000)) // up to 7 bytes
		}
	}
	return out
}

func itoa3(x int) string {
	return string([]byte{'0' + byte(x/100%10), '0' + byte(x/10%10), '0' + byte(x%10)})
}
func itoa4(x int) string {
	return string([]byte{'0' + byte(x/1000%10), '0' + byte(x/100%10), '0' + byte(x/10%10), '0' + byte(x%10)})
}
func itoa5(x int) string {
	return string([]byte{'0' + byte(x/10000%10), '0' + byte(x/1000%10), '0' + byte(x/100%10), '0' + byte(x/10%10), '0' + byte(x%10)})
}

// makeVals returns a deterministic sequence of temperatures as int16 (ten-times integer)
func makeVals(n int) []int16 {
	vals := make([]int16, n)
	r := rand.New(rand.NewSource(42))
	for i := 0; i < n; i++ {
		// range roughly [-50.0, 50.0] in tenths
		v := int16(r.Intn(1000) - 500)
		vals[i] = v
	}
	return vals
}

// -----------------------------
// Baseline: Go map[string]Agg
// -----------------------------

func Benchmark_GoMap(b *testing.B) {
	stations := makeStations(413) // mimic 1BRC station-count
	vals := makeVals(b.N)

	m := make(map[string]Agg, 1024)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		name := string(stations[i%len(stations)]) // new string each op (typical pipeline)
		v := vals[i]
		a := m[name]
		if a.Cnt == 0 {
			a.Min, a.Max = v, v
		} else {
			if v < a.Min {
				a.Min = v
			}
			if v > a.Max {
				a.Max = v
			}
		}
		a.Sum += int64(v)
		a.Cnt++
		m[name] = a
	}

	// keep the map alive (avoid elimination)
	if len(m) == 0 {
		b.Fatalf("unexpected empty map")
	}
}

// ---------------------------------------
// Custom open-addressing table (no GC thrash)
// ---------------------------------------

type entry struct {
	hash     uint64
	keyStart uint32
	keyLen   uint16
	used     bool
	Min, Max int16
	Sum      int64
	Cnt      int64
}

type table struct {
	keys  []byte  // string pool
	slots []entry // open-addressing slots
	mask  uint32  // len(slots)-1 (power of two)
}

func newTable(expectedKeys int) *table {
	// capacity: keep load factor <= 0.5 for speed
	capSlots := 1
	for capSlots < expectedKeys*2 {
		capSlots <<= 1
	}
	return &table{
		keys:  make([]byte, 0, expectedKeys*12),
		slots: make([]entry, capSlots),
		mask:  uint32(capSlots - 1),
	}
}

func (t *table) putOrUpdate(name []byte, hv uint64, v int16) {
	i := uint32(hv) & t.mask
	for {
		e := &t.slots[i]
		if !e.used {
			// first time: copy name into pool once
			start := uint32(len(t.keys))
			t.keys = append(t.keys, name...)
			*e = entry{hash: hv, keyStart: start, keyLen: uint16(len(name)), used: true,
				Min: v, Max: v, Sum: int64(v), Cnt: 1}
			return
		}
		if e.hash == hv && t.equalAt(e, name) {
			if v < e.Min {
				e.Min = v
			}
			if v > e.Max {
				e.Max = v
			}
			e.Sum += int64(v)
			e.Cnt++
			return
		}
		i = (i + 1) & t.mask // linear probing
	}
}

func (t *table) equalAt(e *entry, name []byte) bool {
	n := int(e.keyLen)
	if n != len(name) {
		return false
	}
	base := int(e.keyStart)
	a := t.keys[base : base+n]
	// 8-byte blocks, then tail
	i := 0
	for i+8 <= n {
		if load64(a[i:]) != load64(name[i:]) {
			return false
		}
		i += 8
	}
	for ; i < n; i++ {
		if a[i] != name[i] {
			return false
		}
	}
	return true
}

// Small helpers
func load64(p []byte) uint64 { // little-endian
	_ = p[7]
	return uint64(p[0]) | uint64(p[1])<<8 | uint64(p[2])<<16 | uint64(p[3])<<24 |
		uint64(p[4])<<32 | uint64(p[5])<<40 | uint64(p[6])<<48 | uint64(p[7])<<56
}

func fnv1a64(b []byte) uint64 {
	var h uint64 = 1469598103934665603
	for _, c := range b {
		h ^= uint64(c)
		h *= 1099511628211
	}
	return h
}

// -----------------------------
// Benchmark: custom table
// -----------------------------

func Benchmark_OpenAddressing(b *testing.B) {
	stations := makeStations(413)
	vals := makeVals(b.N)

	t := newTable(1024)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		name := stations[i%len(stations)] // []byte, no per-op string alloc
		hv := fnv1a64(name)

		// Optional: touch the slot early (simulate a prefetch via a harmless load)
		idx := hv & uint64(t.mask)
		_ = atomic.LoadUint64(&t.slots[idx].hash)

		v := vals[i]
		t.putOrUpdate(name, hv, v)
	}

	if len(t.slots) == 0 {
		b.Fatalf("unexpected table state")
	}
}

// -----------------------------
// Extra: micro-bench find-byte SWAR vs naive (optional curiosity)
// -----------------------------

// findByte8 returns [0..7] or -1 if not found
func findByte8(word uint64, target byte) int {
	pat := uint64(target) * 0x0101010101010101
	x := word ^ pat
	y := (x - 0x0101010101010101) & (^x) & 0x8080808080808080
	if y == 0 {
		return -1
	}
	return bits.TrailingZeros64(y) >> 3
}

func Benchmark_findByte8(b *testing.B) {
	w := uint64(0x2E31322DFFFF0000) // "\x00\x00\xFF-21."
	for i := 0; i < b.N; i++ {
		_ = findByte8(w, '.')
	}
}

// goos: darwin
// goarch: arm64
// pkg: code_for_article/optimize/brc1
// cpu: Apple M3
// Benchmark_GoMap-8               27692600                39.77 ns/op            7 B/op          1 allocs/op
// Benchmark_OpenAddressing-8      82038895                13.01 ns/op            0 B/op          0 allocs/op
// Benchmark_findByte8-8           1000000000               0.3009 ns/op          0 B/op          0 allocs/op
// PASS
// ok      code_for_article/optimize/brc1  4.004s
