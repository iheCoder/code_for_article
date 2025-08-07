package rete

import "code_for_article/ruleengine/model"

// AlphaFunc 定义了 AlphaNode 用于过滤事实的函数签名。
// 它接收一个 Fact，如果该 Fact 满足条件，则返回 true。
type AlphaFunc func(f model.Fact) bool

// AlphaNode 是 Rete 网络的第一层，负责对单个事实进行条件过滤。
// 它持有一个 AlphaMemory 来存储所有满足其条件的事实，确保唯一性。
//
// 工作流程:
//  1. AssertFact: 当一个新事实进入时，如果它满足 AlphaNode 的条件且尚未存在于内存中，
//     则将其存入 AlphaMemory，并同时向下游传播该事实及其对应的单元素 Token。
//  2. RetractFact: 当一个事实被撤回时，如果它满足条件且存在于内存中，
//     则从 AlphaMemory 中移除，并向下游传播撤回信号。
type AlphaNode struct {
	baseNode
	cond   AlphaFunc
	memory *AlphaMemory
}

// NewAlphaNode 创建一个新的 AlphaNode。
func NewAlphaNode(f AlphaFunc) *AlphaNode {
	return &AlphaNode{cond: f, memory: NewAlphaMemory()}
}

// AssertFact 检查事实是否满足条件，如果满足，则存入内存并向下传播。
func (a *AlphaNode) AssertFact(f model.Fact) {
	if !a.cond(f) {
		return
	}

	// 检查内存中是否已存在该事实
	// 如果不存在，则插入新事实并传播
	if a.memory.Add(f) {
		// 新事实满足条件，向下游传播
		// - 传播事实本身，供其他 AlphaNode 或 BetaNode 右输入使用。
		// - 传播单元素 Token，供 BetaNode 或逻辑节点的左输入使用。
		token := NewToken([]model.Fact{f})
		a.propagateAssertFact(f)
		a.propagateAssertToken(token)
	}
}

// RetractFact 检查事实是否满足条件，如果满足，则从内存移除并传播撤回信号。
func (a *AlphaNode) RetractFact(f model.Fact) {
	if !a.cond(f) {
		return
	}
	if a.memory.Retract(f) {
		// 事实被成功撤回，向下游传播撤回信号
		token := NewToken([]model.Fact{f})
		a.propagateRetractFact(f)
		a.propagateRetractToken(token)
	}
}

// AssertToken AlphaNode 不直接处理 Token 的断言。
func (a *AlphaNode) AssertToken(t Token) {
	// No-op
}

// RetractToken AlphaNode 不直接处理 Token 的撤回。
func (a *AlphaNode) RetractToken(t Token) {
	// No-op
}
