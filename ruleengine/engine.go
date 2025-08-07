package ruleengine

import (
	"fmt"
	"os"

	"code_for_article/ruleengine/agenda"
	"code_for_article/ruleengine/builder"
	"code_for_article/ruleengine/model"
	"code_for_article/ruleengine/rete"
	"gopkg.in/yaml.v2"
)

// Engine 汇聚 rete 网络、agenda 与工作内存。
type Engine struct {
	alphaRoots []*rete.AlphaNode
	ag         *agenda.Agenda
	builder    *builder.Builder
}

// New 创建一个新的规则引擎实例。
func New() *Engine {
	ag := agenda.New()
	return &Engine{
		ag:      ag,
		builder: builder.NewBuilder(ag),
	}
}

// AddAlphaRoot 将顶层 AlphaNode 注册给引擎。
func (e *Engine) AddAlphaRoot(nodes ...*rete.AlphaNode) {
	e.alphaRoots = append(e.alphaRoots, nodes...)
}

// LoadRulesFromYAML 从 YAML 文件加载规则并构建 Rete 网络。
func (e *Engine) LoadRulesFromYAML(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("读取文件失败: %w", err)
	}

	var ruleSet model.RuleSet
	if err := yaml.Unmarshal(data, &ruleSet); err != nil {
		return fmt.Errorf("解析 YAML 失败: %w", err)
	}

	return e.LoadRules(ruleSet.Rules)
}

// LoadRules 加载规则列表并构建 Rete 网络。
func (e *Engine) LoadRules(rules []model.Rule) error {
	for _, rule := range rules {
		roots, err := e.builder.BuildRule(rule)
		if err != nil {
			return fmt.Errorf("构建规则 '%s' 失败: %w", rule.Name, err)
		}
		e.AddAlphaRoot(roots...)
	}
	return nil
}

// AddFact 插入新事实。
func (e *Engine) AddFact(f model.Fact) {
	for _, n := range e.alphaRoots {
		n.AssertFact(f)
	}
}

// RetractFact 撤回事实。
func (e *Engine) RetractFact(f model.Fact) {
	for _, n := range e.alphaRoots {
		n.RetractFact(f)
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
