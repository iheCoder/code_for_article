# Rete 规则引擎演示案例

本目录包含多个完整的演示案例，展示了 Go 实现的 Rete 规则引擎的各种特性和应用场景。

## 📋 演示案例列表

### 1. 基础优惠券场景 (`coupon_engine_demo.go`)

**特性展示**：
- ✅ 基础的 Alpha/Beta 网络构建
- ✅ 跨事实关联 (User ↔ Cart)
- ✅ 增量匹配和记忆化
- ✅ 事实更新与撤回

**运行命令**：
```bash
go run ./ruleengine/examples/coupon_engine_demo.go
```

**核心规则**：
- VIP 用户 + 大额购物车 (>100) → VIP大额订单优惠
- 超大额购物车 (>500) → 高价值购物车优惠

### 2. 高级优惠券场景 (`advanced_coupon_demo.go`)

**特性展示**：
- ✅ 复杂的多层 Rete 网络
- ✅ NotNode 否定逻辑（无活跃账户检测）
- ✅ ExistsNode 存在性检查（高余额账户）
- ✅ AggregateNode 聚合计数（多购物车统计）
- ✅ 多规则并行执行
- ✅ 复杂的网络拓扑结构

**运行命令**：
```bash
go run ./ruleengine/examples/advanced_coupon_demo.go
```

**核心规则**：
- VIP + 大额购物车 → 15%折扣
- VIP + 高余额账户 → 白金会员升级
- 普通用户 + 无活跃账户 → 开户建议
- VIP + EXISTS(高余额) → 超级VIP福利
- 多购物车聚合 → 满减优惠券
- 超大额订单 → 人工客服跟进

### 3. 反欺诈检测场景 (`simple_demo.go`)

**特性展示**：
- ✅ YAML DSL 规则加载
- ✅ 简化的规则定义语法
- ✅ 多种事实类型处理
- ✅ 动态规则匹配

**运行命令**：
```bash
go run ./ruleengine/examples/simple_demo.go
```

**核心规则**：
- 锁定用户检测
- 大额交易监控 (>10000)
- 失败登录检测
- 可疑用户标记

### 4. 完整反欺诈场景 (`fraud_detection_demo.go`)

**特性展示**：
- ✅ 复杂的 YAML 规则定义
- ✅ 多实体关联检测
- ✅ 高级规则组合
- ✅ 企业级反欺诈逻辑

**运行命令**：
```bash
go run ./ruleengine/examples/fraud_detection_demo.go
```

**核心规则**：
- 锁定账户大额交易预警
- 无效账户交易检测 (NOT逻辑)
- 失败登录用户交易监控 (EXISTS逻辑)
- 多次失败登录聚合检测 (AGGREGATE逻辑)

## 🏗️ 规则定义文件

### YAML 规则文件

- `simple_fraud_rules.yaml` - 简化的反欺诈规则
- `fraud_rules.yaml` - 完整的反欺诈规则集

### 示例规则结构

```yaml
rules:
  - name: "规则名称"
    description: "规则描述"
    salience: 10  # 优先级
    when:
      - type: "fact"
        fact_type: "User"
        field: "Status"
        operator: "=="
        value: "locked"
    then:
      type: "log"
      message: "检测到锁定用户"
```

## 🚀 快速开始

### 运行所有演示

```bash
# 基础优惠券演示
go run ./ruleengine/examples/coupon_engine_demo.go

# 高级优惠券演示  
go run ./ruleengine/examples/advanced_coupon_demo.go

# 反欺诈演示
go run ./ruleengine/examples/simple_demo.go

# 完整反欺诈演示
go run ./ruleengine/examples/fraud_detection_demo.go
```

### 检查代码质量

```bash
# 代码格式化
go fmt ./ruleengine/...

# 编译检查
go build ./ruleengine/...

# 运行测试（如果有）
go test ./ruleengine/...
```

## 📊 特性覆盖矩阵

| 特性 | 基础优惠券 | 高级优惠券 | 简单反欺诈 | 完整反欺诈 |
|------|-----------|-----------|-----------|-----------|
| AlphaNode | ✅ | ✅ | ✅ | ✅ |
| BetaNode | ✅ | ✅ | ❌ | ✅ |
| NotNode | ❌ | ✅ | ❌ | ✅ |
| ExistsNode | ❌ | ✅ | ❌ | ✅ |
| AggregateNode | ❌ | ✅ | ❌ | ✅ |
| YAML DSL | ❌ | ❌ | ✅ | ✅ |
| 撤回机制 | ✅ | ✅ | ✅ | ✅ |
| 复杂网络 | ❌ | ✅ | ❌ | ✅ |

## 💡 学习路径建议

1. **入门**: 从 `coupon_engine_demo.go` 开始，理解基础的 Alpha/Beta 网络
2. **进阶**: 运行 `simple_demo.go`，学习 YAML DSL 的使用
3. **深入**: 探索 `advanced_coupon_demo.go`，理解复杂的网络拓扑
4. **应用**: 研究 `fraud_detection_demo.go`，了解企业级应用场景

## 🔧 扩展指南

要创建自己的规则场景：

1. 定义业务实体（实现 `model.Fact` 接口）
2. 编写规则逻辑（代码或 YAML）
3. 构建 Rete 网络（手动或通过 Builder）
4. 测试和验证规则行为

参考现有演示代码的结构和模式，可以快速搭建适合你业务场景的规则引擎！