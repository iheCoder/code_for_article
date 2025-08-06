package rete

import "code_for_article/ruleengine/model"

// ExistsNode 实现 "exists <pattern>" 语义。
// 只有当左侧输入的 Token 在右侧输入流中至少能找到一个匹配的 Fact 时，该 Token 才会被传播。
//
// 工作流程:
// 与 NotNode 类似，ExistsNode 也需要一个计数器来精确跟踪每个左侧 Token 的匹配数量，
// 以确保只在匹配状态发生关键转变时才传播信号（断言或撤回）。
//
// - 状态转变点:
//   - **Assert**: 当一个 Token 的匹配数从 0 增加到 1 时，传播断言。
//   - **Retract**: 当一个 Token 的匹配数从 1 减少到 0 时，传播撤回。
type ExistsNode struct {
	baseNode
	join        JoinFunc
	leftMemory  *BetaMemory
	rightMemory *AlphaMemory
	counter     map[string]int // token.hash -> match count
}

func NewExistsNode(j JoinFunc) *ExistsNode {
	return &ExistsNode{
		join:        j,
		leftMemory:  NewBetaMemory(),
		rightMemory: NewAlphaMemory(),
		counter:     make(map[string]int),
	}
}

func (e *ExistsNode) AssertToken(t Token) {
	if !e.leftMemory.Assert(t) {
		return
	}
	count := 0
	for _, f := range e.rightMemory.Snapshot() {
		if e.join(t, f) {
			count++
		}
	}
	e.counter[t.Hash()] = count

	if count > 0 {
		e.propagateAssertToken(t)
	}
}

func (e *ExistsNode) RetractToken(t Token) {
	if !e.leftMemory.Retract(t) {
		return
	}
	if count, ok := e.counter[t.Hash()]; ok && count > 0 {
		e.propagateRetractToken(t)
	}
	delete(e.counter, t.Hash())
}

func (e *ExistsNode) AssertFact(f model.Fact) {
	if !e.rightMemory.Assert(f) {
		return
	}
	for _, t := range e.leftMemory.Snapshot() {
		if e.join(t, f) {
			// 匹配数从 0 -> 1，触发断言
			if e.counter[t.Hash()] == 0 {
				e.propagateAssertToken(t)
			}
			e.counter[t.Hash()]++
		}
	}
}

func (e *ExistsNode) RetractFact(f model.Fact) {
	if !e.rightMemory.Retract(f) {
		return
	}
	for _, t := range e.leftMemory.Snapshot() {
		if e.join(t, f) {
			e.counter[t.Hash()]--
			// 匹配数从 1 -> 0，触发撤回
			if e.counter[t.Hash()] == 0 {
				e.propagateRetractToken(t)
			}
		}
	}
}
