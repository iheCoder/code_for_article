package rete

import "code_for_article/ruleengine/model"

// Node 是 rete 网络中所有节点的统一接口。
// Token 从左流向右，Fact 从上游 Alpha 进入。

type Node interface {
	AssertFact(model.Fact)
	AssertToken(Token)
	AddChild(Node)
}

// baseNode 提供通用的 children 管理实现。

type baseNode struct {
	children []Node
}

func (b *baseNode) AddChild(n Node) {
	b.children = append(b.children, n)
}

func (b *baseNode) propagateFactToChildren(f model.Fact) {
	for _, c := range b.children {
		c.AssertFact(f)
	}
}

func (b *baseNode) propagateTokenToChildren(t Token) {
	for _, c := range b.children {
		c.AssertToken(t)
	}
}
