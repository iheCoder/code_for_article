package main

import (
	"fmt"
	"time"

	"code_for_article/ruleengine"
	"code_for_article/ruleengine/agenda"
	"code_for_article/ruleengine/model"
	"code_for_article/ruleengine/rete"
)

func main() {
	fmt.Println("🚀 组合冲突解决策略演示")
	fmt.Println("========================================")
	fmt.Println("本演示重点展示组合冲突解决策略的工作原理：")
	fmt.Println("1. Salience（优先级）: 数字越大越优先")
	fmt.Println("2. Specificity（特殊性）: 条件越多越优先")
	fmt.Println("3. LIFO（后进先出）: 后激活的规则优先")
	fmt.Println("========================================\n")

	// 直接创建引擎并手动添加规则来演示冲突解决
	engine := ruleengine.New()

	// 手动创建一些简单规则来测试冲突解决策略
	rules := []model.Rule{
		{
			Name:     "高优先级规则",
			Salience: 100,
			When: []model.Condition{
				{Type: "fact", FactType: "User", Field: "Status", Operator: "==", Value: "normal"},
			},
			Then: model.Action{Type: "log", Message: "🔥 高优先级规则触发 (Salience: 100)"},
		},
		{
			Name:     "中优先级规则",
			Salience: 50,
			When: []model.Condition{
				{Type: "fact", FactType: "User", Field: "Status", Operator: "==", Value: "normal"},
			},
			Then: model.Action{Type: "log", Message: "⚡ 中优先级规则触发 (Salience: 50)"},
		},
		{
			Name:     "低优先级但高特殊性规则",
			Salience: 50, // 与中优先级相同
			When: []model.Condition{
				{Type: "fact", FactType: "User", Field: "Status", Operator: "==", Value: "normal"},
				{Type: "fact", FactType: "User", Field: "Level", Operator: "==", Value: "VIP"},
			},
			Then: model.Action{Type: "log", Message: "⭐ 低优先级但高特殊性规则触发 (Salience: 50, Specificity: 2)"},
		},
		{
			Name:     "基础规则",
			Salience: 10,
			When: []model.Condition{
				{Type: "fact", FactType: "User", Field: "Status", Operator: "==", Value: "normal"},
			},
			Then: model.Action{Type: "log", Message: "📝 基础规则触发 (Salience: 10)"},
		},
	}

	// 加载规则
	fmt.Println("📖 加载演示规则...")
	err := engine.LoadRules(rules)
	if err != nil {
		fmt.Printf("❌ 规则加载失败: %v\n", err)
		return
	}
	fmt.Println("✅ 规则加载完成\n")

	// ============ 场景 1: 展示 Salience 优先级 ============
	fmt.Println("📋 场景 1: Salience（优先级）演示")
	fmt.Println("插入一个用户，将同时触发多个规则")
	fmt.Println("应该按照优先级顺序执行：100 -> 50（高特殊性）-> 50（低特殊性）-> 10")
	fmt.Println("----------------------------------------")

	user := model.User{ID: 1, Name: "张三", Status: "normal", Level: "VIP"}
	engine.AddFact(user)

	fmt.Println("🔥 触发规则引擎...")
	engine.FireAllRules()
	fmt.Println()

	// ============ 场景 2: 手动测试 LIFO 策略 ============
	fmt.Println("📋 场景 2: LIFO（后进先出）策略演示")
	fmt.Println("手动创建相同优先级和特殊性的激活项，演示 LIFO 顺序")
	fmt.Println("----------------------------------------")

	// 直接操作 agenda 来演示 LIFO
	testAgenda := agenda.New()

	// 创建具有相同优先级和特殊性的激活项，但不同的创建时间
	user1 := model.User{ID: 1, Name: "第一个", Status: "test"}
	user2 := model.User{ID: 2, Name: "第二个", Status: "test"}
	user3 := model.User{ID: 3, Name: "第三个", Status: "test"}

	token1 := rete.NewToken([]model.Fact{user1})
	token2 := rete.NewToken([]model.Fact{user2})
	token3 := rete.NewToken([]model.Fact{user3})

	fmt.Println("添加激活项（相同优先级和特殊性）：")

	// 第一个激活项
	testAgenda.Add("规则A", token1, func() {
		fmt.Println("  📝 规则A执行 - 第一个添加的")
	}, 50, 1)
	fmt.Println("  ➕ 添加规则A激活项")
	time.Sleep(10 * time.Millisecond) // 确保时间差

	// 第二个激活项
	testAgenda.Add("规则B", token2, func() {
		fmt.Println("  📝 规则B执行 - 第二个添加的")
	}, 50, 1)
	fmt.Println("  ➕ 添加规则B激活项")
	time.Sleep(10 * time.Millisecond)

	// 第三个激活项
	testAgenda.Add("规则C", token3, func() {
		fmt.Println("  📝 规则C执行 - 第三个添加的（应该最先执行）")
	}, 50, 1)
	fmt.Println("  ➕ 添加规则C激活项")

	fmt.Println("\n按照LIFO顺序执行（后添加的先执行）：")
	for testAgenda.Size() > 0 {
		if activation, ok := testAgenda.Next(); ok {
			activation.Action()
		}
	}
	fmt.Println()

	// ============ 场景 3: 完整组合策略演示 ============
	fmt.Println("📋 场景 3: 完整组合策略演示")
	fmt.Println("混合不同优先级、特殊性和时间的激活项")
	fmt.Println("----------------------------------------")

	combinedAgenda := agenda.New()

	// 添加不同类型的激活项
	fmt.Println("添加混合激活项：")

	// 低优先级，高特殊性，早添加
	combinedAgenda.Add("低优先级高特殊性", token1, func() {
		fmt.Println("  🎯 低优先级高特殊性规则执行 (Salience: 10, Specificity: 3)")
	}, 10, 3)
	fmt.Println("  ➕ 低优先级(10) 高特殊性(3)")
	time.Sleep(10 * time.Millisecond)

	// 高优先级，低特殊性，中间添加
	combinedAgenda.Add("高优先级低特殊性", token2, func() {
		fmt.Println("  🔥 高优先级低特殊性规则执行 (Salience: 100, Specificity: 1)")
	}, 100, 1)
	fmt.Println("  ➕ 高优先级(100) 低特殊性(1)")
	time.Sleep(10 * time.Millisecond)

	// 中等优先级，中等特殊性，晚添加
	combinedAgenda.Add("中等优先级特殊性", token3, func() {
		fmt.Println("  ⚡ 中等优先级特殊性规则执行 (Salience: 50, Specificity: 2)")
	}, 50, 2)
	fmt.Println("  ➕ 中等优先级(50) 中等特殊性(2)")
	time.Sleep(10 * time.Millisecond)

	// 相同优先级特殊性，测试LIFO
	combinedAgenda.Add("相同参数规则1", token1, func() {
		fmt.Println("  📄 相同参数规则1执行 (先添加)")
	}, 50, 2)
	fmt.Println("  ➕ 相同参数规则1 (50, 2)")
	time.Sleep(10 * time.Millisecond)

	combinedAgenda.Add("相同参数规则2", token2, func() {
		fmt.Println("  📋 相同参数规则2执行 (后添加，应该在规则1前执行)")
	}, 50, 2)
	fmt.Println("  ➕ 相同参数规则2 (50, 2)")

	fmt.Println("\n执行顺序应该是：")
	fmt.Println("1. 高优先级低特殊性 (Salience: 100)")
	fmt.Println("2. 相同参数规则2 (Salience: 50, Specificity: 2, LIFO)")
	fmt.Println("3. 相同参数规则1 (Salience: 50, Specificity: 2, LIFO)")
	fmt.Println("4. 中等优先级特殊性 (Salience: 50, Specificity: 2, LIFO)")
	fmt.Println("5. 低优先级高特殊性 (Salience: 10, Specificity: 3)")
	fmt.Println("\n实际执行顺序：")

	for combinedAgenda.Size() > 0 {
		if activation, ok := combinedAgenda.Next(); ok {
			activation.Action()
		}
	}

	fmt.Println("\n========================================")
	fmt.Println("🎉 组合冲突解决策略演示完成！")
	fmt.Println("通过本演示，您可以清楚地看到：")
	fmt.Println("1. Salience（优先级）是第一层过滤器")
	fmt.Println("2. Specificity（特殊性）是第二层过滤器")
	fmt.Println("3. LIFO（后进先出）是最终的决胜策略")
	fmt.Println("4. 这种组合策略确保了规则执行的可预测性和逻辑性")
}
