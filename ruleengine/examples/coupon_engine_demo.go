package main

import (
	"fmt"

	"code_for_article/ruleengine"
	"code_for_article/ruleengine/model"
	"code_for_article/ruleengine/rete"
)

func buildCouponEngine() *ruleengine.Engine {
	eng := ruleengine.New()
	ag := eng.Agenda()

	// 条件节点
	alphaVIP := rete.NewAlphaNode(func(f model.Fact) bool {
		u, ok := f.(model.User)
		return ok && u.Level == "VIP"
	})

	alphaCart100 := rete.NewAlphaNode(func(f model.Fact) bool {
		c, ok := f.(model.Cart)
		return ok && c.TotalValue > 100
	})

	alphaCart500 := rete.NewAlphaNode(func(f model.Fact) bool {
		c, ok := f.(model.Cart)
		return ok && c.TotalValue > 500
	})

	// Join 用户ID == Cart.UserID
	join := rete.NewBetaNode(func(tok rete.Token, f model.Fact) bool {
		c, ok := f.(model.Cart)
		if !ok {
			return false
		}
		// Token 最后一个肯定是 User
		for _, fact := range tok.Facts {
			if u, ok := fact.(model.User); ok {
				return u.ID == c.UserID
			}
		}
		return false
	})

	// 终端节点
	termRule1 := rete.NewTerminalNode("VIP大额订单优惠", ag, func(tok rete.Token) {
		fmt.Println(">> 执行优惠券逻辑: VIP 大额订单优惠")
	})

	termRule2 := rete.NewTerminalNode("高价值购物车优惠", ag, func(tok rete.Token) {
		fmt.Println(">> 执行优惠券逻辑: 高价值购物车优惠")
	})

	// 构建网络
	alphaVIP.AddChild(join)
	alphaCart100.AddChild(join)
	join.AddChild(termRule1)
	alphaCart500.AddChild(termRule2)

	eng.AddAlphaRoot(alphaVIP, alphaCart100, alphaCart500)

	return eng
}

func main() {
	eng := buildCouponEngine()

	fmt.Println("=== 第一阶段：插入 VIP 用户 ===")
	user := model.User{ID: 1, Level: "VIP"}
	eng.Assert(user)

	fmt.Println("=== 第二阶段：插入购物车 (150) ===")
	cart := model.Cart{ID: 101, UserID: 1, TotalValue: 150}
	eng.Assert(cart)

	fmt.Println("=== 触发规则 ===")
	eng.FireAllRules()

	fmt.Println("\n=== 第三阶段：更新购物车 (600) ===")
	cartUpdated := model.Cart{ID: 101, UserID: 1, TotalValue: 600}
	eng.Assert(cartUpdated) // 简化：直接 assert 新事实

	fmt.Println("=== 触发规则 ===")
	eng.FireAllRules()
}
