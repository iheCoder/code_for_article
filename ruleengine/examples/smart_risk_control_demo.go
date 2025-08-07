package main

import (
	"fmt"
	"time"

	"code_for_article/ruleengine"
	"code_for_article/ruleengine/model"
)

func main() {
	fmt.Println("🚀 智能风控系统演示 - 组合冲突解决策略")
	fmt.Println("========================================")
	fmt.Println("本演示展示：")
	fmt.Println("1. 组合冲突解决策略（Salience + Specificity + LIFO）")
	fmt.Println("2. NotNode 的使用（检测缺失的可信设备）")
	fmt.Println("3. AggregateNode 的使用（统计失败登录次数）")
	fmt.Println("4. ExistsNode 的使用（检测存在的交易记录）")
	fmt.Println("========================================\n")

	// 创建引擎
	engine := ruleengine.New()

	// 加载智能风控规则
	fmt.Println("📖 加载智能风控规则...")
	err := engine.LoadRulesFromYAML("ruleengine/examples/smart_risk_control_rules.yaml")
	if err != nil {
		fmt.Printf("❌ 规则加载失败: %v\n", err)
		return
	}
	fmt.Println("✅ 规则加载完成\n")

	// ============ 场景 1: 测试冲突解决策略 ============
	fmt.Println("📋 场景 1: 冲突解决策略测试")
	fmt.Println("同时触发多个规则，观察执行顺序（Salience -> Specificity -> LIFO）")
	fmt.Println("----------------------------------------")

	// 插入用户数据
	user1 := model.User{ID: 1, Name: "张三", Status: "normal", Level: "VIP"}
	engine.AddFact(user1)

	// 插入用户画像（新用户，高风险评分）
	userProfile1 := model.UserProfile{
		UserID:          1,
		RegistrationAge: 15, // 新用户（小于30天）
		RiskScore:       85, // 高风险评分
		HomeLocation:    "Shanghai",
	}
	engine.AddFact(userProfile1)

	// 插入大额提现交易（会触发多个规则）
	transaction1 := model.Transaction{
		ID:       101,
		UserID:   1,
		Amount:   25000, // 大额交易
		Currency: "CNY",
		Type:     "withdraw", // 提现
		Status:   "pending",
		Location: "Shanghai",
	}
	engine.AddFact(transaction1)

	// 触发规则引擎
	fmt.Println("🔥 触发规则引擎...")
	engine.FireAllRules()
	fmt.Println()

	// ============ 场景 2: NotNode 演示 ============
	fmt.Println("📋 场景 2: NotNode 演示 - 未绑定可信设备的高风险交易")
	fmt.Println("----------------------------------------")

	// 清空前面的激活项，重新开始
	engine = ruleengine.New()
	engine.LoadRulesFromYAML("ruleengine/examples/smart_risk_control_rules.yaml")

	// 插入高额交易，但不插入可信设备信息
	user2 := model.User{ID: 2, Name: "李四", Status: "normal", Level: "normal"}
	engine.AddFact(user2)

	transaction2 := model.Transaction{
		ID:       102,
		UserID:   2,
		Amount:   15000, // 高额交易
		Currency: "CNY",
		Type:     "transfer",
		Status:   "pending",
		Location: "Beijing",
	}
	engine.AddFact(transaction2)

	fmt.Println("💡 插入高额交易但不插入可信设备信息（NotNode 应该触发）")
	engine.FireAllRules()
	fmt.Println()

	// 现在插入可信设备信息，看看是否还会触发
	fmt.Println("💡 现在插入可信设备信息，再次测试...")
	deviceInfo := model.DeviceInfo{
		DeviceID:   "device_trusted_001",
		UserID:     2,
		Trusted:    true,
		LastSeen:   time.Now().Unix(),
		DeviceType: "mobile",
	}
	engine.AddFact(deviceInfo)

	// 插入另一个高额交易测试
	transaction3 := model.Transaction{
		ID:       103,
		UserID:   2,
		Amount:   12000,
		Currency: "CNY",
		Type:     "transfer",
		Status:   "pending",
		Location: "Beijing",
	}
	engine.AddFact(transaction3)
	engine.FireAllRules()
	fmt.Println()

	// ============ 场景 3: AggregateNode 演示 ============
	fmt.Println("📋 场景 3: AggregateNode 演示 - 多次失败登录检测")
	fmt.Println("----------------------------------------")

	// 重新创建引擎
	engine = ruleengine.New()
	engine.LoadRulesFromYAML("ruleengine/examples/smart_risk_control_rules.yaml")

	user3 := model.User{ID: 3, Name: "王五", Status: "normal", Level: "normal"}
	engine.AddFact(user3)

	fmt.Println("💡 模拟连续失败登录...")
	// 插入多次失败登录尝试
	for i := 1; i <= 4; i++ {
		failedLogin := model.LoginAttempt{
			ID:        200 + i,
			UserID:    3,
			Success:   false,
			Timestamp: time.Now().Unix() - int64(i*60), // 每次间隔1分钟
			IP:        fmt.Sprintf("192.168.1.%d", 100+i),
			Location:  "Beijing",
		}
		fmt.Printf("  添加第 %d 次失败登录尝试\n", i)
		engine.AddFact(failedLogin)

		// 每次插入后触发规则
		if i == 3 {
			fmt.Println("  💥 达到阈值（3次），应该触发聚合规则...")
		}
		engine.FireAllRules()
	}
	fmt.Println()

	// ============ 场景 4: ExistsNode 演示 ============
	fmt.Println("📋 场景 4: ExistsNode 演示 - VIP用户优惠检测")
	fmt.Println("----------------------------------------")

	// 重新创建引擎
	engine = ruleengine.New()
	engine.LoadRulesFromYAML("ruleengine/examples/smart_risk_control_rules.yaml")

	// 先插入VIP用户，但没有交易记录
	vipUser := model.User{ID: 4, Name: "赵六", Status: "normal", Level: "VIP"}
	engine.AddFact(vipUser)

	fmt.Println("💡 插入VIP用户但无交易记录（ExistsNode 不应触发）")
	engine.FireAllRules()

	// 现在添加交易记录
	fmt.Println("💡 添加交易记录，再次测试（ExistsNode 应该触发）...")
	completedTransaction := model.Transaction{
		ID:       104,
		UserID:   4,
		Amount:   1000,
		Currency: "CNY",
		Type:     "transfer",
		Status:   "completed", // 已完成的交易
		Location: "Beijing",
	}
	engine.AddFact(completedTransaction)
	engine.FireAllRules()
	fmt.Println()

	// ============ 场景 5: 组合场景演示 - 紧急情况优先级 ============
	fmt.Println("📋 场景 5: 紧急情况优先级演示")
	fmt.Println("同时插入多种风险情况，观察最高优先级的紧急规则先执行")
	fmt.Println("----------------------------------------")

	// 重新创建引擎
	engine = ruleengine.New()
	engine.LoadRulesFromYAML("ruleengine/examples/smart_risk_control_rules.yaml")

	// 插入用户和画像
	emergencyUser := model.User{ID: 5, Name: "紧急用户", Status: "suspicious", Level: "normal"}
	engine.AddFact(emergencyUser)

	emergencyProfile := model.UserProfile{
		UserID:          5,
		RegistrationAge: 25,
		RiskScore:       90,
		HomeLocation:    "Beijing",
	}
	engine.AddFact(emergencyProfile)

	// 插入大额异地交易（最高优先级）
	emergencyTransaction := model.Transaction{
		ID:       105,
		UserID:   5,
		Amount:   60000, // 超大额
		Currency: "CNY",
		Type:     "withdraw",
		Status:   "pending",
		Location: "Shanghai", // 异地（用户常住北京）
	}
	engine.AddFact(emergencyTransaction)

	fmt.Println("💥 插入超大额异地交易，应该最优先触发紧急规则...")
	engine.FireAllRules()

	fmt.Println("\n========================================")
	fmt.Println("🎉 智能风控系统演示完成！")
	fmt.Println("通过本演示，您可以看到：")
	fmt.Println("1. 组合冲突解决策略按 Salience -> Specificity -> LIFO 顺序工作")
	fmt.Println("2. NotNode 成功检测到缺失的条件（无可信设备）")
	fmt.Println("3. AggregateNode 正确统计和触发聚合条件（失败登录次数）")
	fmt.Println("4. ExistsNode 准确检测存在性条件（已完成交易记录）")
	fmt.Println("5. 高优先级规则在复杂场景中优先执行")
}
