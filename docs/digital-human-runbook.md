# 数字人运行手册

## 1. 快速启动

### 1.1 一键启动本地联调

```powershell
cd "D:\go web 01\scenic-guide"
.\scripts\start-local.ps1 -Restart
```

脚本会初始化本地 SQLite 演示数据，并启动 Go 服务与 `127.0.0.1:12393` 的 Open-LLM-VTuber 服务。日志写入 `..\tmp\scenic-guide-start`。

### 1.2 验证服务

```bash
curl http://localhost:8080/health
curl http://localhost:8080/api/v1/dh/health
curl http://localhost:12393/
```

**预期响应：**
```json
{
  "status": "healthy",
  "rag_service": "available",
  "route_service": "available",
  "timestamp": "2024-01-01T12:00:00Z"
}
```

### 1.3 打开主入口

```text
http://127.0.0.1:8080/digital-human#/login
```

## 2. API 测试

`/api/v1/dh/session/create`、`/api/v1/dh/chat/text`、`/api/v1/dh/chat/voice-transcript` 和 `/api/v1/dh/feedback` 都是登录后接口。浏览器主路径使用 `auth_token` HttpOnly Cookie；POST 请求还需要 `csrf_token` Cookie 对应的 `X-CSRF-Token` 请求头。命令行调试时先登录并保存 Cookie。

### 2.0 登录并保存 Cookie

```bash
curl -i -c cookies.txt -X POST http://localhost:8080/api/v1/login \
  -H "Content-Type: application/json" \
  -d '{"username":"visitor","password":"ScenicDemo123456"}'
```

从 `cookies.txt` 读取 `csrf_token` 后写入环境变量：

```bash
CSRF_TOKEN=$(awk '$6 == "csrf_token" {print $7}' cookies.txt)
```

### 2.1 创建会话

```bash
curl -X POST http://localhost:8080/api/v1/dh/session/create \
  -b cookies.txt \
  -H "X-CSRF-Token: $CSRF_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "scene": "lingshan",
    "location": "入口广场",
    "preferences": ["亲子", "2小时"]
  }'
```

**响应示例：**
```json
{
  "session_id": "abc123",
  "expires_at": "2024-01-01T12:30:00Z"
}
```

### 2.2 文本聊天

```bash
curl -X POST http://localhost:8080/api/v1/dh/chat/text \
  -b cookies.txt \
  -H "X-CSRF-Token: $CSRF_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "session_id": "abc123",
    "input_text": "灵山有什么好玩的？",
    "scene": "lingshan",
    "location": "入口广场"
  }'
```

**响应示例：**
```json
{
  "answer_text": "灵山景区有很多景点，推荐您参观灵山大佛、梵宫等著名景点...",
  "emotion": "warm",
  "route_payload": null,
  "trace_id": "xyz789"
}
```

### 2.3 语音识别聊天

```bash
curl -X POST http://localhost:8080/api/v1/dh/chat/voice-transcript \
  -b cookies.txt \
  -H "X-CSRF-Token: $CSRF_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "session_id": "abc123",
    "transcript": "推荐一条亲子路线",
    "confidence": 0.95,
    "location": "入口广场"
  }'
```

**响应示例：**
```json
{
  "answer_text": "为您推荐亲子友好路线：入口广场 -> 百子戏弥勒 -> 灵山胜境...",
  "emotion": "happy",
  "route_payload": {
    "route_id": "亲子路线A",
    "stops": ["入口广场", "百子戏弥勒", "灵山胜境"]
  },
  "trace_id": "xyz012"
}
```

### 2.4 上报反馈

```bash
curl -X POST http://localhost:8080/api/v1/dh/feedback \
  -b cookies.txt \
  -H "X-CSRF-Token: $CSRF_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "session_id": "abc123",
    "trace_id": "xyz789",
    "question_type": "景点推荐",
    "response_time_ms": 1500,
    "rating": 5,
    "comment": "回答很详细，非常满意"
  }'
```

## 3. 与 Open-LLM-VTuber 对接

### 3.1 拉取数字人项目

```bash
git clone https://github.com/Open-LLM-VTuber/Open-LLM-VTuber.git
cd Open-LLM-VTuber
```

### 3.2 配置环境变量

当前本地启动流程会启动 `127.0.0.1:12393`。Open-LLM-VTuber 的 `conf.yaml` 需要让它调用 Go 的 OpenAI 兼容接口：

```env
LLM_API_URL=http://127.0.0.1:8080/v1/chat/completions
LLM_API_KEY=not-needed
```

### 3.3 启动数字人前端

本项目主交付页面使用 Go 托管的 Vue 登录页和数字人页。Open-LLM-VTuber 自带页面 `http://127.0.0.1:12393/` 仅用于确认外部数字人服务是否启动。

## 4. 常见问题

### 4.1 服务启动失败

**问题：**
```
listen tcp 0.0.0.0:8080: bind: Only one usage of each socket address...
```

**解决：**
```bash
# Windows
netstat -ano | findstr :8080
taskkill /F /PID <PID>

# Linux/Mac
lsof -ti:8080 | xargs kill -9
```

### 4.2 RAG服务不可用

**问题：** RAG知识库未初始化

**解决：**
```bash
# 检查配置文件 configs/config.yaml
# 确保 AI.APIKey 和 Embedding.APIKey 已配置

# 检查知识库文件
ls knowledge/
```

### 4.3 API返回错误

**问题：** 404 Not Found

**解决：**
```bash
# 确保服务正在运行
curl http://localhost:8080/health

# 检查API路径是否正确
# 正确路径：/api/v1/dh/chat/text
```

### 4.4 情感识别不准确

**问题：** emotion 总是返回 neutral

**解决：**
- 检查回答内容是否包含情感关键词
- 可在 `digital_human_handler.go` 中调整 `detectEmotion` 函数

## 5. 性能指标

### 5.1 预期响应时间

| API | 预期时间 | 最大时间 |
|-----|---------|---------|
| session/create | < 50ms | 100ms |
| chat/text | < 2000ms | 5000ms |
| voice-transcript | < 2500ms | 6000ms |
| feedback | < 50ms | 100ms |

### 5.2 监控建议

```bash
# 使用 curl 测试响应时间
curl -w "Time: %{time_total}s\n" http://localhost:8080/api/v1/dh/health
```

Prometheus 指标端点为 `/metrics`，需要管理员 Cookie 或 Bearer token 鉴权。

## 6. 部署建议

### 6.1 开发环境

```bash
# 使用 go run
go run main.go
```

### 6.2 生产环境

```bash
# 编译
go build -o scenic-guide .

# 运行
./scenic-guide
```

### 6.3 Docker部署

```bash
docker compose up --build
```

`docker-compose.yml` 已包含 PostgreSQL healthcheck 和应用 `/health` healthcheck。生产环境需通过环境变量提供 `SCENIC_GUIDE_DATABASE_PASSWORD` 和 `SCENIC_GUIDE_SECURITY_JWT_SECRET`。
