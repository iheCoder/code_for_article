package main

import (
	"fmt"
	"log"

	"code_for_article/ruleengine"
	"code_for_article/ruleengine/model"
)

func main() {
	fmt.Println("🚀 简化版规则引擎演示")
	fmt.Println("=" + string(make([]byte, 40)))

	// 1. 创建规则引擎
	engine := ruleengine.New()

	// 2. 从简化的 YAML 文件加载规则
	fmt.Println("📖 加载规则...")
	if err := engine.LoadRulesFromYAML("ruleengine/examples/simple_fraud_rules.yaml"); err != nil {
		log.Fatalf("加载规则失败: %v", err)
	}
	fmt.Println("✅ 规则加载完成")

	// 3. 演示场景1: 插入正常用户
	fmt.Println("\n📋 场景1: 正常用户")
	normalUser := model.User{ID: 1, Name: "张三", Status: "normal", Level: "normal"}
	engine.AddFact(normalUser)
	engine.FireAllRules()

	// 4. 演示场景2: 插入锁定用户
	fmt.Println("\n📋 场景2: 锁定用户")
	lockedUser := model.User{ID: 2, Name: "李四", Status: "locked", Level: "VIP"}
	engine.AddFact(lockedUser)
	engine.FireAllRules()

	// 5. 演示场景3: 插入可疑用户
	fmt.Println("\n📋 场景3: 可疑用户")
	suspiciousUser := model.User{ID: 3, Name: "王五", Status: "suspicious", Level: "normal"}
	engine.AddFact(suspiciousUser)
	engine.FireAllRules()

	// 6. 演示场景4: 插入小额交易
	fmt.Println("\n📋 场景4: 小额交易")
	smallTransaction := model.Transaction{ID: 201, UserID: 1, Amount: 2000, Currency: "CNY", Type: "transfer"}
	engine.AddFact(smallTransaction)
	engine.FireAllRules()

	// 7. 演示场景5: 插入大额交易
	fmt.Println("\n📋 场景5: 大额交易")
	largeTransaction := model.Transaction{ID: 202, UserID: 2, Amount: 15000, Currency: "CNY", Type: "withdraw"}
	engine.AddFact(largeTransaction)
	engine.FireAllRules()

	// 8. 演示场景6: 插入成功登录
	fmt.Println("\n📋 场景6: 成功登录")
	successLogin := model.LoginAttempt{ID: 301, UserID: 1, Success: true, IP: "192.168.1.100"}
	engine.AddFact(successLogin)
	engine.FireAllRules()

	// 9. 演示场景7: 插入失败登录
	fmt.Println("\n📋 场景7: 失败登录")
	failedLogin := model.LoginAttempt{ID: 302, UserID: 2, Success: false, IP: "192.168.1.100"}
	engine.AddFact(failedLogin)
	engine.FireAllRules()

	// 10. 演示撤回功能
	fmt.Println("\n📋 场景8: 撤回演示")
	fmt.Println("撤回锁定用户...")
	engine.RetractFact(lockedUser)
	engine.FireAllRules()

	fmt.Println("\n🎉 演示完成!")
	fmt.Println("\n📊 演示总结:")
	fmt.Println("- ✅ 成功加载并执行 YAML 定义的规则")
	fmt.Println("- ✅ 展示了 AlphaNode 的条件过滤功能")
	fmt.Println("- ✅ 展示了规则引擎的增量匹配")
	fmt.Println("- ✅ 展示了事实撤回机制")
}
