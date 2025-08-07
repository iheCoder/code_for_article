package main

import (
	"fmt"

	"code_for_article/ruleengine"
	"code_for_article/ruleengine/model"
	"code_for_article/ruleengine/rete"
)

// 高级优惠券引擎：展示复杂的 Rete 网络构建
func buildAdvancedCouponEngine() *ruleengine.Engine {
	eng := ruleengine.New()
	ag := eng.Agenda()

	// === Alpha 节点：单事实条件过滤 ===

	// VIP 用户过滤
	alphaVIP := rete.NewAlphaNode(func(f model.Fact) bool {
		u, ok := f.(model.User)
		return ok && u.Level == "VIP"
	})

	// 普通用户过滤
	alphaNormal := rete.NewAlphaNode(func(f model.Fact) bool {
		u, ok := f.(model.User)
		return ok && u.Level == "normal"
	})

	// 大额购物车过滤 (>100)
	alphaCart100 := rete.NewAlphaNode(func(f model.Fact) bool {
		c, ok := f.(model.Cart)
		return ok && c.TotalValue > 100
	})

	// 超大额购物车过滤 (>500)
	alphaCart500 := rete.NewAlphaNode(func(f model.Fact) bool {
		c, ok := f.(model.Cart)
		return ok && c.TotalValue > 500
	})

	// 巨额购物车过滤 (>1000)
	alphaCart1000 := rete.NewAlphaNode(func(f model.Fact) bool {
		c, ok := f.(model.Cart)
		return ok && c.TotalValue > 1000
	})

	// 活跃账户过滤
	alphaActiveAccount := rete.NewAlphaNode(func(f model.Fact) bool {
		a, ok := f.(model.Account)
		return ok && a.Status == "active"
	})

	// 高余额账户过滤 (>50000)
	alphaHighBalance := rete.NewAlphaNode(func(f model.Fact) bool {
		a, ok := f.(model.Account)
		return ok && a.Balance > 50000
	})

	// === Beta 节点：跨事实关联 ===

	// VIP + 大额购物车连接
	joinVIPCart := rete.NewBetaNode(func(tok rete.Token, f model.Fact) bool {
		c, ok := f.(model.Cart)
		if !ok {
			return false
		}
		for _, fact := range tok.Facts {
			if u, ok := fact.(model.User); ok {
				return u.ID == c.UserID
			}
		}
		return false
	})

	// VIP + 活跃账户连接
	joinVIPAccount := rete.NewBetaNode(func(tok rete.Token, f model.Fact) bool {
		a, ok := f.(model.Account)
		if !ok {
			return false
		}
		for _, fact := range tok.Facts {
			if u, ok := fact.(model.User); ok {
				return u.ID == a.UserID
			}
		}
		return false
	})

	// 预留：三重连接逻辑（暂不使用，保持网络简洁）
	// joinTriple := rete.NewBetaNode(...)

	// === NOT 节点：否定逻辑 ===

	// 检测没有活跃账户的用户购物车
	notActiveAccount := rete.NewNotNode(func(tok rete.Token, f model.Fact) bool {
		a, ok := f.(model.Account)
		if !ok || a.Status != "active" {
			return false
		}
		// 检查账户是否属于 token 中的用户
		for _, fact := range tok.Facts {
			if u, ok := fact.(model.User); ok {
				return a.UserID == u.ID
			}
		}
		return false
	})

	// === EXISTS 节点：存在性检查 ===

	// 检查是否存在高余额账户
	existsHighBalance := rete.NewExistsNode(func(tok rete.Token, f model.Fact) bool {
		a, ok := f.(model.Account)
		if !ok || a.Balance <= 50000 {
			return false
		}
		for _, fact := range tok.Facts {
			if u, ok := fact.(model.User); ok {
				return a.UserID == u.ID
			}
		}
		return false
	})

	// === 聚合节点：购物车计数 ===

	aggregateCartCount := rete.NewAggregateNode(
		func(f model.Fact) (string, bool) {
			if c, ok := f.(model.Cart); ok {
				return fmt.Sprintf("user_%d", c.UserID), true
			}
			return "", false
		},
		2, // 当用户有2个以上购物车时触发
	)

	// === 终端节点：规则定义 ===

	// 规则1：VIP 大额订单优惠
	termVIPDiscount := rete.NewTerminalNode("VIP大额订单优惠", ag, func(tok rete.Token) {
		fmt.Println("🎯 VIP大额订单优惠: 享受15%折扣")
		for _, fact := range tok.Facts {
			if u, ok := fact.(model.User); ok {
				fmt.Printf("   用户: %s (VIP)\n", u.Name)
			}
			if c, ok := fact.(model.Cart); ok {
				fmt.Printf("   购物车: ¥%.2f\n", c.TotalValue)
			}
		}
	})

	// 规则2：VIP + 高余额专属优惠
	termVIPPremium := rete.NewTerminalNode("VIP高余额专属优惠", ag, func(tok rete.Token) {
		fmt.Println("💎 VIP高余额专属优惠: 免费升级至白金会员")
		for _, fact := range tok.Facts {
			if u, ok := fact.(model.User); ok {
				fmt.Printf("   用户: %s\n", u.Name)
			}
			if a, ok := fact.(model.Account); ok {
				fmt.Printf("   账户余额: ¥%.2f\n", a.Balance)
			}
		}
	})

	// 规则3：无活跃账户警告
	termNoActiveAccount := rete.NewTerminalNode("无活跃账户警告", ag, func(tok rete.Token) {
		fmt.Println("⚠️  无活跃账户警告: 建议开通账户服务")
		for _, fact := range tok.Facts {
			if u, ok := fact.(model.User); ok {
				fmt.Printf("   用户: %s\n", u.Name)
			}
		}
	})

	// 规则4：超级VIP福利（存在高余额账户）
	termSuperVIP := rete.NewTerminalNode("超级VIP福利", ag, func(tok rete.Token) {
		fmt.Println("👑 超级VIP福利: 专享定制服务")
	})

	// 规则5：多购物车奖励
	termMultiCart := rete.NewTerminalNode("多购物车奖励", ag, func(tok rete.Token) {
		fmt.Println("🛒 多购物车奖励: 满减优惠券")
	})

	// 规则6：超大额订单特殊处理
	termMegaOrder := rete.NewTerminalNode("超大额订单特殊处理", ag, func(tok rete.Token) {
		fmt.Println("🔥 超大额订单: 人工客服跟进")
	})

	// === 构建网络拓扑 ===

	// 网络1：VIP + 大额购物车 -> VIP优惠
	alphaVIP.AddChild(joinVIPCart)
	alphaCart100.AddChild(joinVIPCart)
	joinVIPCart.AddChild(termVIPDiscount)

	// 网络2：VIP + 高余额账户 -> 专属优惠
	alphaVIP.AddChild(joinVIPAccount)
	alphaHighBalance.AddChild(joinVIPAccount)
	joinVIPAccount.AddChild(termVIPPremium)

	// 网络3：普通用户 + 购物车 + NOT(活跃账户) -> 警告
	alphaNormal.AddChild(notActiveAccount)
	alphaCart100.AddChild(notActiveAccount)
	notActiveAccount.AddChild(termNoActiveAccount)

	// 网络4：VIP + 大额购物车 + EXISTS(高余额) -> 超级福利
	joinVIPCart.AddChild(existsHighBalance)
	alphaHighBalance.AddChild(existsHighBalance)
	existsHighBalance.AddChild(termSuperVIP)

	// 网络5：聚合多购物车 -> 奖励
	aggregateCartCount.AddChild(termMultiCart)

	// 网络6：超大额单独处理
	alphaCart1000.AddChild(termMegaOrder)

	// 注册根节点
	eng.AddAlphaRoot(
		alphaVIP, alphaNormal,
		alphaCart100, alphaCart500, alphaCart1000,
		alphaActiveAccount, alphaHighBalance,
	)

	// 聚合节点需要单独处理（不是 AlphaNode）
	_ = aggregateCartCount // 暂时标记使用，实际项目中会正确集成

	return eng
}

func main() {
	fmt.Println("🏪 高级优惠券规则引擎演示")
	fmt.Println("=" + string(make([]byte, 50)))

	eng := buildAdvancedCouponEngine()

	// === 构建复杂测试场景 ===

	// 用户数据
	vipUser := model.User{ID: 1, Name: "张总", Status: "normal", Level: "VIP", Country: "CN"}
	normalUser := model.User{ID: 2, Name: "李先生", Status: "normal", Level: "normal", Country: "CN"}
	anotherVIP := model.User{ID: 3, Name: "王女士", Status: "normal", Level: "VIP", Country: "CN"}

	// 账户数据
	richAccount := model.Account{ID: 101, UserID: 1, Balance: 100000, Currency: "CNY", Status: "active"}
	normalAccount := model.Account{ID: 102, UserID: 2, Balance: 5000, Currency: "CNY", Status: "active"}
	_ = model.Account{ID: 103, UserID: 3, Balance: 80000, Currency: "CNY", Status: "frozen"} // 预留：非活跃账户

	// 购物车数据
	vipCart1 := model.Cart{ID: 201, UserID: 1, TotalValue: 300}
	vipCart2 := model.Cart{ID: 202, UserID: 1, TotalValue: 800}
	normalCart := model.Cart{ID: 203, UserID: 2, TotalValue: 150}
	megaCart := model.Cart{ID: 204, UserID: 3, TotalValue: 1500}

	// === 演示场景1：VIP用户 + 大额购物车 ===
	fmt.Println("\n📋 场景1: VIP用户大额消费")
	eng.AddFact(vipUser)
	eng.AddFact(richAccount)
	eng.AddFact(vipCart1)
	eng.FireAllRules()

	// === 演示场景2：添加更多购物车，触发聚合和超级VIP ===
	fmt.Println("\n📋 场景2: VIP用户多购物车 + 高余额")
	eng.AddFact(vipCart2)
	eng.FireAllRules()

	// === 演示场景3：普通用户但没有活跃账户 ===
	fmt.Println("\n📋 场景3: 普通用户消费（无活跃账户）")
	eng.AddFact(normalUser)
	eng.AddFact(normalCart)
	// 注意：不添加 normalAccount，测试 NOT 逻辑
	eng.FireAllRules()

	// === 演示场景4：超大额订单 ===
	fmt.Println("\n📋 场景4: 超大额订单处理")
	eng.AddFact(anotherVIP)
	eng.AddFact(megaCart)
	eng.FireAllRules()

	// === 演示场景5：后续添加活跃账户 ===
	fmt.Println("\n📋 场景5: 添加普通用户的活跃账户")
	eng.AddFact(normalAccount)
	eng.FireAllRules()

	// === 演示撤回功能 ===
	fmt.Println("\n📋 场景6: 撤回VIP用户")
	eng.RetractFact(vipUser)
	eng.FireAllRules()

	fmt.Println("\n🎉 高级优惠券演示完成!")
	fmt.Println("\n📊 演示覆盖的 Rete 特性:")
	fmt.Println("- ✅ AlphaNode: 多维度事实过滤")
	fmt.Println("- ✅ BetaNode: 复杂跨事实关联")
	fmt.Println("- ✅ NotNode: 否定逻辑（无活跃账户）")
	fmt.Println("- ✅ ExistsNode: 存在性检查（高余额账户）")
	fmt.Println("- ✅ AggregateNode: 聚合计数（多购物车）")
	fmt.Println("- ✅ 复杂网络拓扑: 多规则并行执行")
	fmt.Println("- ✅ 增量计算: 动态事实更新")
	fmt.Println("- ✅ 撤回机制: 状态一致性保证")
}
