package main

import (
	"fmt"
	"log"
	"time"

	"code_for_article/ruleengine"
	"code_for_article/ruleengine/model"
)

func main() {
	fmt.Println("🏦 反欺诈规则引擎演示")
	fmt.Println("=" + string(make([]byte, 50)))

	// 1. 创建规则引擎
	engine := ruleengine.New()

	// 2. 从 YAML 文件加载反欺诈规则
	fmt.Println("📖 加载反欺诈规则...")
	if err := engine.LoadRulesFromYAML("ruleengine/examples/fraud_rules.yaml"); err != nil {
		log.Fatalf("加载规则失败: %v", err)
	}
	fmt.Println("✅ 规则加载完成")

	// 3. 构建测试场景数据
	fmt.Println("\n🏗️  构建测试场景...")

	// 用户数据
	normalUser := model.User{ID: 1, Name: "张三", Status: "normal", Level: "normal", Country: "CN"}
	lockedUser := model.User{ID: 2, Name: "李四", Status: "locked", Level: "VIP", Country: "CN"}
	suspiciousUser := model.User{ID: 3, Name: "王五", Status: "suspicious", Level: "normal", Country: "US"}

	// 账户数据
	activeAccount1 := model.Account{ID: 101, UserID: 1, Balance: 50000, Currency: "CNY", Status: "active"}
	activeAccount2 := model.Account{ID: 102, UserID: 2, Balance: 100000, Currency: "CNY", Status: "active"}
	frozenAccount := model.Account{ID: 103, UserID: 3, Balance: 25000, Currency: "USD", Status: "frozen"}

	// 交易数据
	normalTransaction := model.Transaction{ID: 201, UserID: 1, Amount: 2000, Currency: "CNY", Type: "transfer", Status: "pending"}
	largeTransaction := model.Transaction{ID: 202, UserID: 2, Amount: 15000, Currency: "CNY", Type: "withdraw", Status: "pending"}
	suspiciousTransaction := model.Transaction{ID: 203, UserID: 3, Amount: 8000, Currency: "USD", Type: "transfer", Status: "pending"}
	invalidTransaction := model.Transaction{ID: 204, UserID: 999, Amount: 5000, Currency: "CNY", Type: "deposit", Status: "pending"} // 无对应账户

	// 登录尝试数据（模拟失败登录）
	now := time.Now().Unix()
	failedLogin1 := model.LoginAttempt{ID: 301, UserID: 3, Success: false, Timestamp: now - 300, IP: "192.168.1.100", Location: "Beijing"}
	failedLogin2 := model.LoginAttempt{ID: 302, UserID: 3, Success: false, Timestamp: now - 200, IP: "192.168.1.100", Location: "Beijing"}
	failedLogin3 := model.LoginAttempt{ID: 303, UserID: 3, Success: false, Timestamp: now - 100, IP: "192.168.1.100", Location: "Beijing"}
	failedLogin4 := model.LoginAttempt{ID: 304, UserID: 3, Success: false, Timestamp: now, IP: "192.168.1.100", Location: "Beijing"}

	// 4. 演示场景1: 正常用户交易（不应触发任何规则）
	fmt.Println("\n📋 场景1: 正常用户交易")
	engine.AddFact(normalUser)
	engine.AddFact(activeAccount1)
	engine.AddFact(normalTransaction)
	engine.FireAllRules()

	// 5. 演示场景2: 锁定用户大额交易（应触发规则1）
	fmt.Println("\n📋 场景2: 锁定用户大额交易")
	engine.AddFact(lockedUser)
	engine.AddFact(activeAccount2)
	engine.AddFact(largeTransaction)
	engine.FireAllRules()

	// 6. 演示场景3: 无效账户交易（应触发规则2）
	fmt.Println("\n📋 场景3: 无效账户交易检测")
	engine.AddFact(invalidTransaction)
	engine.FireAllRules()

	// 7. 演示场景4: 有失败登录记录的用户大额交易（应触发规则3）
	fmt.Println("\n📋 场景4: 失败登录用户交易监控")
	engine.AddFact(suspiciousUser)
	engine.AddFact(frozenAccount)
	engine.AddFact(failedLogin1) // 添加一次失败登录
	engine.AddFact(suspiciousTransaction)
	engine.FireAllRules()

	// 8. 演示场景5: 多次失败登录聚合检测（应触发规则4）
	fmt.Println("\n📋 场景5: 多次失败登录聚合检测")
	engine.AddFact(failedLogin2) // 第2次失败
	engine.AddFact(failedLogin3) // 第3次失败 - 应该触发聚合规则
	engine.FireAllRules()

	engine.AddFact(failedLogin4) // 第4次失败 - 再次触发
	engine.FireAllRules()

	// 9. 演示撤回功能
	fmt.Println("\n📋 场景6: 撤回演示")
	fmt.Println("撤回锁定用户...")
	engine.RetractFact(lockedUser)
	engine.FireAllRules() // 应该不再有相关规则触发

	fmt.Println("\n🎉 反欺诈演示完成!")
	fmt.Println("\n📊 演示总结:")
	fmt.Println("- ✅ 展示了 AlphaNode 的单条件过滤")
	fmt.Println("- ✅ 展示了 BetaNode 的跨事实关联")
	fmt.Println("- ✅ 展示了 NotNode 的否定逻辑")
	fmt.Println("- ✅ 展示了 ExistsNode 的存在性检查")
	fmt.Println("- ✅ 展示了 AggregateNode 的聚合计数")
	fmt.Println("- ✅ 展示了事实撤回机制")
	fmt.Println("- ✅ 展示了 YAML DSL 规则定义")
}
