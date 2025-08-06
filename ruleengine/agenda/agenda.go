package agenda

import "code_for_article/ruleengine/rete"

// Activation 存储待执行的规则动作。

type Activation struct {
	RuleName string
	Token    rete.Token
	Action   func()
}

// Agenda 简易先进先出实现。可扩展 salience 等策略。

type Agenda struct {
	list []Activation
}

func New() *Agenda { return &Agenda{} }

func (a *Agenda) Add(ruleName string, tok rete.Token, action func()) {
	act := Activation{RuleName: ruleName, Token: tok, Action: action}
	a.list = append(a.list, act)
}

func (a *Agenda) Next() (Activation, bool) {
	if len(a.list) == 0 {
		return Activation{}, false
	}
	act := a.list[0]
	a.list = a.list[1:]
	return act, true
}

func (a *Agenda) Size() int { return len(a.list) }
