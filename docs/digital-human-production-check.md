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

3. 打开数字人页面：

```text
http://127.0.0.1:12393/
```

## 检查项

- 顶部品牌面板显示“灵山智慧导览数字人”。
- 联调状态面板显示 WebSocket、麦克风、SSE、Trace 四类状态。
- WebSocket 连接建立后状态从“正在连接”变为“导览在线”。
- 麦克风被拒绝时弹出权限说明，并保留文本输入演示入口。
- 发送文本问题后，SSE/流式状态会短暂显示“流式回答中”。
- 后端返回 `trace_id` 或 `traceId` 时，面板显示当前 trace。
- 点击“打断回答”会停止本地音频并向已打开 WebSocket 发送 `interrupt-signal`。
- 点击“重连会话”会尝试使用最近一次 WebSocket URL 重新连接。

## 验证命令

```powershell
go run ./cmd/rag-eval -knowledge knowledge/lingshan_scale_3000.jsonl -eval knowledge/lingshan_eval_300.json -k 8 -bench -concurrency 16 -repeat 1 -retrieval-only
go test ./...
```

## 口径边界

- Open-LLM-VTuber 和 Live2D/PixiJS 是外部框架能力。
- 本项目负责景区业务后端、RAG 问答、OpenAI 兼容接口、WebSocket 代理、前端二开注入层和联调状态增强。
- 简历中应写“协议适配 + 前端二开 + 联调测试”，不要暗示从零实现数字人底层框架。
