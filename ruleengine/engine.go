package ruleengine

import (
	"fmt"

	"code_for_article/ruleengine/agenda"
	"code_for_article/ruleengine/model"
	"code_for_article/ruleengine/rete"
)

// Engine 汇聚 rete 网络、agenda 与工作内存。

type Engine struct {
	alphaRoots []*rete.AlphaNode
	ag         *agenda.Agenda
}

func New() *Engine {
	return &Engine{ag: agenda.New()}
}

// AddAlphaRoot 将顶层 AlphaNode 注册给引擎。
func (e *Engine) AddAlphaRoot(nodes ...*rete.AlphaNode) {
	e.alphaRoots = append(e.alphaRoots, nodes...)
}

// Assert 插入新事实。
func (e *Engine) Assert(f model.Fact) {
	for _, n := range e.alphaRoots {
		n.AssertFact(f)
	}
}

// FireAllRules 持续触发 agenda 直到为空。
func (e *Engine) FireAllRules() {
	for {
		act, ok := e.ag.Next()
		if !ok {
			return
		}
		fmt.Printf("🔥 RULE FIRED: %s | Facts: %v\n", act.RuleName, act.Token.Facts)
		if act.Action != nil {
			act.Action()
		}
	}
}

// Agenda 返回内部 agenda 引用，便于 TerminalNode 使用。
func (e *Engine) Agenda() *agenda.Agenda { return e.ag }
