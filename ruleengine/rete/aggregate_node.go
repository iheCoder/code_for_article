package rete

import (
	"code_for_article/ruleengine/model"
	"fmt"
)

// AggregateFunc 定义了从事实中提取分组键的函数。
type AggregateFunc func(f model.Fact) (groupKey string, ok bool)

// AggregateNode 实现聚合功能，例如 "count(事实) > N"。
//
// 工作流程:
// 1. AssertFact: 当一个 Fact 到达时，使用 groupBy 函数提取其分组键。
//   - 对应分组的计数器加一。
//   - 如果计数值 **首次** 达到或超过阈值 (threshold)，则生成一个特殊的聚合结果事实
//     (AggregateResult) 并向下游传播。
//
// 2. RetractFact: (为保持示例简洁，本实现 **未** 完整支持撤回)
//   - 理想情况下，撤回一个 Fact 会使其分组计数减一。
//   - 如果计数值从阈值之上降到阈值之下，则应向下游传播对聚合结果事实的撤回。
type AggregateNode struct {
	baseNode
	groupBy    AggregateFunc
	threshold  int
	rightFacts *AlphaMemory
	counts     map[string]int // groupKey -> count
}

// AggregateResult 是一个特殊的事实，代表聚合运算的结果。
type AggregateResult struct {
	GroupKey string
	Count    int
}

func (ar AggregateResult) Key() string { return fmt.Sprintf("agg:%s", ar.GroupKey) }

func NewAggregateNode(groupBy AggregateFunc, threshold int) *AggregateNode {
	return &AggregateNode{
		groupBy:    groupBy,
		threshold:  threshold,
		rightFacts: NewAlphaMemory(),
		counts:     make(map[string]int),
	}
}

func (a *AggregateNode) AssertFact(f model.Fact) {
	// 仅当事实满足分组条件时，才进行聚合
	if !a.rightFacts.Add(f) {
		return
	}

	// 使用 groupBy 函数提取分组键
	// 如果无法提取分组键，则忽略该事实
	key, ok := a.groupBy(f)
	if !ok {
		return
	}

	// 仅当计数从 threshold-1 上升到 threshold 时，才传播断言
	if a.counts[key] == a.threshold-1 {
		resultFact := AggregateResult{GroupKey: key, Count: a.threshold}
		// 聚合节点将产生新的事实流
		a.propagateAssertFact(resultFact)
	}
	a.counts[key]++
}

func (a *AggregateNode) RetractFact(f model.Fact) {
	// 简化：本实现不支持聚合节点的撤回。
	// 在生产环境中，需要在这里实现计数减少和结果撤回的逻辑。
}

func (a *AggregateNode) AssertToken(t Token)  {}
func (a *AggregateNode) RetractToken(t Token) {}
