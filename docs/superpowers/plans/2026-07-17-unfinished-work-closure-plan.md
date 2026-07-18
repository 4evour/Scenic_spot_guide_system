# 未完成任务收尾实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use `superpowers:executing-plans` to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 恢复可被可靠验证的本地 RAG 问答质量，并完成已交接的安全策略、测试覆盖、真实链路验证和文档收尾。

**Architecture:** RAG 先把评测变为可重复、可解释的离线基线，再修复“已召回内容在本地生成阶段被丢弃”的问题。认证撤销、JWT 格式与 CSP 属于跨请求契约，先完成明确的策略决策，再沿模型、令牌、鉴权中间件和测试自底向上实现。外部数字人、ASR、TTS 与模型仅作为联调依赖，不修改 `Open-LLM-VTuber/`。

**Tech Stack:** Go 1.25、Gin、GORM、SQLite/PostgreSQL、Vue 3、Vite、Node.js。

## 全局约束

- 只在 `scenic-guide/` 修改；不覆盖当前 61 个已修改和 43 个未跟踪文件中的既有工作。
- 所有新增 API 字段使用 `snake_case`；认证、Cookie、CSRF 和 WebSocket 沿用现有约定。
- RAG 离线评测不得读取、打印或依赖真实 API Key、数据库和外部模型。
- 每个任务先写失败测试，再以最小改动实现；每次项目文件修改后更新根目录 `CHANGELOG.md`。
- 不将 `static/vue-app/`、运行日志、评测临时文件或凭据加入 Git。

## 已确认的根因

1. 默认命令 `go run ./cmd/rag-eval -k 8 -fail-on-miss` 使用 `knowledge/lingshan_chunks.jsonl` 和 5 条旧评测用例；每条都没有 `expected_chunk_ids`。`retrievalMetrics` 因此直接返回 `Recall@K=1`，报告中的 100% Recall 不是有效召回证据。
2. 同一数据集以 `-retrieval-only` 运行时为 5/5，说明 Top-8 中包含所需文本；普通模式仅 1/5，故失败发生在 `generateAnswerFromChunksWithContext` 的本地句子选择，而非“没有召回任何资料”。
3. 当前本地生成器对“多高”等事实题将输出限制为一条句子；`extractRelevantSnippets` 仅按 BM25 排序，可能优先选择同名背景句，漏掉包含数字、工艺或文化事实的句子。
4. 当前未提交变更新增证据置信度策略：只对 `buildRAGSources(chunks, 3)` 的标题和预览计算证据，并可能在正确证据位于第 4–8 位或预览之后时拒答。这会让在线问答看起来像检索失败，必须单独覆盖回归。
5. `f283118` 基线在同样的本地无模型评测下为 0/5，当前为 1/5；此次问题不是 `f569c31` 安全修复新引入的回归，而是长期存在的离线本地生成质量缺口。

---

## Phase 1：RAG 评测可信化与回答修复

### Task 1：固定离线评测的提供者和召回真值

**Files:**
- Modify: `cmd/rag-eval/main.go`
- Modify: `cmd/rag-eval/main_test.go`
- Modify: `internal/service/rag_evaluation.go`
- Modify: `internal/service/rag_service_test.go`
- Modify: `knowledge/lingshan_eval_qa.json`

**Interfaces:**
- `EvaluationOptions` 新增明确的本地生成/配置生成模式，默认本地模式不得加载外部模型。
- `RAGEvaluationResult` 新增 `retrieval_evaluated`；无 `expected_chunk_ids` 的用例不计入 Recall/MRR 聚合。

- [ ] 编写失败测试：无 `config.yaml`、无任何环境变量时，默认离线评测可以启动且 `generation_provider=local`。
- [ ] 编写失败测试：没有 `expected_chunk_ids` 的用例将 `retrieval_evaluated=false`，不会得到伪造的 Recall/MRR 100%。
- [ ] 为 5 个基础用例补充稳定切片 ID；ID 来自 `knowledge/lingshan_chunks.jsonl`，不得按数组位置推断。
- [ ] 将 `cmd/rag-eval` 的默认路径改为纯本地 BM25 + 本地生成；仅显式指定配置生成或 embedding 时读取配置。
- [ ] 运行：`go test ./cmd/rag-eval ./internal/service`。

**验收标准：** 默认评测不需要密钥；报告明确列出真实的召回评估样本数；篡改一个 expected chunk ID 后 Recall/MRR 测试失败。

### Task 2：修复本地事实答案的句子覆盖

**Files:**
- Modify: `internal/service/generation_service.go`
- Modify: `internal/service/generation_service_test.go`
- Modify: `knowledge/lingshan_eval_qa.json`

**Interfaces:**
- 保留 `generateAnswerFromChunksWithContext(query, chunks, sessionContext)` 签名。
- 新增私有的事实意图评分函数；它只影响本地降级回答，不改变外部 LLM prompt 或检索排序。

- [ ] 编写 5 条失败单测：高度题必须覆盖通高/佛体/总高；九龙灌浴覆盖诞生场景；梵宫覆盖东阳木雕、琉璃和传统工艺；坛城覆盖藏传佛教、五方五佛和转经筒；位置题覆盖省市、区域和马山镇。
- [ ] 在句子评分中保留原 BM25 与 profile boost；对“高度/多少”要求数字和单位、对“为什么”提升原因/工艺说明、对“主要表现/文化”提升主体与解释共现。不得写入只匹配这 5 道题的固定答案。
- [ ] 对事实题只在首条候选已覆盖其意图时使用单句；否则追加互补句，最多 4 条且总输出仍受 700 rune 限制。
- [ ] 运行：`go test ./internal/service -run 'Test.*(Local|Snippet|RAG|Generation)'`。

**验收标准：** 5 条基础用例在本地生成模式均通过；同名背景句不能挤掉包含所问事实的句子。

### Task 3：让证据拒答依据完整召回结果

**Files:**
- Modify: `internal/service/rag_service.go`
- Modify: `internal/service/generation_service.go`
- Modify: `internal/service/rag_service_test.go`
- Modify: `internal/service/generation_service_test.go`

**Interfaces:**
- 保留 API 返回的 `sources` 最多 3 项。
- 证据评分改为基于完整 `chunks` 的标题和全文；不得把截断的 API 预览作为唯一证据。

- [ ] 编写失败测试：正确证据只排在第 4 项时，查询不得被 `should_abstain` 错误拒绝。
- [ ] 编写失败测试：所有切片均无关时仍返回 `should_abstain=true` 和既有无依据答复。
- [ ] 新增内部 `calculateChunkEvidence(query, chunks)`，用于置信度；`buildRAGSources(chunks, 3)` 只用于响应展示。
- [ ] 保持实时/票务/开放类边界问题的拒答规则不变，并覆盖其回归测试。
- [ ] 运行：`go test ./internal/service -run 'Test.*(Evidence|Abstain|RAG)'`。

**验收标准：** 证据排名第 4–8 时可回答；无关资料不会因数量而放行；前端响应契约不变。

### Checkpoint 1：RAG 完整验证

- [ ] `go test ./cmd/rag-eval ./internal/service`
- [ ] `go run ./cmd/rag-eval -k 8 -fail-on-miss`
- [ ] `go run ./cmd/rag-eval -k 8 -retrieval-only -fail-on-miss`
- [ ] 保存不含敏感信息的 JSON 报告到 `docs/eval-results/`，并在报告中区分本地生成和真实端到端生成。

## Phase 2：安全策略决策与实现

### 决策门 A：Token 撤销

**推荐：** 在 `User` 中增加单调递增的 `token_version`，JWT claims 携带该值，`AuthMiddleware`、`WSTokenAuth` 和 `OptionalAuthMiddleware` 均核对数据库当前值。改密、降权、禁用/删除用户时递增版本。

| 方案 | 优点 | 代价 |
|---|---|---|
| 每请求查询用户版本（推荐） | 撤销立即生效，逻辑最少 | 每次认证多一次轻量 DB 查询 |
| 版本缓存 30–60 秒 | 降低查询量 | 撤销最多延迟缓存 TTL，需失效策略 |
| JWT denylist | 可撤销单个 token | 需要 Redis/持久化、过期清理和额外运维 |

**需要确认：** 采用推荐的“每请求版本校验”，还是接受缓存延迟。

### Task 4：实现已确认的 TokenVersion 撤销

**Files:**
- Modify: `internal/model/models.go`
- Modify: `internal/pkg/jwt.go`
- Modify: `internal/pkg/middleware.go`
- Modify: `internal/service/user_service.go`
- Modify: `internal/handler/user_handler.go`
- Modify: `internal/pkg/jwt_test.go`
- Modify: `internal/pkg/middleware_test.go`
- Modify: `internal/handler/user_handler_test.go`

- [ ] 编写失败测试：版本为 0 的旧 token 在密码修改后返回 401；新 token 返回 200。
- [ ] 迁移 `User.TokenVersion`，默认 0 且不向 JSON 输出。
- [ ] 为 `Claims` 和 `GenerateToken` 加入版本；更新登录、游客登录、CSRF token 与所有测试调用方。
- [ ] 在三种 JWT 中间件中统一校验用户存在、角色/版本与 claims 一致；普通可选身份认证在失效时保持匿名，不注入旧身份。
- [ ] 在改密、角色降权、禁用/删除路径递增版本，并清除 `auth_token` Cookie。
- [ ] 运行：`go test ./internal/pkg ./internal/service ./internal/handler`。

**验收标准：** 改密或权限降级立即使旧 HTTP/WS token 失效；游客和 CSRF 流程保持可用。

### 决策门 B：JWT 密钥格式

强制“64 位 hex”是编码格式限制，不等同于更高熵。推荐兼容 **64 hex 字符** 或 **base64 解码后至少 32 bytes**，继续拒绝占位符与短密钥；这既保证 256-bit 密钥材料，也不会强制现有部署改用 hex。

**需要确认：** 接受兼容 hex/base64 的推荐，或强制仅 64 hex（后者是破坏性部署迁移）。

### Task 5：实现已确认的 JWT 密钥校验与 CSP/代理收尾

**Files:**
- Modify: `internal/pkg/jwt.go`
- Modify: `internal/pkg/jwt_test.go`
- Modify: `internal/handler/routes.go`
- Modify: `internal/handler/routes_test.go`
- Modify: `.env.example`
- Modify: `README.md`
- Modify: `docs/digital-human-production-check.md`

- [ ] 编写 JWT 格式正反例测试；任何错误信息不得回显密钥。
- [ ] 按决策门 B 实现格式/长度校验，并提供迁移说明和生成命令。
- [ ] 先为游客主站和数字人路径记录实际 CSP 违规来源；只在验证通过后将 Live2D 必需的 `unsafe-eval` 限制到数字人路由的单独 CSP，禁止直接全局删除而导致 Naive UI 样式失效。
- [ ] 文档化 `SCENIC_GUIDE_TRUSTED_PROXIES` 的生产必填项、逗号分隔示例和启动失败行为；不在示例中写真实 IP。
- [ ] 运行：`go test ./internal/pkg ./internal/handler`、`node scripts/check-secrets.mjs`、`node scripts/check-compose-healthcheck.mjs`。

**验收标准：** 密钥策略可测试、可迁移；反代配置未设置时仍默认拒绝 XFF；CSP 改动经过浏览器验证。

### Checkpoint 2：安全回归

- [ ] `go test ./...`
- [ ] `go vet ./...`
- [ ] `go build ./...`
- [ ] `go build -tags dev ./... && go test -tags dev ./...`
- [ ] 覆盖：旧 token 撤销、CSRF、代理 IP、WS token、CSP 路由策略。

## Phase 3：测试覆盖与真实链路验收

### Task 6：补齐交接文档标出的核心单测缺口

**Files:**
- Modify: `internal/handler/ai_handler_test.go`
- Modify: `internal/service/stats_service_test.go`
- Create: `internal/service/rag_llm_enhance_test.go`
- Create: `internal/service/knowledge_manager_test.go`

- [ ] 为 `/api/v1/ai/chat` 补流式成功、非流式成功、客户端取消、无依据拒答和本地降级测试；断言不会双写会话。
- [ ] 为 Dashboard 30 秒缓存补首次聚合、缓存命中、TTL 失效和并发安全测试。
- [ ] 为 LLM 改写/重排补空配置、本地回退、无效上游响应和排序稳定性测试。
- [ ] 为知识文件导入补 ID upsert、向量/分类字段回填和非法 JSONL 原子失败测试。
- [ ] 运行：`go test ./internal/handler ./internal/service -race`。

**验收标准：** 交接中列出的四个测试缺口均有针对性回归；不引入真实网络调用。

### Task 7：真实依赖联调与评测证据刷新

**Files:**
- Modify: `scripts/e2e_eval.js`
- Modify: `scripts/voice_latency_eval.js`
- Modify: `scripts/e2e_eval.test.js`
- Modify: `scripts/voice_latency_eval.test.js`
- Modify: `docs/digital-human-production-check.md`

- [ ] 为两份脚本加入明确模式：`local/mock`、`external`；报告必须包含依赖状态、服务版本、跳过原因和时间戳。
- [ ] 外部模式只从环境变量读取端点与凭据，日志和 JSON 仅记录“已配置/未配置”，绝不记录值。
- [ ] 在真实 Go 服务、Open-LLM-VTuber、ASR/TTS 和模型均可用时，执行 15 题端到端准确率、语音首字节/完整音频、ASR→回答→播放全链路测量。
- [ ] 外部依赖不可用时，报告标为 `skipped`/`not_measured`，不得将模拟值表述为真实端到端结果。
- [ ] 运行：`node --test scripts/e2e_eval.test.js scripts/voice_latency_eval.test.js`；外部服务就绪后再运行对应外部模式命令。

**验收标准：** 最新报告与当前 commit 同日生成；本地与真实外部结果不可混淆。

### Task 8：同步实施状态并准备提交

**Files:**
- Modify: `docs/superpowers/plans/2026-07-14-p0-qwen-omni-consumption-analysis-plan.md`
- Modify: `HANDOFF_TO_CODEX.md`
- Modify: `CHANGELOG.md`
- Modify: `D:\go web 01\CHANGELOG.md`

- [ ] 仅勾选已由命令和报告证实的验收项；外部联调未执行时保留未勾选并写明原因。
- [ ] 将 RAG 根因、TokenVersion 决策、CSP 路由策略和代理部署要求写入交接文档。
- [ ] 按功能主题拆分提交，避免把 61 个现有修改和 43 个未跟踪文件无审查地一次性提交。
- [ ] 修改完成后以 `mode=full`、`persistence=true` 更新 `D-go-web-01` 代码图谱；若 MCP 不可用，记录失败原因。

**验收标准：** 文档状态和真实验证一致；工作树中没有本次验证产生的临时文件；每个提交只包含一个可审查主题。

## 最终验收矩阵

- [ ] RAG 本地评测与 retrieval-only 评测均通过，Recall/MRR 有有效 ground truth。
- [ ] `go test ./...`、`go vet ./...`、`go build ./...`、dev 标签测试与构建通过。
- [ ] `npm run check`、`npm run lint`、`npm run test:geolocation`、`npm run test:voice-emotion`、`npm run build` 通过。
- [ ] 全部 `check:*`、密钥检查、Compose 检查通过。
- [ ] Token 撤销、JWT 密钥、CSRF、可信代理、WebSocket 和 CSP 有回归测试。
- [ ] 真实外部服务不可用时报告明确跳过；可用时有同 commit 的完整端到端报告。

## 风险与处理

| 风险 | 处理 |
|---|---|
| 现有评测数据和现代 e2e 口径不同 | 先定义每题的事实与 chunk 真值，再统一关键字/事实断言，不以降低阈值掩盖生成遗漏。 |
| TokenVersion 每请求查询增加 DB 压力 | 先实现即时正确性并观测；只有出现证据化瓶颈时再单独设计可失效缓存。 |
| 收紧 CSP 破坏 Naive UI/Live2D | 先收集违规报告，按路由隔离策略逐步收紧，并执行浏览器回归。 |
| 外部服务、模型或凭据不可用 | 仍完成本地/模拟验证，外部报告显式 `skipped`，不伪造成功。 |
| 工作树已有大量未提交改动 | 不使用 reset/checkout；按文件和主题审查、提交，先确认改动归属。 |

## 不纳入本轮最小闭环

- `digital_human_handler.go` 去重、四组 CRUD 泛型化、35 个 i18n 脚本合并和六份文档重写是可选重构，不是已验证 bug 或安全缺口。它们应另立 RFC 和独立计划，避免与认证/RAG 修复混合。
