package brc1

import (
	"math/rand"
	"testing"
)

type Key uint64

type Val struct {
	A uint64
	B uint64
	C uint64
}

// heavy 模拟“值相关的中等偏重计算”：不需要太重，但能把等待时间塞满。
func heavy(v Val, k Key) uint64 {
	// 纯算，不访问 map，缩短依赖链，给 OOO 喂活
	x := v.A ^ uint64(k)
	y := v.B*33 + v.C*7
	// 防止被优化掉
	return (x<<1 | y>>1) ^ (x*y + 0x9e3779b97f4a7c15)
}

// 构造一批数据：
// - m 里 50 万项，keys 的命中率约 60%（可调）
// - 注意：不需要稳定重现，只要分布“够随机”
func makeData(nMap, nKeys int, hitRatio float64) (map[Key]Val, []Key) {
	m := make(map[Key]Val, nMap)
	rnd := rand.New(rand.NewSource(1))
	for i := 0; i < nMap; i++ {
		k := Key(rnd.Uint64())
		m[k] = Val{rnd.Uint64(), rnd.Uint64(), rnd.Uint64()}
	}
	keys := make([]Key, nKeys)
	i := 0
	for i < nKeys {
		if rnd.Float64() < hitRatio {
			// 命中：从 m 里抽一个 key
			for k := range m { // 取第一个即可，省事
				keys[i] = k
				break
			}
		} else {
			// miss：随机新 key
			keys[i] = Key(rnd.Uint64())
		}
		i++
	}
	return m, keys
}

// processV1: 朴素写法，依赖链长，容易被单次 map miss 卡住。
func processV1(m map[Key]Val, keys []Key) uint64 {
	var acc uint64
	for _, k := range keys {
		if v, ok := m[k]; ok {
			acc ^= heavy(v, k)
		}
	}

	return acc
}

// processV2: 小批次两阶段。把慢的随机访问聚在第一趟，把纯算集中在第二趟。
// 这样 miss 的等待期能被另一半批次的 heavy 填满。
func processV2(m map[Key]Val, keys []Key, batch int) uint64 {
	var acc uint64

	// 复用缓存，避免反复分配
	vals := make([]Val, 0, batch)
	masks := make([]uint8, 0, batch) // // 0/1 标记命中（byte 比 bool 更可控）

	for i := 0; i < len(keys); i += batch {
		end := i + batch
		if end > len(keys) {
			end = len(keys)
		}
		b := end - i

		// 阶段 1：只做查表，写入 vals 和 mask。纯 load，易乱序填充。
		for j := 0; j < b; j++ {
			if v, ok := m[keys[i+j]]; ok {
				vals = append(vals, v)
				masks = append(masks, 1)
			} else {
				// 占位，保持 vals 和 keys 对齐
				vals = append(vals, Val{})
				masks = append(masks, 0)
			}
		}

		// 阶段 2：对命中项做纯计算。此时没有 map 依赖，ALU 可持续工作。
		for j := 0; j < b; j++ {
			if masks[j] == 1 {
				acc ^= heavy(vals[j], keys[i+j])
			}
		}
	}

	return acc
}

// processV3: 保留两阶段，但在第二阶段用“掩码选择”替代分支。
// 对某些数据分布（高抖动、难预测）会更稳。
func processV3(m map[Key]Val, keys []Key, batch int) uint64 {
	var acc uint64
	vals := make([]Val, batch)
	mask := make([]uint64, batch) // 用 0/全 1 掩码，便于位选择

	for i := 0; i < len(keys); i += batch {
		end := i + batch
		if end > len(keys) {
			end = len(keys)
		}
		b := end - i

		// 阶段 1：只做查表，写入 vals 和 mask。纯 load，易乱序填充。
		for j := 0; j < b; j++ {
			if v, ok := m[keys[i+j]]; ok {
				vals[j] = v
				mask[j] = ^uint64(0) // 全 1
			} else {
				// 占位，保持 vals 和 keys 对齐
				vals[j] = Val{}
				mask[j] = 0
			}
		}

		// 阶段 2：无分支聚合
		for j := 0; j < b; j++ {
			// 计算总是执行，但结果根据 mask 决定是否采纳
			v := heavy(vals[j], keys[i+j])
			acc ^= v & mask[j]
		}
	}

	return acc
}

const MaxBatch = 256

func processV2Stack(m map[Key]Val, keys []Key, batch int) uint64 {
	if batch > MaxBatch {
		batch = MaxBatch
	}
	var acc uint64
	var vals [MaxBatch]Val
	var mask [MaxBatch]uint64

	for i := 0; i < len(keys); i += batch {
		b := batch
		if remain := len(keys) - i; remain < b {
			b = remain
		}

		// 阶段 1：只查表
		for j := 0; j < b; j++ {
			if v, ok := m[keys[i+j]]; ok {
				vals[j] = v
				mask[j] = ^uint64(0) // 全 1
			} else {
				mask[j] = 0
			}
		}
		// 阶段 2：纯算（可改回有分支版）
		for j := 0; j < b; j++ {
			h := heavy(vals[j], keys[i+j])
			acc ^= h & mask[j]
		}
	}
	return acc
}

type BatchBuf struct {
	vals []Val
	mask []uint64
}

func NewBatchBuf(batch int) *BatchBuf {
	return &BatchBuf{
		vals: make([]Val, batch),
		mask: make([]uint64, batch),
	}
}
func processV2Buf(m map[Key]Val, keys []Key, buf *BatchBuf) uint64 {
	batch := len(buf.vals)
	var acc uint64
	for i := 0; i < len(keys); i += batch {
		b := batch
		if r := len(keys) - i; r < b {
			b = r
		}
		vs, ms := buf.vals[:b], buf.mask[:b]
		for j := 0; j < b; j++ {
			if v, ok := m[keys[i+j]]; ok {
				vs[j], ms[j] = v, ^uint64(0)
			} else {
				ms[j] = 0
			}
		}
		for j := 0; j < b; j++ {
			acc ^= (heavy(vs[j], keys[i+j]) & ms[j])
		}
	}
	return acc
}

func BenchmarkProcessV1(b *testing.B) {
	m, keys := makeData(500_000, 2_000_000, 0.6)
	b.ReportAllocs()
	b.ResetTimer()
	var sink uint64
	for i := 0; i < b.N; i++ {
		sink ^= processV1(m, keys)
	}
	_ = sink
}

func BenchmarkProcessV2(b *testing.B) {
	m, keys := makeData(500_000, 2_000_000, 0.6)
	b.ReportAllocs()
	b.ResetTimer()
	var sink uint64
	for i := 0; i < b.N; i++ {
		sink ^= processV2(m, keys, 32) // 批次先用 64
	}
	_ = sink
}

func BenchmarkProcessV2Stack(b *testing.B) {
	m, keys := makeData(500_000, 2_000_000, 0.6)
	b.ReportAllocs()
	b.ResetTimer()
	var sink uint64
	for i := 0; i < b.N; i++ {
		sink ^= processV2Stack(m, keys, 32) // 批次先用 64
	}
	_ = sink
}

func BenchmarkProcessV2Buf(b *testing.B) {
	m, keys := makeData(500_000, 2_000_000, 0.6)
	buf := NewBatchBuf(128)
	b.ReportAllocs()
	b.ResetTimer()
	var sink uint64
	for i := 0; i < b.N; i++ {
		sink ^= processV2Buf(m, keys, buf) // 批次先用 64
	}
	_ = sink
}

func BenchmarkProcessV3(b *testing.B) {
	m, keys := makeData(500_000, 2_000_000, 0.6)
	b.ReportAllocs()
	b.ResetTimer()
	var sink uint64
	for i := 0; i < b.N; i++ {
		sink ^= processV3(m, keys, 32)
	}
	_ = sink
}
