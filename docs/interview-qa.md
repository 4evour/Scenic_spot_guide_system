# 灵山胜境智能导览系统面试问答

本文档用于面试复习，按“项目整体 -> 架构 -> RAG/AI -> 数字人 -> 安全 -> 数据 -> 前端 -> 性能 -> 测试 -> 优化”的顺序整理常见问题和参考回答。

## 一、项目整体

### 1. 这个项目是做什么的？

这是一个面向景区的智能导览系统，目标是把景区信息管理、游客智能问答、路线推荐、知识库检索和 Live2D 数字人语音交互整合起来。

系统分为游客侧和管理员侧：游客可以查询景点、路线、导览内容，也可以通过 AI 问答或数字人获取导览服务；管理员可以维护景点、路线、导览内容、知识库、数字人配置，并查看运营数据大屏。

技术上，后端使用 Go + Gin + GORM，主数据库配置是 PostgreSQL，SQLite 只保留为本地开发和轻量测试配置；前端使用 Vue 3 + TypeScript + Vite，AI 部分使用 RAG 检索增强生成，并通过 OpenAI 兼容接口和 SSE/WebSocket 对接数字人服务。

### 2. 这个项目解决了什么痛点？

传统景区导览常见问题是信息分散、人工咨询压力大、游客个性化需求难满足。这个项目把静态景区信息、知识库问答、路线推荐和数字人交互整合成一个系统，让游客可以用自然语言问问题，管理员也能持续维护知识内容。

它不只是展示页面，而是形成了“知识维护 -> 智能问答 -> 数字人讲解 -> 交互数据统计”的闭环。

### 3. 你负责了哪些部分？

我主要负责后端服务架构、REST API、RAG 问答流程、知识库管理、JWT 鉴权、数字人接口集成，以及部分 Vue 管理后台和数据大屏联调。

重点工作包括知识文件导入与分片、Embedding/BM25 检索降级、OpenAI 兼容代理接口、管理员权限控制、交互日志统计和数字人会话接口。

如果面试官追问“是不是你一个人做的”，可以补充：

> 这个项目偏完整实践型项目，我对核心链路都做过实现和调试，尤其是后端分层、RAG、鉴权和数字人联调部分是我重点掌握的。

## 二、系统架构

### 4. 项目整体架构是什么？

整体采用前后端分离加后端统一托管的结构。开发阶段 Vue 通过 Vite dev server 运行，并把 `/api`、`/v1` 等请求代理到 Go 后端；生产构建后 Vue 产物输出到 `static/vue-app/`，由 Go 服务统一托管。

后端按 Handler、Service、Repository、Model 分层：

- Handler 负责 HTTP 参数解析、路由和响应。
- Service 负责业务逻辑，比如 RAG、统计、用户、景点、路线。
- Repository 负责 GORM 数据库访问。
- Model 定义数据库模型。

服务启动时会加载配置、初始化日志、JWT、数据库、自动迁移表结构、初始化 RAG 知识库，然后注册路由并启动 HTTP Server。

### 5. 为什么用这种分层？

因为项目不只是简单 CRUD，还有 RAG、知识库导入、鉴权、统计、数字人集成等业务。如果把逻辑都写在 Handler 里，会导致接口层臃肿、难测试、难维护。

分层后，Handler 只处理 HTTP，Service 聚合业务规则，Repository 屏蔽数据访问细节。现在数据库主配置已经支持 PostgreSQL，SQLite 只是本地开发配置；后续如果把 RAG 检索换成 pgvector、Milvus 或 Qdrant，影响范围会更可控。

### 6. 项目启动流程是什么？

入口在 `main.go`，启动流程如下：

1. 读取 `configs/config.yaml`，并支持 `SCENIC_GUIDE_` 前缀环境变量覆盖配置。
2. 初始化日志。
3. 初始化 JWT，校验密钥不能为空、不能是默认弱密钥、长度至少 32 位。
4. 初始化数据库连接。
5. 通过 GORM AutoMigrate 自动迁移模型。
6. 初始化 RAG 服务；如果知识库为空，会从 `knowledge/lingshan_chunks.jsonl` 导入默认知识。
7. 创建各 Repository、Service、Handler，完成依赖注入。
8. 注册静态资源、Vue SPA、REST API、OpenAI 兼容接口和数字人 WebSocket 代理。
9. 启动 HTTP Server，并支持 SIGINT/SIGTERM 优雅关闭。

## 三、RAG 和 AI

### 7. RAG 是怎么实现的？

RAG 流程分成知识导入、检索、Prompt 构造和大模型生成四步。

知识导入时，系统支持 JSONL、JSON、Markdown、TXT。JSONL/JSON 可以直接包含 `title`、`content`、`source`、`metadata` 字段；Markdown/TXT 会按段落切成约 1200 字以内的知识片段。每个片段会生成 ID、标题、来源和向量信息，然后存入数据库。

用户提问时，系统先从知识库中取出片段，通过 BM25/词面检索、Embedding 余弦相似度、加权混合、RRF 融合或轻量 rerank 计算相关度，默认筛选 TopK=8。Embedding 口径写清楚为 DashScope `text-embedding-v2`，1536 维、Cosine、Float32；没有 Key 时保留 BM25-only 和 light-rerank 作为本地可复现路径。然后把相关知识片段拼进 Prompt，要求模型优先基于景区资料回答，不能编造票价、开放时间等实时敏感信息。如果没有检索到相关知识，就走通用 Chat 模式，并明确提示知识库不足。

评估命令 `cmd/rag-eval` 支持 `bm25-local`、`embedding`、`hybrid-weighted`、`rrf-fusion`、`light-rerank` 多模式对比。RRF 的好处是融合排名而不是直接融合原始分数，所以比加权分数更不依赖归一化；light-rerank 目前用标题命中、查询词覆盖、景区实体词、来源类型等本地规则做可解释重排，不引入 Cross-Encoder。

### 8. 为什么需要 BM25 降级？

Embedding 依赖外部 API，一旦 API Key 没配置、网络异常或服务不可用，RAG 就会失效。BM25 是本地关键词检索，不依赖外部服务，所以可以作为降级方案。

这样即使没有 Embedding，系统仍然能基于关键词匹配做基础问答，保证可用性。

### 9. RAG 如何避免胡说？

主要有三层约束：

1. 检索阶段只把相关知识片段传给模型，减少模型自由发挥空间。
2. Prompt 明确要求优先使用知识库资料，不要编造票价、开放时间、演出时间、交通方式等信息；资料不足时要说明“根据当前资料无法确认”或建议咨询服务中心。
3. 如果大模型调用失败，系统会退回到基于知识片段的简单回答，而不是随便生成。

不过，这不能 100% 消除幻觉。后续可以增加引用来源展示、答案置信度、敏感问题白名单和人工审核机制。

### 10. 为什么不用真正的向量数据库？

当前项目先用 PostgreSQL/GORM 保存知识片段和业务数据，RAG 检索逻辑封装在服务层；3000 个知识切片和 300 条评测问答是合成闭集实验，用于验证本地检索链路可复现、可评估。引入 Milvus、Qdrant 或 pgvector 会增加部署复杂度，所以目前没有把向量数据库作为默认依赖。

当前设计把检索逻辑封装在 RAGService 里，后续如果知识规模扩大，可以把 Repository 或检索 Provider 替换为真正的向量数据库。

### 11. RAG 的缓存做了什么？

RAGService 里做了几类缓存：

- 查询回答缓存：相同问题短时间内直接返回，降低模型调用成本。
- Embedding 缓存：避免同一文本重复生成向量。
- 知识库缓存：避免每次检索都全量查数据库。

缓存 TTL 是 5 分钟，最大缓存数量 1000。知识库发生增删改时会清理相关缓存，避免旧知识继续影响回答。

### 12. 多轮追问是怎么做的？

用户请求可以带 `session_id`。服务端只保留最近 5 轮短期上下文，每轮回答后提取主题实体、意图类型和实时边界状态，例如“灵山大佛 / 属性追问”或“九龙灌浴 / 实时信息边界”。下一轮如果用户问“它有多高”“门票呢”“下雨呢”“现在人多吗”，系统会用这些元信息补全检索 query，但不会改前端展示的原始问题，也不会把 `rewritten_query` 暴露到公开 API。

接大模型时，prompt 会加入很短的会话上下文，让回答先承接上一轮主题；无 Key 本地 fallback 也按事实、路线、边界三类组织回答。涉及票价、开放、演出、客流、排队、无人机、宠物等实时或现场规则时，回答必须说明不能直接承诺，以官方最新公告或现场公示为准。

## 四、数字人集成

### 13. 数字人是怎么接入的？

项目里有两条数字人集成路径。

第一条是 Go 后端提供 `/api/v1/dh/*` 接口，包括创建会话、文本聊天、语音转写聊天、反馈提交和健康检查。数字人前端可以把用户文本或语音转写结果发给后端，后端调用 RAG 生成回答，再返回回答文本、情绪标签、trace_id 和可选路线信息。

第二条是 OpenAI 兼容接口 `/v1/chat/completions`，用于对接 Open-LLM-VTuber。它兼容普通 OpenAI Chat Completion 请求，也支持 `stream=true` 的 SSE 流式响应。Open-LLM-VTuber 可以像调用 OpenAI 接口一样调用本项目的 RAG 服务。

### 14. 为什么要做 OpenAI 兼容接口？

因为很多 AI 前端或数字人框架默认支持 OpenAI Chat Completions 协议。如果后端暴露兼容接口，Open-LLM-VTuber 不需要深度改造，只要把 Base URL 指向 Go 服务即可。

这样集成成本低，也方便后续替换模型或接入其他兼容 OpenAI 协议的客户端。

### 15. 情绪和 Live2D 表情怎么关联？

后端会根据回答内容做简单情绪检测，比如 happy、sadness 等，然后把情绪标签拼到回答开头，例如 `[happy] 回答内容`。前端或数字人服务读取这个标签后，可以映射到 Live2D 表情文件，比如 `exp_01` 等，实现回答内容和表情联动。

Vue 里的 Live2DStage 还会根据音频音量驱动口型参数，实现口型同步。

### 16. WebSocket 代理是做什么的？

Go 服务把 `/vtuber-ws/*path` 反向代理到本地 Open-LLM-VTuber 默认端口 `127.0.0.1:12393`。

这样前端可以通过统一路径访问数字人 WebSocket 服务，避免前端直接硬编码多个服务地址，也方便开发环境和部署环境统一入口。

## 五、鉴权和安全

### 17. 项目怎么做登录鉴权？

项目使用 JWT，但浏览器主路径不是把 token 暴露给前端。用户登录成功后，后端生成包含 `user_id`、`username`、`role` 的 token，并写入 `auth_token` HttpOnly Cookie；前端通过 Cookie 会话访问需要登录的接口，并通过 `/api/v1/user/me` 恢复用户状态。`Authorization: Bearer <token>` 仍保留给非浏览器客户端或接口调试使用。

管理员接口会额外经过 AdminMiddleware，检查 role 是否为 `admin`。普通用户只能访问自己的基础接口，景点、路线、导览内容、知识库管理、系统设置、数据大屏等写操作或后台接口需要管理员权限。

### 18. 哪些接口是公开的，哪些需要权限？

公开接口主要包括景点、路线、导览内容的 GET 查询、AI 聊天、TTS、数字人健康检查、数字人形象列表和健康检查。数字人文本聊天、语音转写聊天、反馈和会话管理需要用户或游客会话。

需要登录的接口包括游客查询创建、查询详情、用户信息等。

需要管理员权限的接口包括景点/路线/内容的新增修改删除、知识库管理、后台数据大屏、系统设置、数字人配置、用户列表管理等。

### 19. JWT 安全上做了哪些处理？

JWT 初始化时会拒绝空密钥、常见默认密钥和小于 32 字符的短密钥。Token 有过期时间，过期时间来自配置。接口层通过 AuthMiddleware 校验 HttpOnly Cookie，并兼容 Bearer token；通过 AdminMiddleware 控制管理员权限。测试里覆盖了过期 token、伪造签名、异常签名算法、普通用户访问管理员接口、知识库高风险删除接口鉴权，以及限流窗口和并发行为。

如果继续强化，可以加刷新 token、退出登录黑名单、密码复杂度策略，以及生产环境强制 HTTPS。

### 20. 注册接口为什么做限流？

注册接口容易被刷账号或爆破，所以加了基于客户端 IP 的轻量限流，默认每分钟 5 次。实现上用内存 map 记录 IP 的请求次数和窗口重置时间。

这个方案简单，适合单机演示；如果部署多实例，应该换成 Redis 之类的集中式限流。

### 21. 日志脱敏怎么做？

AI/RAG/数字人相关日志避免打印完整用户问题、完整回答、请求体和响应体，更多记录长度、状态码、错误类型、trace_id、session_id 等元信息。RAG trace 会记录 `retrieval_ms`、`embedding_ms`、`generation_ms`、`total_ms`、`provider`、`cache_hit`、`chunk_count` 和 `retrieval_mode`；超过 5 秒的请求会输出结构化 WARN 日志，方便定位是检索、生成还是外部服务慢。

这样既方便排查问题，又减少游客隐私和敏感信息泄露风险。

## 六、数据库和数据模型

### 22. 为什么主配置用 PostgreSQL，还保留 SQLite？

之前 SQLite 的优势是部署简单，适合课程项目、本地演示和快速复现。但它不适合高写并发和更接近真实多用户的场景，所以现在项目把 PostgreSQL 作为主数据库配置，并在 Docker Compose 中提供 `postgres:16-alpine` 服务。

SQLite 仍然保留为本地开发和轻量测试配置，方便没有数据库服务时快速跑通。它不是 PostgreSQL 故障后的自动接管方案，也没有双写或数据同步语义。正式表达时，我会讲 PostgreSQL + GORM + 索引 + 连接池，而不是把 SQLite 包装成生产数据库。

如果面试官追问“为什么还保留 SQLite”，可以这样答：

> 我保留 SQLite 是为了本地开发和测试便利，不是为了做数据库高可用。这个项目的主数据库配置是 PostgreSQL；真正上线还要补版本化 migration、备份恢复、慢查询监控，并把缓存和限流迁到 Redis。

### 23. 数据库表大概有哪些？

主要有用户、景点、路线、导览内容、游客问题、知识片段、交互日志、系统设置和数字人配置等。

知识片段表用于 RAG 检索，交互日志用于数据大屏统计，数字人配置用于维护数字人名称、形象、语气、欢迎语、默认表情等运营配置。

### 24. 运营数据大屏的数据从哪来？

数据大屏的数据来源主要是 `InteractionLog`。数字人聊天和 OpenAI 兼容代理在生成回答后会记录交互，包括问题、回答、情绪、响应耗时、问题类别和来源。

后台统计服务再基于这些日志聚合今日服务人数、问答次数、热门问题、关注点分布、响应延迟、满意度趋势和最近对话。

## 七、前端

### 25. 前端结构是什么？

前端主要是 Vue 3 + TypeScript + Vite。核心页面包括管理后台 AdminView、数据大屏 DashboardView、数字人相关视图 DigitalHumanView，以及 Live2DStage、KpiCard、TrendChart、DonutChart 等组件。

开发时通过 Vite 运行，生产构建输出到 Go 后端的 `static/vue-app/` 目录，由后端托管。

### 26. Vue 前端和 Go 后端如何联调？

开发环境下 Vite 配置代理，把 `/api`、`/v1`、`/static`、`/vtuber-ws` 等路径转发到对应服务。

生产环境则由 Go 后端直接托管构建后的 Vue SPA，像 `/app`、`/dashboard`、`/admin`、`/digital-human` 都返回同一个 `index.html`，再由前端路由接管。

## 八、性能和稳定性

### 27. 项目有哪些性能优化？

主要有：

- RAG 查询缓存，避免重复调用大模型。
- Embedding 缓存，避免重复生成向量。
- 知识库缓存，减少频繁全表读取。
- BM25 倒排候选索引，避免 3000 个切片下每次全量排序。
- RAG/AI 链路记录 `trace_id`、`retrieval_ms`、`generation_ms`、`total_ms`、`chunk_count`、`provider` 和 `cache_hit`，慢请求会输出结构化 WARN 日志。
- HTTP Client 设置连接池、超时和 KeepAlive。
- 知识库列表接口做数据库分页，避免全量加载后内存分页。
- 服务端设置 ReadTimeout、WriteTimeout、IdleTimeout 和 ReadHeaderTimeout，避免慢请求拖垮服务。

### 28. 如果并发高了会有什么问题？

当前项目适合单机中小规模场景。并发高时可能有几个瓶颈：

- PostgreSQL 主路径能支撑比 SQLite 更可靠的多用户读写，但生产化还需要版本化 migration、备份、慢查询监控和容量评估。
- 当前 BM25 倒排候选已经覆盖 3000 切片验证，更大规模仍应接 pgvector、Milvus、Qdrant 或专门搜索服务。
- 内存限流和缓存不能跨实例共享。
- 大模型接口调用延迟和限额会成为瓶颈。
- SSE 流式响应会占用连接。

优化方向是继续完善 PostgreSQL 迁移和索引治理，加 Redis 缓存和限流，引入向量数据库或 pgvector，异步记录日志，并对 AI 调用做队列、超时、熔断和降级。

### 29. 服务如何优雅关闭？

后端使用 `http.Server` 启动，监听 SIGINT 和 SIGTERM。收到信号后，会创建一个 10 秒超时的 context 调用 `server.Shutdown`，让已有请求尽量处理完再退出，避免直接中断连接或写坏数据。

## 九、测试和工程化

### 30. 项目怎么测试？

后端可以在 `scenic-guide/` 下运行：

```powershell
go test ./...
```

前端可以运行 TypeScript 类型检查和生产构建：

```powershell
npm run check
npm run build
```

项目还有 `make check` 作为全量检查入口，当前会串联编码检查、密钥形态扫描、后端测试和前端类型检查。数字人相关接口可在本地后端启动后用接口文档中的请求示例手动验证。

RAG 还有独立评估命令：基础样例用 `go run ./cmd/rag-eval -k 8 -fail-on-miss` 做 smoke test；真实资料评估用 `go run ./cmd/rag-eval -knowledge knowledge/real/lingshan_real_chunks.jsonl -eval knowledge/real/lingshan_real_eval_open.json -k 8 -bench -concurrency 16 -repeat 3 -retrieval-only -report-env -format json -out docs/eval-results/lingshan-real-rag-eval-bench.json`。真实资料集包含 122 个切片、203 条独立评测问答。优化前本地 retrieval-only 基线是 Recall@8 85.5%、MRR@8 0.749、关键词覆盖率 94.3%、纯检索 p50/p95 约 7ms/10ms。这个结果只代表本地检索链路，不包含外部 Embedding、大模型生成、ASR 或 TTS。

如果要看多模式对比，可以运行 `go run ./cmd/rag-eval -knowledge knowledge/real/lingshan_real_chunks.jsonl -eval knowledge/real/lingshan_real_eval_open.json -k 8 -bench -concurrency 16 -repeat 1 -retrieval-only -compare-modes bm25-local,light-rerank`。失败样例定向优化后，`bm25-local` 单轮结果为通过率 98.5%、Recall@8 94.8%、MRR@8 0.793；`light-rerank` 为通过率 99.5%、Recall@8 95.3%、MRR@8 0.802。这里要强调：提升来自 query expansion、切片问法补强和本地可解释重排，只说明当前真实资料评测集上的问法映射变稳，不能说成线上准确率或大模型能力提升。

### 31. 遇到过什么问题，怎么解决的？

一个典型问题是中文文档和源码在 Windows PowerShell 里显示乱码。排查后发现很多情况不是文件本身损坏，而是终端编码页问题。所以项目里补了 `.editorconfig` 和编码检查脚本，统一 UTF-8/LF，并通过 `npm run check:encoding` 检查替换字符和常见乱码模式，避免误判和提交脏编码。

另一个问题是数字人框架和主系统接口协议不一致。解决方式是后端额外提供 OpenAI 兼容的 `/v1/chat/completions` 接口，并支持流式 SSE，这样数字人框架可以低成本接入。

## 十、项目亮点和不足

### 32. 这个项目最大的亮点是什么？

亮点主要有三个：

1. 它不是单纯 CRUD，而是把景区业务、RAG 知识库、数字人交互和数据统计串成了闭环。
2. AI 接入做了工程化处理，包括知识导入、检索降级、缓存、Prompt 约束、OpenAI 兼容协议和日志脱敏。
3. 权限和后台运营能力比较完整，管理员可以维护知识和配置，前台游客可以直接获得智能导览服务。

### 33. 这个项目有哪些不足？

不足也比较明确：

- 当前向量检索还不是专业向量数据库，知识规模大后性能会下降。
- 目前已经补了 BM25、Embedding、加权混合、RRF 和 light-rerank 评估，但仍未接专业向量数据库或 Cross-Encoder。
- PostgreSQL 已作为主数据库配置，但还缺少版本化 migration、备份恢复和真实生产流量验证。
- 数字人和 AI 问答已支持 session 级短期上下文，会基于最近主题实体、意图类型和边界状态处理“它、那里、门票呢、下雨呢、现在人多吗”等追问，但这不是长期记忆，也没有复杂用户画像。
- AI 回答还缺少来源引用、置信度和人工审核流程。
- 限流和缓存是内存级，多实例部署时需要 Redis 等共享组件。

这样回答会显得你清楚项目边界，而不是盲目吹项目。

### 34. 这个项目会不会有过度包装的问题？

这个问题要主动降调回答：

> 我会把它定位为演示系统和作品集项目，而不是已经大规模上线的商业系统。它的价值在于我把景区内容管理、RAG 问答、OpenAI 兼容协议、数字人接入、权限控制和运营统计这些链路跑通，并且知道每一层的边界。

可以继续补充：

当前保留了 32 个知识切片和 5 条评测问答作为 smoke test，同时保留 3000/300 合成规模验证集作为内部回归口径；主口径改为真实资料评估：基于官网/政府公开资料清洗 122 个切片，并设计 203 条独立问答覆盖事实问答、游客自然问法和实时信息边界。最近一轮优化是失败样例驱动的检索优化，不是删题刷分：半天游、亲子路线、文化建筑、无人机/宠物、排队和导览服务等原失败样例被写进回归集，再通过检索扩展和切片措辞补强解决。真正落地时，仍需要继续扩大语料、增加来源引用、独立人工标注、向量数据库、rerank 和监控。

### 35. 哪些是你自己实现的，哪些是接入第三方？

自己实现的部分主要是 Go 后端分层、REST API、GORM 数据模型、JWT 鉴权、知识库管理、RAG 调用链路、BM25 降级、OpenAI 兼容代理、SSE 响应、数字人相关业务接口和 Vue 管理/看板联调。

第三方或外部能力包括 Gin、GORM、PostgreSQL 驱动、SQLite、DashScope Embedding、DeepSeek/OpenAI 兼容模型、Open-LLM-VTuber、Live2D/PixiJS。我的工作不是从零实现大模型或 Live2D 框架，而是完成协议适配、OpenAI 兼容接口、SSE/WebSocket 联调、前端二开、本地检索兜底和工程化封装。

### 36. 如果继续优化，你会怎么做？

我会从四个方向优化：

1. RAG 侧继续扩大真实资料标注，基于失败原因补切片和查询改写；数据规模扩大后再接 pgvector、Milvus、Qdrant 或 Cross-Encoder rerank。
2. 工程侧完善 PostgreSQL migration、备份、慢查询监控，并引入 Redis 做缓存和限流。
3. 数字人侧在现有短期 session 追问改写基础上，继续完善打断机制、语音识别和 TTS 链路监控。
4. 运维侧补充 Docker Compose、CI 检查、结构化日志、指标监控和错误追踪，提升可部署性。

## 十一、高频追问短答

### 37. Handler、Service、Repository 分别干什么？

Handler 处理 HTTP，Service 处理业务规则，Repository 处理数据库访问。这样职责清晰，方便测试和替换实现。

### 38. AutoMigrate 有什么风险？

AutoMigrate 适合开发和演示，但生产环境要谨慎，因为复杂字段变更、索引调整、数据迁移不一定可控。生产更适合用版本化 migration 工具。

### 39. 为什么知识库更新后要清缓存？

因为查询缓存和知识库缓存可能包含旧内容。如果管理员更新知识后不清缓存，用户可能继续拿到过期答案。

### 40. 为什么 `/v1/chat/completions` 不放在 `/api/v1` 下面？

因为它是为了兼容 OpenAI 协议和第三方客户端习惯。很多框架默认请求 `/v1/chat/completions`，保持这个路径能减少适配成本。

### 41. TTS 接口要注意什么？

文本参数要正确 URL 编码，避免中文或特殊字符破坏请求参数。同时日志里不要打印完整文本，避免泄露用户输入。

### 42. 如果 AI API 挂了怎么办？

RAG 检索仍可以拿到知识片段，系统会退回到基于片段的简单回答；如果没有知识或 API Key，也会给出明确提示，而不是让服务整体不可用。

### 43. 怎么证明这不是套壳大模型？

因为系统有自己的景区知识库、知识导入管理、检索逻辑、Prompt 约束、权限后台、数字人协议适配和交互数据统计。大模型只是生成层，项目核心是围绕景区导览业务做了完整工程链路。

## 十二、复习重点

建议重点背熟以下五块：

1. 项目整体介绍。
2. RAG 流程。
3. 数字人集成。
4. 权限安全。
5. 项目不足与优化。
