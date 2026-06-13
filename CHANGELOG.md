# CHANGELOG

## 2026-06-13 15:05 - 景区导览系统全面修复

### 变更内容

**P0 安全修复**
- configs/config.yaml — 敏感凭据（API Key、JWT Secret）改为空占位符，通过环境变量注入
- configs/config.example.yaml — 更新模板，同步 qwen-vl-max 模型配置
- .env.example — 新建，记录所有必需的环境变量
- internal/config/config.go — LoadConfig() 末尾增加敏感字段启动校验（ai.api_key、security.jwt_secret 为空则拒绝启动）
- scripts/cleanup-git-history.sh — 新建 git-filter-repo 脚本清理历史凭据

**P1 架构修复**
- internal/handler/ai_handler.go — Chat() 流式分支改为真正的 token-by-token SSE 流式，新增心跳保活（15秒间隔）
- internal/service/generation_service.go — 新增 CallLLMStreaming() 方法，支持 stream:true 调用 LLM API
- internal/service/rag_service.go — 新增 QueryWithRAGStreaming() 方法，支持检索+流式生成；新增 SlowRequestThresholdMs 常量
- internal/service/session_manager.go — 提取 appendTurnLocked() 公共方法消除 appendSessionTurn/AppendSessionTurnWithUser 重复代码；会话持久化失败日志从 Debug 改为 Warn
- internal/service/embedding_service.go — BM25 CalculateScore 改用 BM25-style TF 归一化 + [0,1] 标准化

**P2 质量改进**
- web-vue/src/services/vtuberSocket.ts — 新增指数退避自动重连（最多10次，最长30秒间隔）
- web-vue/src/utils/fingerprint.ts — 增强设备指纹：新增 Canvas 指纹 + WebGL 渲染器信息 + screen.colorDepth + navigator.platform

**P3 工程化**
- internal/service/embedding_service_test.go — 新建 BM25 分词和评分单元测试
- internal/service/session_manager_test.go — 新建追问改写、意图检测、会话清理单元测试
- internal/service/generation_service_test.go — 新建 prompt 构建单元测试
- static/digital-human-live2d.html — 标记 @deprecated
- static/preview-digital-human.html — 标记 @deprecated
- static/preview-tourist.html — 标记 @deprecated
- static/preview.html — 标记 @deprecated
- web-vue/src/views/DigitalHumanView.vue — persistMessage 新增 7 天过期 localStorage 清理

### 原因
修复安全漏洞、架构缺陷、代码质量问题，提升系统可靠性

### 影响范围
- 安全：配置加载、环境变量依赖
- AI Chat：SSE 流式响应机制、LLM 调用方式
- 会话管理：持久化逻辑、日志级别
- 前端：WebSocket 重连、设备指纹、localStorage 清理
## 2026-06-13 16:30 - 补充比赛官方数据到知识库和数据大屏

### 变更内容

**知识库补充（knowledge/real/lingshan_real_chunks.jsonl）**
- 新增 22 条结构化景点切片（real-struct-ls-001~016 灵山胜境 16 景点, real-struct-nh-001~006 拈花湾 6 景点），来源：比赛官方结构化数据集 docx
- 新增 9 条游览指南切片（real-guide-history-001~003 历史沿革, real-guide-ticket-001~002 门票信息, real-guide-engineering-001~002 工程数据, real-guide-culture-001~002 文化内涵），来源：比赛官方游览指南 docx
- 原有 122 条切片未修改，经逐条对比无事实冲突

**数据大屏数据（static/data/）**
- 新建 tourist_overview.json — 140,447 条游客行为记录总览统计
- 新建 attraction_stats.json — 152 个景区各自统计（满意度、停留时长、消费结构等）
- 新建 attraction_type_stats.json — 8 种景区类型统计
- 新建 cost_breakdown.json — 各类型消费结构占比
- 新建 lingshan_detail.json — 灵山大佛/灵山胜境/拈花湾月度趋势、年龄段、消费结构

### 原因
比赛官方提供了三份数据文件（xlsx 游客行为数据 + 两份 docx 景区资料），项目此前只使用了网页抓取数据，未利用官方结构化数据集。补充后知识库切片从 122 增加到 153，同时为数据大屏前端提供了可直接使用的游客行为统计数据。

### 影响范围
- 知识库：lingshan_real_chunks.jsonl 切片数 122→153，RAG 检索覆盖面提升（新增景点细节、历史、工程参数、门票信息）
- 前端数据：static/data/ 新增 5 个 JSON 文件，可直接 fetch 用于数据大屏展示
- 不涉及代码逻辑变更，不影响现有检索和生成功能
