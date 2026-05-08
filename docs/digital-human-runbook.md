# 数字人运行手册

## 1. 快速启动

### 1.1 启动 Go 后端

```bash
cd d:\go web 01\scenic-guide

# 安装依赖
go mod tidy

# 启动服务
go run main.go
```

**预期输出：**
```
=== 启动景区导览服务 ===
步骤1: 加载配置...
配置加载成功
步骤2: 初始化日志...
日志初始化成功
步骤2.5: 初始化JWT...
JWT初始化成功
步骤3: 初始化数据库...
数据库连接成功
步骤4: 数据库迁移...
数据库迁移成功
步骤4.5: 初始化RAG知识库...
RAG知识库初始化成功
步骤5: 设置路由...
路由设置成功
步骤6: 启动服务器，监听地址: 0.0.0.0:8080
```

### 1.2 验证服务

```bash
# 检查健康状态
curl http://localhost:8080/api/v1/dh/health
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

## 2. API 测试

### 2.1 创建会话

```bash
curl -X POST http://localhost:8080/api/v1/dh/session/create \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "test_user",
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

创建 `.env` 文件：

```env
GO_BACKEND_BASE_URL=http://127.0.0.1:8080
GO_DH_SESSION_API=/api/v1/dh/session/create
GO_DH_TEXT_API=/api/v1/dh/chat/text
GO_DH_VOICE_API=/api/v1/dh/chat/voice-transcript
GO_DH_FEEDBACK_API=/api/v1/dh/feedback

ASR_ENGINE=faster_whisper
ASR_LANGUAGE=zh
ASR_MODEL_SIZE=small

TTS_ENGINE=edge_tts
TTS_VOICE=zh-CN-XiaoxiaoNeural
```

### 3.3 启动数字人前端

参考 Open-LLM-VTuber 官方文档启动。

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

### 6.3 Docker部署（可选）

```dockerfile
FROM golang:1.21-alpine
WORKDIR /app
COPY . .
RUN go build -o scenic-guide .
EXPOSE 8080
CMD ["./scenic-guide"]
```