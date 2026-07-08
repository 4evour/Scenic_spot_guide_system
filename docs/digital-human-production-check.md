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

## 检查项

- 页面能自动创建或恢复游客 Cookie 会话，并进入数字人导览界面。
- 页面左侧加载当前 Live2D 模型，默认模型为 `mao_pro`，管理员允许切换时可选择 `shizuku`。
- WebSocket 连接建立后状态从“正在连接”变为在线；外部数字人服务未启动时，文字聊天仍可通过 Go 后端降级运行。
- 发送文本问题后，聊天面板显示用户问题和数字人回答；回答会同步写入 localStorage、Pinia 状态和后端会话消息接口。
- 点击“启用声音”后，后续回答会尝试走 Go TTS；TTS 失败时降级到浏览器朗读，不阻塞文字输出。
- 点击“打断回答”会停止本地音频并向已打开 WebSocket 发送 `interrupt-signal`。
- 历史会话抽屉可加载后端会话列表；搜索框可搜索当前会话和后端历史消息。
- 管理端 `/admin/avatar` 修改默认数字人或禁止游客切换后，游客页可选模型列表随 `/api/v1/digital-human/avatar-options` 同步变化。

## 验证命令

```powershell
go run ./cmd/rag-eval -knowledge knowledge/lingshan_scale_3000.jsonl -eval knowledge/lingshan_eval_300.json -k 8 -bench -concurrency 16 -repeat 1 -retrieval-only
go test ./...
```

## 口径边界

- Open-LLM-VTuber 和 Live2D/PixiJS 是外部框架能力。
- 本项目负责景区业务后端、RAG 问答、OpenAI 兼容接口、WebSocket 代理、前端二开注入层和联调状态增强。
- 简历中应写“协议适配 + 前端二开 + 联调测试”，不要暗示从零实现数字人底层框架。
