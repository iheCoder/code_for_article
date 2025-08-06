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

func (t *TerminalNode) RetractFact(fact model.Fact) {
	// Terminal 不处理单独 Fact
}

func (t *TerminalNode) RetractToken(tok Token) {
	// Terminal 不处理 Token 的撤回
	// 但可以在需要时添加逻辑来处理 Token 的撤回
	// 例如：如果需要在 Token 被撤回时执行某些清理操作
}
