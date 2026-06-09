# 项目总览

## 项目目标

Scenic Spot Guide System 是一个景区智能导览系统，提供游客问答、景点/导览内容管理、路线管理、运营数据看板和数字人导览能力。

## 技术栈

- 后端：Go 1.25.0、Gin、GORM、PostgreSQL（主数据库配置）、SQLite（本地开发/轻量测试配置）。
- 前端：Vue 3、Vite、TypeScript、PixiJS、Live2D。
- 知识库：本地 JSONL 分块语料，启动时导入到数据库；基础 smoke 集为 32 切片/5 问答，合成规模验证集为 3000 切片/300 问答。
- AI：DeepSeek 兼容接口；Embedding 示例配置使用 DashScope `text-embedding-v2`，口径为 1536 维、Cosine、Float32；未配置 Embedding API 时回退到 BM25/词面检索。

## 目录结构

- `main.go`：服务启动、依赖组装、RAG 初始化和优雅关闭。
- `internal/config`：配置结构和加载逻辑，默认读取 `configs/config.yaml`，支持 `SCENIC_GUIDE_` 环境变量覆盖。
- `internal/handler`：HTTP 路由与接口处理。
- `internal/service`：业务逻辑、RAG、统计和 AI 相关服务。
- `internal/repository`：数据库访问层。
- `internal/model`：GORM 模型与自动迁移。
- `knowledge`：景区知识库语料、基础分块/评测样例、`lingshan_scale_3000.jsonl`、`lingshan_eval_300.json` 和 `DATASET.md` 数据边界说明。
- `web-vue`：Vue 管理端、数据看板和数字人前端源码。
- `static`：Go 服务直接托管的静态资源和 Vue 构建产物。
- `docs`：数字人集成、可复现评估报告、博客草稿、运行说明和 API 文档。
- `scripts`：项目辅助脚本，例如编码检查和作品集录屏生成。
- `cmd/rag-eval`：离线 RAG 评估和基准命令，默认使用 `knowledge/lingshan_chunks.jsonl` 和 `knowledge/lingshan_eval_qa.json`，支持 `-k`、`-bench`、`-concurrency`、`-repeat`、`-retrieval-only`、`-mode`、`-compare-modes`、`-embedding-weight`、`-bm25-weight`、`-rrf-k`，输出通过率、Recall@K、MRR@K、关键词覆盖率、失败原因和检索 p50/p95。
- `cmd/demo-seed`：演示数据初始化命令，向当前配置数据库写入演示账号、景点、路线、交互日志，并在知识库为空时导入默认知识。

## 核心流程

1. 启动时加载 `configs/config.yaml`，再叠加 `SCENIC_GUIDE_` 环境变量。
2. 初始化日志、JWT、PostgreSQL 或显式配置的 SQLite 本地数据库和模型迁移；配置未指定 driver 时按 PostgreSQL 主配置处理。
3. 初始化 RAG 服务：优先使用 Embedding 检索，缺少配置时使用 BM25/词面本地检索；BM25 路径会维护 token cache、倒排候选索引和 chunk ID 映射以支撑 3000 切片实验。评估链路支持 `bm25-local`、`embedding`、`hybrid-weighted`、`rrf-fusion`、`light-rerank`，其中 `light-rerank` 是本地可复现规则重排，`rrf-fusion` 作为需要 Embedding 配置的混合召回实验方向。真实资料检索会在召回前做本地 query expansion，只影响检索打分和候选召回，不改写用户原始问题或生成 prompt。
4. 若知识库表为空，从 `knowledge/lingshan_chunks.jsonl` 导入数据。
5. Gin 挂载 API、静态页面、Vue 应用入口和 Open-LLM-VTuber 代理。
6. 服务收到 `SIGINT` 或 `SIGTERM` 后执行优雅关闭。

## 运行与测试

- 复制 `configs/config.example.yaml` 为 `configs/config.yaml`，再填写本地密钥。
- `security.jwt_secret` 必须替换为至少 32 位的随机字符串，不能使用示例占位值。
- 后端依赖：`go mod download`。
- 前端构建：在 `web-vue` 目录运行 `npm install` 和 `npm run build`。
- 容器启动：仓库根目录运行 `docker compose up --build`，默认启动 PostgreSQL 16 和 Go 服务。
- 启动服务：仓库根目录运行 `go run .`，本地直启需准备 PostgreSQL 或显式改为 SQLite 本地配置。
- 验证命令：`go test ./...`、`go vet ./...`、`npm run check`、`npm run check:encoding`、`npm run build`。
- RAG smoke 评估：`go run ./cmd/rag-eval -k 8 -fail-on-miss`。
- RAG 合成规模实验：`go run ./cmd/rag-eval -knowledge knowledge/lingshan_scale_3000.jsonl -eval knowledge/lingshan_eval_300.json -k 8 -bench -concurrency 16 -repeat 1 -retrieval-only -fail-on-miss`。
- RAG 真实资料多模式本地对比：`go run ./cmd/rag-eval -knowledge knowledge/real/lingshan_real_chunks.jsonl -eval knowledge/real/lingshan_real_eval_open.json -k 8 -bench -concurrency 16 -repeat 1 -retrieval-only -compare-modes bm25-local,light-rerank`。
- 演示数据：`go run ./cmd/demo-seed`，默认管理员账号为 `admin / DemoAdmin123456`；演示密码只用于本地演示，不应作为生产凭据。
- 作品集录屏：服务启动后运行 `node scripts/record_lingshan_demo.js`，会用 Playwright 打开本地页面并输出带中文字幕的灵山项目演示视频到 `tmp/demo-video`；可通过 `SCENIC_DEMO_OUTPUT_DIR` 覆盖输出目录。

## 关键约定

- 不提交 `configs/config.yaml`、数据库、日志、可执行文件、缓存目录和本地环境文件。
- 数据库主路径为 PostgreSQL，`database.driver` 支持 `postgres`/`postgresql` 和 `sqlite`；连接池配置包括 `max_open_conns`、`max_idle_conns`、`conn_max_lifetime_minutes`。
- RAG 评测 JSON 支持 `expected_keywords`、`expected_chunk_ids`、`category`、`difficulty`；3000/300 是合成闭集实验数据，不宣称覆盖真实景区完整知识库。
- 3000/300 合成实验只作为内部回归口径，不能外推为开放域真实召回率；简历和面试主口径使用 `knowledge/real/` 真实资料评估集。
- 真实资料 RAG 评估的优化前本地基线：BM25-only 单轮约 Recall@8 85.5%、MRR@8 0.749、通过率 88.2%；light-rerank 单轮约 Recall@8 86.0%、MRR@8 0.761、通过率 88.2%。按失败样例定向优化后，当前 retrieval-only 单轮口径为：`bm25-local` 通过率 98.5%、Recall@8 94.8%、MRR@8 0.793；`light-rerank` 通过率 99.5%、Recall@8 95.3%、MRR@8 0.802。该结果来自语料与问法映射增强，仅代表当前真实资料评测集，不包含外部 Embedding、大模型生成、ASR 或 TTS，也不是线上 SLA。
- AI/RAG 请求可传 `session_id` 启用短期会话上下文；服务默认保留最近 5 轮，并在内部记录最近主题实体、意图类型和实时边界状态。追问改写使用这些元信息补全“它、那里、门票呢、下雨呢、现在人多吗”等省略问法；改写只影响检索和 prompt 上下文，不新增公开 API 字段，不属于长期记忆或用户画像。
- `static/vue-app` 是 Vue 构建输出，会随 `web-vue` 构建刷新。
- Vue 应用包含数据看板、管理后台、数字人导览和游客地图四个主要视图。
- 游客地图优先从 `/api/v1/spots` 读取真实景点数据；接口为空或不可用时才回退到前端内置演示点位。
- 旧静态 `/static/admin.html` 与 `/static/dashboard.html` 不再承载 mock 管理/大屏页面，仅跳转到 Vue 正式入口；对应旧 mock 脚本已从 `static/js` 移除，避免演示时误用模拟数据。
- 前端生产构建关闭 sourcemap，避免提交大体积调试映射文件。
- `static/digital-human/libs/live2dcubismcore.min.js` 被 Vue 入口引用，不能随意删除。
- 当前 Live2D 主模型使用 `static/live2d-models/mao_pro`。
- 未被业务入口引用的旧 `static/live2d-models/shizuku` 模型已清理。
- scripts/check-secrets.mjs 新增高德地图 API Key 检测规则（map_webapi_key、map_config_key）。
- internal/handler/errors.go 提供 isRecordNotFound 统一封装 gorm.ErrRecordNotFound，handler 层统一使用。
- user_handler.go 提取 alidateUsername、alidateEmail、alidateRole、userPayload 共享函数；创建/编辑用户统一走密码策略校验。
- docs/architecture.md 记录景区系统与数字人系统的解耦架构、独立迁移清单和 API 连接点。
- web-vue 新增 .eslintrc.cjs、.prettierrc.json、.prettierignore，前端代码风格统一由 ESLint + Prettier 管理。
- 源码和文档统一使用 UTF-8；提交前应运行编码检查，防止中文内容 mojibake。

## 安全与鉴权补充

- 登录使用 `auth_token` HttpOnly Cookie；`POST /api/v1/login` 不在响应体返回 JWT，前端通过 `/api/v1/user/me` 恢复会话。`Authorization: Bearer <token>` 仍作为非浏览器客户端兼容路径保留。
- `/api/v1/admin/*` 管理接口必须通过 `AuthMiddleware` 与 `AdminMiddleware`；浏览器端依赖 Cookie 会话，非浏览器客户端可使用 Bearer token。
- `/api/v1/knowledge/*` 知识库管理接口按管理员接口处理，读取、创建、上传、更新和删除都需要管理员权限。
- `/api/v1/register` 不接受客户端传入角色，后端会强制把新注册用户角色设为 `visitor`。
- 静态首页和 Vue 前端不再读写 `localStorage.authToken`；登录、登出、数字人 API 和管理接口统一依赖 Cookie 会话。
- 对外 JSON 字段统一使用 `snake_case`，例如 `image_url`、`sort_order`、`spot_id`、`content_type`、`audio_url`、`created_at`、`updated_at`；Vue 管理页按该契约直接收发字段。
- `/api/v1/admin/users` 支持管理员用户分页、创建、编辑和删除；创建/改密复用密码策略和 bcrypt，编辑时密码留空表示不修改。
- `/api/v1/contents` 是管理员分页列表；公开导览内容查询仅保留单条与按景点/类型查询路径，避免公开全量导览内容。
- `/vtuber-ws/*` WebSocket 代理支持 `auth_token` Cookie 鉴权，同时保留子协议 token 和 query token 兼容路径。
- 旧静态游客页和大屏中由用户或接口返回的文本插入 HTML 前必须经过 `escapeHtml` 转义。
- RAG 通用问答日志不得打印 API Key 或其前缀；调试日志应避免输出完整请求体和敏感响应体。
- 日志使用 Go `slog` 作为默认结构化 logger；AI、RAG、数字人和 OpenAI 兼容代理链路优先记录长度、耗时、状态码、trace/session 等元信息，不记录完整敏感正文。RAG trace 记录 `trace_id`、`retrieval_ms`、`embedding_ms`、`generation_ms`、`total_ms`、`provider`、`cache_hit`、`chunk_count` 和 `retrieval_mode`；总耗时超过 5 秒时输出结构化 WARN 日志。
- 安全测试覆盖 JWT 过期、伪造签名、异常签名算法、普通用户访问管理员接口、知识库高风险删除接口鉴权，以及 rate limit 的并发和窗口恢复行为。
- HTTP 服务显式配置读取、写入、空闲和请求头超时，并设置不破坏同源 iframe、语音输入和数字人媒体能力的基础安全响应头。
- `SCENIC_GUIDE_DEV_ADMIN_BYPASS` 仅用于本机开发调试，且只接受真实 `RemoteAddr` 为 loopback 的请求，不信任转发头。
- 公开仓库前已重写 Git 历史，移除旧提交中出现过的 API Key 形态文本和旧 sourcemap 文件；相关真实密钥仍应在服务商控制台轮换。

## 已知风险

- `configs/config.yaml` 本地可能含真实 API Key，必须保持忽略状态。
- SQLite 仅作为本地开发/轻量测试配置；简历和面试中不要把它说成 PostgreSQL 的高可用接管、故障自动切换或生产数据库能力。
- 3000/300 RAG 验证集是合成闭集 fixture，适合说明检索链路可评估、可复现；真实景区落地仍需要真实资料、独立人工标注、来源引用、向量数据库/pgvector、监控和运营闭环。
- `static/digital-human` 保留了旧数字人静态前端及运行库，删除前需确认没有外部入口依赖。
- 服务启动日志在部分 Windows 终端可能出现编码显示问题，但 Go 编译和测试不受影响。
