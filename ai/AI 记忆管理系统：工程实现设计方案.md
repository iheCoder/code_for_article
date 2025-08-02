# AI 记忆管理系统：工程实现设计方案

本文档为《从“健忘”到“懂我”：构建新一代AI记忆系统》中所述理念的详细工程实现方案。它将聚焦于技术选型、模块设计、数据流转和核心算法，为开发团队提供清晰的落地指引。

## 1. 系统架构与技术选型

为实现分层记忆与读写分离的设计理念，我们将记忆系统构建为一套独立的微服务，通过 RESTful API 与上层应用（如聊天服务、任务助手）交互。

| **记忆层级**                  | **核心存储**                                           | **备选/补充**              | **技术选型 rationale**                                                                       |
|---------------------------|----------------------------------------------------|------------------------|------------------------------------------------------------------------------------------|
| **工作记忆 (Working Memory)** | **Redis**                                          | N/A                    | 内存数据库，提供微秒级读写延迟，完美匹配高频、易失的会话缓冲需求。其 `LIST` 和 `HASH` 数据结构天然契合滚动窗口与锚点实现。                    |
| **偏好/画像 (Preference)**    | **PostgreSQL / MySQL**                             | MongoDB                | 关系型数据库能强力约束画像数据的 Schema，保证数据一致性与可解释性。使用 JSONB 字段可兼顾结构化查询与非固定偏好的灵活性。                      |
| **情节记忆 (Episodic)**       | **Vector DB (Pinecone / Milvus)** + **PostgreSQL** | OpenSearch (with k-NN) | 向量数据库是实现高效语义检索的核心。将元数据与原始文本存储在 PostgreSQL 中，可实现“元数据过滤 + 向量搜索”的混合检索策略，极大提升精度和效率。          |
| **语义知识 (Semantic KB)**    | **Vector DB (Pinecone / Milvus)** + **PostgreSQL** | Elasticsearch          | 同情节记忆，RAG 的核心是向量检索。PostgreSQL 用于存储文档的结构化信息（版本、来源、层级），保证知识的可追溯性和版本管理。                     |
| **服务间通信**                 | **RabbitMQ / Kafka**                               | gRPC                   | 采用消息队列实现写操作的异步化，避免阻塞主流程的用户响应。                                                            |
| **核心计算**                  | **Python Service (FastAPI / Flask)**               | Go                     | Python 拥有最丰富的 AI/ML 生态（Hugging Face, Scikit-learn 等），是实现 embedding、reranking、NER 等任务的首选。 |

**整体架构图:**

```mermaid
graph TD
    subgraph AI Application Layer
        direction LR
        App[AI 应用前端] --> GW[API 网关]
    end

    subgraph Memory System Microservice
        direction TB
        GW -->|REST API| ReadPath[读取路径 Read Path]
        GW -->|Async via MQ| WritePath[写入路径 Write Path]

        subgraph Read Path Components
            ReadPath --> Intent[1. 意图分析与路由]
            Intent --> ParallelRet[2. 并行检索]
            ParallelRet -->|Working| Redis[(Redis)]
            ParallelRet -->|Episodic| PG_E[PostgreSQL] & Pinecone_E[Pinecone]
            ParallelRet -->|Semantic| PG_S[PostgreSQL] & Pinecone_S[Pinecone]
            ParallelRet -->|Preference| PG_P[PostgreSQL]

            subgraph Fusion
                Redis --> FusionEngine[3. 上下文融合器]
                PG_E --> FusionEngine
                Pinecone_E --> FusionEngine
                PG_S --> FusionEngine
                Pinecone_S --> FusionEngine
                PG_P --> FusionEngine
            end
            FusionEngine -->|Context Prompt| LLM
        end

        subgraph Write Path Components Async Workers
            MQ[Message Queue] --> WriteGating[1. 写入门控]
            WriteGating -->|Pass| Store[2. 存储/更新]
            Store --> Redis
            Store --> Pinecone_E & PG_E
            Store --> Pinecone_S & PG_S
            Store --> PG_P
        end
    end

    LLM[(LLM Service)] --> GW
    classDef db fill: #D6EAF8, stroke: #5DADE2, stroke-width: 2px;
    class Redis, PG_E, Pinecone_E, PG_S, Pinecone_S, PG_P db;
```

## 2. 记忆读取路径 (Read Path)：完整流程

当用户请求到达时，读取路径被同步调用，其核心目标是在预算内（延迟、Token 数）构建最优质的 Prompt。

### Step 1: 意图分析与路由 (Intent Analysis & Routing)

接收到用户请求后，采用**分层意图分析**策略，结合了低成本的初步筛选和强大的大语言模型深度解析，生成一份指导后续所有检索行为的“检索计划”，通过
`target_memories`字段精确控制检索范围。

以下是典型的意图到模块的映射关系，由LLM在分析时决定：

- **查询“我们上次聊到哪了？”或“总结一下”** -> `target_memories: ["working"]` -> 只会从Redis中检索对话历史、摘要和锚点。
- **查询“我们关于苍穹项目的那个决策是什么？”** -> `target_memories: ["episodic", "working"]` ->
  重点检索情节记忆中的“决策”事件，并结合工作记忆的上下文。
- **查询“给我解释一下什么是RRF算法”** -> `target_memories: ["semantic"]` -> 只在语义知识库中进行查找。
- **复杂的、开放式查询** -> `target_memories: ["episodic", "semantic", "preference", "working"]` -> 执行最全面的检索。

#### Layer 0：**缓存层 (Cache)**

使用 Redis `HASH` 存储。键为 `retrieval_plan_cache`，字段为用户查询的哈希值，值为 LLM 生成的完整“检索计划”JSON。

收到请求后，首先计算查询语句的哈希值，在缓存中查找。如果命中，直接返回缓存的检索计划，**完全避免 LLM 调用**。设置合理的过期时间（如
24 小时）以适应潜在的意图变化。

#### Layer 1: 轻量级分类器 (Triage)

使用基于关键词、正则表达式的规则引擎或一个轻量级的本地模型，比如 `fastText`、`DistilBERT` 。

快速处理简单、高频的命令式意图（如“清空对话”、“总结一下”），或识别出无需调用记忆的闲聊。若匹配成功，则直接执行或跳过后续复杂步骤；否则，进入
Layer 2。

如果查询被成功分类，则使用预先定义好的检索计划模板，填充关键词后直接进入下一步。例如，识别为 `查询决策` 意图，则直接套用模板：
`{"target_memories": ["episodic"], "episodic": {"keywords": [...], "metadata_filter": {"field": "event_type", "op": "eq", "value": "decision"}}}`。

#### Layer 2: LLM 驱动的深度分析 (Deep Analysis)

调用一个低成本、高速度的 LLM（如 GPT-3.5-Turbo, Llama3-8B-Instruct）进行类似 "Function Calling" 的分析。

- **输入**: `{"user_id": "...", "session_id": "...", "query": "我们上次关于苍穹项目的决策是什么？"}`

- **Prompt 指令**: `"分析用户查询，识别其核心意图，并为记忆系统规划一个检索计划。提取用于筛选的元数据实体、用于稀疏检索的关键词，并判断是否需要生成假设性答案（HyDE）。以 JSON 格式输出检索计划。"`

- **输出 (优化后)**: 一个详尽的检索计划。

  ```json
  {
  "query": "我们上次关于苍穹项目的决策是什么？",
  "retrieval_plan": {
    "target_memories": [
      "episodic",
      "semantic"
    ],
    "episodic": {
      "keywords": [
        "苍穹项目",
        "决策"
      ],
      "metadata_filter": {
        "and": [
          {
            "field": "project_id",
            "op": "eq",
            "value": "proj_123_cangqiong"
          },
          {
            "field": "event_type",
            "op": "eq",
            "value": "decision"
          }
        ]
      }
    },
    "semantic": {
      "keywords": [
        "苍穹项目",
        "决策",
        "Redis",
        "缓存"
      ],
      "metadata_filter": {
        "and": [
          {
            "field": "tags",
            "op": "contains",
            "value": "project_cangqiong"
          },
          {
            "field": "status",
            "op": "eq",
            "value": "active"
          }
        ]
      },
      "hyde_needed": true
    },
    "preference": {
      "keys": [
        "communication_tone",
        "project_roles"
      ]
    }
  }
}
  ```

- **作用**:

    1. **精确路由**: `target_memories` 字段明确了需要查询的记忆库，避免了对所有模块的无效检索。
    2. **关键词提取**: `keywords` 字段直接为后续的稀疏检索（BM25）提供弹药。
    3. **元数据过滤**: `metadata_filter` 为混合检索的“先过滤”步骤提供了精确的结构化查询条件。

### **Step 2: 并行检索 (Parallel Retrieval)**

根据生成的检索计划，系统向目标记忆模块分发检索任务。

1. **查询扩展 (Multi-Query Expansion)**:

    - **HyDE (Hypothetical Document Embeddings)**: 生成的检索计划在需要查询拓展的情况下，系统会先请求 LLM
      生成一个针对该问题的“假设性理想答案”。
    - **调用**: `LLM("为问题'X'生成一个理想的回答") -> hypothetical_answer`
    - **后续**: 使用 `hypothetical_answer` 的 embedding 进行向量检索，这比直接用问题的 embedding 效果更好。

2. **混合检索 (Hybrid Search)**: 目标记忆模块执行“先过滤，再搜索”的策略。

    - **稀疏检索 (BM25)**: 擅长捕捉**字面量匹配**
      。当用户的查询包含必须精确匹配的专有名词、ID、代码片段或特定术语（如项目名 "苍穹"、错误码 "404"
      ）时，稀疏检索能确保包含这些关键词的文档被优先召回。关键词来源是`retrieval_plan`中LLM提取的`keywords`字段。

      可在 PostgreSQL 中使用 `tsvector` 实现，或调用独立的 Elasticsearch/OpenSearch 服务，擅长关键词匹配。

    - **稠密检索 (Vector)**: 擅长捕捉**语义相似性**
      。当用户的查询是概念性的或换了种说法（如用户问“项目进度落后的原因”，而原文是“交付延期的风险分析”）时，向量检索能够理解其背后的语义，召回内容相关但措辞不同的文档。

      使用 `user_query` 或 `hypothetical_answer` 的 embedding 查询 Pinecone/Milvus。擅长语义匹配。

### **Step 3: 结果融合与重排 (Fusion & Reranking)**

各模块召回 Top-K 结果后，进入融合与重排阶段。

1. **结果合并 (Reciprocal Rank Fusion - RRF)**:

    - 算法: RRF 是一种无需调参的、效果出色的结果合并算法。它根据每个文档在不同检索器（BM25, Vector
      Search）结果列表中的排名倒数来计算最终得分。
    - 公式: $\text{RRF\_score}(d) = \sum_{i=1}^{n} \frac{1}{k + rank_i(d)}
      $，(d 表示某个文档，$rank_i(d)$ 表示文档 d 在第 i 个排序列表中的排名，k 是一个小的平滑常数，避免分母为 0，通常为 60)
    - 作用: 将多个异构检索源的结果公平地融合在一起

2. **重排 (Reranking) - 模型选型对比**:

    - 模型：使用 Cross-Encoder 模型对 RRF 排序后的 Top-K（如 K=100）结果进行精排。具体选择哪款模型，取决于对成本、延迟和效果的权衡。
    - 过程: Cross-Encoder 会同时接收 `query` 和每个 `document_chunk`，输出一个更精确的相关性分数。
    - 输出: 重排得到最终的 Top-N（例如 N=10）个最相关的记忆片段。

#### 高级检索策略

长文档分层检索 (Hierarchical Retrieval): 当需要检索超长文档（如百页 PDF）时，采用分层策略。

1. 将文档按章节/大段落分块并生成摘要，先对摘要进行检索和粗排，定位到最相关的几个章节。
2. 仅在这些选中的章节内部，进行更细粒度的段落检索和精排。这种方法极大减少了送入 Reranker 的 Token 数量，显著降低成本和延迟。

多语言环境优化 (Multilingual Enhancement): 当知识库包含多种语言（如中、英、日）时，为保证检索效果，可采取两种策略：

- 在“检索计划”生成阶段，让 LLM 将原始查询扩展为包含多种语言的“种子查询”(Seed Queries)；
- 替换 Embedding 模型为原生支持多语言的优秀模型，如 `LaBSE` 或 `E5-large-multilingual`，从根源上提升多语言语义理解能力。

#### 重排模型/算法选型

| **方案类别**                 | **推荐模型/算法**                                      | **核心原理**                                                 | **优点**                                                     | **缺点**                                          | **适用场景**                                               |
| ---------------------------- | ------------------------------------------------------ | ------------------------------------------------------------ | ------------------------------------------------------------ | ------------------------------------------------- | ---------------------------------------------------------- |
| **效果最佳 (Cross-Encoder)** | `Cohere Rerank`, `BGE-Reranker-large`                  | 全文交叉比对，深度分析 Query 和文档的交互关系。              | 精度极高，能深刻理解细微关联。                               | 成本高，算力要求高，延迟大（>100ms）。            | 对答案质量有极致要求的金融、医疗、法律等专业问答场景。     |
| **综合最佳 (Cross-Encoder)** | `BGE-Reranker-base`                                    | 全文交叉比对。                                               | 在效果、成本和速度之间取得了出色的平衡。                     | 对于非常专业或模糊的查询，效果略逊于 large 模型。 | **通用场景**，如日常助手、企业知识库问答等，是主推荐方案。 |
| **低延迟/高吞吐 (矢量重排)** | `ColBERT-v2`, `SPLADE++`                               | 将文档预计算为“词袋”式的稀疏向量或多向量表示，重排时仅计算向量相似度，而非全文交叉。 | **速度极快 (x3-x5)**，效果优于 Bi-Encoder，逼近 Cross-Encoder。 | 实现相对复杂，模型需要特殊训练。                  | QPS 要求高的在线服务，或需要快速从海量结果中精筛的场景。   |
| **成本最低**                 | `ms-marco-MiniLM-L-6-v2` (Bi-Encoder) 或**不使用重排** | 将 Query 和文档分别编码为单个向量，计算其相似度。            | 速度最快，计算成本最低。                                     | 效果较差，可能无法准确捕捉语义相关性。            | 对响应速度要求严苛，且对精度容忍度较高的场景。             |

### **Step 4: 预算感知与提示词构建 (Budget-Aware Prompting)**

这是将信息呈现给 LLM 的最后一步。

- **实现**:

    1. 定义一个严格的 Token 预算（如 6000 tokens）。
    2. 按照固定优先级填充内容：**锚点 > 用户偏好 > 工作记忆 > (重排后的)情节记忆 > (重排后的)语义知识**。
    3. 每个记忆片段前都应附带其来源，实现引用注入 (Citation Injection)。

- **示例 Prompt 结构**:

  ```
  # System Instructions
  You are a helpful AI assistant.
  
  # User Profile (from Preference Memory)
  - timezone: Asia/Tokyo
  - communication_tone: concise and professional
  - project_roles: {"proj_123_cangqiong": "lead_engineer"}
  
  # Anchors (from Working Memory)
  - language_preference: respond_in_chinese
  
  # Conversation History (from Working Memory)
  User: 上次我们聊到哪了？
  AI: 我们正在讨论“苍穹”项目的性能瓶颈。
  
  # Retrieved Memories (Top 3 from Reranker)
  ---
  [Source: Episodic, doc_id: event_89, date: 2025-07-28]
  - Decision: 会议决定采用 Redis 替代现有的本地缓存方案。
  ---
  [Source: Semantic, doc_id: tech_doc_v1.2, section: 3.4]
  - Document Content: Redis 在高并发场景下相比本地缓存，具有...
  ---
  [Source: Episodic, doc_id: chat_log_45, date: 2025-07-25]
  - Discussion: 阿泽提到目前的本地缓存存在锁竞争问题。
  ---
  
  # Current User Request
  User: 我们上次关于苍穹项目的决策是什么？
  ```

#### 偏好取舍

以上对于锚点、工作记忆、情节记忆、语义记忆的上下文构建都说的很清楚了，那么偏好会如何取舍呢？

1. 批量拉取：拉取所有可能相关的偏好

   ```sql
   SELECT * FROM user_preferences
   WHERE user_id = 'user_abc' 
     AND (scope = 'current_project_xyz' OR scope = 'global');
   ```

2. 应用层处理：在应用代码中，对拉取到的结果按`preference_key`进行分组。对于每个key，如果存在多个值（即存在冲突），则根据`scope`
   和`source`的优先级（场景 > 全局，显式 > 推断）选择唯一最优的那个

最终，将所有不存在冲突的、以及冲突已解决的偏好一并注入到Prompt中。这种“先拉取，再处理”的策略完美实现了原文描述的优先级：*
*场景瞬时偏好 > 用户全局偏好 > ... > 模型推断偏好**。

## 3. 记忆写入路径 (Write Path)：异步处理

写入路径在响应用户后被异步触发，确保记忆质量。

### Step 1:  工作记忆预处理与写入触发

在将交互信息发送至消息队列前，应用层会实时进行工作记忆的预处理。

#### 滚动摘要机制

为防止工作记忆无限增长，系统采用滚动摘要策略。当`wm:history`列表长度或Token数超限时，系统会取出最旧的一部分对话记录，调用LLM将其提炼成单个摘要字符串，并存入独立的
`wm:summary`列表中。这确保了`wm:history`始终保持短小精悍，同时历史信息也得以保留。

以Redis中的`LIST`为例说明：

1. **状态**: 假设工作记忆`wm:history:{session_id}`的最大长度 `N=20`。当前已有20条对话记录。
2. **新交互**: 第21轮交互产生，包含用户问题和AI回答，共2条新记录。它们被`LPUSH`到列表头部。列表长度变为22。
3. **触发滚动**: 系统检测到长度（22 > 20）超限。
4. **取出旧记录**: 系统从列表**尾部**（即最旧的记录）取出一定数量的记录，例如12条（`LTRIM`命令可以保留头部10条，其余的被丢弃，应用程序在丢弃前获取它们）。
5. **摘要**: 将这12条旧记录打包，发送给LLM进行**一次**摘要调用。
6. **存入摘要区**: LLM返回的**单个摘要字符串**（例如：“...讨论了项目的初期预算和人员分配问题。”）被`LPUSH`到**另一个独立的列表
   ** `wm:summary:{session_id}`的头部。
7. **最终状态**: `wm:history`现在只包含最近的10条对话，保持了上下文的即时性。`wm:summary`列表则包含了历史对话的摘要，供需要时回顾。

#### 鲁棒的话题切换检测

为准确识别话题转换，系统采用一种复合策略。它结合了**embedding余弦相似度**的启发式检测、对话中**关键命名实体的漂移分析**
，以及在信号触发时调用LLM进行最终**语义裁决**，从而可靠地判断话题是否发生切换。

1. 相似度启发式检测: 将当前交互与过去3轮交互的embedding平均值进行比较。相似度急剧下降（例如，从0.9降至0.6）作为一个**初步信号
   **。

2. 实体漂移分析: 跟踪对话中出现的关键命名实体（项目、人名、技术等）。如果当前交互的实体集合与前几轮的实体集合几乎没有交集，这是一个强烈的切换信号。

3. LLM裁决: 当上述一个或多个信号被触发时，发起一个极低成本的LLM调用进行最终确认。

   Prompt**: `"分析最后两段对话。用户是否显著地改变了话题？请回答'是'或'否'。"`

   这种方法结合了定量分析和模型的语义理解能力，准确率会高得多。

> 相似度阈值（如从0.9降至0.6）不应设为固定值，因为不同 Embedding
> 模型的向量分布差异很大。系统应在上线后，采集一周左右的真实对话数据，对“相似度下降值”进行聚类分析，观察其分布，从而科学地设定一个更合理的动态阈值。

#### 锚点提取与生命周期

锚点是用户在会话中明确提出的、必须遵守的指令（如“接下来都用中文回答”）。锚点是**会话级别**的，其生命周期贯穿整个会话，*
*不受话题切换的影响**，除非被用户的新指令明确覆盖。

系统采用**混合模式**进行提取，以兼顾速度与准确性。

1. 快速路径 (Regex + Trie-Tree): 对于高频、模式固定的锚点（如“用中文回答”、“总结一下”），预先定义正则表达式或构建 Trie
   树进行瞬时匹配。这覆盖了 80% 的常见情况，几乎零成本。
2. LLM 兜底: 如果快速路径未命中，则调用 LLM 进行更深度的语义理解，以提取更复杂、形式不固定的指令。
    - Prompt: `"分析以下用户语句，判断其中是否包含任何必须在当前整个会话中遵守的指令、命令或约束。如果有，以JSON键值对格式提取该约束。如果没有，返回空对象。用户语句：..."`
    - 示例: 用户说“请注意，接下来的讨论，保密等级为绝密”，LLM会输出`{"security_level": "top_secret"}`。

### Step 2: 写入门控 (Write Gating)

消费消息的 Worker 执行严格的检查。

#### 提炼记忆原子

调用 LLM 将原始对话提炼成结构化的“记忆原子”。

为保证结构化检索时标签的一致性，采用**受控词汇表（Controlled Vocabulary）或枚举（Enum）**的方式约束LLM的输出，而非让其自由生成。

- **Prompt改造**:

  ```
  "从以下对话中，提取关键事实、决策、任务、实体关系和新学到的用户偏好。
  1. 提炼核心内容。
  2. 从以下列表中为该内容选择最合适的类型：['decision', 'task_assignment', 'fact_statement', 'Youtubeed', 'user_opinion']。
  3. 以JSON格式输出。
  对话：..."
  ```

- **优点**: 这强制LLM成为一个**分类器**而非生成器，极大地保证了`event_type`、`scope`等关键元数据的一致性和可查询性。

#### 多级优先队列

为不同类型的记忆提供不同的处理优先级，解决“所有记忆都等待同样长的时间”的问题

实时总线 (Real-time Bus - P0)

- 处理内容: 会话锚点 (Anchors)。例如用户指令“接下来用中文回答”。
- 处理方式: 这类信息直接**同步写入**
  或通过一个超高优先级的内存队列处理，绕过所有复杂的门控检查，确保在下一轮对话中立即生效。延迟目标：< 50ms。

高优队列 (High-Priority Queue - P1)

- 处理内容: 显式形成的关键记忆。如用户明确的偏好设定、会议中产生的明确决策 (Decision) 或 任务分配 (Task)。
- 处理方式: 进入一个**独立的、消费者资源充足的队列**
  。执行轻量级门控（如仅查重，不进行复杂的置信度评估），快速写入数据库。延迟目标：< 5s。

标准队列 (Standard Queue - P2)

- 处理内容: 普通对话内容、待提炼的情节记忆、需要置信度评估的推断偏好 (Inferred Preference)。
- 处理方式: 进入**主处理队列**，执行完整的写入门控流程（提炼、查重、查信、查敏）。这是原始方案描述的路径。延迟目标：< 60s。

批量处理 (Batch Processing - P3)

- 处理内容: 大规模、非紧急的知识导入，如批量上传知识库文档、旧有记忆的重新索引等。
- 处理方式: 在系统负载较低的时段（如夜间）进行**批处理**，避免冲击在线服务的资源。

#### 查重 (Deduplication):

- 算法: 对提炼出的每个“情节记忆原子”，生成其 embedding。使用 `MinHash LSH`
  或在向量数据库中进行近似最近邻搜索，检查近期（如过去7天）是否存在向量余弦相似度 > 0.95 的记忆。
- 处理: 若重复，则更新现有记忆的 `last_accessed_at` 和重要性评分，而不是新增。

#### 查信 (Confidence Check)

- 算法: 主要针对推断出的偏好。其置信度采用**贝叶斯更新 (Bayesian Updating)** 模型，而非简单的线性增减，能更平滑、更合理地反映置信度的变化。

- 模型: P(Pref∣Evidence)∝P(Evidence∣Pref)×P(Pref)

- 简化实现: 将置信度视为概率。初始置信度为 p0=0.6。

    - 当用户采纳建议（正反馈），新置信度
      $$
      p_{\text{new}} = \frac{p_{\text{old}} \times p(\text{pos\_ev} \mid H)}{p_{\text{old}} \times p(\text{pos\_ev} \mid H) + (1 - p_{\text{old}}) \times p(\text{pos\_ev} \mid \neg H)}
      $$

    其中 $p({pos_ev}|H)$（假设为真时看到正反馈的概率，如0.8）> $p({pos_ev}|\neg H)$（假设为假时看到正反馈的概率，如0.3）。

- 当用户纠正（负反馈），应用类似的更新逻辑。

- **优势**: 这种方法可以自然地处理证据的强度，并能生成更可靠的置信度曲线。

**处理**: 只有当 `confidence > 0.7` 时，该偏好才能被读取路径直接使用。

#### 查敏 (Sensitivity Check)

采用**多层防御**策略实现

1. **PII检测库**: 使用 `presidio` (by Microsoft) 或类似库检测常规个人身份信息（姓名、电话、邮箱等）。
2. **自定义规则**: 使用正则表达式匹配内部定义的敏感信息格式（如员工ID、API Key格式）。
3. **信息熵与关键词检测**: 增加一层专门检测凭证（如 API Key, Token）的机制。通过计算字符串的**香农熵 (Shannon Entropy)**
   识别高信息密度的随机字符串，并结合关键词（如 `key`, `token`, `secret`, `password`）进行精准捕获。这种方法对 GitGuardian
   等工具发现的意外泄露非常有效。

检测到敏感信息后，根据预设策略进行拒写、脱敏或告警。

### Step 3: 存储与索引

通过门控后，记忆原子被写入相应的数据库。

- **工作记忆**: 此部分在读取路径中已实时更新，写入路径不处理。
- **偏好记忆**: 将通过验证的偏好写入或更新 PostgreSQL 的 `user_preferences` 表。
- **情节/语义记忆**: 将提炼的记忆原子、其 embedding 和元数据分别写入 PostgreSQL 和 Pinecone。

#### 智能化的偏好更新

用户的偏好会发生变化，比如用户之前经常用 Go 语言编写代码，可是后面逐渐越来越多次数用 Python 了，那么该如何更新此偏好变化呢？

1. 提取新偏好: 写入门控提炼出一个新的**推断偏好**，例如
   `{"key": "language_preference", "value": "Python", "confidence": 0.6}`。

2. 查询现有偏好: 在写入`user_preferences`表之前，系统会先查询：

   ```sql
   SELECT id, preference_value, confidence, source 
   FROM user_preferences 
   WHERE user_id = '...' AND preference_key = 'language_preference';
   ```


3. 执行更新逻辑:

- **情况A：找到冲突偏好**: 查询返回了`'Go'`的偏好。
    - 如果`'Go'`的`source`是`'explicit'`（用户明确设置的），则**忽略**本次推断出的`'Python'`偏好，不进行任何操作。
    - 如果`'Go'`的`source`是`'inferred'`或`'confirmed'`，则执行**置信度衰减**。例如，将`'Go'`的`confidence`降低
      `UPDATE user_preferences SET confidence = confidence * 0.7 WHERE id = ...`。同时，插入新的`'Python'`偏好。当任何偏好的
      `confidence`低于某个阈值（如0.1）时，可由一个后台任务将其归档或删除。
- **情况B：未找到冲突偏好**: 直接将新的`'Python'`偏好插入表中。

## 4. 核心模块：数据 Schema 与实现细节

### 4.1 工作记忆 (Working Memory)

- 存储: Redis

- 数据结构:

    - **对话历史**: `LIST`，键为 `wm:history:{session_id}`。每次交互 `LPUSH` 一个 JSON 字符串，并 `LTRIM` 保持窗口大小（如最近
      20 条）。

      JSON

      ```json
      // 一个 LIST 元素
      {"role": "user", "content": "你好", "timestamp": "..."}
      ```

    - **层级摘要**: `LIST`，键为 `wm:summary:{session_id}`。当历史长度或 Token 数超限时，触发 LLM 生成摘要，`LPUSH`
      到此列表，并清空部分历史记录。

    - **锚点**: `HASH`，键为 `wm:anchors:{session_id}`。存储会话级指令，如 `{"language": "german"}`。

    - **上轮交互向量**：`STRING`，键为`wm:last_interaction_embedding:{session_id}`。存储上一轮交互向量，用于话题切花检测

### 4.2 情节记忆 (Episodic Memory)

- 存储: PostgreSQL (元数据) + Pinecone (向量)

- PostgreSQL Table: `episodic_events`

  ```sql
  CREATE TABLE episodic_events
  (
      event_id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
      user_id               VARCHAR(255) NOT NULL,
      session_id            VARCHAR(255),
      created_at            TIMESTAMPTZ      DEFAULT NOW(),
      last_accessed_at      TIMESTAMPTZ      DEFAULT NOW(),
      event_type            VARCHAR(50),                  -- 'decision', 'task', 'qa', 'fact'
      content_text          TEXT         NOT NULL,
      content_embedding_id  VARCHAR(255),                 -- ID in Pinecone
      importance_score      FLOAT            DEFAULT 0.5, -- 用于衰减和排序
      metadata              JSONB,                        -- {"project_id": "...", "tags": [...]}
      source_interaction_id UUID                          -- 关联到原始交互记录
  );
  ```

- **时间衰减 (Time Decay) 实现**: 在检索排序时，结合检索得分和时间衰减函数。

    - **公式**: `FinalScore = RetrievalScore * exp(-k * (NOW() - last_accessed_at))`。`k` 是衰减系数。`last_accessed_at`
      在每次被成功引用时更新。

### 4.3 语义记忆 (Semantic Memory)

- 存储: PostgreSQL (元数据) + Pinecone (向量)

- PostgreSQL Table: `knowledge_chunks`

  ```sql
  CREATE TABLE knowledge_chunks (
      chunk_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
      document_id VARCHAR(255) NOT NULL,
      document_version VARCHAR(50) NOT NULL,
      chunk_text TEXT NOT NULL,
      chunk_embedding_id VARCHAR(255),
      source_metadata JSONB, -- {"filepath": "...", "title": "..."}
      hierarchy_info JSONB, -- {"h1": "Sec 1", "h2": "Sub Sec 1.1"}
      status VARCHAR(20) DEFAULT 'active' -- 'active', 'archived'
  );
  CREATE INDEX ON knowledge_chunks (document_id, document_version);
  ```

- 分块策略: 使用 Markdown 解析器，按 `#`, `##`, `###` 标题进行语义分块。每个块都保留其标题层级作为 `hierarchy_info`
  ，以便在引用时生成精确来源。

### 4.4 偏好/画像 (Preference Memory)

- 存储: PostgreSQL

- PostgreSQL Table: `user_preferences`

  ```sql
  CREATE TABLE user_preferences (
      id SERIAL PRIMARY KEY,
      user_id VARCHAR(255) NOT NULL,
      preference_key VARCHAR(100) NOT NULL,
      preference_value JSONB NOT NULL,
      scope VARCHAR(100) DEFAULT 'global', -- 'global', 'project_id_123'
      source VARCHAR(50) NOT NULL, -- 'explicit', 'confirmed', 'inferred'
      confidence FLOAT DEFAULT 1.0,
      created_at TIMESTAMPTZ DEFAULT NOW(),
      last_updated_at TIMESTAMPTZ DEFAULT NOW(),
      UNIQUE (user_id, preference_key, scope)
  );
  ```

## 5. 进阶架构与未来方向

本设计方案提供了一套稳健的通用记忆系统实现。在特定场景下，或为了追求极致性能，可以考虑以下更前沿的架构演进。

| **架构模式**              | **核心思想**                                                                                                                      | **优点**                                            | **缺点**                                                                       | **适用场景**                                               |
|-----------------------|-------------------------------------------------------------------------------------------------------------------------------|---------------------------------------------------|------------------------------------------------------------------------------|--------------------------------------------------------|
| Graph-RAG             | 将知识库中的信息块（Chunks）作为图的**节点**，将它们之间的引用、上下文或逻辑关系定义为**边**。检索时，不仅考虑内容的语义相似性，还利用图算法（如最短路径、社区发现）在知识图谱上游走，寻找关联最紧密的知识簇。                | 检索精度极高，能发现深层次、跨文档的隐含关系，提供更具逻辑和可解释性的答案。            | 架构复杂，需要额外的知识图谱构建和维护成本。                                                       | 高度结构化、内部关联性强的知识领域，如**代码库分析、法律文书研究、药物研发**等，需要进行复杂推理的场景。 |
| Streaming RAG         | 将数据源（如实时日志、监控指标、社交媒体流）视为一个无限的数据流。通过流处理平台（如 Flink, Spark Streaming）直接对流入的数据进行实时处理、向量化，并写入向量数据库。                                | **近乎零延迟**的知识更新，数据产生后数秒内即可被检索到。                    | 对基础设施要求高，需要稳定的流处理管道。                                                         | **实时监控与告警、金融市场动态分析、舆情监控**等需要对最新信息做出即时反应的场景。            |
| LLM+VectorStore 一体化服务 | 使用云厂商提供的托管服务，如 `OpenAI Assistants API`, `Chroma Cloud`, `Pinecone Serverless` 等。这些服务将 LLM 调用、向量存储、检索逻辑（甚至 Reranking）封装在一起。    | **极大简化开发和运维**，开发者无需自行管理向量数据库和复杂的 RAG 流程，可以快速搭建原型。 | **锁定特定厂商**，灵活性和可定制性较低，可能无法进行细粒度的性能优化（如替换 Reranker），且对于数据隐私和私有化部署有严格要求的场景不适用。 | **初创公司、快速原型验证、非核心业务集成**，或者运维资源有限的团队。                   |
| 领域自适应 Embedding       | 针对不同领域分别训练或选择专有 Embedding 模型（如医疗用`SapBERT`，代码用`CodeBERT`）。在检索时，先用一个轻量级分类器（Router）判断查询属于哪个领域，然后将查询路由到对应的 Embedding 模型和向量数据库分区。 | 增强在在特定专业领域（如医疗、法律、代码）的语义理解能力                      | 架构更复杂，需要维护 Router 和多个向量索引；Router 的准确性会影响整体效果。                                | 需要同时处理**多个差异巨大且边界清晰**的领域知识的平台，如一个同时提供法律、医疗和编程问答的企业大脑。  |

## 6. 结语

本设计方案将原文的理念具体化，提供了一套可操作、可扩展的工程蓝图。通过采用微服务架构、成熟的技术栈和明确的数据流，该系统能够高效地为
AI 应用赋予强大而可靠的记忆能力。成功的关键在于对读写路径的精细控制、对记忆质量的持续监控，以及一个能够将用户反馈融入系统迭代的闭环机制。