package ruleengine

// Engine 提供对外规则引擎 API 的骨架结构。
// 详细实现将在后续步骤逐步完善。

type Engine struct {
	// TODO: WorkingMemory, rete network, agenda
}

// New 创建并返回一个新的规则引擎实例。
func New() *Engine {
	return &Engine{}
}
