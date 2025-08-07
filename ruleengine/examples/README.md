# 规则引擎演示程序

本目录包含了各种规则引擎功能的演示程序，展示了从基础功能到高级特性的完整应用。

## 📁 文件列表

### 🎯 核心演示

#### `conflict_resolution_demo.go` - 组合冲突解决策略演示
**重点功能**: 展示组合冲突解决策略的完整工作流程

**演示内容**:
- ✅ **Salience（优先级）**: 数字越大越优先执行
- ✅ **Specificity（特殊性）**: 条件越多越优先执行  
- ✅ **LIFO（后进先出）**: 相同优先级和特殊性时，后激活的规则先执行

**运行命令**:
```bash
go run ruleengine/examples/conflict_resolution_demo.go
```

**演示效果**:
```
🚀 组合冲突解决策略演示
========================================
📋 场景 1: Salience（优先级）演示
🔥 RULE FIRED: 高优先级规则 | Facts: [{1 张三 normal VIP }]
🔥 RULE FIRED: 低优先级但高特殊性规则 | Facts: [{1 张三 normal VIP } {1 张三 normal VIP }]
🔥 RULE FIRED: 中优先级规则 | Facts: [{1 张三 normal VIP }]
🔥 RULE FIRED: 基础规则 | Facts: [{1 张三 normal VIP }]

📋 场景 2: LIFO（后进先出）策略演示
📝 规则C执行 - 第三个添加的（应该最先执行）
📝 规则B执行 - 第二个添加的  
📝 规则A执行 - 第一个添加的
```

#### `advanced_nodes_demo.go` - 高级节点功能演示  
**重点功能**: 展示NotNode、AggregateNode、ExistsNode的应用

**演示内容**:
- 🔍 **NotNode概念**: 检测缺失条件的逻辑
- 📊 **AggregateNode概念**: 聚合统计功能
- ✅ **ExistsNode概念**: 存在性检查
- 🎯 **复杂场景**: 多规则冲突解决的实际应用

**运行命令**:
```bash
go run ruleengine/examples/advanced_nodes_demo.go
```

#### `smart_risk_control_demo.go` - 智能风控系统演示
**重点功能**: 完整的业务场景应用示例

**演示内容**:
- 🚨 紧急风险检测（最高优先级）
- ⚠️ 聚合风险分析
- 🔍 安全设备验证（NotNode应用）
- 🎁 用户优惠检测（ExistsNode应用）
- 💰 基础交易监控

**运行命令**:
```bash
go run ruleengine/examples/smart_risk_control_demo.go
```

### 📋 配置文件

#### `smart_risk_control_rules.yaml` - 智能风控规则配置
完整的YAML规则配置示例，包含：

```yaml
rules:
  # 最高优先级：紧急安全检测（Salience: 100）
  - name: "紧急_大额异地交易检测"
    salience: 100
    when:
      - type: "fact"
        fact_type: "Transaction"
        field: "Amount"
        operator: ">"
        value: 50000
    then:
      type: "log"
      message: "🚨 紧急警报：检测到大额异地交易！"

  # 聚合检测（Salience: 80）
  - name: "聚合_多次失败登录检测"
    salience: 80
    when:
      - type: "aggregate"
        fact_type: "LoginAttempt"
        group_by: "UserID"
        threshold: 3
    then:
      type: "log"
      message: "⚠️ 检测到多次失败登录"

  # NOT逻辑检测（Salience: 50）
  - name: "NOT_未绑定可信设备的高风险交易"
    salience: 50
    when:
      - type: "fact"
        fact_type: "Transaction"
        field: "Amount"
        operator: ">"
        value: 10000
      - type: "not"
        fact_type: "DeviceInfo"
        field: "Trusted"
        operator: "=="
        value: true
    then:
      type: "log"
      message: "🔍 高额交易未在可信设备上操作"
```

## 🚀 快速开始

### 1. 运行基础演示
```bash
# 克隆项目
cd /path/to/code_for_article

# 运行冲突解决策略演示
go run ruleengine/examples/conflict_resolution_demo.go

# 运行高级节点演示
go run ruleengine/examples/advanced_nodes_demo.go
```

### 2. 理解输出
每个演示都会清晰地展示：
- 📋 **场景描述**: 当前测试的具体功能
- 🔥 **规则触发**: 显示哪个规则被激活及其处理的事实
- ⭐ **执行顺序**: 演示冲突解决策略的实际效果
- 📚 **总结说明**: 解释关键概念和实现细节

### 3. 自定义规则
参考 `smart_risk_control_rules.yaml` 创建自己的规则配置：

```yaml
rules:
  - name: "我的自定义规则"
    salience: 80  # 设置优先级
    when:
      - type: "fact"
        fact_type: "MyEntity"
        field: "MyField"
        operator: "=="
        value: "MyValue"
    then:
      type: "log"
      message: "我的规则被触发了！"
```

## 🎯 核心特性展示

### ✅ 已实现功能

1. **组合冲突解决策略**
   - Salience（优先级）排序
   - Specificity（特殊性）排序
   - LIFO（后进先出）最终排序

2. **基础规则引擎**
   - AlphaNode（单事实过滤）
   - BetaNode（多事实连接）
   - TerminalNode（规则激活）

3. **智能议程管理**
   - 动态规则排序
   - 可插拔冲突解决策略
   - 实时优先级调整

### 🚧 待完善功能

1. **高级节点类型**
   - NotNode（需要Builder层完善）
   - AggregateNode（需要Builder层完善）
   - ExistsNode（需要Builder层完善）

2. **增强功能**
   - 动态规则热加载
   - 规则性能监控
   - 更复杂的条件表达式

## 📖 学习路径

推荐按以下顺序学习：

1. **入门**: 运行 `conflict_resolution_demo.go` 理解基础概念
2. **进阶**: 运行 `advanced_nodes_demo.go` 了解高级特性
3. **实战**: 研究 `smart_risk_control_demo.go` 学习业务应用
4. **深入**: 阅读源码理解Rete算法实现

## 🤝 贡献指南

欢迎提交更多演示场景和改进建议！

### 添加新演示的步骤：
1. 创建新的 `*_demo.go` 文件
2. 如需要，创建对应的 YAML 配置文件
3. 更新本 README 文件
4. 确保代码有清晰的注释和输出说明

---

**💡 提示**: 所有演示都包含详细的输出说明，运行时请仔细观察控制台输出中的规则执行顺序和逻辑说明。