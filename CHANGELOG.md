# CHANGELOG

## 2026-06-16 19:02 - 解耦数字人文字输出和语音播放

### 变更内容
- web-vue/src/views/DigitalHumanView.vue — 游客发送问题后固定优先调用 Go 后端 `/api/v1/ai/chat` 输出文字，不再依赖 Open-LLM-VTuber WebSocket 是否返回音频消息；发送期间屏蔽旧数字人音频回放，避免语音链路阻塞文字气泡。
- web-vue/src/views/DigitalHumanView.vue — 音频错误提示增加去重，避免同一自动播放/TTS 错误重复刷屏。
- web-vue/src/views/DigitalHumanView.vue — 未点击“启用声音”前不再请求 TTS 或尝试自动播放，只提示用户启用声音，避免触发浏览器自动播放拦截。

### 原因
浏览器阻止自动播放是正常安全策略：页面必须先经过用户点击才能播放声音。但当前实现里，WebSocket 已连接时文字也会等待数字人语音链路返回；一旦浏览器拦截播放、TTS 返回空音频或 Open-LLM-VTuber 语音链路卡住，用户会感觉“连文字都不输出”。

### 影响范围
- 影响数字人游客端发送问题后的文字回答路径。
- 语音播放、口型驱动仍保留，但不能再阻塞文字回答。
- 不改变后端 RAG、LLM、知识库数据和 Open-LLM-VTuber 配置。

## 2026-06-16 18:52 - 修复数字人声音提示、口型兜底和聊天面板调整

### 变更内容
- web-vue/src/services/audioPlayback.ts — 增加音频解锁方法和播放错误回调；流式 TTS 无音频数据时自动切换到浏览器朗读；浏览器阻止播放时向页面返回明确提示；朗读 fallback 继续驱动口型脉冲。
- web-vue/src/services/ttsApi.ts — 统一使用 `getCSRFToken()` 读取 CSRF token，避免 TTS 请求与普通 API 使用不同的 token 读取逻辑。
- web-vue/src/views/DigitalHumanView.vue — 增加“启用声音”按钮和语音状态提示；TTS 失败时提示用户并自动退到浏览器朗读；桌面端聊天面板支持拖动调整宽度并记住宽度；移动端隐藏拖拽条并保留全宽聊天视图。
- static/vue-app/ — 重新构建 Vue 静态产物，让 Go 服务托管页面加载到本次前端修复。

### 原因
当前 TTS 后端可能返回 403、500 或 200 但无音频数据，前端此前会静默吞掉错误，用户无法知道需要启用声音或语音服务不可用；没有真实音频时口型不会随回答变化；桌面聊天面板宽度固定，无法按屏幕和使用习惯调整。

### 影响范围
- 影响数字人游客端的声音启用、语音播放失败提示、浏览器朗读兜底和口型驱动。
- 影响数字人页面桌面端聊天面板宽度调整和移动端布局。
- 不改变 RAG、LLM、知识库数据和 Open-LLM-VTuber 后端配置。

## 2026-06-16 18:28 - 正确优先修复数字人 RAG 流式回答

### 变更内容
- internal/handler/ai_proxy_handler.go — OpenAI 兼容 `/v1/chat/completions` 在 `stream=true` 时改为直接调用 `QueryWithRAGStreaming`，不再先生成完整回答后按字符伪流式输出；上游 LLM 失败时返回明确 SSE error；流式完成后记录交互日志并写入会话历史。
- internal/handler/ai_proxy_handler_test.go — 新增数字人流式接口回归测试，覆盖必须向上游 LLM 发送 `stream:true`，以及 LLM 失败时不能把“游客常问/问答素材”等知识库元说明拼成答案。
- internal/service/generation_service.go — 流式 LLM 扫描响应出错时返回错误；有回调时才发送 token，避免空回调导致异常。
- internal/service/rag_service.go、internal/service/retrieval_engine.go — 移除为 3 秒首段准备的快速本地检索开关，恢复配置了 LLM 时的查询改写和重排序增强，优先保证检索准确性。
- internal/service/generation_service_test.go — 移除快速首段本地答案测试，保留 LLM 失败不得静默降级为素材拼接的回归测试。
- web-vue/package.json、web-vue/package-lock.json — 版本号更新为 `0.1.1`。

### 原因
用户明确要求舍弃输出效率、回答必须正确；此前 3 秒首段方案会引入快速本地摘要，可能在完整 RAG/LLM 生成前展示不完整或不够准确的内容。数字人接口此前是伪流式，且模型失败时容易被本地素材拼接掩盖真实错误。

### 影响范围
- 影响数字人通过 Open-LLM-VTuber 调用 Go 后端的 OpenAI-compatible RAG 流式问答链路。
- 影响 RAG 检索增强策略：配置真实 LLM 后继续使用改写与重排序，不为速度跳过。
- 影响模型不可用时的用户体验：会明确失败，不再伪装成正式导游回答。
- 不改变知识库数据、不提交本地 `.env.local` 或任何 API Key。

## 2026-06-16 14:55 - 修复数字人导游口吻和聊天连续性

### 变更内容
- scripts/start-local.ps1 — 本地启动环境只在未设置 `SCENIC_GUIDE_AI_API_KEY` 时使用占位符，不再覆盖用户已有真实 LLM Key；启动日志会提示当前是保留真实 Key 还是使用本地兜底；脚本输出改为 ASCII，避免 Windows PowerShell 5 按 ANSI 解析无 BOM UTF-8 时启动失败；启动前读取 `.env`/`.env.local`，并自动把 `DEEPSEEK_API_KEY`、`QWEN_API_KEY`、`DASHSCOPE_API_KEY` 映射到项目需要的 AI 环境变量；DashScope/Qwen 默认使用 `qwen-turbo` 文本模型。
- .env.local — 写入本机 DashScope/Qwen 启动变量，供一键启动脚本读取；该文件已被 `.gitignore` 忽略，不进入版本库；本地模型设为 `qwen-turbo`，避免纯文本导游问答使用视觉模型超时。
- web-vue/src/services/audioPlayback.ts — 播放提示增加 `showText` 开关，允许音频播放时只驱动口型和表情，不重复插入聊天文本。
- web-vue/src/views/DigitalHumanView.vue — 数字人音频分片到达时即时追加到同一条助手消息，避免每个分片生成独立气泡造成“截断”；后端历史为空时也会恢复 localStorage 本地聊天记录；中断和新一轮对话会重置当前助手分片状态。
- internal/service/generation_service.go — RAG prompt 增加导游回答约束，要求直接回答游客当前问题，不复述“游客常问”“问答素材”等知识库元说明。
- internal/service/generation_service_test.go — 新增 RAG prompt 回归测试，覆盖导游口吻约束。

### 原因
一键启动脚本无条件把 `SCENIC_GUIDE_AI_API_KEY` 改成占位符，导致真实 LLM 调用失败后走本地规则兜底，回答口吻不像导游；项目只读取 `SCENIC_GUIDE_AI_API_KEY`，不会自动识别 DeepSeek/Qwen/DashScope 常见变量名；知识库里存在“游客常问”等面向素材组织的元说明，原 prompt 未禁止模型复述这些内容；Open-LLM-VTuber 返回的分段语音文本被前端当成多条助手消息，视觉上像关键内容被截断；本地 session 尚未落库或后端返回空历史时，刷新页面不会回退到 localStorage 备份；Windows PowerShell 5 会把无 BOM UTF-8 脚本按 ANSI 解析，中文日志可能被误读成语法错误。

### 影响范围
- 影响本地一键启动时 RAG 是否能使用真实 LLM 和 ScenicProfile 导游提示词。
- 影响 RAG 回答口吻和知识库素材说明的过滤方式。
- 影响数字人游客端回答展示、语音播放文本同步、刷新后的本地聊天记录恢复。
- 不改变 Open-LLM-VTuber 配置文件和后端 RAG 检索逻辑。
- `.env.local` 仅影响当前机器本地启动，不影响仓库提交内容。

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

## 2026-06-15 22:29 - 修复管理前端页面样式和追踪上报

### 变更内容
- internal/handler/routes.go — 放宽 CSP 的 `style-src` 以允许前端 UI 组件运行时样式注入；CORS 允许 `X-CSRF-Token` 请求头；追踪接口允许 `/admin/*` 管理子路由。
- internal/handler/routes_test.go — 新增追踪页面白名单测试，覆盖 `/admin/content` 等管理子路由。
- web-vue/index.html — 将 favicon 引用改为项目中已存在的 `/static/digital-human/favicon.ico`，避免页面请求不存在的 `/favicon.svg`。
- web-vue/src/router/index.ts — 页面和操作追踪在缺少 CSRF token 时跳过上报，并把管理子路由归一为 `/admin`。
- static/vue-app/ — 重新构建 Vue 静态产物，让 Go 服务托管页面加载到本次前端修复。

### 原因
当前管理前端页面被 CSP 阻止运行时样式，导致 Naive UI 空状态和菜单箭头异常放大；登录页首次追踪缺少 CSRF token 返回 403；管理子路由追踪返回 400。

### 影响范围
- 影响 Go 服务托管的 Vue 管理后台、数据大屏和其他 Naive UI 页面样式渲染。
- 影响 `/api/v1/track` 页面访问与用户操作上报。

## 2026-06-15 23:06 - 补齐官方知识库和数字人文字问答降级

### 变更内容
- main.go — 启动时不再因数据库已有旧知识而跳过默认知识文件，改为每次幂等补齐配置中的 `knowledge/lingshan_chunks.jsonl` 和 `knowledge/real/lingshan_real_chunks.jsonl`。
- cmd/demo-seed/main.go — 本地演示数据初始化时同时幂等导入旧知识库和官方真实知识库。
- cmd/demo-seed/main_test.go — 新增测试，覆盖多知识文件重复导入不会漏数据、不会重复插入。
- internal/service/knowledge_manager.go — 文件导入新增知识后刷新 RAG 检索缓存。
- web-vue/src/views/AdminKnowledge.vue — 知识库管理增加分类筛选、服务端搜索和分页，分类识别 metadata 中的 `category/topic/source_type/type/domain/filename`。
- web-vue/src/views/DigitalHumanView.vue — 数字人 WebSocket 不可用时改走 `/api/v1/ai/chat` 文本问答，TTS/数字人播放失败不影响文字回答；扫码自动提问不再无限等待数字人连接。
- web-vue/src/components/MarkdownRenderer.vue — 监听 streaming 状态结束并补渲染全文，避免数字人文本答案只显示前半段。
- internal/service/chat_session_service.go — 新本地会话尚未落库时，读取历史消息返回空列表而不是 400。
- internal/service/chat_session_service_test.go — 新增空会话读取历史的回归测试。
- web-vue/src/views/DigitalHumanView.vue — 移除不存在的单条消息保存接口调用，避免聊天时产生 `/sessions/{id}/messages` 404。
- static/vue-app/ — 重新构建 Vue 静态产物，让 Go 服务托管页面加载到本次前端修复。

### 原因
本地 SQLite 已有旧知识时，启动和 demo-seed 都不会补导入官方真实景点知识；知识库管理页只显示当前 20 条且缺少分类筛选；数字人问答与 WebSocket/TTS 强耦合，数字人服务未启动时文字问答也不可用；文本打字机结束时未补渲染全文；新本地会话加载历史和额外保存消息会产生 400/404。

### 影响范围
- 影响本地启动、演示数据初始化和 RAG 知识库补齐。
- 影响管理后台知识库管理页面。
- 影响数字人游客端文字问答、语音播放降级、扫码自动提问和会话历史加载。

## 2026-06-16 00:12 - 新增本地一键启动脚本

### 变更内容
- scripts/start-local.ps1 — 新增本地启动脚本，按顺序启动 Open-LLM-VTuber、初始化本地 SQLite 演示账号和知识库、启动 Go 服务，并输出访问地址和日志目录。
- start-local.bat — 新增双击入口，调用 PowerShell 启动脚本。

### 原因
本地运行项目时需要同时启动景区 Go 服务和已集成调教好的 Open-LLM-VTuber；手动分别执行命令容易漏启动数字人服务。

### 影响范围
- 影响本地开发/演示启动流程。
- 不改 Open-LLM-VTuber 的 `conf.yaml`，继续复用现有数字人接口配置。

## 2026-06-16 14:24 - 修复本地启动脚本端口重启

### 变更内容
- scripts/start-local.ps1 — 将端口监听进程变量从 `$pid` 改为 `$listenerPid`，避免覆盖 PowerShell 内置只读变量 `$PID`。

### 原因
`-Restart` 模式需要先停止 8080 和 12393 端口上的旧进程；使用 `$pid` 会与 PowerShell 内置变量冲突，可能导致重启流程失败。

### 影响范围
- 影响 `scripts/start-local.ps1 -Restart` 的本地一键重启流程。

## 2026-06-16 14:35 - 修复数字人 WebSocket 代理连接

### 变更内容
- internal/pkg/wsproxy.go — WebSocket 代理不再提前返回残缺的 101 响应，改为读取 Open-LLM-VTuber 后端握手响应后，将包含 `Sec-WebSocket-Accept` 的头部转发给浏览器。
- internal/pkg/wsproxy.go — 修复大于 125 字节的 WebSocket 数据帧转发，避免 `set-model-and-conf` 等大消息被截断后触发浏览器 `Invalid frame header`。
- internal/pkg/wsproxy_test.go — 新增代理握手和扩展长度数据帧回归测试，覆盖浏览器侧必须收到后端 `Sec-WebSocket-Accept`，以及大消息帧不能被截断。

### 原因
浏览器连接 `/vtuber-ws/client-ws` 时先提示 `Sec-WebSocket-Accept header is missing`，修复握手后继续提示 `Invalid frame header`；Open-LLM-VTuber 后端虽然已启动并接受连接，但 Go 代理返回给浏览器的握手响应不完整，且扩展长度数据帧转发时会丢失长度字段。

### 影响范围
- 影响数字人页面通过 Go 服务代理连接 Open-LLM-VTuber 的 WebSocket 链路。
- 不改 Open-LLM-VTuber 配置和前端聊天协议。

## 2026-06-16 14:38 - 修复数字人调用 RAG 内部接口鉴权

### 变更内容
- internal/pkg/middleware.go — 内部 API key 校验在保留 `X-API-Key` 的同时兼容 `Authorization: Bearer ...`，匹配 OpenAI 兼容客户端的默认鉴权方式。
- internal/pkg/middleware_test.go — 新增 Bearer API key 回归测试。
- scripts/start-local.ps1 — 本地启动 Go 服务和 demo-seed 时设置 `SCENIC_GUIDE_API_KEY=not-needed`，与 Open-LLM-VTuber `conf.yaml` 中的 `llm_api_key` 保持一致。

### 原因
Open-LLM-VTuber 已连上 WebSocket 后，调用 Go 后端 `/v1/chat/completions` 返回 403 `API key not configured on server`；一键启动脚本没有设置 `SCENIC_GUIDE_API_KEY`，且 Go 中间件未兼容 OpenAI 兼容客户端发送的 Bearer key。

### 影响范围
- 影响 Open-LLM-VTuber 通过 Go 的 OpenAI-compatible RAG 接口生成回答。
- 不影响普通游客端 `/api/v1/ai/chat` 文本问答接口。

## 2026-06-16 14:42 - 允许数字人 Live2D 运行时

### 变更内容
- internal/handler/routes.go — CSP 的 `script-src` 增加 `'unsafe-eval'`，允许 Live2D/Pixi 运行时加载模型。
- internal/handler/routes_test.go — 新增数字人页面 CSP 回归测试，覆盖 Live2D 运行时所需策略。

### 原因
浏览器控制台提示 `Live2D SDK unavailable ... unsafe-eval`，导致数字人页面只能显示备用动效预览，不能加载真实 Live2D 运行时。

### 影响范围
- 影响 Go 服务托管的数字人页面 Live2D 模型加载。
- 会放宽页面脚本 CSP；仅针对当前项目内已集成的 Live2D/Pixi 运行时需求。

## 2026-06-16 19:46 - 增强景区导览闭环

### 变更内容
- internal/model/models.go、internal/model/rag_models.go — 扩展景点电子围栏字段、知识库关联字段，并新增游客反馈、AI 分析结果和知识候选模型。
- internal/repository/scenic_spot.go、internal/repository/knowledge.go、internal/service/knowledge_manager.go、internal/service/rag_service.go — 保存景点围栏配置，支持知识类型、景点分类、景点组合筛选，并在知识入库时同步新字段。
- internal/service/generation_service.go、internal/service/visitor_insight_service.go — 增加 OpenAI-compatible LLM 调用能力，新增游客会话脱敏分析、满意度结果保存、知识候选生成、批准入库和拒绝流程。
- internal/handler/ai_handler.go、internal/handler/digital_human_handler.go、internal/handler/admin_handler.go、internal/handler/qr_handler.go、main.go — 增强反馈保存接口，新增管理端分析/候选接口，新增二维码 PNG/SVG 图片接口，并把分析服务注入现有路由。
- internal/repository/knowledge_filters_test.go、internal/repository/scenic_spot_geofence_test.go、internal/service/visitor_insight_service_test.go、internal/handler/qr_handler_test.go — 新增知识筛选、电子围栏保存、AI 分析与候选入库、二维码图片响应测试。
- web-vue/src/composables/useSeniorMode.ts、web-vue/src/composables/useProximityGuide.ts、web-vue/src/services/audioPlayback.ts、web-vue/src/styles/global.css — 新增游客端老年模式和跨页面电子围栏触发冷却逻辑，老年模式下降低语速并放大游客端控件。
- web-vue/src/views/MapView.vue、web-vue/src/views/DigitalHumanView.vue、web-vue/src/views/QRScanView.vue — 地图页、数字人页、扫码页接入老年模式；地图页和数字人页共享到点自动讲解开关和定位触发逻辑。
- web-vue/src/views/AdminSpots.vue、web-vue/src/views/AdminKnowledge.vue、web-vue/src/views/AdminQRCode.vue、web-vue/src/router/index.ts、web-vue/src/layout/GlobalSider.vue、web-vue/src/types/admin.ts、web-vue/src/locales/zh-CN.json、web-vue/src/locales/en-US.json — 管理端增加景点围栏配置、知识组合筛选、AI 知识候选处理和二维码管理页。
- go.mod、go.sum — 新增二维码图片生成依赖。
- static/vue-app/ — 重新构建 Vue 静态产物，包含本次前端页面和样式变更。

### 原因
需要按分阶段计划补齐景点分类筛选、电子围栏自动讲解、游客端老年模式、满意度反馈分析、知识库反馈迭代和二维码管理闭环；AI 分析必须基于现有 OpenAI-compatible 配置，不能在无 API Key 时伪造结果。

### 影响范围
- 影响景点管理、知识库管理、二维码管理、游客地图页、数字人页和扫码页。
- 影响聊天反馈、会话满意度分析、知识候选审核入库和数据大屏可用的后续分析数据来源。
- 新增数据库字段和表，启动时由现有 AutoMigrate 自动迁移。
