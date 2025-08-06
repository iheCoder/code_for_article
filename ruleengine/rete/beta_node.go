package rete

import "code_for_article/ruleengine/model"

// JoinFunc 判断 (左 token, 右 fact) 是否满足 join 条件。

type JoinFunc func(Token, model.Fact) bool

// BetaNode 负责两侧输入的连接。

type BetaNode struct {
	baseNode
	join        JoinFunc
	leftMemory  *BetaMemory  // Token
	rightMemory *AlphaMemory // Fact
}

func NewBetaNode(j JoinFunc) *BetaNode {
	return &BetaNode{
		join:        j,
		leftMemory:  NewBetaMemory(),
		rightMemory: NewAlphaMemory(),
	}
}

func (b *BetaNode) AssertToken(t Token) {
	// 左输入 token
	if !b.leftMemory.Assert(t) {
		return
	}
	for _, f := range b.rightMemory.Snapshot() {
		if b.join(t, f) {
			newToken := NewToken(append(append([]model.Fact(nil), t.Facts...), f))
			b.propagateTokenToChildren(newToken)
		}
	}
}

func (b *BetaNode) AssertFact(f model.Fact) {
	// 右输入 fact
	if !b.rightMemory.Assert(f) {
		return
	}
	for _, tok := range b.leftMemory.Snapshot() {
		if b.join(tok, f) {
			newToken := NewToken(append(append([]model.Fact(nil), tok.Facts...), f))
			b.propagateTokenToChildren(newToken)
		}
	}
}
