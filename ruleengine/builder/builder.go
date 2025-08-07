package builder

import (
	"fmt"
	"reflect"
	"strconv"

	"code_for_article/ruleengine/agenda"
	"code_for_article/ruleengine/model"
	"code_for_article/ruleengine/rete"
)

// Builder 负责将声明式的规则定义编译成 Rete 网络。
type Builder struct {
	alphaNodes map[string]*rete.AlphaNode // key: 条件描述
	agenda     *agenda.Agenda
}

// NewBuilder 创建一个新的规则构建器。
func NewBuilder(ag *agenda.Agenda) *Builder {
	return &Builder{
		alphaNodes: make(map[string]*rete.AlphaNode),
		agenda:     ag,
	}
}

// BuildRule 将单条规则编译成 Rete 网络节点，并返回根节点列表。
// 简化实现：每个条件都创建独立的 AlphaNode，并通过 BetaNode 连接。
func (b *Builder) BuildRule(rule model.Rule) ([]*rete.AlphaNode, error) {
	var rootNodes []*rete.AlphaNode
	var currentNode rete.Node

	// 计算规则特殊性（条件数量）
	specificity := len(rule.When)

	// 创建终端节点
	terminalNode := rete.NewTerminalNode(rule.Name, b.agenda, b.createAction(rule.Then), rule.Salience, specificity)

	// 简化：处理第一个条件作为根节点
	if len(rule.When) == 0 {
		return rootNodes, fmt.Errorf("规则 '%s' 没有条件", rule.Name)
	}

	firstCondition := rule.When[0]
	switch firstCondition.Type {
	case "fact":
		alphaNode, err := b.buildFactCondition(firstCondition)
		if err != nil {
			return nil, err
		}
		rootNodes = append(rootNodes, alphaNode)
		currentNode = alphaNode

	case "aggregate":
		aggNode := b.buildAggregateNode(firstCondition)
		// 聚合节点需要包装在 AlphaNode 中作为根节点
		dummyAlpha := rete.NewAlphaNode(func(f model.Fact) bool { return true })
		dummyAlpha.AddChild(aggNode)
		rootNodes = append(rootNodes, dummyAlpha)
		currentNode = aggNode

	default:
		return nil, fmt.Errorf("不支持的根节点类型: %s", firstCondition.Type)
	}

	// 处理后续条件（简化：仅支持基本的 fact 条件连接）
	for i := 1; i < len(rule.When); i++ {
		condition := rule.When[i]

		switch condition.Type {
		case "fact":
			alphaNode, err := b.buildFactCondition(condition)
			if err != nil {
				return nil, err
			}

			// 创建 BetaNode 连接
			betaNode := b.buildJoinNode(condition.Join)

			// 连接网络
			if alpha, ok := currentNode.(*rete.AlphaNode); ok {
				alpha.AddChild(betaNode)
			}
			alphaNode.AddChild(betaNode)
			currentNode = betaNode

		case "not":
			// 简化：not 节点直接接在当前节点后
			notNode := b.buildNotNode(condition)
			currentNode.AddChild(notNode)
			currentNode = notNode

		case "exists":
			// 简化：exists 节点直接接在当前节点后
			existsNode := b.buildExistsNode(condition)
			currentNode.AddChild(existsNode)
			currentNode = existsNode
		}
	}

	// 连接终端节点
	currentNode.AddChild(terminalNode)

	return rootNodes, nil
}

// buildFactCondition 根据条件创建 AlphaNode。
func (b *Builder) buildFactCondition(condition model.Condition) (*rete.AlphaNode, error) {
	key := fmt.Sprintf("%s.%s %s %v", condition.FactType, condition.Field, condition.Operator, condition.Value)

	if node, exists := b.alphaNodes[key]; exists {
		return node, nil // 节点复用
	}

	alphaFunc := func(f model.Fact) bool {
		return b.evaluateCondition(f, condition)
	}

	node := rete.NewAlphaNode(alphaFunc)
	b.alphaNodes[key] = node
	return node, nil
}

// buildJoinNode 创建 BetaNode 进行条件连接。
func (b *Builder) buildJoinNode(joinClause *model.JoinClause) *rete.BetaNode {
	if joinClause == nil {
		// 默认连接：简单的 AND 关系，不需要特殊条件
		return rete.NewBetaNode(func(t rete.Token, f model.Fact) bool {
			return true // 总是成功连接
		})
	}

	return rete.NewBetaNode(func(t rete.Token, f model.Fact) bool {
		// 实现基于字段的连接逻辑
		if len(t.Facts) == 0 {
			return false
		}
		leftFact := t.Facts[len(t.Facts)-1]
		leftVal := b.getFieldValue(leftFact, joinClause.LeftField)
		rightVal := b.getFieldValue(f, joinClause.RightField)
		return leftVal == rightVal
	})
}

// buildNotNode 创建 NotNode。
func (b *Builder) buildNotNode(condition model.Condition) *rete.NotNode {
	return rete.NewNotNode(func(t rete.Token, f model.Fact) bool {
		return b.evaluateCondition(f, condition)
	})
}

// buildExistsNode 创建 ExistsNode。
func (b *Builder) buildExistsNode(condition model.Condition) *rete.ExistsNode {
	return rete.NewExistsNode(func(t rete.Token, f model.Fact) bool {
		return b.evaluateCondition(f, condition)
	})
}

// buildAggregateNode 创建 AggregateNode。
func (b *Builder) buildAggregateNode(condition model.Condition) *rete.AggregateNode {
	groupFunc := func(f model.Fact) (string, bool) {
		val := b.getFieldValue(f, condition.GroupBy)
		if val != nil {
			return fmt.Sprintf("%v", val), true
		}
		return "", false
	}
	return rete.NewAggregateNode(groupFunc, condition.Threshold)
}

// evaluateCondition 评估单个条件是否满足。
func (b *Builder) evaluateCondition(fact model.Fact, condition model.Condition) bool {
	// 检查事实类型
	factType := reflect.TypeOf(fact).Name()
	if factType != condition.FactType {
		return false
	}

	// 获取字段值
	fieldValue := b.getFieldValue(fact, condition.Field)
	if fieldValue == nil {
		return false
	}

	// 执行比较操作
	return b.compareValues(fieldValue, condition.Operator, condition.Value)
}

// getFieldValue 使用反射获取结构体字段值。
func (b *Builder) getFieldValue(fact model.Fact, fieldName string) interface{} {
	val := reflect.ValueOf(fact)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return nil
	}

	field := val.FieldByName(fieldName)
	if !field.IsValid() {
		return nil
	}
	return field.Interface()
}

// compareValues 比较两个值。
func (b *Builder) compareValues(left interface{}, operator string, right interface{}) bool {
	switch operator {
	case "==":
		return left == right
	case "!=":
		return left != right
	case ">":
		return b.compareNumeric(left, right) > 0
	case ">=":
		return b.compareNumeric(left, right) >= 0
	case "<":
		return b.compareNumeric(left, right) < 0
	case "<=":
		return b.compareNumeric(left, right) <= 0
	default:
		return false
	}
}

// compareNumeric 比较数值。
func (b *Builder) compareNumeric(left, right interface{}) int {
	leftFloat := b.toFloat64(left)
	rightFloat := b.toFloat64(right)

	if leftFloat > rightFloat {
		return 1
	} else if leftFloat < rightFloat {
		return -1
	}
	return 0
}

// toFloat64 将值转换为 float64。
func (b *Builder) toFloat64(val interface{}) float64 {
	switch v := val.(type) {
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case float32:
		return float64(v)
	case float64:
		return v
	case string:
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return 0
}

// createAction 创建规则执行动作。
func (b *Builder) createAction(action model.Action) func(rete.Token) {
	return func(token rete.Token) {
		switch action.Type {
		case "log":
			fmt.Printf("🔥 规则触发: %s | 事实: %v\n", action.Message, token.Facts)
		case "callback":
			// 可以在这里添加自定义回调逻辑
			fmt.Printf("📞 回调执行: %s\n", action.Message)
		default:
			fmt.Printf("⚡ 动作执行: %s\n", action.Message)
		}
	}
}
