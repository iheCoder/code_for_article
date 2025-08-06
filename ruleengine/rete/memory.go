package rete

import (
	"sync"

	"code_for_article/ruleengine/model"
)

// AlphaMemory 存储通过 AlphaNode 条件过滤后的单一事实集合。
// 使用 map 确保按 Key 唯一。

type AlphaMemory struct {
	mu   sync.RWMutex
	data map[string]model.Fact
}

func NewAlphaMemory() *AlphaMemory {
	return &AlphaMemory{data: make(map[string]model.Fact)}
}

// Assert 插入新 fact，若已存在则返回 false（未变更）。
func (m *AlphaMemory) Assert(f model.Fact) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.data[f.Key()]; ok {
		return false
	}
	m.data[f.Key()] = f
	return true
}

// Retract 删除 fact，返回是否确实删除。
func (m *AlphaMemory) Retract(f model.Fact) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.data[f.Key()]; ok {
		delete(m.data, f.Key())
		return true
	}
	return false
}

// Snapshot 返回当前 memory 的只读副本。
func (m *AlphaMemory) Snapshot() []model.Fact {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]model.Fact, 0, len(m.data))
	for _, v := range m.data {
		out = append(out, v)
	}
	return out
}

// Size 返回当前 memory 中事实数量。
func (m *AlphaMemory) Size() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.data)
}

// -------------------------------------------------------------------------

// BetaMemory 存储 Join 结果（Token）。

type BetaMemory struct {
	mu   sync.RWMutex
	data map[string]Token // key = token.Hash()
}

func NewBetaMemory() *BetaMemory {
	return &BetaMemory{data: make(map[string]Token)}
}

// Assert 插入新 token；若已存在返回 false。
func (m *BetaMemory) Assert(t Token) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.data[t.Hash()]; ok {
		return false
	}
	m.data[t.Hash()] = t
	return true
}

// Retract 删除 token；返回是否确实删除。
func (m *BetaMemory) Retract(t Token) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.data[t.Hash()]; ok {
		delete(m.data, t.Hash())
		return true
	}
	return false
}

func (m *BetaMemory) Snapshot() []Token {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Token, 0, len(m.data))
	for _, v := range m.data {
		out = append(out, v)
	}
	return out
}

func (m *BetaMemory) Size() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.data)
}
