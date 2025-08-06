package model

// Fact 是所有业务实体插入规则引擎前需实现的接口。
// Key 必须在工作内存中唯一，用于快速定位与撤回。
// 建议使用业务主键或复合键（如 "User:42"）。
//
// 业务侧可直接在其结构体上实现 Key() 方法，或使用 GenericFact 包装。

type Fact interface {
	Key() string
}

// GenericFact 作为简易示例，适用于无须自定义结构体的场景。
// 生产中更常见做法是：直接让业务结构体实现 Fact 接口。

type GenericFact struct {
	ID      string
	Payload any
}

func (g GenericFact) Key() string { return g.ID }
