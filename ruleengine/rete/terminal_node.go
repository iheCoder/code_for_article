package rete

// TerminalNode 在规则最终满足时产生 Activation 并放入 Agenda。

import (
	"code_for_article/ruleengine/model"
)

// AgendaAdder 接口，避免循环依赖
type AgendaAdder interface {
	Add(ruleName string, tok Token, action func())
}

type TerminalNode struct {
	baseNode
	ruleName string
	ag       AgendaAdder
	action   func(Token)
}

func NewTerminalNode(ruleName string, ag AgendaAdder, action func(Token)) *TerminalNode {
	return &TerminalNode{ruleName: ruleName, ag: ag, action: action}
}

func (t *TerminalNode) AssertFact(fact model.Fact) {
	// Terminal 不处理单独 Fact
}

func (t *TerminalNode) AssertToken(tok Token) {
	t.ag.Add(t.ruleName, tok, func() { t.action(tok) })
}
