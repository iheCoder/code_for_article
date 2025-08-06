package rete

import "code_for_article/ruleengine/model"

// JoinFunc 定义了 BetaNode 用于连接左侧 Token 和右侧 Fact 的函数签名。
type JoinFunc func(t Token, f model.Fact) bool

// BetaNode 是 Rete 网络中的核心连接节点。
// 它接收来自左侧（通常是另一个 Rete 节点）的 Token 和来自右侧（通常是 AlphaNode）的 Fact，
// 并根据指定的 Join 条件将它们组合成新的、更长的 Token。
//
// 工作流程:
// - Assert:
//  1. 当左侧 Token 到达时，存入 leftMemory，并与 rightMemory 中的所有 Fact 进行匹配。
//  2. 当右侧 Fact 到达时，存入 rightMemory，并与 leftMemory 中的所有 Token 进行匹配。
//  3. 每当匹配成功，就创建一个新的组合 Token 并向下游传播。
//
// - Retract:
//  1. 当 Token 或 Fact 被撤回时，从相应内存中移除。
//  2. 同时，找到所有由它参与构成的下游 Token，并对它们发起撤回传播。
type BetaNode struct {
	baseNode
	join        JoinFunc
	leftMemory  *BetaMemory
	rightMemory *AlphaMemory
}

// NewBetaNode 创建一个新的 BetaNode。
func NewBetaNode(j JoinFunc) *BetaNode {
	return &BetaNode{
		join:        j,
		leftMemory:  NewBetaMemory(),
		rightMemory: NewAlphaMemory(),
	}
}

// AssertToken 处理来自左侧的 Token 断言。
func (b *BetaNode) AssertToken(t Token) {
	if !b.leftMemory.Assert(t) {
		return
	}
	// 与右侧所有事实进行 Join
	for _, f := range b.rightMemory.Snapshot() {
		if b.join(t, f) {
			newToken := extendToken(t, f)
			b.propagateAssertToken(newToken)
		}
	}
}

// RetractToken 处理来自左侧的 Token 撤回。
func (b *BetaNode) RetractToken(t Token) {
	if !b.leftMemory.Retract(t) {
		return
	}
	// 撤回所有相关的下游 Token
	for _, f := range b.rightMemory.Snapshot() {
		if b.join(t, f) {
			staleToken := extendToken(t, f)
			b.propagateRetractToken(staleToken)
		}
	}
}

// AssertFact 处理来自右侧的 Fact 断言。
func (b *BetaNode) AssertFact(f model.Fact) {
	if !b.rightMemory.Assert(f) {
		return
	}
	// 与左侧所有 Token 进行 Join
	for _, t := range b.leftMemory.Snapshot() {
		if b.join(t, f) {
			newToken := extendToken(t, f)
			b.propagateAssertToken(newToken)
		}
	}
}

// RetractFact 处理来自右侧的 Fact 撤回。
func (b *BetaNode) RetractFact(f model.Fact) {
	if !b.rightMemory.Retract(f) {
		return
	}
	// 撤回所有相关的下游 Token
	for _, t := range b.leftMemory.Snapshot() {
		if b.join(t, f) {
			staleToken := extendToken(t, f)
			b.propagateRetractToken(staleToken)
		}
	}
}

// extendToken 是一个辅助函数，用于将一个 Fact 追加到一个旧的 Token 上，生成一个新 Token。
func extendToken(t Token, f model.Fact) Token {
	newFacts := make([]model.Fact, 0, len(t.Facts)+1)
	newFacts = append(newFacts, t.Facts...)
	newFacts = append(newFacts, f)
	return NewToken(newFacts)
}
