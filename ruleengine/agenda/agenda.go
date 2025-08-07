package agenda

import (
	"code_for_article/ruleengine/rete"
	"sort"
	"time"
)

// Activation 存储待执行的规则动作。
type Activation struct {
	RuleName string
	Token    rete.Token
	Action   func()

	Salience    int       // 规则优先级（数字越大优先级越高）
	Specificity int       // 规则特殊性（条件越多越特殊）
	CreateTime  time.Time // 创建时间（用于LIFO策略）
}

// ConflictResolutionStrategy 定义冲突解决策略的接口
type ConflictResolutionStrategy interface {
	Compare(a, b Activation) bool // 如果a应该排在b前面则返回true
}

// CompositeStrategy 组合冲突解决策略：Salience -> Specificity -> LIFO
type CompositeStrategy struct{}

func (cs CompositeStrategy) Compare(a, b Activation) bool {
	// 1. 首先按Salience（优先级）排序，数字越大越优先
	if a.Salience != b.Salience {
		return a.Salience > b.Salience
	}

	// 2. 如果Salience相同，按Specificity（特殊性）排序，数字越大越优先
	if a.Specificity != b.Specificity {
		return a.Specificity > b.Specificity
	}

	// 3. 如果Specificity也相同，按LIFO（后进先出）排序
	return a.CreateTime.After(b.CreateTime)
}

// Agenda 智能议程，支持组合冲突解决策略。
type Agenda struct {
	activations []Activation
	strategy    ConflictResolutionStrategy
	sorted      bool // 标记是否已排序
}

func New() *Agenda {
	return &Agenda{
		strategy: CompositeStrategy{},
		sorted:   true,
	}
}

// SetStrategy 设置冲突解决策略
func (a *Agenda) SetStrategy(strategy ConflictResolutionStrategy) {
	a.strategy = strategy
	a.sorted = false
}

// Add 添加新的激活项
func (a *Agenda) Add(ruleName string, tok rete.Token, action func(), salience, specificity int) {
	act := Activation{
		RuleName:    ruleName,
		Token:       tok,
		Action:      action,
		Salience:    salience,
		Specificity: specificity,
		CreateTime:  time.Now(),
	}
	a.activations = append(a.activations, act)
	a.sorted = false // 标记需要重新排序
}

// AddLegacy 为了兼容性保留的旧方法
func (a *Agenda) AddLegacy(ruleName string, tok rete.Token, action func()) {
	a.Add(ruleName, tok, action, 0, 1) // 默认优先级0，特殊性1
}

// Next 获取下一个要执行的激活项
func (a *Agenda) Next() (Activation, bool) {
	if len(a.activations) == 0 {
		return Activation{}, false
	}

	// 如果未排序，先排序
	if !a.sorted {
		a.sort()
	}

	act := a.activations[0]
	a.activations = a.activations[1:]
	return act, true
}

// sort 根据冲突解决策略对激活项进行排序
func (a *Agenda) sort() {
	sort.Slice(a.activations, func(i, j int) bool {
		return a.strategy.Compare(a.activations[i], a.activations[j])
	})
	a.sorted = true
}

func (a *Agenda) Size() int { return len(a.activations) }

// Clear 清空议程
func (a *Agenda) Clear() {
	a.activations = nil
	a.sorted = true
}

// Remove 移除特定的激活项（用于撤回）
func (a *Agenda) Remove(ruleName string, token rete.Token) bool {
	for i, act := range a.activations {
		if act.RuleName == ruleName && act.Token.Hash() == token.Hash() {
			// 移除元素
			a.activations = append(a.activations[:i], a.activations[i+1:]...)
			return true
		}
	}
	return false
}
