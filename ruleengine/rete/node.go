package rete

import "code_for_article/ruleengine/model"

// Node 是 rete 网络中所有节点的统一接口。
// 它支持对 Fact 和 Token 的断言 (Assert) 与撤回 (Retract)。
type Node interface {
	AssertFact(f model.Fact)
	RetractFact(f model.Fact)

	AssertToken(t Token)
	RetractToken(t Token)

	AddChild(n Node)
}

// baseNode 提供通用的 children 管理及传播实现。
type baseNode struct {
	children []Node
}

// AddChild 向节点添加一个子节点。
func (b *baseNode) AddChild(n Node) {
	b.children = append(b.children, n)
}

// --- Propagate Assertions ---

func (b *baseNode) propagateAssertFact(f model.Fact) {
	for _, child := range b.children {
		child.AssertFact(f)
	}
}

func (b *baseNode) propagateAssertToken(t Token) {
	for _, child := range b.children {
		child.AssertToken(t)
	}
}

// --- Propagate Retractions ---

func (b *baseNode) propagateRetractFact(f model.Fact) {
	for _, child := range b.children {
		child.RetractFact(f)
	}
}

func (b *baseNode) propagateRetractToken(t Token) {
	for _, child := range b.children {
		child.RetractToken(t)
	}
}
