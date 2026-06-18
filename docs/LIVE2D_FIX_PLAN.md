# Live2D 数字人遗留项完成记录

> 项目: scenic-guide
> 最后更新: 2026-06-18
> 状态: 遗留计划已审查并按当前代码实现校准

## 结论

Live2D Phase 1-4 后留下的 6 项计划已经处理完毕，旧计划中部分操作已过时或不适用于当前 Vite 构建路径。

| # | 原计划项 | 当前状态 | 说明 |
|---|---|---|---|
| 1 | ROADMAP.md 变更日志更新 | 已完成 | `docs/ROADMAP.md` 已包含 2026-06-11 Live2D 优化记录，并在 2026-06-18 同步当前路线图调整。 |
| 2 | 动作映射精确化 | 已完成 | `Live2DStage.vue` 使用 `STATE_MOTION_INDEX` 和 `SPEAKING_MOTION_POOL`，按 `mao_pro.model3.json` 的真实动作组映射状态。 |
| 3 | 移动端底部 Tab 切换 | 已完成 | `DigitalHumanView.vue` 已有 `mobileTab`、`isMobileView` 和底部 Tab；locale key 为 `dh.tabAvatar` / `dh.tabChat`。 |
| 4 | ThinkingIndicator 独立组件 | 已完成 | `ThinkingIndicator.vue` 是纯展示组件，父级用 `v-if` 控制显示。 |
| 5 | manifest.json 迁移 | 已完成且路径已校准 | `web-vue/public/manifest.json` 已存在；因 `vite.config.ts` build base 为 `/static/vue-app/`，`web-vue/index.html` 和构建产物中的 manifest 路径保持 `/static/vue-app/manifest.json`。旧计划中改为 `/manifest.json` 不适用于当前部署。 |
| 6 | resize/scroll 性能处理 | 已完成 | `Live2DStage.vue` 使用 `onResize` + `requestAnimationFrame` 同步布局；移动端聊天视图采用底部 Tab 分屏，避免同时渲染挤压。 |

## 当前 Live2D 实装边界

- 默认模型为 `mao_pro`，另有 `shizuku` 可选模型。
- 后端公开 `/api/v1/digital-human/avatar-options`，管理员可通过 `/api/v1/admin/digital-human/config` 配置默认数字人和是否允许游客切换。
- 游客偏好通过 `/api/v1/user/avatar-preference` 读写，前端选择后会同步 Open-LLM-VTuber 配置文件。
- 会话消息采用 localStorage 兜底、Pinia 实时展示和后端 `/api/v1/sessions/:session_id/messages` 持久化。

## 验证命令

```bash
cd web-vue
npm run check
npm run build

cd ..
go test ./internal/service ./internal/handler
```
