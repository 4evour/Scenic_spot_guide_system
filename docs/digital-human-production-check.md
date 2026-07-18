# 数字人联调检查

本文档用于验证景区主系统与 Open-LLM-VTuber 前端二开层的集成状态。本项目不声明自研完整数字人框架，重点验证协议适配、OpenAI 兼容接口、SSE/WebSocket 联调和前端异常状态。

## 启动顺序

1. 启动 PostgreSQL 与 Go 服务：

```powershell
docker compose up --build
```

2. 启动 Open-LLM-VTuber 服务，确认默认 WebSocket 地址为：

```text
ws://127.0.0.1:12393/client-ws
```

3. 打开 Go 托管的 Vue 数字人页面：

```text
http://127.0.0.1:8080/digital-human#/digital-human
```

Open-LLM-VTuber 自带页面 `http://127.0.0.1:12393/` 只用于确认外部数字人服务是否启动，不作为本项目主交付入口。

## 认证与生产代理配置

- Go 服务的 JWT 密钥仅接受 64 个 hex 字符，或 base64 解码后至少 32 bytes。可用 `openssl rand -hex 32` 生成（需安装 OpenSSL）；不依赖 OpenSSL 的 PowerShell 生成方式见 `README.md`。旧版本把配置文本直接用于签名，新版本会先解码 hex/base64：即使升级前后的 64-hex 文本完全不变，新版也会改用解码后的 32 bytes，已有 JWT 仍会失效。多实例不得让新旧版本普通滚动混跑，应先同步同一个合规值，再协调或原子切换版本，并安排用户重新登录。格式错误不会回显输入密钥。
- `/vtuber-ws/*` 只接受同源浏览器自动携带的 HttpOnly `auth_token` Cookie，或 `auth.token.<JWT>` WebSocket 子协议。URL query 不再接受 JWT，不要使用旧的 `?token=` 接入方式。
- 生产环境位于反向代理后时，必须将 `SCENIC_GUIDE_TRUSTED_PROXIES` 配置为实际代理 IP/CIDR，多个值用逗号分隔，例如 `<reverse-proxy-ip>,<reverse-proxy-cidr>`。未配置时服务仍会启动并拒绝信任 `X-Forwarded-For`，按直连地址限流；当前程序不会因“生产环境未配置”而主动失败。非法 IP/CIDR 会导致可信代理初始化失败并终止启动。

## CSP 路由隔离检查

普通游客主站和管理端路由的 `script-src` 不允许 `'unsafe-eval'`；只有 `/digital-human` 及其子路由为 Live2D 运行时保留必要的 `'unsafe-eval'`。

- [x] 2026-07-18 使用 Chromium/Playwright 打开 `/map`：文档响应的 `script-src` 不含 `'unsafe-eval'`，Vue 与 Naive UI 资源均返回 200，控制台无 error 或 CSP 违规。
- [x] 2026-07-19 使用同一浏览器会话打开 `/digital-human`：文档响应的 `script-src` 包含 Live2D 所需的 `'unsafe-eval'`，控制台无 CSP 违规；Go RAG 兜底和移动端无横向溢出均已实际复测。
- [x] 2026-07-19 Open-LLM-VTuber 已启动并建立连接；外部生成错误和连接失速均能切换到 Go 后端 RAG，实测“灵山大佛有多高？”返回包含“通高 88 米”。
- [ ] Live2D Cubism Core 未随仓库部署：`/static/digital-human/libs/live2dcubismcore.min.js` 缺失时页面明确启用备用动效。生产发布前必须按 Live2D 授权将 Core 文件部署到该路径，不能把备用动效当作正式 Live2D 模型通过。
- [ ] ASR 和真实外部 TTS 尚未完成真实设备端到端测量；当前只确认文字链路、TTS 输入校验、Edge TTS 分段/异常帧拒绝及浏览器朗读降级测试通过。

## 检查项

- 页面能自动创建或恢复游客 Cookie 会话，并进入数字人导览界面。
- 页面左侧加载当前 Live2D 模型，默认且唯一公开模型为 Live2D 官方 `mao_pro` 临时示例；正式发布前应替换为具有独立授权的灵山专属模型。
- WebSocket 连接建立后状态从“正在连接”变为在线；外部数字人服务未启动时，文字聊天仍可通过 Go 后端降级运行。
- 发送文本问题后，聊天面板显示用户问题和数字人回答；回答会同步写入 localStorage、Pinia 状态和后端会话消息接口。
- 点击“启用声音”后，后续回答会尝试走 Go TTS；TTS 失败时降级到浏览器朗读，不阻塞文字输出。
- 点击“打断回答”会停止本地音频并向已打开 WebSocket 发送 `interrupt-signal`。
- 历史会话抽屉可加载后端会话列表；搜索框可搜索当前会话和后端历史消息。
- 管理端 `/admin/avatar` 修改默认数字人或禁止游客切换后，游客页可选模型列表随 `/api/v1/digital-human/avatar-options` 同步变化。

## 发布前置条件

- 必须部署已获授权的 Live2D Cubism Core 文件，并重新执行模型加载、表情、口型和 WebSocket 联调。
- 必须在目标浏览器和目标音频设备上完成一次 ASR -> 回答 -> TTS -> 播放的真实端到端测量，并导出语音 trace。
- 未满足以上两项时，Go RAG、文字问答、取消旧回答、异常降级和备用动效可以验收，但数字人“完整生产语音/Live2D”不能标记为通过。

## 验证命令

```powershell
go run ./cmd/rag-eval -knowledge knowledge/lingshan_scale_3000.jsonl -eval knowledge/lingshan_eval_300.json -k 8 -bench -concurrency 16 -repeat 1 -retrieval-only
go test ./...
```

## 口径边界

- Open-LLM-VTuber 和 Live2D/PixiJS 是外部框架能力。
- 本项目负责景区业务后端、RAG 问答、OpenAI 兼容接口、WebSocket 代理、前端二开注入层和联调状态增强。
- 简历中应写“协议适配 + 前端二开 + 联调测试”，不要暗示从零实现数字人底层框架。
