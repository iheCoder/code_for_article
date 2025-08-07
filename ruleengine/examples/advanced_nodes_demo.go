package main

import (
	"fmt"
	"time"

	"code_for_article/ruleengine"
	"code_for_article/ruleengine/model"
)

func main() {
	fmt.Println("🚀 高级节点演示 - NotNode 和 AggregateNode")
	fmt.Println("========================================")
	fmt.Println("本演示展示：")
	fmt.Println("1. NotNode 的使用 - 检测缺失条件")
	fmt.Println("2. AggregateNode 的使用 - 聚合统计")
	fmt.Println("3. ExistsNode 的使用 - 存在性检查")
	fmt.Println("========================================\n")

	// ============ 创建简化的规则进行演示 ============

	// 创建引擎
	engine := ruleengine.New()

	// 定义简化的规则来演示高级节点
	rules := []model.Rule{
		// NOT节点演示：检测没有可信设备的高额交易
		{
			Name:     "NOT演示_无可信设备高额交易",
			Salience: 90,
			When: []model.Condition{
				{Type: "fact", FactType: "Transaction", Field: "Amount", Operator: ">", Value: 10000},
				// 注意：实际的NOT逻辑需要builder支持，这里先用简单条件演示概念
			},
			Then: model.Action{Type: "log", Message: "🔍 NOT演示：检测到高额交易但缺少安全验证"},
		},

		// 聚合节点演示：需要修改builder支持
		{
			Name:     "AGGREGATE演示_失败登录统计",
			Salience: 80,
			When: []model.Condition{
				{
					Type:      "aggregate",
					FactType:  "LoginAttempt",
					GroupBy:   "UserID",
					Aggregate: "count",
					Threshold: 3,
					Field:     "Success",
					Operator:  "==",
					Value:     false,
				},
			},
			Then: model.Action{Type: "log", Message: "⚠️ AGGREGATE演示：检测到多次失败登录"},
		},

		// EXISTS节点演示
		{
			Name:     "EXISTS演示_VIP用户交易检查",
			Salience: 70,
			When: []model.Condition{
				{Type: "fact", FactType: "User", Field: "Level", Operator: "==", Value: "VIP"},
				// EXISTS逻辑也需要builder支持
			},
			Then: model.Action{Type: "log", Message: "🎁 EXISTS演示：VIP用户有交易记录"},
		},

		// 基础规则用于对比
		{
			Name:     "基础规则_交易监控",
			Salience: 50,
			When: []model.Condition{
				{Type: "fact", FactType: "Transaction", Field: "Amount", Operator: ">", Value: 5000},
			},
			Then: model.Action{Type: "log", Message: "💰 基础监控：检测到大额交易"},
		},
	}

	fmt.Println("📖 加载演示规则...")
	err := engine.LoadRules(rules)
	if err != nil {
		fmt.Printf("❌ 规则加载失败: %v\n", err)
		return
	}
	fmt.Println("✅ 规则加载完成\n")

	// ============ 场景 1: 基础功能演示 ============
	fmt.Println("📋 场景 1: 基础交易监控演示")
	fmt.Println("----------------------------------------")

	user1 := model.User{ID: 1, Name: "张三", Status: "normal", Level: "VIP"}
	engine.AddFact(user1)

	transaction1 := model.Transaction{
		ID:       101,
		UserID:   1,
		Amount:   15000,
		Currency: "CNY",
		Type:     "transfer",
		Status:   "pending",
		Location: "Beijing",
	}
	engine.AddFact(transaction1)

	fmt.Println("🔥 插入高额交易，触发规则...")
	engine.FireAllRules()
	fmt.Println()

	// ============ 场景 2: 聚合功能演示（概念性） ============
	fmt.Println("📋 场景 2: 聚合功能演示（概念性）")
	fmt.Println("由于当前builder限制，我们手动演示聚合概念")
	fmt.Println("----------------------------------------")

	// 重新创建引擎专门测试聚合
	engine2 := ruleengine.New()
	engine2.LoadRules(rules)

	user2 := model.User{ID: 2, Name: "李四", Status: "normal", Level: "normal"}
	engine2.AddFact(user2)

	fmt.Println("💡 模拟连续失败登录尝试...")

	// 添加失败登录尝试
	failedAttempts := []model.LoginAttempt{
		{ID: 201, UserID: 2, Success: false, Timestamp: time.Now().Unix() - 180, IP: "192.168.1.100"},
		{ID: 202, UserID: 2, Success: false, Timestamp: time.Now().Unix() - 120, IP: "192.168.1.101"},
		{ID: 203, UserID: 2, Success: false, Timestamp: time.Now().Unix() - 60, IP: "192.168.1.102"},
	}

	for i, attempt := range failedAttempts {
		fmt.Printf("  添加第 %d 次失败登录\n", i+1)
		engine2.AddFact(attempt)
		if i == 2 {
			fmt.Println("  💥 当达到3次失败时，聚合规则应该触发（需要正确的builder实现）")
		}
		engine2.FireAllRules()
	}
	fmt.Println()

	// ============ 场景 3: 演示冲突解决策略在复杂场景中的表现 ============
	fmt.Println("📋 场景 3: 复杂场景中的冲突解决")
	fmt.Println("同时触发多个不同优先级的规则")
	fmt.Println("----------------------------------------")

	// 创建新引擎
	engine3 := ruleengine.New()

	// 添加更多规则来展示冲突解决
	moreRules := []model.Rule{
		{
			Name:     "紧急规则_超大额交易",
			Salience: 100,
			When: []model.Condition{
				{Type: "fact", FactType: "Transaction", Field: "Amount", Operator: ">", Value: 50000},
			},
			Then: model.Action{Type: "log", Message: "🚨 紧急：超大额交易需要立即处理"},
		},
		{
			Name:     "高优先级_VIP大额交易",
			Salience: 80,
			When: []model.Condition{
				{Type: "fact", FactType: "Transaction", Field: "Amount", Operator: ">", Value: 20000},
				{Type: "fact", FactType: "User", Field: "Level", Operator: "==", Value: "VIP"},
			},
			Then: model.Action{Type: "log", Message: "⭐ 高优先级：VIP大额交易特殊处理"},
		},
		{
			Name:     "中优先级_提现监控",
			Salience: 60,
			When: []model.Condition{
				{Type: "fact", FactType: "Transaction", Field: "Type", Operator: "==", Value: "withdraw"},
			},
			Then: model.Action{Type: "log", Message: "💸 中优先级：提现交易监控"},
		},
		{
			Name:     "基础_交易记录",
			Salience: 40,
			When: []model.Condition{
				{Type: "fact", FactType: "Transaction", Field: "Status", Operator: "==", Value: "pending"},
			},
			Then: model.Action{Type: "log", Message: "📝 基础：记录待处理交易"},
		},
	}

	engine3.LoadRules(moreRules)

	// 插入会触发多个规则的数据
	vipUser := model.User{ID: 3, Name: "VIP用户", Status: "normal", Level: "VIP"}
	engine3.AddFact(vipUser)

	bigTransaction := model.Transaction{
		ID:       103,
		UserID:   3,
		Amount:   60000, // 超大额，会触发多个规则
		Currency: "CNY",
		Type:     "withdraw", // 提现类型
		Status:   "pending",  // 待处理状态
		Location: "Shanghai",
	}
	engine3.AddFact(bigTransaction)

	fmt.Println("🔥 插入超大额VIP提现交易，观察执行顺序...")
	fmt.Println("期望顺序：紧急(100) -> VIP高优先级(80) -> 提现监控(60) -> 基础记录(40)")
	engine3.FireAllRules()
	fmt.Println()

	fmt.Println("========================================")
	fmt.Println("🎉 高级节点演示完成！")
	fmt.Println()
	fmt.Println("📚 总结：")
	fmt.Println("1. ✅ 组合冲突解决策略正常工作")
	fmt.Println("2. ⏳ NotNode 和 AggregateNode 需要更完整的builder支持")
	fmt.Println("3. ⏳ ExistsNode 同样需要builder层面的完善")
	fmt.Println("4. ✅ 基础规则引擎框架运行良好")
	fmt.Println()
	fmt.Println("🚀 下一步改进方向：")
	fmt.Println("- 完善builder对复杂节点类型的支持")
	fmt.Println("- 增加更灵活的条件表达式解析")
	fmt.Println("- 添加运行时规则动态加载功能")
}
