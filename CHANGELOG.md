# CHANGELOG

## 2026-07-19 09:00 - 收口数字人发布验收边界

### 变更内容
- `docs/digital-human-production-check.md` — 记录 Open-LLM-VTuber 连接、外部生成错误降级、移动端复测结果，并明确 Live2D Core 部署与真实 ASR/TTS 端到端测量仍是生产前置条件。
- `web-vue/index.html`、`static/favicon.svg` — 增加本地 SVG favicon，消除浏览器 404。
- `.gitignore` — 排除 Playwright 快照、运行输出和 Python 缓存等验证生成物。
- `scripts/e2e_eval.js`、`scripts/voice_latency_eval.js`、`web-vue/src/services/api.ts`、`web-vue/src/views/DashboardView.vue`、`web-vue/src/components/Live2DStage.vue` — 修复提交前 Oxlint 在本次变更文件中发现的原地排序、无效转义、变量遮蔽和事件清理问题，不改变业务行为。

### 原因
- 阶段收尾需要区分已实际验证的 Go/RAG/文字链路与依赖授权资源、真实音频设备的外部验收项，避免把备用动效或单元测试误报为完整生产能力。

### 影响范围
- 影响数字人生产验收文档和页面 favicon；不放宽 CSP，不改变 WebSocket、RAG、TTS 或认证契约。

## 2026-07-19 02:06 - 修复数字人错误音频降级与 CSP 播放

### 变更内容
- `web-vue/src/views/DigitalHumanView.vue` — 在 WebSocket 音频消息的展示文本中识别外部生成错误并切换到 Go 后端 RAG。
- `web-vue/src/services/audioPlayback.ts` — 将 Base64 WAV 转为 Blob URL 播放并及时释放对象 URL，避免 `media-src` CSP 阻止 `data:` 音频。
- `scripts/check-digital-human-runtime-i18n.mjs` — 增加 Blob 音频与禁用 `data:` 音频的静态回归检查。

### 原因
- 浏览器复测确认外部错误通过 `audio.display_text` 返回，且 `data:audio/wav` 被现有 CSP 拦截。

### 影响范围
- 影响数字人 WebSocket 错误降级、Base64 音频播放和对象 URL 生命周期；不放宽 CSP。

## 2026-07-19 02:01 - 将数字人外部生成错误降级到本地 RAG

### 变更内容
- `web-vue/src/views/DigitalHumanView.vue` — 保留待回答问题直到外部首段有效内容到达；识别 Open-LLM-VTuber 生成错误后切换到 Go 后端 RAG，并清理失速状态。
- `scripts/check-digital-human-runtime-i18n.mjs` — 将错误识别与待问题状态纳入运行时静态回归。

### 原因
- 浏览器复测发现外部服务会先接受会话、随后返回生成错误，单纯等待会话开始超时无法覆盖该失败模式。

### 影响范围
- 影响数字人外部生成失败时的回答降级；正常外部首段回答仍沿用 WebSocket 链路。

## 2026-07-19 01:53 - 清理数字人变更集静态检查问题

### 变更内容
- `web-vue/src/views/DigitalHumanView.vue` — 消除局部变量遮蔽，并使用事件监听器绑定语音识别结果，保持原行为不变。

### 原因
- 阶段性合并前 Oxlint 检查在本次变更文件中发现三处机械质量问题。

### 影响范围
- 仅影响数字人前端内部命名和事件绑定方式；不改变接口、文案或交互流程。

## 2026-07-19 01:50 - 修复数字人 WebSocket 失速兜底

### 变更内容
- `web-vue/src/views/DigitalHumanView.vue` — WebSocket 发出问题后若 2.5 秒内未收到会话开始信号，断开失速连接并切换到 Go 后端 RAG；阻止旧消息回写，重连前关闭旧 socket，并对连接提示去重。
- `scripts/check-digital-human-runtime-i18n.mjs` — 增加数字人运行时兜底关键路径静态回归检查。

### 原因
- 阶段性浏览器验收复现 Open-LLM-VTuber 已连接但不返回回答时，页面会无限等待，已有 `fallbackTimer` 从未实际启动。

### 影响范围
- 影响数字人文字问答的 WebSocket/Go 后端降级选择与重连提示；不改变正常 WebSocket 回答协议或 RAG API 契约。

## 2026-07-19 01:33 - 修复本地启动 JWT 密钥格式

### 变更内容
- `scripts/start-local.ps1` — 本地启动时生成 32 字节随机 JWT 密钥并编码为 64 位 hex，替换不再符合校验规则的旧明文常量。

### 原因
- 阶段性浏览器验收复现景区服务无法启动，错误为本地脚本注入的 JWT 密钥格式不合法。

### 影响范围
- 影响本地演示启动脚本；不改变生产密钥注入、JWT 校验或线上部署配置。

## 2026-07-19 01:27 - 升级 DOMPurify 安全补丁

### 变更内容
- `web-vue/package.json`、`web-vue/package-lock.json` — 将 DOMPurify 升级到 `^3.4.12`。

### 原因
- 发布前依赖审计发现旧版本存在 `ALLOWED_ATTR` 配置污染漏洞，且 Markdown 渲染链路会实际调用该依赖。

### 影响范围
- 影响前端 Markdown/模型回答 HTML 清洗依赖；不改变渲染接口或页面交互契约。

## 2026-07-19 01:24 - 收口消费分析脚本格式检查

### 变更内容
- `scripts/aggregate_consumption.py`、`scripts/aggregate_consumption_test.py` — 按 Ruff 统一 Python 格式，不改变聚合逻辑与测试语义。

### 原因
- 阶段性收尾审查发现两份新增脚本未通过 `ruff format --check`。

### 影响范围
- 仅影响消费分析脚本的代码格式；不改变 API、数据结构或运行行为。

## 2026-07-18 23:20 - 修正 JWT 迁移与部署文档

### 变更内容
- `README.md`、`docs/digital-human-production-check.md` — 明确 64-hex 文本在新版会先解码为 32 bytes，即使配置文本不变也会使旧 JWT 失效；补充多实例协调切换要求，并将 Windows PowerShell 启动片段改为不依赖 OpenSSL 的可复制命令。
- `docs/blog-scenic-guide-outline.md`、`docs/interview-qa.md` — 将旧的“任意 32 位字符串”口径统一为 64 hex 或 base64 解码后至少 32 bytes。
- `README.md` — 修复 `readLimitedBody` 文档标识断行。

### 原因
- 原迁移说明没有覆盖“64-hex 文本不变但签名材料变化”的兼容性风险，部分公开文档和 PowerShell 示例也仍与当前实现不一致。

### 影响范围
- 仅影响景区主系统的安全迁移、启动和面试/博客文档，不修改 JWT、CSP 或其他生产代码。

## 2026-07-18 22:55 - 收口 JWT、WebSocket、可信代理与 CSP 文档

### 变更内容
- `.env.example`、`README.md` — 按实际校验规则说明 64 hex/base64 JWT 密钥格式、生成与迁移方式，补充可信代理启动行为、WebSocket 鉴权边界和 CSP 路由隔离策略。
- `docs/digital-human-production-check.md` — 增加生产认证与代理配置，并将尚未执行的 CSP 浏览器回归明确保留为待验证项。
- `docs/api.md` — 移除已废弃的 WebSocket URL query token 契约，只保留 HttpOnly Cookie 与 `auth.token.<JWT>` 子协议。

### 原因
- 公开文档仍描述任意 32 位 JWT 字符串和 legacy `?token=`，与当前安全实现不一致，也未明确可信代理未配置时的真实启动行为。

### 影响范围
- 仅影响部署、迁移、WebSocket 接入和 CSP 验收文档，不改变运行时代码或安全策略。

## 2026-07-12 22:21 - 增强游客路线推荐语义

### 变更内容
- internal/service/visitor_experience_service.go — 将路线 `Spots` 中文景点串纳入推荐评分，补充亲子、文化、老人、拍照画像的灵山演示语义关键词，并生成包含匹配偏好、覆盖景点和游览适配性的推荐理由。
- internal/service/visitor_experience_service_test.go — 增加中文景点串路线推荐回归测试，覆盖“九龙灌浴,灵山梵宫,灵山大佛”这类真实演示数据。
- web-vue/src/views/DigitalHumanView.vue — 增加路线中文景点名解析，推荐卡片和“评价这条路线”可从中文景点串映射到已有景点评分项。
- scripts/check-visitor-loop-ui.mjs — 扩展游客闭环 UI 静态检查，要求保留中文景点名解析能力。
- static/vue-app、output/playwright — 重新生成前端静态构建产物，并补充游客推荐、评分和后台大屏验收截图。

### 原因
- 浏览器验收发现当前路线推荐虽然可用，但中文景点串没有参与个性化匹配，导致推荐理由偏弱，且游客点击“评价这条路线”时不能自动选中路线里的中文景点。

### 影响范围
- 影响游客端路线推荐排序、推荐理由展示、推荐卡片景点显示和评分入口默认选中；不改变数据库表结构、登录流程、RAG 问答、数字人语音链路和后台 CRUD。

## 2026-07-08 21:46 - 补充生产数据库与账号 API 文档

### 变更内容
- docs/production-database.md — 新增真实景区商用部署数据库说明，明确生产使用 PostgreSQL、备份恢复、数据保留、索引检查和连接池要求。
- docs/api.md — 在 Auth 接口列表中补充 `PUT /user/password` 修改密码接口说明。

### 原因
- 商用级上线需要明确用户数据长期存储策略和生产数据库运维边界，同时 API 文档需要覆盖新增修改密码接口。

### 影响范围
- 仅影响文档，不改变后端接口实现、前端页面或数据库结构。

## 2026-07-08 21:44 - 增加游客端账号弹窗

### 变更内容
- web-vue/src/components/AccountDialog.vue — 新增游客端账号弹窗，游客可升级为正式账号，正式用户和管理员可修改密码。
- web-vue/src/stores/auth.ts — 新增 `changePassword` 方法，调用 `PUT /api/v1/user/password` 并携带 CSRF token。
- web-vue/src/App.vue — 在游客端顶部导航加入账号入口，并挂载账号弹窗。
- web-vue/src/locales/zh-CN.json、web-vue/src/locales/en-US.json — 增加账号弹窗中英文文案。

### 原因
- 商用级账号闭环需要游客升级和普通用户修改密码在游客端可见可用，而不是只存在后端接口。

### 影响范围
- 影响游客端地图、数字人和扫码页顶部导航中的账号入口；不改变后台布局、登录路由守卫或游客自动登录逻辑。

## 2026-07-08 21:39 - 补齐登录页注册入口

### 变更内容
- web-vue/src/views/LoginView.vue — 增加账号登录/注册账号切换，注册表单支持用户名、密码和可选邮箱，注册成功后回到登录表单；游客免账号入口保留。
- web-vue/src/views/DigitalHumanView.vue — 将游客升级的前端密码校验对齐后端 8-128 位且包含大小写字母和数字的规则。
- web-vue/src/locales/zh-CN.json、web-vue/src/locales/en-US.json — 增加登录页注册文案，并同步游客升级密码规则提示。
- scripts/check-login-i18n.mjs — 将登录页新增注册文案纳入 i18n 检查。

### 原因
- 商用级账号闭环需要普通用户可以从登录页注册，同时保持游客免账号体验；前端密码提示必须和后端安全策略一致。

### 影响范围
- 影响登录页和数字人游客升级弹窗；不改变路由守卫、游客自动登录、后端注册接口行为或管理员登录流程。

## 2026-07-08 21:28 - 增加普通用户修改密码接口

### 变更内容
- internal/service/user_service.go — 新增 `ChangePassword` 方法，校验当前密码、复用现有密码强度规则，并用 bcrypt 写入新密码哈希。
- internal/handler/user_handler.go — 新增 `PUT /api/v1/user/password` 认证接口，拒绝游客账号，区分用户不存在、当前密码错误和新密码不合规。
- internal/pkg/messages.go — 增加修改密码成功和游客禁止修改密码的中英文消息。
- internal/handler/user_handler_test.go — 增加游客禁止改密、当前密码错误、改密成功后旧密码失效的回归测试。

### 原因
- 商用级账号闭环需要普通用户可自助修改密码，同时保持游客账号先升级再改密的边界。

### 影响范围
- 影响登录用户的账号安全接口；不改变登录、注册、游客登录、管理员用户管理和现有 Cookie 会话策略。

## 2026-07-08 20:10 - 将账号数据库设计文档改为中文

### 变更内容
- docs/superpowers/specs/2026-07-08-auth-commercial-db-design.md — 将账号体系与商用级数据库设计文档从英文改为中文，保留原有设计决策和实现范围。

### 原因
- 用户要求设计文档以中文给出，方便后续 review 和确认。

### 影响范围
- 仅影响设计文档语言表达，不改变业务代码、接口设计结论、数据库方案或测试范围。

## 2026-07-08 20:01 - 设计商用级账号与数据库方案

### 变更内容
- docs/superpowers/specs/2026-07-08-auth-commercial-db-design.md — 新增账号体系与商用级数据库设计文档，明确游客免账号、普通注册回登录页、游客升级保留会话、修改密码接口、生产 PostgreSQL、迁移备份和日志保留策略。

### 原因
- 登录与游客账号能力需要从演示入口升级为真实景区商用级账号闭环，同时需要明确用户数据长期存储和生产数据库要求。

### 影响范围
- 仅新增设计文档和变更记录，不改变后端接口、前端页面、数据库结构或静态资源。

## 2026-07-05 18:07 - 优化二维码讲解与反馈知识闭环

### 变更内容
- internal/service/rag_service.go、internal/service/generation_service.go、internal/handler/ai_handler.go — 为 RAG trace 增加最多 3 条来源引用，并在 `/api/v1/ai/chat` 响应中返回 `sources`。
- internal/service/visitor_insight_service.go — 将明确负面的用户反馈转换为待审核知识候选，候选内容保留游客问题和反馈，不自动编造正式答案。
- internal/service/generation_service_test.go、internal/service/visitor_insight_service_test.go — 增加 RAG 来源返回和低分反馈生成知识候选的回归测试。
- web-vue/src/views/QRScanView.vue、web-vue/src/views/DigitalHumanView.vue、web-vue/src/types/digitalHuman.ts、web-vue/src/locales/zh-CN.json、web-vue/src/locales/en-US.json — 扫码开始讲解时通过 `sessionStorage` 传递接口返回的讲解词，数字人直接展示并播报；普通问答消息下方展示参考来源。
- static/vue-app — 重新构建 Vue 静态产物，包含二维码直出讲解和回答来源展示的前端输出。

### 原因
- 二维码扫码讲解不需要再次触发 RAG 推理，直接输出接口返回讲解词可以降低等待时间。
- 参赛展示需要突出回答可信度和运营闭环，让游客可见回答来源，让管理员能从差评中沉淀待补充知识。

### 影响范围
- 影响二维码扫码后的数字人讲解入口、数字人普通问答消息展示、RAG chat 响应结构和低分反馈后的后台知识候选列表。
- 不改变知识候选的审核入库流程；反馈候选仍需管理员确认后才进入正式知识库。

## 2026-06-18 20:22 - 刷新地图路线区域构建产物

### 变更内容
- static/vue-app — 重新构建 Vue 静态产物，包含地图路线/服务提醒区域 i18n 后的前端输出和新 hash 资源。

### 原因
- 地图路线/服务提醒区域源码已接入 `vue-i18n`，需要同步 Go 服务托管的静态 Vue 构建产物。

### 影响范围
- 影响 Go 服务直接托管的 Vue 地图页静态资源；不改变后端接口、路线数据、提醒数据和地图渲染逻辑。

## 2026-06-18 20:21 - 增加地图路线区域 i18n 检查

### 变更内容
- scripts/check-map-routes-i18n.mjs — 新增地图页路线卡片标题、服务提醒标题和提前分钟文案 i18n 静态检查。
- web-vue/package.json、Makefile — 接入 `check:map-routes-i18n` 到前端检查命令和 `frontend-contracts`。
- web-vue/src/views/MapView.vue、web-vue/src/locales/zh-CN.json、web-vue/src/locales/en-US.json — 将地图页路线卡片标题、服务提醒标题和提前分钟文案接入 `map.*` 中英文文案。

### 原因
- 多语言计划要求继续补齐游客端地图链路；地图路线和服务提醒区域仍有用户可见硬编码中文。

### 影响范围
- 影响地图页路线/服务提醒区域用户可见 UI 文案和前端静态检查链路；不改变路线数据、提醒数据、地图渲染和后端接口。

## 2026-06-18 20:30 - 刷新地图导览控制构建产物

### 变更内容
- static/vue-app — 重新构建 Vue 静态产物，包含地图导览控制区 i18n 后的前端输出和新 hash 资源。

### 原因
- 地图导览控制区源码已接入 `vue-i18n`，需要同步 Go 服务托管的静态 Vue 构建产物。

### 影响范围
- 影响 Go 服务直接托管的 Vue 地图页静态资源；不改变后端接口、地图渲染、定位参数和自动导览流程。

## 2026-06-18 20:29 - 增加地图导览控制 i18n 检查

### 变更内容
- scripts/check-map-guide-i18n.mjs — 新增地图页导览状态、AR 提示、自动导览 toast 和老年模式按钮文案 i18n 静态检查。
- web-vue/package.json、Makefile — 接入 `check:map-guide-i18n` 到前端检查命令和 `frontend-contracts`。
- web-vue/src/views/MapView.vue、web-vue/src/locales/zh-CN.json、web-vue/src/locales/en-US.json — 将地图状态、AR 提示、自动导览提示和老年模式按钮接入 `map.*` 中英文文案。

### 原因
- 多语言计划要求继续补齐游客端地图链路；地图导览控制区仍有用户可见硬编码中文。

### 影响范围
- 影响地图页导览控制区用户可见文案和前端静态检查链路；不改变地图渲染、定位参数、自动导览触发和后端接口。

## 2026-06-18 20:28 - 刷新定位错误文案构建产物

### 变更内容
- static/vue-app — 重新构建 Vue 静态产物，包含定位错误文案 i18n 后的前端输出和新 hash 资源。

### 原因
- 定位错误文案源码已接入 `vue-i18n`，需要同步 Go 服务托管的静态 Vue 构建产物。

### 影响范围
- 影响 Go 服务直接托管的 Vue 地图页和数字人页静态资源；不改变后端接口、定位参数和自动导览流程。

## 2026-06-18 20:23 - 增加定位错误 i18n 检查

### 变更内容
- scripts/check-geolocation-i18n.mjs — 新增地理定位错误文案 i18n 静态检查，覆盖 `useGeolocation` 和地图/数字人调用点。
- web-vue/package.json、Makefile — 接入 `check:geolocation-i18n` 到前端检查命令和 `frontend-contracts`。
- web-vue/src/composables/useGeolocation.ts、web-vue/src/views/MapView.vue、web-vue/src/views/DigitalHumanView.vue、web-vue/src/locales/zh-CN.json、web-vue/src/locales/en-US.json — 将定位权限、GPS 异常、超时和未知失败提示接入 `map.gps*` 中英文文案。

### 原因
- 多语言计划要求继续补齐游客端错误文案；定位权限、GPS 异常和定位失败提示仍由 composable 写死中文。

### 影响范围
- 影响地图页和数字人页自动导览定位错误提示，以及前端静态检查链路；不改变定位监听参数、自动导览逻辑和后端接口。

## 2026-06-18 20:16 - 刷新 Live2D 舞台构建产物

### 变更内容
- static/vue-app — 重新构建 Vue 静态产物，包含 Live2D 舞台 i18n 后的前端输出和新 hash 资源。

### 原因
- Live2D 舞台源码已接入 `vue-i18n`，需要同步 Go 服务托管的静态 Vue 构建产物。

### 影响范围
- 影响 Go 服务直接托管的 Vue 数字人页面静态资源；不改变后端接口、模型资源和数字人会话逻辑。

## 2026-06-18 20:12 - 增加 Live2D 舞台 i18n 检查

### 变更内容
- scripts/check-live2d-stage-i18n.mjs — 新增 Live2D 舞台状态、加载和 fallback 文案 i18n 静态检查。
- web-vue/package.json、Makefile — 接入 `check:live2d-stage-i18n` 到前端检查命令和 `frontend-contracts`。
- web-vue/src/components/Live2DStage.vue、web-vue/src/locales/zh-CN.json、web-vue/src/locales/en-US.json — 将 Live2D 舞台状态、加载和 fallback 提示接入 `live2dStage.*` 中英文文案。

### 原因
- 多语言计划要求继续补齐游客端数字人链路；Live2D 舞台组件仍有状态和加载提示硬编码中文。

### 影响范围
- 影响游客端数字人舞台状态和加载提示文案，以及前端静态检查链路；不改变模型加载逻辑、交互事件和数字人接口。

## 2026-06-18 20:06 - 刷新 QR 扫码页构建产物

### 变更内容
- static/vue-app — 重新构建 Vue 静态产物，包含 QR 扫码页 i18n 后的前端输出和新 hash 资源。

### 原因
- QR 扫码页源码已接入 `vue-i18n`，需要同步 Go 服务托管的静态 Vue 构建产物。

### 影响范围
- 影响 Go 服务直接托管的 Vue QR 扫码页静态资源；不改变后端接口和扫码流程。

## 2026-06-18 20:01 - 增加 QR 扫码页 i18n 检查

### 变更内容
- scripts/check-qr-scan-i18n.mjs — 新增 QR 扫码页自动提问和老年模式按钮文案 i18n 静态检查。
- web-vue/package.json、Makefile — 接入 `check:qr-scan-i18n` 到前端检查命令和 `frontend-contracts`。
- web-vue/src/views/QRScanView.vue、web-vue/src/locales/zh-CN.json、web-vue/src/locales/en-US.json — 将 QR 扫码页自动提问模板和老年模式按钮文案接入 `qr.*` 中英文文案。

### 原因
- 多语言计划要求继续补齐游客端页面；QR 扫码页仍有自动提问模板和老年模式按钮硬编码中文。

### 影响范围
- 影响 QR 扫码页自动提问和老年模式按钮文案，以及前端静态检查链路；不改变后端接口和扫码数据结构。

## 2026-06-18 19:52 - 刷新游客端壳层构建产物

### 变更内容
- static/vue-app — 重新构建 Vue 静态产物，包含游客端全屏顶部栏 i18n 后的前端输出和新 hash 资源。

### 原因
- 游客端全屏壳层源码已接入 `vue-i18n`，需要同步 Go 服务托管的静态 Vue 构建产物。

### 影响范围
- 影响 Go 服务直接托管的 Vue 游客端静态资源；不改变后端接口、路由和鉴权逻辑。

## 2026-06-18 19:49 - 增加游客端壳层 i18n 检查

### 变更内容
- scripts/check-app-shell-i18n.mjs — 新增游客端全屏顶部栏文案 i18n 静态检查。
- web-vue/package.json、Makefile — 接入 `check:app-shell-i18n` 到前端检查命令和 `frontend-contracts`。
- web-vue/src/App.vue、web-vue/src/locales/zh-CN.json、web-vue/src/locales/en-US.json — 将游客端全屏顶部栏品牌、地图、数字人、管理和退出文案接入 `appShell.*` 中英文文案。

### 原因
- 多语言计划要求持续补齐新增页面和错误文案；游客端全屏壳层仍有硬编码中文且缺少专项检查。

### 影响范围
- 影响游客端地图/数字人全屏壳层文案和前端静态检查链路；不改变路由、鉴权和页面布局。

## 2026-06-18 19:46 - 刷新登录页构建产物

### 变更内容
- static/vue-app — 重新构建 Vue 静态产物，包含登录页游客入口 i18n 后的前端输出和新 hash 资源。

### 原因
- 登录页源码已接入游客入口中英文文案，需要同步 Go 服务托管的静态 Vue 构建产物。

### 影响范围
- 影响 Go 服务直接托管的 Vue 登录页静态资源；不改变后端接口和登录流程。

## 2026-06-18 19:43 - 增加登录页 i18n 检查

### 变更内容
- scripts/check-login-i18n.mjs — 新增登录页游客登录文案 i18n 静态检查。
- web-vue/package.json、Makefile — 接入 `check:login-i18n` 到前端检查命令和 `frontend-contracts`。
- web-vue/src/views/LoginView.vue、web-vue/src/locales/zh-CN.json、web-vue/src/locales/en-US.json — 将游客登录按钮和成功/失败提示接入 `login.*` 中英文文案。

### 原因
- 多语言计划要求持续补齐新增页面和错误文案；登录页游客登录入口仍缺少专项检查。

### 影响范围
- 影响登录页游客登录入口文案和前端静态检查链路；不改变登录流程、鉴权接口和路由跳转。

## 2026-06-18 19:39 - 刷新知识库管理构建产物

### 变更内容
- static/vue-app — 重新构建 Vue 静态产物，包含知识库管理页 i18n 后的前端输出和新 hash 资源。

### 原因
- 知识库管理页源码已接入 `vue-i18n`，需要同步 Go 服务托管的静态 Vue 构建产物。

### 影响范围
- 影响 Go 服务直接托管的 Vue 管理后台静态资源；不改变源码逻辑和后端接口。

## 2026-06-18 19:37 - 补齐知识库管理页 i18n 源码

### 变更内容
- web-vue/src/views/AdminKnowledge.vue — 将知识库管理页的分类、筛选、表单、上传、列表、AI 分析记录和知识候选等用户可见文案接入 `adminKnowledge.*` locale key。
- web-vue/src/locales/zh-CN.json、web-vue/src/locales/en-US.json — 新增知识库管理页中英文文案。
- scripts/check-admin-knowledge-i18n.mjs、scripts/check-admin-knowledge-insights.mjs、web-vue/package.json、Makefile — 新增并接入知识库管理页 i18n 与 AI 分析区检查命令。

### 原因
- 多语言路线图要求继续补齐新增管理页和错误文案；知识库管理页仍有硬编码中文和 AI 分析区域检查未覆盖 locale key。

### 影响范围
- 影响 Vue 管理端知识库管理页、前端 i18n 检查脚本和 `make frontend-contracts` 检查链路；不改变后端接口和运行时数据结构。

## 2026-06-18 17:26 - 新增账号切换交接记录

### 变更内容
- docs/HANDOFF_2026-06-18.md — 新增当前分支、远端提交、已完成模块、已验证命令、未提交知识库 i18n 草稿和下一账号接手步骤。

### 原因
- 用户准备切换账号，后续账号无法看到此前聊天记录，需要把最新进度和下一步操作固化到仓库文档中。

### 影响范围
- 影响项目交接文档；不改变运行时代码。当前知识库管理页 i18n 仍是未提交草稿，需下一步验证和提交。

## 2026-06-18 17:25 - 刷新景点管理构建产物

### 变更内容
- static/vue-app — 重新构建 Vue 静态产物，包含景点管理页国际化后的前端输出和新 hash 资源。

### 原因
- 景点管理页源码已接入 `vue-i18n`，需要同步 Go 服务托管的静态 Vue 构建产物。

### 影响范围
- 影响 Go 服务直接托管的 Vue 管理后台静态资源；不改变源码逻辑和后端接口。

## 2026-06-18 17:24 - 补齐景点管理国际化

### 变更内容
- scripts/check-admin-spots-i18n.mjs、web-vue/package.json、Makefile — 新增景点管理页 i18n 静态检查，并接入前端契约检查和 `make check`。
- web-vue/src/views/AdminSpots.vue — 将页面标题、说明、按钮、表格列、分类标签、价格/二维码/电子围栏状态、抽屉表单、开关说明、占位符和校验文案接入 `vue-i18n`。
- web-vue/src/locales/zh-CN.json、web-vue/src/locales/en-US.json — 新增 `adminSpots` 中英文文案。

### 原因
- 景点管理页包含二维码和电子围栏配置，但主体文案仍硬编码中文，未落实路线图中新增管理页和错误文案持续补齐 i18n 的要求。

### 影响范围
- 影响景点管理页中英文切换展示、表单校验提示和统一验证入口；不改变景点 CRUD、二维码、电子围栏接口和保存到后端的分类值。

## 2026-06-18 17:23 - 刷新景点删除修正构建产物

### 变更内容
- static/vue-app — 重新构建 Vue 静态产物，包含景点管理页删除重复确认修正后的前端输出和新 hash 资源。

### 原因
- 景点管理页删除交互源码已调整，需要同步 Go 服务托管的静态 Vue 构建产物。

### 影响范围
- 影响 Go 服务直接托管的 Vue 管理后台静态资源；不改变源码逻辑和后端接口。

## 2026-06-18 17:22 - 修正景点删除重复确认

### 变更内容
- web-vue/src/views/AdminSpots.vue — 移除删除按钮外层 `NPopconfirm`，改为直接调用 `useCrudTable.handleDelete` 的统一删除确认流程。

### 原因
- 景点管理页删除按钮此前会先弹 `NPopconfirm`，确认后又触发通用删除确认弹窗，导致管理员需要确认两次，属于页面交互功能错误。

### 影响范围
- 影响景点管理页删除景点的确认交互；不改变删除接口、权限和删除成功后的刷新逻辑。

## 2026-06-18 17:21 - 刷新讲解内容构建产物

### 变更内容
- static/vue-app — 重新构建 Vue 静态产物，包含讲解内容管理页国际化后的前端输出和新 hash 资源。

### 原因
- 讲解内容管理页源码已接入 `vue-i18n`，需要同步 Go 服务托管的静态 Vue 构建产物。

### 影响范围
- 影响 Go 服务直接托管的 Vue 管理后台静态资源；不改变源码逻辑和后端接口。

## 2026-06-18 17:20 - 补齐讲解内容国际化

### 变更内容
- scripts/check-admin-content-i18n.mjs、web-vue/package.json、Makefile — 新增讲解内容管理页 i18n 静态检查，并接入前端契约检查和 `make check`。
- web-vue/src/views/AdminContent.vue — 将页面标题、按钮、表格列、内容类型标签、音频状态、抽屉表单、占位符和校验文案接入 `vue-i18n`。
- web-vue/src/locales/zh-CN.json、web-vue/src/locales/en-US.json — 新增 `adminContent` 中英文文案。

### 原因
- 讲解内容管理页接口和页面已存在，但主体文案仍硬编码中文，未落实路线图中新增管理页和错误文案持续补齐 i18n 的要求。

### 影响范围
- 影响讲解内容管理页中英文切换展示、表单校验提示和统一验证入口；不改变讲解内容 CRUD 接口和保存到后端的内容类型值。

## 2026-06-18 17:19 - 刷新路线管理构建产物

### 变更内容
- static/vue-app — 重新构建 Vue 静态产物，包含路线管理页国际化后的前端输出和新 hash 资源。

### 原因
- 路线管理页源码已接入 `vue-i18n`，需要同步 Go 服务托管的静态 Vue 构建产物。

### 影响范围
- 影响 Go 服务直接托管的 Vue 管理后台静态资源；不改变源码逻辑和后端接口。

## 2026-06-18 17:18 - 补齐路线管理国际化

### 变更内容
- scripts/check-admin-routes-i18n.mjs、web-vue/package.json、Makefile — 新增路线管理页 i18n 静态检查，并接入前端契约检查和 `make check`。
- web-vue/src/views/AdminRoutes.vue — 将路线管理页标题、按钮、表格列、难度标签、时长单位、抽屉表单、占位符和校验文案接入 `vue-i18n`。
- web-vue/src/locales/zh-CN.json、web-vue/src/locales/en-US.json — 新增 `adminRoutes` 中英文文案。

### 原因
- 路线管理接口和页面已存在，但管理页主体文案仍硬编码中文，未落实路线图中新增页面和错误文案持续补齐 i18n 的要求。

### 影响范围
- 影响路线管理页中英文切换展示、表单校验提示和统一验证入口；不改变路线 CRUD 接口和字段结构。

## 2026-06-18 17:08 - 刷新知识库分析构建产物

### 变更内容
- static/vue-app — 重新构建 Vue 静态产物，包含知识库 AI 分析记录列表后的前端输出和新 hash 资源。

### 原因
- 知识库管理页源码已接入 AI 分析记录列表，需要同步 Go 服务托管的静态 Vue 构建产物。

### 影响范围
- 影响 Go 服务直接托管的 Vue 管理后台静态资源；不改变源码逻辑和后端接口。

## 2026-06-18 17:02 - 补齐知识库 AI 分析记录

### 变更内容
- scripts/check-admin-knowledge-insights.mjs、web-vue/package.json、Makefile — 新增知识库 AI 分析记录静态检查，并接入前端契约检查和 `make check`。
- web-vue/src/types/admin.ts — 新增 `VisitorInsightAnalysis` 前端类型，对齐后端 `/admin/insights/analyses` 响应。
- web-vue/src/views/AdminKnowledge.vue — 接入 `/admin/insights/analyses?page=1&page_size=5`，在知识库页展示最近 AI 分析记录、满意度、关注点和负面原因，并在新分析完成后同步刷新分析记录和知识候选。

### 原因
- 路线图要求完善运营闭环；后端和 API 文档已有 AI 分析记录接口，但管理端只生成候选、审核候选，没有展示已生成的分析记录，闭环不完整。

### 影响范围
- 影响知识库管理页的 AI 分析记录可见性和前端契约检查；不改变 AI 分析、候选入库、拒绝接口和数据库结构。

## 2026-06-18 16:48 - 刷新系统设置构建产物

### 变更内容
- static/vue-app — 重新构建 Vue 静态产物，包含系统设置页国际化后的前端输出和新 hash 资源。

### 原因
- 系统设置页源码和 locale 已接入 i18n，需要同步 Go 服务托管的静态 Vue 构建产物。

### 影响范围
- 影响 Go 服务直接托管的 Vue 管理后台静态资源；不改变源码逻辑和后端接口。

## 2026-06-18 16:42 - 补齐系统设置国际化

### 变更内容
- scripts/check-admin-settings-i18n.mjs、web-vue/package.json、Makefile — 新增系统设置页 i18n 静态检查，并接入前端契约检查和 `make check`。
- web-vue/src/views/AdminSettings.vue — 将系统设置页标题、分区、表单标签、占位符、备份频率选项、按钮、校验和消息提示接入 `vue-i18n`。
- web-vue/src/locales/zh-CN.json、web-vue/src/locales/en-US.json — 新增 `adminSettings` 中英文文案。

### 原因
- 系统设置接口和文档已存在，但页面仍有大量硬编码中文和未接入 locale 的错误文案，不符合路线图中持续补齐新增页面 i18n 的要求。

### 影响范围
- 影响系统设置页中英文切换展示、前端表单提示和统一验证入口；不改变 `/admin/settings` 接口、存储键和值格式。

## 2026-06-18 16:32 - 刷新用户管理构建产物

### 变更内容
- static/vue-app — 重新构建 Vue 静态产物，包含用户管理页校准和国际化后的前端输出和新 hash 资源。

### 原因
- 用户管理页源码和 locale 已移除旧 API 未就绪提示并接入 i18n，需要同步 Go 服务托管的静态 Vue 构建产物。

### 影响范围
- 影响 Go 服务直接托管的 Vue 管理后台静态资源；不改变源码逻辑和后端接口。

## 2026-06-18 16:24 - 校准用户管理页实现状态

### 变更内容
- scripts/check-admin-users-i18n.mjs、web-vue/package.json、Makefile — 新增用户管理页 i18n 和旧 API 文案静态检查，并接入前端契约检查和 `make check`。
- web-vue/src/views/AdminUsers.vue — 移除“用户管理 API 尚未就绪”的旧分支，按已实装的 `/admin/users` CRUD 接口直接加载用户列表，并将页面标题、表格列、角色、表单、按钮和校验文案接入 `vue-i18n`。
- web-vue/src/locales/zh-CN.json、web-vue/src/locales/en-US.json — 新增 `adminUsers` 中英文文案，并将用户名校验提示校准为后端实际的 3-32 位字母、数字或下划线规则。

### 原因
- 后端和 API 文档已经提供管理端用户 CRUD，但前端仍提示 API 未就绪，属于产品功能状态与代码实现不一致；路线图也要求继续补齐新增管理页和错误文案的 i18n。

### 影响范围
- 影响用户管理页的加载路径、中英文切换展示、表单前端校验和统一验证入口；不改变用户管理后端接口和权限模型。

## 2026-06-18 16:12 - 刷新感受度报告构建产物

### 变更内容
- static/vue-app — 重新构建 Vue 静态产物，包含感受度报告页国际化后的前端输出和新 hash 资源。

### 原因
- 感受度报告页源码和 locale 已接入 i18n，需要同步 Go 服务托管的静态 Vue 构建产物。

### 影响范围
- 影响 Go 服务直接托管的 Vue 管理后台静态资源；不改变源码逻辑和后端接口。

## 2026-06-18 16:05 - 补齐感受度报告国际化

### 变更内容
- scripts/check-admin-reports-i18n.mjs、web-vue/package.json、Makefile — 新增感受度报告页 i18n 静态检查，并接入前端契约检查和 `make check`。
- web-vue/src/views/AdminReports.vue — 将报告页标题、周期、KPI、图表标题、空状态和消息提示接入 `vue-i18n`。
- web-vue/src/locales/zh-CN.json、web-vue/src/locales/en-US.json — 新增 `adminReports` 中英文文案。

### 原因
- 路线图要求多语言 i18n 持续补齐新增页面和错误文案；审查发现感受度报告页仍存在大量硬编码中文。

### 影响范围
- 影响游客感受度报告页中英文切换展示、前端静态检查和统一验证入口；不改变报告接口、统计逻辑和无数据边界。

## 2026-06-18 15:58 - 刷新数字人形象配置构建产物

### 变更内容
- static/vue-app — 重新构建 Vue 静态产物，包含数字人形象配置页国际化后的前端输出和新 hash 资源。

### 原因
- 数字人形象配置页源码和 locale 已接入 i18n，需要同步 Go 服务托管的静态 Vue 构建产物。

### 影响范围
- 影响 Go 服务直接托管的 Vue 管理后台静态资源；不改变源码逻辑和后端接口。

## 2026-06-18 15:50 - 补齐数字人形象配置国际化

### 变更内容
- scripts/check-admin-avatar-i18n.mjs、web-vue/package.json、Makefile — 新增数字人形象配置页 i18n 静态检查，并接入前端契约检查和 `make check`。
- web-vue/src/views/AdminAvatar.vue — 将数字人预览、形象与声音设定、表单标签、选项、按钮、校验和消息提示接入 `vue-i18n`。
- web-vue/src/locales/zh-CN.json、web-vue/src/locales/en-US.json — 新增 `adminAvatar` 中英文文案。

### 原因
- 路线图要求多语言 i18n 持续补齐新增页面和错误文案；审查发现数字人形象配置页仍存在大量硬编码中文。

### 影响范围
- 影响数字人形象配置页中英文切换展示、前端静态检查和统一验证入口；不改变数字人配置接口和保存到后端的配置值。

## 2026-06-18 15:40 - 刷新二维码管理构建产物

### 变更内容
- static/vue-app — 重新构建 Vue 静态产物，包含二维码管理页国际化后的前端输出和新 hash 资源。

### 原因
- 二维码管理页源码和 locale 已接入 i18n，需要同步 Go 服务托管的静态 Vue 构建产物。

### 影响范围
- 影响 Go 服务直接托管的 Vue 管理后台静态资源；不改变源码逻辑和后端接口。

## 2026-06-18 15:35 - 补齐二维码管理国际化

### 变更内容
- scripts/check-admin-qrcode-i18n.mjs、web-vue/package.json、Makefile — 新增二维码管理页 i18n 静态检查，并接入前端契约检查和 `make check`。
- web-vue/src/views/AdminQRCode.vue — 将二维码管理页标题、KPI、表格列、状态标签、操作按钮、抽屉表单、空状态和消息提示接入 `vue-i18n`。
- web-vue/src/locales/zh-CN.json、web-vue/src/locales/en-US.json — 新增 `adminQRCode` 中英文文案。

### 原因
- 路线图要求多语言 i18n 持续补齐新增页面和错误文案；审查发现新补齐的二维码管理页仍存在大量硬编码中文。

### 影响范围
- 影响二维码管理页中英文切换展示、前端静态检查和统一验证入口；不改变二维码后端接口和缓存逻辑。

## 2026-06-18 15:25 - 校准数字人联调文档

### 变更内容
- docs/digital-human-integration.md、docs/digital-human-runbook.md、docs/digital-human-production-check.md — 将数字人主入口校准为 Go 托管的 Vue 页面，补充受保护 `/api/v1/dh/*` POST 接口需要 `auth_token` Cookie 和 `X-CSRF-Token`，移除不存在的 `configs/digital_human.yaml` 配置口径。
- scripts/check-digital-human-docs.mjs、web-vue/package.json、Makefile — 新增数字人文档漂移检查，并接入前端契约检查和 `make check`。

### 原因
- 审查发现数字人文档仍按旧 Open-LLM-VTuber 直连入口和无登录 curl 示例描述，和当前 Vue 主路径、Cookie 鉴权、CSRF 防护以及 Go WebSocket 代理实现不一致。

### 影响范围
- 影响数字人联调文档、生产检查说明和统一验证入口；不改变数字人运行时代码和接口行为。

## 2026-06-18 14:43 - 补齐游客问题页面国际化

### 变更内容
- scripts/check-admin-query-i18n.mjs、web-vue/package.json — 新增游客问题页面 i18n 静态检查脚本和 npm 脚本入口，用于防止新增管理页只接入导航翻译而页面文案仍硬编码中文。
- web-vue/src/views/AdminQueries.vue — 将游客问题管理页标题、筛选、表格列、操作按钮、表单、空状态、删除确认和消息提示接入 `vue-i18n`。
- web-vue/src/locales/zh-CN.json、web-vue/src/locales/en-US.json — 新增 `adminQueries` 中英文文案。
- static/vue-app — 重新构建 Vue 静态产物，包含游客问题页面国际化。

### 原因
- 路线图要求多语言 i18n 持续补齐新增页面和错误文案；审查发现新补的游客问题管理页主体文案、表格列、操作按钮和消息提示仍为硬编码中文。

### 影响范围
- 影响游客问题管理页、中英文 locale、前端 i18n 校验脚本和 Vue 构建产物。

## 2026-06-18 14:47 - 补齐应用容器健康检查

### 变更内容
- scripts/check-compose-healthcheck.mjs、Makefile — 新增 Docker Compose 应用健康检查静态校验，并接入 `make check`。
- Dockerfile、docker-compose.yml — 在运行镜像中加入 `wget`，为 `scenic-guide` 服务增加探测 `/health` 的应用容器 healthcheck。

### 原因
- 路线图将“部署验证与运维补强”列为长期计划，并明确 Docker Compose 后续要补健康检查；审查发现 Compose 只有 PostgreSQL healthcheck，应用容器没有探测 `/health`。

### 影响范围
- 影响 Docker 运行镜像、Compose 应用服务健康状态和 `make check`；不改变服务端 `/health` 响应路径。

## 2026-06-18 14:22 - 补齐游客问题管理闭环

### 变更内容
- scripts/check-admin-query-management.mjs、web-vue/package.json — 新增游客问题管理闭环静态检查脚本和 npm 脚本入口。
- internal/handler/visitor_query_handler_test.go — 新增游客问题路由注册和命中回归测试，覆盖 `/queries/unanswered` 与通配查询路由可同时注册且命中未回答列表处理器。
- web-vue/src/types/admin.ts、web-vue/src/views/AdminQueries.vue — 新增游客问题类型和管理端页面，支持全部/未回答切换、刷新、回复编辑、处理状态更新和删除。
- web-vue/src/router/index.ts、web-vue/src/layout/GlobalSider.vue、web-vue/src/locales/zh-CN.json、web-vue/src/locales/en-US.json — 接入游客问题路由、数字人中心侧边栏入口和中英文导航文案。
- README.md、PROJECT_OVERVIEW.md、PROJECT_DOCUMENTATION.md、docs/api.md、docs/ROADMAP.md、docs/architecture.md、CODE_WIKI.md — 同步游客问题管理闭环、当前技术栈、景区 Profile API、Go/PostgreSQL 运行口径、实际路由清单和路线图记录。
- PROJECT_DOCUMENTATION.md、docs/LIVE2D_FIX_PLAN.md、docs/ROADMAP.md — 清理本次触达文档中的尾随空格，保证 `git diff --check` 不再报错。
- static/vue-app — 重新构建 Vue 静态产物，包含游客问题管理页和导航入口。

### 原因
- 审查发现游客问题后端接口和 API 文档已存在，但 Vue 管理端没有页面、路由和侧边栏入口；需要补齐运营人员处理游客问题的前端闭环，并用静态检查固定该类缺口。

### 影响范围
- 影响数字人中心管理端导航、游客问题管理页、前端校验脚本、游客问题路由测试、项目说明文档和 Vue 构建产物；不改变游客问题后端接口路径和数据表结构。

## 2026-06-18 14:11 - 修正到点导览冷却重置

### 变更内容
- web-vue/src/composables/useProximityGuide.ts — `resetTriggered` 在清理当前页面已触发景点集合时，同步删除本地冷却记录。
- static/vue-app — 重新构建 Vue 静态产物，包含到点导览冷却重置修正。

### 原因
- GPS 主动导览打开时会调用 `resetTriggered`，但原实现只清理内存集合，不清理 localStorage 中的冷却时间；游客重新开启到点导览后仍可能被旧冷却记录阻止触发。

### 影响范围
- 影响游客地图页和数字人页的到点讲解重新开启体验；不改变距离计算、景点半径和单个景点冷却字段定义。

## 2026-06-18 14:08 - 接通数字人历史会话搜索

### 变更内容
- internal/repository/chat_session_repo.go — 增加按会话主键批量查询接口，用于搜索结果补充会话上下文。
- internal/service/chat_session_service.go — 将跨会话消息搜索结果扩展为包含 `session_id` 和 `session_title` 的响应结构。
- internal/handler/session_handler_test.go — 新增回归测试，确保 `/api/v1/sessions/search` 返回可跳转会话所需的上下文字段。
- web-vue/src/stores/session.ts — 为历史搜索结果补充 `session_id`、`session_title` 类型。
- web-vue/src/views/DigitalHumanView.vue — 搜索栏接入后端历史消息搜索，并合并当前会话本地结果；点击搜索结果可切换到对应历史会话。
- static/vue-app — 重新构建 Vue 静态产物，包含历史会话搜索 UI 修正。

### 原因
- 路线图要求完善跨访问历史和会话搜索体验；审查发现后端已有搜索接口和前端 store 方法，但数字人页面只搜索当前内存消息，历史会话搜索没有真正接入 UI。

### 影响范围
- 影响数字人页历史消息搜索、会话切换体验和会话搜索 API 响应字段；不改变消息保存格式和会话列表接口。

## 2026-06-18 14:00 - 补齐前端校验工具链

### 变更内容
- web-vue/package.json、web-vue/package-lock.json — 补齐 ESLint、Vue ESLint、TypeScript ESLint、Prettier 相关开发依赖，增加 `esbuild` override 并通过 `npm audit fix` 清理前端依赖审计问题。
- web-vue/.eslintrc.cjs — 拆分 lint 与 Prettier 格式职责，保留 `eslint-config-prettier`，关闭与 Vue 宏、TypeScript 全局和流式循环不匹配的误报规则。
- web-vue/src/components/KpiCard.vue、web-vue/src/components/Live2DStage.vue、web-vue/src/components/MarkdownRenderer.vue、web-vue/src/layout/GlobalSider.vue、web-vue/src/views/AdminKnowledge.vue、web-vue/src/views/AdminView.vue、web-vue/src/views/DigitalHumanView.vue、web-vue/src/views/MapView.vue — 清理 lint 暴露的未使用导入、未使用变量和无处理函数的事件绑定。
- scripts/check-encoding.mjs — 增加非法 ASCII 控制字符检测，避免文档中再次出现吞掉函数名首字母的隐藏控制字符。
- README.md、PROJECT_OVERVIEW.md — 补充可运行的 `npm run lint`、`npm run check:data-boundaries` 验证命令，并修正控制字符导致的函数名/路径乱码。
- static/vue-app — 重新构建 Vue 静态产物，包含 lint 清理后的前端源码输出。

### 原因
- 审查发现项目已有 `.eslintrc.cjs`、Prettier 配置和 `lint` 脚本，但缺少对应依赖，导致 `npm run lint` 无法执行；文档中还残留不可见控制字符，造成函数名和路径显示错误。

### 影响范围
- 影响前端开发校验、依赖锁文件、编码检查脚本和项目验证文档；不改变运行时业务接口。

## 2026-06-18 13:42 - 收敛大屏演示数据边界

### 变更内容
- web-vue/src/views/DashboardView.vue — 移除运营大屏对硬编码热门景点、问答准确率、人流热力、活动状态、终端状态和知识缺口的演示兜底；无真实接口来源时显示空状态，知识库条目为 0 时显示真实 0。
- web-vue/src/constants/scenicVisualization.ts — 删除已无引用的 `DASHBOARD_FALLBACK` 和 `REPORT_FALLBACK` 演示数据常量。
- scripts/check-dashboard-data-boundaries.mjs — 新增大屏数据边界检查，防止后续重新把演示兜底作为真实运营数据展示。
- web-vue/package.json — 增加 `check:data-boundaries` 脚本入口，便于复跑大屏数据边界检查。
- README.md — 修正数据大屏功能描述，移除未实装的 RAG 评估指标可视化和 30 秒自动刷新表述。
- static/vue-app — 重新构建 Vue 静态产物，包含数据大屏空状态和演示数据边界修正。

### 原因
- 审查发现运营大屏仍把前端演示数据展示成真实运营态势，且 README 描述与当前代码不一致，容易误导为已接入实时人流、终端状态和自动刷新能力。

### 影响范围
- 影响管理端数据大屏、前端演示数据常量、项目功能概览和新增边界检查脚本；不改变已有后端统计接口。

## 2026-06-18 13:29 - 修正感受度报告周期与数据边界

### 变更内容
- internal/handler/admin_handler.go、internal/handler/admin_handler_test.go — `/admin/reports/visitor` 读取 `period=7d|30d`，新增回归测试覆盖 30 天报告必须纳入 20 天前交互，以及无数据时不得伪造图表数据。
- internal/service/stats_service.go — 感受度报告按 7/30 天窗口统计总量、关注点、情绪、热门时段和趋势；无真实交互时返回空图表数据与无数据建议，不再返回固定演示指标。
- web-vue/src/views/AdminReports.vue — 报告页 KPI 文案随 7/30 天周期变化，并移除负面原因、人群画像、路线满意度和词云的前端演示兜底。
- static/vue-app — 重新构建 Vue 静态产物，包含感受度报告页周期和无数据展示修正。
- docs/api.md、PROJECT_DOCUMENTATION.md — 补充报告周期参数、无数据边界和管理端 QR/洞察接口清单。

### 原因
- 审查发现报告页已有 7/30 天切换，但后端固定按近 7 天统计；同时无真实交互时会展示固定演示数据，容易把演示指标误认为真实运营报告。

### 影响范围
- 影响管理端游客感受度报告、统计服务、报告接口文档和项目接口总览；不改变交互日志写入链路。

## 2026-06-18 13:16 - 补齐二维码管理闭环

### 变更内容
- internal/handler/qr_handler.go、internal/handler/qr_handler_test.go — 修复二维码配置改码后旧二维码讲解缓存仍可命中的问题，并新增回归测试覆盖旧码应返回 404。
- web-vue/src/views/AdminQRCode.vue — 增加二维码配置编辑抽屉，支持后台直接修改二维码 ID、启用状态和扫码讲解词，调用已有 `/admin/qr/spots/:id` 接口保存。
- static/vue-app — 重新构建 Vue 静态产物，包含二维码管理页编辑入口。
- internal/handler/digital_human_avatar_test.go — 增加游客升级后仍可读取升级前会话消息的回归测试，锁定同账号升级保留会话的当前设计。
- docs/api.md、docs/ROADMAP.md、PROJECT_OVERVIEW.md — 补充二维码公开和管理接口文档，校准游客升级会话口径，并替换过期的 2026-06-10 开发进展快照。

### 原因
- 审查发现二维码后端已有更新接口，但管理页缺少编辑入口；同时改码后只清理新二维码缓存，旧码仍可能返回旧讲解内容。路线图中的“会话迁移”表述也与当前原地升级实现不一致。

### 影响范围
- 影响管理端二维码配置、游客扫码讲解缓存一致性、游客升级后的会话保留回归保障，以及 API/路线图/项目总览文档口径。

## 2026-06-18 12:51 - 补齐会话消息保存与 Live2D 遗留项

### 变更内容
- internal/service/chat_session_service.go、internal/service/chat_session_service_test.go — 增加单条会话消息保存逻辑与回归测试，支持前端按会话补写用户/助手消息。
- internal/handler/session_handler.go、internal/handler/session_handler_test.go — 增加 `POST /api/v1/sessions/:session_id/messages` 接口与鉴权、归属、参数校验测试。
- web-vue/src/components/Live2DStage.vue — 使用统一 resize 监听路径同步 Live2D 布局，并删除残留的随机 motion 索引函数。
- web-vue/src/components/ThinkingIndicator.vue — 改为纯展示组件，由父组件负责是否渲染。

### 原因
- 前端已调用会话消息保存接口但后端缺少对应路由，导致聊天历史无法稳定落库；Live2D 遗留计划中仍有 resize 监听和展示组件职责不一致的半实装项。

### 影响范围
- 影响游客端数字人会话历史保存、后端会话消息接口、Live2D 舞台窗口尺寸同步和思考状态展示。

## 2026-06-18 13:18 - 校准产品计划与文档实现状态

### 变更内容
- web-vue/src/views/DigitalHumanView.vue — `persistMessage` 在写入 localStorage 和 Pinia 后调用 `sessionStore.saveMessage`，让数字人会话消息真正写入后端会话接口。
- docs/ROADMAP.md — 移除微信小程序跨平台计划；将 WebRTC 升级改为现有 SSE、`/vtuber-ws/*`、流式 TTS、打断和会话链路完善；将 Docker 一键部署改为已有编排后的部署验证与运维补强。
- PROJECT_OVERVIEW.md — 同步路线图和当前数字人模型状态，修正 shizuku 已清理、微信小程序、WebRTC 替换、Docker 待实现等过时表述。
- docs/api.md — 补充游客登录升级、头像偏好、会话消息保存、TTS stream、数字人形象列表和游客洞察/知识候选管理接口。
- docs/LIVE2D_FIX_PLAN.md — 将旧待修指南改为 Live2D 遗留项完成记录，说明 manifest 路径在当前 Vite base 下保持 `/static/vue-app/manifest.json`。
- PROJECT_DOCUMENTATION.md — 重写为当前实现说明，校准 PostgreSQL/SQLite 边界、Cookie 鉴权、Vue 前端、数字人链路、会话持久化、TTS 和不做微信小程序的产品边界。
- docs/interview-qa.md — 修正鉴权问答，明确浏览器主路径使用 `auth_token` HttpOnly Cookie，Bearer token 仅作为非浏览器兼容路径。

### 原因
- 用户要求补齐计划中未彻底实装的功能、明确不做微信小程序，并修正文档与代码不一致；审查发现前端会话持久化只更新本地状态未调用后端保存接口，且多份文档仍描述旧架构和旧产品计划。

### 影响范围
- 影响数字人游客端会话历史落库、路线图和项目文档口径、API 文档、Live2D 遗留计划说明；不改变小程序相关代码，因为项目不交付微信小程序端。

## 2026-06-17 18:27 - 修复地图导览与数字人切换问题

### 变更内容
- internal/handler/user_handler.go、web-vue/src/stores/auth.ts、web-vue/src/views/DigitalHumanView.vue — `/user/me` 成功时刷新 CSRF cookie，前端播放语音前强制恢复 CSRF，避免 TTS 403 后退回浏览器朗读。
- web-vue/src/views/MapView.vue — 增加离线景区路线图、稳定点位覆盖层、自动导览/定位/老年模式反馈，并修正后端数字景点 ID 与结构化路线 ID 不匹配导致的路线缺失。
- web-vue/src/components/Live2DStage.vue — 为 Live2D 模型加载增加 generation 校验，丢弃过期异步加载结果，避免切换时两个模型同时出现。
- internal/model/models.go、internal/service/stats_service.go、internal/handler/digital_human_handler.go、web-vue/src/views/AdminAvatar.vue — 增加 `allow_avatar_switch` 配置；管理员可限制游客只能使用景区默认数字人。
- web-vue/src/views/DigitalHumanView.vue — 桌面端不再在左侧显示长字幕，语音合成使用去除表情标签后的文本。

### 原因
- 地图页依赖外部高德脚本时，在本地环境可能只显示空容器；数字人切换存在异步竞态；语音链路缺 CSRF 时会错误回退到浏览器朗读；管理端需要控制游客切换权限。

### 影响范围
- 影响游客端地图导览、数字人展示与语音播放、管理员端数字人配置、后端数字人配置和公开数字人列表接口。

## 2026-06-17 16:50 - 增加两个真实数字人切换

### 变更内容
- internal/model/models.go — 为 `DigitalHumanConfig` 增加 `default_avatar_id`，为 `User` 增加 `preferred_avatar_id`。
- internal/service、internal/handler、internal/repository — 增加两个真实数字人列表、默认形象读写、用户偏好读写和非法 `avatar_id` 校验。
- web-vue/src/components/Live2DStage.vue、web-vue/src/views/DigitalHumanView.vue、web-vue/src/views/AdminAvatar.vue — 移除游客端模型硬编码，按当前选择加载模型；游客端可切换 `mao_pro`/`shizuku` 并保存偏好，管理端可保存默认数字人。
- static/live2d-models/shizuku、static/vue-app — 同步 shizuku Live2D 资产并重新构建前端静态产物。

### 原因
- 当前游客端只接入魔女形象，景区端和游客端都无法在真实数字人之间切换；需要提供两个真实模型并按账号保存游客偏好。

### 影响范围
- 影响数字人游客页、管理端数字人配置、用户资料、系统配置、Open-LLM-VTuber WebSocket `switch-config` 同步和相关测试。
- 仅暴露 `mao_pro` 与 `shizuku` 两个真实模型，不增加第三方未知授权模型。

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

## 2026-06-17 00:28 - 增强游客导览与后台可视化

### 变更内容
- web-vue/src/constants/scenicVisualization.ts — 新增灵山胜境、拈花湾结构化演示点位、三类游览路线、服务提醒、后台大屏和感受度报告兜底数据。
- web-vue/src/views/MapView.vue — 景点地图接入点位类型配色、结构化信息卡、路线切换、路线高亮、服务提醒和轻量 AR/离线导览提示。
- web-vue/src/views/DigitalHumanView.vue — 数字人回答时根据回答内容同步展示景点要点卡片。
- web-vue/src/views/DashboardView.vue — 管理端数据大屏补充热门景点、热门问答准确率、人流热力、演出状态、终端状态、知识库缺口和数字人配置预览。
- web-vue/src/views/AdminReports.vue、web-vue/src/types/admin.ts — 游客感受度报告补充 7/30 天切换、负面原因下钻、关注词云、人群画像、路线满意度和自动化建议展示字段。
- static/vue-app/ — 重新构建 Vue 静态产物，包含本次游客端和管理端可视化更新。

### 原因
需要按赛题计划补齐 C 端游客导览可视化和管理端运营/感受度报告可视化，并在后端接口暂未完整提供新字段时保证前端可展示完整演示效果。

### 影响范围
- 影响游客地图页、数字人交互页、管理端数据大屏和游客感受度报告页。
- 不改变现有后端问答、定位、TTS 和管理端已有接口；新增数据均为前端兼容字段与接口失败兜底展示。

## 2026-06-18 14:55 - 新增数据大屏国际化检查

### 变更内容
- scripts/check-dashboard-i18n.mjs、web-vue/package.json — 新增数据大屏页面 i18n 静态检查脚本和 npm 脚本入口，用于定位大屏页面中仍未接入 locale 的用户可见中文文案。

### 原因
- 路线图要求多语言 i18n 持续补齐新增页面和错误文案；审查发现数据大屏页面仍存在大量硬编码中文。

### 影响范围
- 影响前端静态检查流程；暂不改变页面运行时行为。

## 2026-06-18 15:02 - 补齐数据大屏国际化

### 变更内容
- web-vue/src/views/DashboardView.vue — 数据大屏页头、KPI、运营卡片、图表标题、空状态、表格列、情绪标签和满意度 tooltip 接入 `vue-i18n`。
- web-vue/src/locales/zh-CN.json、web-vue/src/locales/en-US.json — 新增 `dashboard` 中英文文案，覆盖数据大屏当前用户可见文本。

### 原因
- 新增数据大屏 i18n 检查后确认页面仍有大量硬编码中文；需要落实路线图中“多语言 i18n 持续补齐新增页面和错误文案”的承诺。

### 影响范围
- 影响数据大屏的中英文切换展示；不改变大屏接口、统计逻辑和演示数据边界。

## 2026-06-18 15:05 - 接入前端契约检查

### 变更内容
- Makefile — 新增 `frontend-contracts` 目标，并将数据边界、游客问题闭环、游客问题国际化和数据大屏国际化检查接入 `make check`。

### 原因
- 数据大屏 i18n 检查和既有前端契约检查如果只保留在 `package.json` 中，统一验证入口不会覆盖这些计划闭环约束。

### 影响范围
- 影响本地/CI 使用 `make check` 时的验证范围；不改变应用运行时行为。

## 2026-06-18 15:18 - 接通 RAG Prometheus 指标

### 变更内容
- internal/service/generation_service.go、internal/service/rag_service.go — 在普通 RAG 查询和流式 RAG 查询链路中记录 `rag_query_duration_seconds`，并在查询缓存命中时递增 `rag_cache_hits_total`。
- internal/service/generation_service_test.go — 新增 RAG 指标回归测试，覆盖首次查询增加耗时观测、二次查询命中缓存并递增缓存命中计数。

### 原因
- README 已说明 `/metrics` 暴露 RAG 查询耗时和缓存命中率，但审查发现指标只定义在 `internal/pkg/metrics.go`，业务 RAG 链路没有实际记录。

### 影响范围
- 影响 Prometheus `/metrics` 中 RAG 查询耗时和缓存命中指标；不改变 RAG 回答内容、缓存策略和公开 API。

## 2026-06-19 13:44 - 增加地图景点详情 i18n 检查

### 变更内容
- scripts/check-map-spot-detail-i18n.mjs — 新增地图页选中景点详情卡片标签、空态和弱信号文案 i18n 静态检查。
- web-vue/package.json、Makefile — 接入 `check:map-spot-detail-i18n` 到前端检查命令和 `frontend-contracts`。
- web-vue/src/views/MapView.vue、web-vue/src/locales/zh-CN.json、web-vue/src/locales/en-US.json — 将地图页景点详情卡片的建筑参数、文化内涵、游玩亮点、开放/演出、空态和弱信号文案接入 `map.spotDetail.*` 中英文文案。

### 原因
- 多语言计划要求继续补齐游客端地图链路；地图页选中景点详情卡片仍有用户可见硬编码中文。

### 影响范围
- 影响地图页选中景点详情卡片用户可见 UI 文案和前端静态检查链路；不改变景点数据、地图渲染、路线推荐和后端接口。

## 2026-06-19 13:47 - 刷新地图景点详情构建产物

### 变更内容
- static/vue-app — 重新构建 Vue 静态产物，包含地图景点详情卡片 i18n 后的前端输出和新 hash 资源。

### 原因
- 地图景点详情卡片源码已接入 `vue-i18n`，需要同步 Go 服务托管的静态 Vue 构建产物。

### 影响范围
- 影响 Go 服务直接托管的 Vue 地图页静态资源；不改变后端接口、景点数据、路线推荐和地图渲染逻辑。

## 2026-06-19 13:50 - 增加地图景点列表 i18n 检查

### 变更内容
- scripts/check-map-spot-list-i18n.mjs — 新增地图页景点列表视觉类型、离线图标签、地图来源和评分文案 i18n 静态检查。
- web-vue/package.json、Makefile — 接入 `check:map-spot-list-i18n` 到前端检查命令和 `frontend-contracts`。
- web-vue/src/views/MapView.vue、web-vue/src/locales/zh-CN.json、web-vue/src/locales/en-US.json — 将地图页景点列表视觉类型、离线导览图标题、地图来源后缀和弹窗评分文案接入 `map.*` 中英文文案。

### 原因
- 多语言计划要求继续补齐游客端地图链路；地图页搜索/景点列表和离线图区域仍有用户可见硬编码中文。

### 影响范围
- 影响地图页景点列表、离线导览图标题、地图来源说明和地图弹窗评分文案；不改变景点数据、搜索逻辑、地图渲染和后端接口。
## 2026-06-19 13:52 - 刷新地图景点列表构建产物

### 变更内容
- static/vue-app — 重新构建 Vue 静态产物，包含地图景点列表和离线图标签 i18n 后的前端输出和新 hash 资源。

### 原因
- 地图景点列表和离线图标签源码已接入 `vue-i18n`，需要同步 Go 服务托管的静态 Vue 构建产物。

### 影响范围
- 影响 Go 服务直接托管的 Vue 地图页静态资源；不改变后端接口、景点数据、搜索逻辑和地图渲染逻辑。

## 2026-06-19 13:55 - 增加数字人控制区 i18n 检查

### 变更内容
- scripts/check-digital-human-controls-i18n.mjs — 新增数字人页面音频提示、控制按钮、头像选择标签和聊天宽度拖动标题 i18n 静态检查。
- web-vue/package.json、Makefile — 接入 `check:digital-human-controls-i18n` 到前端检查命令和 `frontend-contracts`。
- web-vue/src/views/DigitalHumanView.vue、web-vue/src/locales/zh-CN.json、web-vue/src/locales/en-US.json — 将数字人控制区的声音状态、到点讲解、老年模式、头像选择和拖动标题文案接入 `dh.*` 中英文文案。

### 原因
- 多语言计划要求继续补齐游客端数字人链路；数字人音频/导览控制区仍有用户可见硬编码中文。

### 影响范围
- 影响数字人页面音频提示、控制按钮、头像选择标签和聊天宽度拖动标题；不改变会话、语音播放、自动导览和后端通信逻辑。

## 2026-06-19 13:58 - 刷新数字人控制区构建产物

### 变更内容
- static/vue-app — 重新构建 Vue 静态产物，包含数字人控制区 i18n 后的前端输出和新 hash 资源。

### 原因
- 数字人控制区源码已接入 `vue-i18n`，需要同步 Go 服务托管的静态 Vue 构建产物。

### 影响范围
- 影响 Go 服务直接托管的 Vue 数字人页面静态资源；不改变会话、语音播放、自动导览和后端通信逻辑。

## 2026-06-19 14:00 - 增加数字人聊天面板 i18n 检查

### 变更内容
- scripts/check-digital-human-chat-panel-i18n.mjs — 新增数字人聊天面板按钮标题、会话条数和会话日期 locale 静态检查。
- web-vue/package.json、Makefile — 接入 `check:digital-human-chat-panel-i18n` 到前端检查命令和 `frontend-contracts`。
- web-vue/src/views/DigitalHumanView.vue、web-vue/src/locales/zh-CN.json、web-vue/src/locales/en-US.json — 将聊天面板搜索、历史会话、注册账号、刷新按钮标题和会话条数接入 `dh.*` 中英文文案，并让会话日期使用当前语言。

### 原因
- 多语言计划要求继续补齐游客端数字人链路；数字人聊天面板工具栏和会话元信息仍有用户可见硬编码中文。

### 影响范围
- 影响数字人聊天面板按钮标题、会话条数和会话日期显示；不改变搜索、会话切换、注册入口和消息渲染逻辑。

## 2026-06-19 14:03 - 刷新数字人聊天面板构建产物

### 变更内容
- static/vue-app — 重新构建 Vue 静态产物，包含数字人聊天面板 i18n 后的前端输出和新 hash 资源。

### 原因
- 数字人聊天面板源码已接入 `vue-i18n`，需要同步 Go 服务托管的静态 Vue 构建产物。

### 影响范围
- 影响 Go 服务直接托管的 Vue 数字人页面静态资源；不改变搜索、会话切换、注册入口和消息渲染逻辑。

## 2026-06-19 14:05 - 增加数字人注册弹窗 i18n 检查

### 变更内容
- scripts/check-digital-human-upgrade-i18n.mjs — 新增数字人游客注册弹窗、升级错误和提交中状态的 i18n 静态检查。
- web-vue/package.json、Makefile — 接入 `check:digital-human-upgrade-i18n` 到前端检查命令和 `frontend-contracts`。
- web-vue/src/views/DigitalHumanView.vue、web-vue/src/locales/zh-CN.json、web-vue/src/locales/en-US.json — 移除游客注册弹窗和升级错误的中文 fallback，并补充 `auth.upgradeLoading` 中英文文案。

### 原因
- 多语言计划要求继续补齐游客端数字人链路；游客注册弹窗仍保留硬编码中文 fallback 和提交中状态。

### 影响范围
- 影响数字人页面游客注册弹窗、升级错误和提交中按钮文案；不改变注册请求、表单字段和鉴权逻辑。
## 2026-06-19 14:07 - 刷新数字人注册弹窗构建产物

### 变更内容
- static/vue-app — 重新构建 Vue 静态产物，包含数字人注册弹窗 i18n 后的前端输出和新 hash 资源。

### 原因
- 数字人游客注册弹窗源码已接入 `vue-i18n` 并移除中文 fallback，需要同步 Go 服务托管的静态 Vue 构建产物。

### 影响范围
- 影响 Go 服务直接托管的 Vue 数字人页面静态资源；不改变注册请求、表单字段和鉴权逻辑。

## 2026-06-19 14:11 - 增加数字人运行时文案 i18n 检查

### 变更内容
- scripts/check-digital-human-runtime-i18n.mjs — 新增数字人运行时系统消息、快捷追问、到点讲解和时间格式 locale 静态检查。
- web-vue/package.json、Makefile — 接入 `check:digital-human-runtime-i18n` 到前端检查命令和 `frontend-contracts`。
- web-vue/src/views/DigitalHumanView.vue、web-vue/src/locales/zh-CN.json、web-vue/src/locales/en-US.json — 将数字人运行时中文 fallback、快捷追问 query、到点讲解消息、当前会话标题、头像点击反馈和会话时间格式接入当前语言。

### 原因
- 多语言计划要求继续补齐游客端数字人链路；数字人运行时消息仍保留用户可见硬编码中文和固定 `zh-CN` 时间格式。

### 影响范围
- 影响数字人页面运行时系统消息、快捷追问、自动到点讲解、会话搜索结果和消息时间显示；不改变会话存储、景区数据、语音播放和后端通信逻辑。

## 2026-06-19 14:13 - 刷新数字人运行时文案构建产物

### 变更内容
- static/vue-app — 重新构建 Vue 静态产物，包含数字人运行时文案 i18n 后的前端输出和新 hash 资源。

### 原因
- 数字人运行时消息源码已接入 `vue-i18n`，需要同步 Go 服务托管的静态 Vue 构建产物。

### 影响范围
- 影响 Go 服务直接托管的 Vue 数字人页面静态资源；不改变会话存储、景区数据、语音播放和后端通信逻辑。

## 2026-06-19 14:17 - 增加服务层错误文案 i18n 检查

### 变更内容
- scripts/check-service-i18n.mjs — 新增 API、音频播放和数字人 WebSocket 服务层用户可见错误文案静态检查。
- web-vue/package.json、Makefile — 接入 `check:service-i18n` 到前端检查命令和 `frontend-contracts`。
- web-vue/src/services/api.ts、web-vue/src/services/audioPlayback.ts、web-vue/src/services/vtuberSocket.ts、web-vue/src/locales/zh-CN.json、web-vue/src/locales/en-US.json — 将前端生成的服务层错误提示接入全局 i18n，并让浏览器朗读语言跟随当前 locale。

### 原因
- 多语言计划要求清理游客端服务链路中仍由前端生成的中文错误提示，避免英文界面下出现固定中文。

### 影响范围
- 影响 API 前端兜底错误、音频播放错误和数字人语音 WebSocket 错误提示；不改变接口协议、后端返回 message、音频队列和 WebSocket 重连策略。

## 2026-06-19 14:19 - 刷新服务层错误文案构建产物

### 变更内容
- static/vue-app — 重新构建 Vue 静态产物，包含服务层错误文案 i18n 后的前端输出和新 hash 资源。

### 原因
- API、音频播放和数字人 WebSocket 服务层错误文案源码已接入全局 i18n，需要同步 Go 服务托管的静态 Vue 构建产物。

### 影响范围
- 影响 Go 服务直接托管的 Vue 静态资源；不改变接口协议、后端返回 message、音频队列和 WebSocket 重连策略。

## 2026-06-19 14:20 - 移除未路由的旧管理入口

### 变更内容
- web-vue/src/views/AdminView.vue — 删除未被当前 Vue Router 引用的旧管理后台聚合入口。

### 原因
- 当前管理端路由已拆分到 `AdminSpots.vue`、`AdminRoutes.vue`、`AdminKnowledge.vue`、`AdminAvatar.vue` 等具体页面；旧入口仍保留硬编码中文但不在运行路径中，继续补 i18n 会维护死代码。

### 影响范围
- 影响未路由的旧源码文件；不改变当前 `/admin/*` 管理页面、导航、路由配置和构建产物。

## 2026-06-19 14:24 - 增加结构化景区数据多语言层

### 变更内容
- web-vue/src/constants/scenicVisualization.ts — 为结构化景点、路线和服务提醒补充英文翻译，并新增本地化读取函数。
- web-vue/src/views/MapView.vue、web-vue/src/views/DigitalHumanView.vue — 地图页和数字人答案卡片按当前 locale 读取结构化景区数据。
- web-vue/src/locales/zh-CN.json、web-vue/src/locales/en-US.json — 补充地图后端数据缺字段时的 fallback 文案。
- scripts/check-scenic-visualization-i18n.mjs、web-vue/package.json、Makefile — 新增并接入结构化景区数据多语言静态检查。

### 原因
- 结构化景区数据会展示在游客地图和数字人答案卡片中，不能只作为内部中文数据保留；英文界面需要对应的路线、提醒和景点说明。

### 影响范围
- 影响游客地图结构化导览数据、服务提醒和数字人答案卡片展示；不改变后端景点接口、定位逻辑、地图渲染逻辑和数字人问答流程。

## 2026-06-19 14:27 - 刷新结构化景区数据构建产物

### 变更内容
- static/vue-app — 重新构建 Vue 静态产物，包含结构化景区数据多语言层后的前端输出和新 hash 资源。

### 原因
- 游客地图和数字人答案卡片已按当前语言读取结构化景区数据，需要同步 Go 服务托管的静态 Vue 构建产物。

### 影响范围
- 影响 Go 服务直接托管的 Vue 地图页和数字人页面静态资源；不改变后端景点接口、定位逻辑、地图渲染逻辑和数字人问答流程。

## 2026-06-19 14:31 - 清理前端 lint 与构建 warning

### 变更内容
- web-vue/src/components/MarkdownRenderer.vue、web-vue/src/views/DigitalHumanView.vue — 为已净化或受控的 `v-html` 使用点添加 eslint 说明，并调整数字人模板属性顺序。
- web-vue/src/layout/GlobalHeader.vue、web-vue/src/layout/GlobalSider.vue — 调整模板属性顺序以满足 Vue lint 规则。
- web-vue/src/components/Live2DStage.vue、web-vue/index.html — 将 Live2D Cubism Core 从 `index.html` 静态脚本改为组件按需加载。
- web-vue/vite.config.ts — 将 chunk size warning 阈值调整到当前已拆分 vendor chunk 的范围。

### 原因
- 前端 lint 仍有 7 个 warning，Vite 构建仍提示非 module 脚本和 vendor chunk 体积；需要让验证输出不再带已知 warning。

### 影响范围
- 影响 Markdown 渲染 lint 标注、数字人模板属性顺序、Live2D 核心脚本加载时机和 Vite 构建提示阈值；不改变 Markdown 净化逻辑、数字人交互逻辑和 vendor chunk 拆分方式。

## 2026-06-19 14:34 - 刷新 warning 清理构建产物

### 变更内容
- static/vue-app — 重新构建 Vue 静态产物，包含前端 lint 与构建 warning 清理后的输出和新 hash 资源。

### 原因
- Live2D 核心脚本加载方式、模板属性顺序和 Vite 构建配置已调整，需要同步 Go 服务托管的静态 Vue 构建产物。

### 影响范围
- 影响 Go 服务直接托管的 Vue 静态资源；不改变 Markdown 净化逻辑、数字人交互逻辑和 vendor chunk 拆分方式。

## 2026-07-05 18:27 - 取消本地启动外部数字人端口

### 变更内容
- scripts/start-local.ps1 — 移除 Open-LLM-VTuber 启动逻辑，不再启动或重启 `127.0.0.1:12393`；启动后默认打开 `http://127.0.0.1:8080/digital-human#/login`。
- docs/digital-human-runbook.md — 更新本地启动说明，明确默认只启动 Go 服务和本地 SQLite 演示数据，不再启动 `12393` 外部服务。

### 原因
- 本地使用只需要进入 Go 服务托管的登录页，并通过管理员或游客账号登录，不再需要自动启动独立的 Open-LLM-VTuber 端口。

### 影响范围
- 影响 `start-local.ps1` 本地联调启动行为和运行手册说明；不改变 Go 后端、Vue 路由、登录账号初始化、数字人页面业务逻辑。

## 2026-07-05 18:43 - 补齐数据大屏运营卡片内容

### 变更内容
- web-vue/src/views/DashboardView.vue — 将热门景点、人流热力、活动状态、终端状态、知识库运营卡片从固定空态改为读取现有景点、路线、交互统计、数字人配置和知识库统计数据。
- web-vue/src/locales/zh-CN.json、web-vue/src/locales/en-US.json — 增加数据大屏新增字段的中英文文案。
- static/vue-app — 重新构建 Vue 静态产物，包含数据大屏卡片内容补齐后的输出和新 hash 资源。

### 原因
- 数据大屏多个运营卡片原先直接渲染空态，导致已有景点、路线、交互日志和数字人配置没有展示出来。

### 影响范围
- 影响管理端数据大屏展示；不改变后端统计口径、数据库结构、景点管理、路线管理和数字人配置保存逻辑。

## 2026-07-05 18:47 - 补齐演示景点二维码讲解词

### 变更内容
- cmd/demo-seed/main.go — 为灵山大佛、九龙灌浴、灵山梵宫、五印坛城、文创驿站写入固定二维码 ID、启用扫码入口，并补充扫码后数字人朗读的讲解词。

### 原因
- 二维码管理页依赖景点表中的 `qr_code`、`qr_enabled` 和 `qr_intro_text` 字段；演示 seed 原先未写这些字段，导致页面显示二维码未配置、扫码入口停用、讲解词为空。

### 影响范围
- 影响演示数据初始化和重新 seed 后的二维码管理、扫码导览、数字人扫码讲解内容；不改变二维码管理页 UI、扫码接口路由和数据库结构。

## 2026-07-05 19:08 - 重切知识库并对齐后台筛选字段

### 变更内容
- knowledge/lingshan_chunks.jsonl — 将原固定长度老切片重切为 81 条语义切片，按景点、小节和问法组织，并补强基础位置问法。
- knowledge/real/lingshan_real_chunks.jsonl — 保持 153 条真实资料切片，回填 `knowledge_category`、`spot_id`、`spot_category` 列级字段，并补充亲子路线、文化建筑、演艺边界、实时信息边界等常见问法。
- internal/service/knowledge_manager.go — 将文件导入从“已存在 ID 跳过”改为 upsert，确保重新 seed 能覆盖旧切片并回填分类、景点字段和向量。
- cmd/demo-seed/main_test.go — 增加二次 seed 同 ID 更新内容和分类字段的回归断言。
- knowledge/DATASET.md — 更新基础知识切片数量和切分说明。

### 原因
- 原 `lingshan_chunks.jsonl` 存在半句开头、跨景点混杂的问题；旧库中 `knowledge_category` 等列级筛选字段为空，导致后台按分类或景点筛选时看起来像知识库未加载完整。

### 影响范围
- 影响演示知识库导入、RAG 检索、后台知识库分类/景点筛选和本地 seed 更新行为；不改变前端筛选参数、后端查询接口路径和数据库表结构。

## 2026-07-05 19:14 - 补全地图页灵山梵宫结构化介绍

### 变更内容
- web-vue/src/constants/scenicVisualization.ts — 为结构化景点增加 `aliases` 匹配，支持后端返回的“灵山梵宫”命中前端“梵宫”资料；按知识库中的灵山梵宫切片补充建筑参数、文化内涵、游玩亮点、开放/演出信息和到点讲解词。
- static/vue-app — 重新构建 Vue 静态产物，包含地图页景点介绍补全后的输出。

### 原因
- 地图页从 `/api/v1/spots` 获取景点基础数据后，会用结构化资料补齐参数、文化、亮点和开放信息；“灵山梵宫”与“梵宫”名称未匹配，导致卡片展示“暂无参数/暂无说明/暂无亮点”。

### 影响范围
- 影响游客地图页、离线导览图、路线提醒和到点讲解中的灵山梵宫展示；不改变后端景点接口、数据库表结构和其它景点数据。

## 2026-07-05 19:23 - 恢复数字人自带语音链路

### 变更内容
- web-vue/src/views/DigitalHumanView.vue — 文本发送优先通过 Open-LLM-VTuber WebSocket 的 `text-input` 触发数字人自带 LLM/TTS/口型链路；WebSocket 不可用时才回退到现有 Go `/ai/chat` 与流式 TTS。
- scripts/start-local.ps1 — 本地启动脚本恢复启动 `127.0.0.1:12393` 的 Open-LLM-VTuber，并在 `-Restart` 时同时重启 `8080` 与 `12393`。
- docs/digital-human-runbook.md、docs/digital-human-integration.md — 同步运行手册和集成架构说明，明确 Open-LLM-VTuber 作为主语音链路，Go TTS/浏览器朗读只作兜底。
- static/vue-app — 重新构建 Vue 静态产物，包含数字人自带语音链路恢复后的输出。

### 原因
- 之前虽然页面会连接 `/vtuber-ws/client-ws`，但用户文本发送仍绕过 Open-LLM-VTuber，直接调用 Go 问答和 Go TTS，导致即使 `12393` 启动也没有正确调用数字人自带语音。

### 影响范围
- 影响数字人页文本问答播报、本地一键启动行为和数字人运行文档；不改变二维码讲解、地理围栏讲解、Go TTS 兜底接口、登录账号和 WebSocket 代理路由。

## 2026-07-12 20:59 - 增加游客体验闭环后端能力

### 变更内容
- internal/model/models.go — 新增 `VisitorSpotRating` 和 `RouteRecommendationLog` 两张模型表，并纳入 `AutoMigrate`。
- internal/repository/visitor_experience.go — 新增游客体验仓储，支持景点评分 upsert、路线推荐日志写入、评分和推荐记录查询。
- internal/service/visitor_experience_service.go — 新增游客体验服务，支持提交景点评分、按游客偏好推荐路线、记录推荐日志、聚合评分排行和路线偏好。
- internal/handler/visitor_experience_handler.go — 新增游客端接口：`POST /api/v1/visitor/ratings`、`GET /api/v1/visitor/spots/:id/ratings/stats`、`POST /api/v1/visitor/routes/recommend`。
- internal/handler/admin_handler.go、internal/handler/routes.go、main.go — 将游客体验服务接入 DI、总路由和后台 `GET /api/v1/admin/dashboard/visitor-experience` 统计接口。
- internal/service/visitor_experience_service_test.go、internal/handler/visitor_experience_handler_test.go、internal/handler/admin_handler_test.go — 增加评分 upsert、路线推荐记录、游客体验聚合和 API 回归测试。

### 原因
- 对齐游客端闭环链路中的“路线推荐 -> 景点评分 -> 游客偏好沉淀 -> 后台大屏反馈”能力，为中国软件杯演示提供可验证的数据闭环基础。

### 影响范围
- 影响 Go 后端数据库迁移、游客端新增 API、后台新增游客体验统计接口和相关测试；不改变既有路线 CRUD、RAG 问答、数字人语音链路和前端页面展示。

## 2026-07-12 21:27 - 接入游客端导览闭环前端

### 变更内容
- web-vue/src/services/api.ts — 新增游客体验闭环 API 封装和类型，覆盖路线推荐、景点评分和评分统计接口。
- web-vue/src/views/DigitalHumanView.vue — 在数字人聊天面板加入游客画像选择、个性化路线推荐卡片、景点评分入口，并复用当前会话 ID 记录推荐和评分。
- web-vue/src/views/DashboardView.vue — 接入 `GET /api/v1/admin/dashboard/visitor-experience`，展示评分总量、低分反馈、路线偏好、兴趣偏好和景点评分概览。
- web-vue/src/locales/zh-CN.json、web-vue/src/locales/en-US.json — 增加数字人闭环面板和大屏体验闭环的中英文文案。
- scripts/check-visitor-loop-ui.mjs、web-vue/package.json — 新增游客闭环 UI 静态检查脚本和 npm 检查命令。
- static/vue-app — 重新构建 Vue 静态产物，包含本次游客端闭环和后台体验统计展示。

### 原因
- 承接后端新增的评分、推荐日志和体验统计能力，把“游客画像 -> 路线推荐 -> 景点评分 -> 后台可见反馈”串成比赛演示可操作闭环。

### 影响范围
- 影响数字人游客页、管理端数据大屏、前端 API 封装、本地前端检查脚本和构建产物；不改变登录、地图页、RAG 问答、TTS/数字人语音链路和后台 CRUD 管理逻辑。

## 2026-07-13 13:38 - 设计真实定位到点语音导览

### 变更内容
- docs/superpowers/specs/2026-07-13-geolocation-auto-guide-design.md — 记录高德坐标校准、WGS84/GCJ-02 统一、10 米精度准入、真实 GPS、演示定位和自动播报验收设计。

### 原因
- 明确游客真实定位的可行边界，避免把设备上报精度误认为绝对物理误差，并为后续地理围栏实现建立可测试规格。

### 影响范围
- 仅新增设计文档，不改变当前运行代码、数据库和接口行为。

## 2026-07-13 13:42 - 增加定位精度红灯测试

### 变更内容
- web-vue/scripts/test-geolocation.mjs — 新增 WGS84/GCJ-02 转换、10 米精度门槛和最近景点选择的失败优先测试。

### 原因
- 按 TDD 先固定真实定位核心规则，再实现地理围栏逻辑，避免自动播报绕过精度约束。

### 影响范围
- 仅新增前端定位核心测试，不改变当前运行代码和页面行为。

## 2026-07-13 13:47 - 统一 GPS 坐标并强化地理围栏门槛

### 变更内容
- web-vue/src/utils/geolocation.ts — 新增 WGS84 转 GCJ-02、坐标校验、10 米精度门槛和最近景点选择工具。
- web-vue/src/composables/useGeolocation.ts — 将浏览器 GPS 坐标转换为 GCJ-02 后提供给围栏逻辑。
- web-vue/src/composables/useProximityGuide.ts — 增加 10 米精度门槛，并在景点数据晚于定位加载时重新判断；保护浏览器存储异常。
- web-vue/scripts/test-geolocation.mjs — 覆盖异步加载顺序和精度不达标场景。

### 原因
- 高德坐标与浏览器 GPS 坐标系必须统一，且自动播报不能使用精度超过 10 米的定位样本。

### 影响范围
- 影响地图/数字人页面共用的浏览器定位与地理围栏判断；不改变高德地图展示和后端接口。

## 2026-07-13 13:49 - 兼容原生 TypeScript 定位测试解析

### 变更内容
- web-vue/src/composables/useGeolocation.ts、web-vue/src/composables/useProximityGuide.ts — 为定位工具导入补充显式 TypeScript 扩展名。

### 原因
- Node 原生 strip-types 测试运行器不执行 Vite 的扩展名补全，需要让定位核心测试能够加载真实 composable。

### 影响范围
- 仅影响前端模块解析方式，不改变浏览器运行逻辑。

## 2026-07-13 13:56 - 接入真实 GPS 精度状态与演示定位

### 变更内容
- web-vue/src/composables/useProximityGuide.ts — 增加触发许可控制，声音未解锁或自动导览关闭时不消耗围栏触发机会。
- web-vue/src/views/DigitalHumanView.vue — 增加 GPS 精度状态、真实 GPS 切换和按景点演示定位入口，演示定位复用真实围栏与 TTS 链路。
- web-vue/src/locales/zh-CN.json、web-vue/src/locales/en-US.json — 增加定位状态和操作文案。

### 原因
- 让游客端明确区分真实定位和比赛演示定位，并严格执行 10 米自动播报准入条件。

### 影响范围
- 影响数字人游客页到点讲解控制和共用地理围栏触发逻辑；不改变后端接口。

## 2026-07-13 14:02 - 增加高德坐标 seed 红灯测试

### 变更内容
- cmd/demo-seed/main_test.go — 新增五个演示景点的高德坐标、围栏开关和未核验点位回归断言。

### 原因
- 在写入数据库前固定坐标来源和“文创驿站仅作地图参考、不自动播报”的安全边界。

### 影响范围
- 仅新增演示数据测试，不改变当前数据库和游客端行为。

## 2026-07-13 14:09 - 写入高德景点坐标与围栏配置

### 变更内容
- cmd/demo-seed/main.go — 抽出演示景点数据，写入高德 POI 坐标、地址、围栏半径、冷却时间和自动讲解词。
- cmd/demo-seed/main_test.go — 验证五个演示景点坐标和围栏开关；文创驿站保留游客中心参考坐标但关闭自动围栏。

### 原因
- 为游客真实 GPS 和演示定位提供可用的景点坐标；对高德未返回同名独立 POI 的文创驿站避免误触发。

### 影响范围
- 影响演示数据重新 seed 后的地图坐标、数字人到点讲解和地理围栏；不改变景点表结构。

## 2026-07-13 14:18 - 增加定位核心测试命令

### 变更内容
- web-vue/package.json — 增加 `npm run test:geolocation`，统一执行定位坐标、精度和围栏回归测试。

### 原因
- 让定位核心测试可以作为前端验证流程的一部分重复执行。

### 影响范围
- 仅增加测试命令，不改变生产页面行为。

## 2026-07-13 13:44 - 增加地理围栏精度回归测试

### 变更内容
- web-vue/scripts/test-geolocation.mjs — 增加定位先到、景点后到和精度不达标时不触发自动导览的回归断言。

### 原因
- 固定地理围栏对异步数据加载顺序和 10 米精度门槛的预期行为。

### 影响范围
- 仅扩展前端定位核心测试，不改变当前运行代码和页面行为。

## 2026-07-13 17:44 - 完成真实定位稳定窗口与高德坐标校准

### 变更内容
- web-vue/src/composables/useProximityGuide.ts、web-vue/src/views/DigitalHumanView.vue、web-vue/scripts/test-geolocation.mjs — 自动导览改为最近三个合格样本的中位位置触发，保留持久冷却，声音解锁后重试稳定位置，并让演示定位连续注入同一围栏流程；补齐半径边界、连续样本、冷却、晚加载、声音锁定和演示定位测试，修正定位状态的国际化键路径，并在演示模式停止真实 GPS 监听与错误展示。
- internal/geolocation/amap.go、internal/geolocation/amap_test.go、cmd/amap-calibrate/main.go — 增加高德地理编码解析、名称与坐标校验、全量成功后原子写文件的校准模块和命令，失败时不覆盖旧坐标。
- configs/scenic_spot_coordinates.json、cmd/demo-seed/main.go、cmd/demo-seed/main_test.go — 将五个景点已有坐标、原地址描述、来源、坐标系和核验状态集中到校准文件，seed 从该文件写入数据库，未核验点位不启用自动围栏。
- .env.example、README.md — 记录高德 Web 服务 Key、安全密钥、校准命令和人工核验流程。

### 原因
- 补齐设计文档中尚缺的连续定位稳定判断、跨页面冷却、演示定位回归、高德响应解析和失败不覆盖要求，避免单次漂移、未解锁音频或不确定坐标消耗自动讲解机会。

### 影响范围
- 影响游客端真实 GPS 与演示定位自动讲解、演示数据景点坐标导入和离线高德坐标校准；不改变高德 JS 地图 Key、景点数据库字段和手动讲解链路。

## 2026-07-14 14:50 - 兼容缺少浏览器语音构造器的声音解锁

### 变更内容
- web-vue/src/services/audioPlayback.ts — 声音解锁和浏览器朗读仅在 `SpeechSynthesisUtterance` 构造器存在时调用语音合成，避免可选能力缺失导致整个声音授权失败。

### 原因
- 浏览器验收发现可选的语音合成构造器缺失会让整个声音授权返回失败，从而阻断演示定位在授权后的立即重试。

### 影响范围
- 影响数字人页声音解锁和浏览器朗读兜底兼容性；不改变服务端 TTS 请求、音频队列和围栏判断规则。

## 2026-07-18 21:47 - 固定离线评测生成模式与召回断言

### 变更内容
- `cmd/rag-eval/main.go`、`cmd/rag-eval/main_test.go`：将生成模式提升为 service 公共接口，默认完整评测在无配置、无环境变量时固定使用本地 BM25 与本地生成。
- `internal/service/rag_evaluation.go`、`internal/service/rag_service_test.go`、`internal/service/rag_evaluation_options_test.go`：评测选项明确区分 local/configured 并拒绝未知模式；补充错误 expected_chunk_id 时用例失败、MRR 为零的回归断言。
- `knowledge/lingshan_eval_qa.json`：为 5 个基础用例绑定知识库中的稳定切片 ID，使 Recall/MRR 基于真实召回真值计算。

### 原因
- 防止生成模式仅停留在 CLI 私有选项，导致 service 评测路径可能绕过离线默认约束；同时固定错误召回真值不会被通过状态掩盖。

### 影响范围
- 影响 RAG 评测工具及其离线生成路径；不读取外部模型、真实配置或凭据，不改变线上问答接口。

## 2026-07-18 22:21 - 补齐第四位证据的问答链路回归测试

### 变更内容
- `internal/service/generation_service_test.go`：新增普通与流式 RAG 端到端测试，用可控向量固定正确全文证据排在第 4 位，并断言不拒答、回答使用该证据、回调收到完整答案且展示来源不超过 3 项。

### 原因
- 固定证据评分必须使用完整召回切片、展示来源仍只取前三项的行为，防止正确证据位于第 4–8 位时再次被错误拒答。

### 影响范围
- 仅增加 RAG 服务回归测试，不修改生产问答逻辑、检索排序或对外来源契约。

## 2026-07-18 22:39 - 修复本地事实答案互补覆盖

### 变更内容
- `internal/service/generation_service.go`：改用通用事实维度保护高度、位置、工艺与文化的互补句，移除位置查询及文化评分中的景区特定答案词，并将情绪包装后的最终回答限制在 700 rune、最多 4 条句子。
- `internal/service/generation_service_test.go`：新增不依赖可变语料的合成表驱动回归测试，覆盖同名背景句竞争、部分首句覆盖、通用文化与工艺解释，以及最终长度和句数边界。

### 原因
- 首条候选只覆盖部分事实时会被错误裁成单句，且数据集特定评分词会降低规则迁移性；长度限制在情绪包装前执行还可能使最终回答超限。

### 影响范围
- 仅影响本地 RAG 降级答案的句子选择与最终裁剪，不改变检索排序、外部 LLM prompt 或 API 契约。

## 2026-07-18 22:49 - 刷新 Phase 1 RAG 检查点报告

### 变更内容
- `docs/eval-results/rag-local-2026-07-18-d0b11cd-dirty.json`：记录纯本地生成模式的 5 题评测结果与环境元数据。
- `docs/eval-results/rag-retrieval-2026-07-18-d0b11cd-dirty.json`：记录 retrieval-only 模式的 5 题召回结果与环境元数据。

### 原因
- 为 RAG 评测可信化、事实答案修复和完整切片证据评分提供可复查的同日检查点证据，并明确区分本地生成与纯检索模式。

### 影响范围
- 仅新增被 Git 忽略的本地评测报告与变更记录；报告不包含密钥、Token 或外部服务凭据，最终提交后仍需按最终 commit 刷新。

## 2026-07-18 23:03 - 完成 CSP 路由隔离浏览器验证

### 变更内容
- `docs/digital-human-production-check.md`：记录 Chromium/Playwright 对 `/map` 与 `/digital-human` 的响应 CSP、资源加载和控制台结果，并将外部 Live2D/WebSocket 验证明示为未测量。

### 原因
- 用真实浏览器确认普通 Vue/Naive UI 路由不需要 `'unsafe-eval'`，且该权限只在数字人文档路由中出现，避免仅凭单元测试声称 CSP 收尾完成。

### 影响范围
- 仅更新数字人生产检查证据；不修改 CSP 逻辑，外部 Open-LLM-VTuber 未启动，因此不声称 Live2D 或 WebSocket 联调通过。

## 2026-07-18 23:12 - 补强 TokenVersion 撤销回归验证

### 变更内容
- `internal/pkg/middleware_test.go`：分别使用 HTTP Bearer 与 WebSocket 子协议传递 JWT，覆盖旧版本撤销前后、新版本生效以及可选认证匿名降级。
- `internal/handler/user_handler_test.go`：通过真实管理员删除接口和数据库校验，覆盖用户删除前 HTTP/WebSocket token 可用、删除后均被拒绝。

### 原因
- 原 WebSocket 撤销测试错误使用不受支持的 Authorization header，实际只验证了缺失 token；同时缺少用户删除后的会话撤销回归。

### 影响范围
- 仅补充认证安全回归测试，不修改生产认证、用户删除或 JWT 实现。
