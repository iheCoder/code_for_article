package rete

import "code_for_article/ruleengine/model"

// NotNode 实现 "not <pattern>" 语义。
// 只有当左侧输入的 Token 在右侧输入流中找不到任何匹配的 Fact 时，该 Token 才会被传播。
//
// 工作流程:
// 1. 左侧 Token (t) 到达:
//   - 存入 leftMemory。
//   - 检查 rightMemory 中是否有 Fact (f) 满足 join(t, f)。
//   - 如果一个匹配都没有，则立即向下游传播 t。
//
// 2. 右侧 Fact (f) 到达:
//   - 存入 rightMemory。
//   - 检查 leftMemory 中是否有 Token (t) 满足 join(t, f)。
//   - 对于每个新匹配上的 Token，如果它之前是“无匹配”状态（即已被传播过），
//     则需要向下游传播对该 Token 的撤回信号。
//
// 3. 撤回:
//   - 当左侧 Token 被撤回时，如果它之前没有匹配（已被传播），则传播撤回信号。
//   - 当右侧 Fact 被撤回时，检查它之前匹配了哪些左侧 Token。对于这些 Token，
//     如果这是它们最后一个匹配的 Fact，它们现在进入了“无匹配”状态，需要被传播。
//
// 为了精确实现撤回，我们使用一个 counter 来记录每个左侧 Token 的匹配数量。
type NotNode struct {
	baseNode
	join        JoinFunc
	leftMemory  *BetaMemory
	rightMemory *AlphaMemory
	counter     map[string]int // token.hash -> match count
}

func NewNotNode(j JoinFunc) *NotNode {
	return &NotNode{
		join:        j,
		leftMemory:  NewBetaMemory(),
		rightMemory: NewAlphaMemory(),
		counter:     make(map[string]int),
	}
}

func (n *NotNode) AssertToken(t Token) {
	if !n.leftMemory.Assert(t) {
		return
	}

	count := 0
	for _, f := range n.rightMemory.Snapshot() {
		if n.join(t, f) {
			count++
		}
	}
	n.counter[t.Hash()] = count

	if count == 0 {
		n.propagateAssertToken(t)
	}
}

func (n *NotNode) RetractToken(t Token) {
	if !n.leftMemory.Retract(t) {
		return
	}
	// 如果这个 token 之前没有匹配项（即曾被传播过），则传播撤回
	if count, ok := n.counter[t.Hash()]; ok && count == 0 {
		n.propagateRetractToken(t)
	}
	delete(n.counter, t.Hash())
}

func (n *NotNode) AssertFact(f model.Fact) {
	if !n.rightMemory.Assert(f) {
		return
	}
	for _, t := range n.leftMemory.Snapshot() {
		if n.join(t, f) {
			// 匹配数从 0 -> 1，意味着之前传播的 Token 需要被撤回
			if n.counter[t.Hash()] == 0 {
				n.propagateRetractToken(t)
			}
			n.counter[t.Hash()]++
		}
	}
}

func (n *NotNode) RetractFact(f model.Fact) {
	if !n.rightMemory.Retract(f) {
		return
	}
	for _, t := range n.leftMemory.Snapshot() {
		if n.join(t, f) {
			n.counter[t.Hash()]--
			// 匹配数从 1 -> 0，意味着这个 Token 现在没有匹配了，需要被传播
			if n.counter[t.Hash()] == 0 {
				n.propagateAssertToken(t)
			}
		}
	}
}
