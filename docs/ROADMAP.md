# Scenic Guide 路线图 (Roadmap)

> 最后更新: 2026-06-10  
> 基于开源项目竞争调研制定

## 调研背景

调研了 GitHub/Gitee 上与景区导览相关的开源项目，与 scenic-guide 对比分析后制定本路线图。

核心参考项目:
- [TripStar](https://github.com/1sdv/TripStar) — ⭐1,800+ AI 多智能体旅行规划
- [WeTravel](https://github.com/nanbouking/WeTravel) — ⭐265 微信小程序景区平台
- [TravelGuide3D](https://github.com/kiranbaby14/Travel-Guide-3D) — 3D 电影级路线可视化
- [CyberVerse](https://github.com/Lynpoint/CyberVerse) — WebRTC 实时数字人平台

---

## 短期计划 (P0/P1)

| # | 任务 | 优先级 | 现状 | 目标 |
|---|------|--------|------|------|
| 1 | **实时语音 WebRTC 升级** | P0 | 语音转写→文本→TTS 三阶段 | WebRTC 流式语音 + 打断 |
| 2 | **微信小程序跨平台** | P0 | 仅 Web 端 | uni-app/Taro 多端覆盖 |
| 3 | **地图体验优化** | P1 | 2D 标注，无配图 | 景点配图 + 美观标注 |
| 4 | **多语言 i18n** | P1 | ~~仅中文~~ → ✅ 中/英 | 中/英双语 |
| 5 | **游客账号 + 会话记忆** | P1 | 30分钟超时清除 | 注册/登录 + 持久化记忆 |
| 6 | **GPS 定位主动导览** | P1 | 被动问答 | 到达景点自动触发讲解 |

## 长期计划 (P2)

| # | 任务 | 备注 |
|---|------|------|
| 7 | **预约订阅板块** | 独立票务系统，体量大需单独立项 |
| 8 | **Docker 一键部署** | docker-compose 完整编排 |

## 不做

- 多 Agent 架构: 当前单 Agent + RAG 已足够
- 知识图谱可视化: 非刚需
- 离线模式: 以联网场景为主

---

## 变更日志

| 日期 | 变更 |
|------|------|
| 2026-06-10 | 第4项多语言i18n实施：前端 vue-i18n + 中英 locale 文件，GlobalHeader 语言切换按钮，MapView/DigitalHumanView/LoginView/Admin 页面全面 i18n，后端 messages.go 消息目录 (~90 keys) + LanguageMiddleware，所有 handler 层用户可见消息改为 `pkg.T(c, key)`，LLM 系统提示追加语言指令，ChatRequest 新增 `lang` 参数 |
| 2026-06-10 | 初始创建，完成竞品调研 |
| 2026-06-11 | Live2D 数字人体验全面优化 Phase 1-4：8 表情 + 唇形同步 + Hit Area 交互 + 打字机效果 + Markdown 渲染 + 上下文感知快捷提问 + 消息搜索 + 会话抽屉 + 后端情感系统强化（80+关键词+强度评分）+ 会话持久化双写 + 触屏手势 + 长按语音 + PWA 基础支持 |
