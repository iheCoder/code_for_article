package model

// RuleSet 表示一组规则的集合，通常从 YAML 或 JSON 文件加载。
type RuleSet struct {
	Rules []Rule `yaml:"rules" json:"rules"`
}

// Rule 表示单条业务规则的声明式定义。
type Rule struct {
	Name        string      `yaml:"name" json:"name"`
	Description string      `yaml:"description,omitempty" json:"description,omitempty"`
	Salience    int         `yaml:"salience,omitempty" json:"salience,omitempty"` // 优先级，默认为 0
	When        []Condition `yaml:"when" json:"when"`
	Then        Action      `yaml:"then" json:"then"`
}

// Condition 表示规则的一个条件子句。
type Condition struct {
	Type     string      `yaml:"type" json:"type"`           // "fact", "not", "exists", "aggregate"
	FactType string      `yaml:"fact_type" json:"fact_type"` // 事实类型，如 "User", "Order"
	Field    string      `yaml:"field,omitempty" json:"field,omitempty"`
	Operator string      `yaml:"operator,omitempty" json:"operator,omitempty"` // "==", ">", "<", ">=", "<=", "!="
	Value    interface{} `yaml:"value,omitempty" json:"value,omitempty"`
	Join     *JoinClause `yaml:"join,omitempty" json:"join,omitempty"` // 用于连接条件

	// 聚合相关
	GroupBy   string `yaml:"group_by,omitempty" json:"group_by,omitempty"`
	Aggregate string `yaml:"aggregate,omitempty" json:"aggregate,omitempty"` // "count", "sum"
	Threshold int    `yaml:"threshold,omitempty" json:"threshold,omitempty"`
}

// JoinClause 定义两个条件之间的连接关系。
type JoinClause struct {
	LeftField  string `yaml:"left_field" json:"left_field"`
	RightField string `yaml:"right_field" json:"right_field"`
}

// Action 定义规则触发时的执行动作。
type Action struct {
	Type    string                 `yaml:"type" json:"type"` // "log", "assert", "callback"
	Message string                 `yaml:"message,omitempty" json:"message,omitempty"`
	Data    map[string]interface{} `yaml:"data,omitempty" json:"data,omitempty"`
}
