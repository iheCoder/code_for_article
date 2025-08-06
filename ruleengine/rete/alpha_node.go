package rete

import "code_for_article/ruleengine/model"

// AlphaFunc 判断 fact 是否满足节点条件。

type AlphaFunc func(model.Fact) bool

// AlphaNode 处理单一 Fact 条件过滤。

type AlphaNode struct {
	baseNode
	cond   AlphaFunc
	memory *AlphaMemory
}

func NewAlphaNode(f AlphaFunc) *AlphaNode {
	return &AlphaNode{cond: f, memory: NewAlphaMemory()}
}

func (a *AlphaNode) AssertFact(f model.Fact) {
	if !a.cond(f) {
		return
	}
	if a.memory.Assert(f) {
		// 新 fact 满足条件，向下传播
		// 对于需要 Token 的子节点，生成单元素 Token
		token := NewToken([]model.Fact{f})
		a.propagateFactToChildren(f)
		a.propagateTokenToChildren(token)
	}
}

func (a *AlphaNode) AssertToken(Token) {
	// AlphaNode 不处理 Token
}
