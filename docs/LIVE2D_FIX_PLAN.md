# Live2D 数字人优化遗留问题修复指南

> **项目**: scenic-guide (D:\go web 01\scenic-guide)  
> **日期**: 2026-06-11  
> **状态**: Phase 1-4 核心功能已完成（~90%），6 项遗留问题待修复  
> **验证通过**: `vue-tsc --noEmit` 零错误, `vite build` 成功, `go test ./internal/handler/` 70/70 通过

---

## 背景

Live2D 数字人体验全面优化（Phase 1-4）的核心功能已实现：

- ✅ 8 表情 (exp_01–08) 映射 + 唇形同步（Live2DStage.vue）
- ✅ Hit Area 点击交互（头部/身体）+ 点击涟漪动画
- ✅ 打字机效果（useTypewriter.ts）+ Markdown 渲染（MarkdownRenderer.vue）
- ✅ 上下文感知快捷提问 + 消息搜索 + 会话抽屉
- ✅ 后端情感系统（emotion.go: 80+ 关键词 + 权重 + 强度评分）
- ✅ localStorage + 后端双写会话持久化
- ✅ 触屏手势（双指缩放 0.05–0.25 / 单指拖动 + 300ms 弹性回弹）
- ✅ 长按语音按钮（500ms 阈值）
- ✅ PWA manifest + viewport meta

但审计发现有 **6 项与计划偏差**，本指南逐一说明修复方法。

---

## 修复概览

| # | 优先级 | 修复内容 | 涉及文件 |
|---|--------|----------|----------|
| 1 | 🔴 高 | ROADMAP.md 变更日志更新 | `docs/ROADMAP.md` |
| 2 | 🔴 高 | 动作映射精确化（状态→特定动作） | `web-vue/src/components/Live2DStage.vue` |
| 3 | 🟡 中 | Mobile 底部 Tab 切换（<768px） | `web-vue/src/views/DigitalHumanView.vue` |
| 4 | 🟡 中 | 提取 ThinkingIndicator 为独立组件 | 新建 + 修改 Live2DStage.vue |
| 5 | 🟡 中 | manifest.json 迁移至 public/ | 移动文件 + 修改 index.html |
| 6 | 🟡 中 | resize/scroll 事件节流 | Live2DStage.vue + DigitalHumanView.vue |

---

## 🔴 修复 1: ROADMAP.md 变更日志更新

**文件**: `docs/ROADMAP.md`

### 当前状态
变更日志表最后一条是 `2026-06-10`，无 Live2D 优化记录。

### 修复操作
在变更日志表追加一行：

```markdown
| 2026-06-11 | Live2D 数字人体验全面优化 Phase 1-4：8 表情 + 唇形同步 + Hit Area 交互 + 打字机效果 + Markdown 渲染 + 上下文感知快捷提问 + 消息搜索 + 会话抽屉 + 后端情感系统强化（80+关键词+强度评分）+ 会话持久化双写 + 触屏手势 + 长按语音 + PWA 基础支持 |
```

### 验证
- 打开 `docs/ROADMAP.md`，确认变更日志表包含 `2026-06-11` 条目

---

## 🔴 修复 2: 动作映射精确化

**文件**: `web-vue/src/components/Live2DStage.vue`

### 问题
当前 `STATE_MOTION_MAP` 中所有状态都映射为 `['Idle']`，watch handler 只对 `speaking` 和 `interrupted` 播放随机动作。不同对话状态（listening/thinking/connecting）缺少差异化动作表现。

### 计划要求的状态→动作映射

| 状态 | 应播放动作 | mao_pro 模型中的位置 |
|------|-----------|---------------------|
| `idle` | `Idle` 组（随机） | `motion('Idle')` |
| `connecting` | `mtn_01` | `motion('', 0)` |
| `listening` | `special_01` | `motion('', 3)` |
| `thinking` | `special_02` | `motion('', 4)` |
| `speaking` | `mtn_02`/`mtn_03`/`mtn_04` 随机 | `motion('', 1)` / `motion('', 2)` / `motion('', 5)`（注：special_03 是 index 5） |
| `interrupted` | `special_03` | `motion('', 5)` |
| `error` | `Idle` | `motion('Idle')` |

### mao_pro 动作索引说明

模型文件位于 `static/live2d-models/mao_pro/runtime/mao_pro.model3.json`。  
空字符串组 `""` 包含 6 个动作，其索引对应：

| 索引 | 动作名 | 用途 |
|------|--------|------|
| 0 | mtn_01 | 等待 / 连接中 |
| 1 | mtn_02 | 说话（随机池） |
| 2 | mtn_03 | 说话（随机池） |
| 3 | special_01 | 专注听 |
| 4 | special_02 | 思考 |
| 5 | special_03 (或 mtn_04) | 惊讶 |

> **注意**: 索引与具体 mao_pro 模型的 .model3.json 中 Motions 数组顺序对应。最可靠的方式是读取 `model3.json` 的 `Motions` 字段确认顺序。以上索引基于常见的 mao_pro 配置。

### 修复步骤

#### 步骤 A: 修改 `STATE_MOTION_MAP`（约第 299 行）

找到：
```ts
const STATE_MOTION_MAP: Record<ConversationState, MotionGroup[]> = {
  idle: ['Idle'],
  connecting: ['Idle'],
  listening: ['Idle'],
  thinking: ['Idle'],
  speaking: ['Idle'],
  interrupted: ['Idle'],
  error: ['Idle'],
};
```

替换为（使用空字符串组 + 索引）：
```ts
// Motion index in the "" group of mao_pro:
// 0=mtn_01(waiting), 1=mtn_02(speaking), 2=mtn_03(speaking),
// 3=special_01(listening), 4=special_02(thinking), 5=special_03(surprised)
const STATE_MOTION_INDEX: Record<ConversationState, number | null> = {
  idle: null,        // Uses Idle group, not "" group
  connecting: 0,     // mtn_01
  listening: 3,      // special_01
  thinking: 4,       // special_02
  speaking: -1,      // -1 means random from speaking pool [1, 2, 5]
  interrupted: 5,    // special_03
  error: null,       // Uses Idle group
};

const SPEAKING_MOTION_POOL = [1, 2, 5]; // mtn_02, mtn_03, mtn_04/special_03
```

> 如果你想保守一点，也可以保留原 `STATE_MOTION_MAP` 结构但改用 `['']` 配合索引。这里用 `STATE_MOTION_INDEX` 更清晰地表达意图。

#### 步骤 B: 修改 watch handler（约第 553 行）

找到：
```ts
watch(() => props.state, (newState, oldState) => {
  if (!live2dLoaded.value) return;
  if (newState === oldState) return;

  // Play motion when entering speaking state
  if (newState === 'speaking' && oldState !== 'speaking') {
    playMotion('', getRandomMotionIndex());
  }
  if (newState === 'interrupted') {
    playMotion('', getRandomMotionIndex());
  }
});
```

替换为：
```ts
watch(() => props.state, (newState, oldState) => {
  if (!live2dLoaded.value) return;
  if (newState === oldState) return;

  const index = STATE_MOTION_INDEX[newState];
  if (index === null || index === undefined) {
    // Use Idle group for idle/error
    if (newState === 'idle' || newState === 'error') {
      playMotion('Idle');
    }
    return;
  }

  if (index === -1) {
    // Speaking: pick random from speaking pool
    const randomIdx = SPEAKING_MOTION_POOL[Math.floor(Math.random() * SPEAKING_MOTION_POOL.length)];
    playMotion('', randomIdx);
  } else {
    playMotion('', index);
  }
});
```

#### 步骤 C: 可删除 `getRandomMotionIndex`（约第 333 行）

如果原 `getRandomMotionIndex()` 不再被其他地方调用，可以删除。

### 验证
- 打开数字人页面
- `connecting` 状态时应看到 mtn_01 动作
- 发送消息 → `thinking` 时应看到 special_02
- 开始说话 → `speaking` 时应随机看到 mtn_02/mtn_03/mtn_04
- 打断 → `interrupted` 时应看到 special_03

---

## 🟡 修复 3: Mobile 底部 Tab 切换

**文件**: `web-vue/src/views/DigitalHumanView.vue`

### 问题
`<768px` 时计划要求 **底部 Tab 切换**（Tab 1: 数字人, Tab 2: 聊天），但当前只是简单的 `flex-direction: column` 堆叠。

### 修复步骤

#### 步骤 A: 添加响应式状态（在 `<script setup>` 顶部附近，约第 49 行后）

```ts
// Mobile tab switching
const mobileTab = ref<'stage' | 'chat'>('stage');
const isMobileView = ref(false);

function checkMobile() {
  isMobileView.value = window.innerWidth < 768;
}
```

在 `onMounted` 中添加：
```ts
checkMobile();
window.addEventListener('resize', checkMobile);
```

在 `onUnmounted` 中添加：
```ts
window.removeEventListener('resize', checkMobile);
```

#### 步骤 B: 在 template 中添加底部 Tab 栏

在 `</main>` 闭合标签之前（约第 877 行前），即 `</aside>` 之后、`<!-- 会话抽屉 -->` 之前，添加：

```html
<!-- 移动端底部 Tab 栏 -->
<nav v-if="isMobileView" class="mobile-tabs">
  <button
    class="mobile-tab"
    :class="{ active: mobileTab === 'stage' }"
    @click="mobileTab = 'stage'"
  >
    🤖 {{ $t('dh.tabDigitalHuman') }}
  </button>
  <button
    class="mobile-tab"
    :class="{ active: mobileTab === 'chat' }"
    @click="mobileTab = 'chat'"
  >
    💬 {{ $t('dh.tabChat') }}
  </button>
</nav>
```

#### 步骤 C: 给 stage 和 chat 区域添加 v-show

在 `<section class="dh-stage">` 上添加：
```html
<section class="dh-stage" v-show="!isMobileView || mobileTab === 'stage'">
```

在 `<aside class="dh-chat">` 上添加：
```html
<aside class="dh-chat" v-show="!isMobileView || mobileTab === 'chat'">
```

#### 步骤 D: 添加 CSS（在 `</style>` 之前，约第 1286 行）

```css
/* 移动端底部 Tab 栏 */
.mobile-tabs {
  display: flex;
  height: 48px;
  border-top: 1px solid rgba(255, 255, 255, 0.08);
  background: rgba(10, 10, 15, 0.95);
  backdrop-filter: blur(10px);
  flex-shrink: 0;
}
.mobile-tab {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  border: none;
  background: transparent;
  color: rgba(255, 255, 255, 0.4);
  font-size: 13px;
  cursor: pointer;
  transition: all 0.2s;
  border-bottom: 2px solid transparent;
}
.mobile-tab.active {
  color: var(--sg-jade-bright, #63e2b7);
  border-bottom-color: var(--sg-jade-bright, #63e2b7);
  background: rgba(99, 226, 183, 0.06);
}
```

#### 步骤 E: 修正 `<768px` 的 CSS（约第 1266 行）

将现有的 `@media (max-width: 768px)` 块修改为让 stage 全高、chat 全高：

```css
@media (max-width: 768px) {
  .dh-view {
    flex-direction: column;
  }
  .dh-stage {
    flex: 1;
    height: 100%;
    min-height: 0;
  }
  .dh-chat {
    width: 100%;
    min-width: unset;
    flex: 1;
    border-left: none;
    border-top: none;
    min-height: 0;
  }
  .subtitle-bubble { max-width: 90%; }
  .emotion-bar { gap: 4px; }
  .emotion-btn { width: 28px; height: 28px; font-size: 13px; }
  .session-drawer { width: 100%; max-width: 100vw; }
}
```

### i18n keys 需要添加
如果 `$t('dh.tabDigitalHuman')` 和 `$t('dh.tabChat')` 不存在：

在 `web-vue/src/locales/zh-CN.json` 的 `dh` 节点下添加：
```json
"tabDigitalHuman": "数字人",
"tabChat": "聊天"
```

在 `web-vue/src/locales/en-US.json` 的 `dh` 节点下添加：
```json
"tabDigitalHuman": "Avatar",
"tabChat": "Chat"
```

### 验证
- Chrome DevTools 模拟 375px 宽度
- 应看到底部 Tab 栏，默认显示数字人
- 点击"聊天" Tab → 切换到聊天面板
- 点击"数字人" Tab → 切回数字人

---

## 🟡 修复 4: 提取 ThinkingIndicator 为独立组件

### 问题
思考动画（三个跳动圆点）内嵌在 Live2DStage.vue 的 template 中（第 599-603 行），计划要求独立为 `ThinkingIndicator.vue` 组件。

### 修复步骤

#### 步骤 A: 新建 `web-vue/src/components/ThinkingIndicator.vue`

```vue
<script setup lang="ts">
// Pure presentational component — no props needed.
// Parent controls visibility via v-if.
</script>

<template>
  <div class="thinking-indicator">
    <span class="think-dot" style="animation-delay: 0s" />
    <span class="think-dot" style="animation-delay: 0.2s" />
    <span class="think-dot" style="animation-delay: 0.4s" />
  </div>
</template>

<style scoped>
.thinking-indicator {
  position: absolute;
  top: 16%;
  left: 50%;
  transform: translateX(-50%);
  display: flex;
  gap: 6px;
  z-index: 3;
  pointer-events: none;
}
.think-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: linear-gradient(135deg, var(--sg-cyan, #52f0ee), var(--sg-jade-bright, #63e2b7));
  animation: dot-bounce 1.2s ease-in-out infinite;
}
@keyframes dot-bounce {
  0%, 80%, 100% { transform: translateY(0); opacity: 0.4; }
  40% { transform: translateY(-10px); opacity: 1; }
}
</style>
```

#### 步骤 B: 修改 Live2DStage.vue

**在 `<script setup>` 中添加 import**（约第 8 行后）：
```ts
import ThinkingIndicator from './ThinkingIndicator.vue';
```

**替换 template 中的内联 thinking-indicator**（第 599-603 行）：
```html
<!-- 旧代码：删除以下 5 行 -->
<div v-if="state === 'thinking' && live2dLoaded" class="thinking-indicator">
  <span class="think-dot" style="animation-delay: 0s" />
  <span class="think-dot" style="animation-delay: 0.2s" />
  <span class="think-dot" style="animation-delay: 0.4s" />
</div>

<!-- 新代码：替换为 -->
<ThinkingIndicator v-if="state === 'thinking' && live2dLoaded" />
```

**删除 Live2DStage.vue 的 `<style scoped>` 中旧的 thinking-indicator 样式**（第 658-677 行）：
```css
/* 删除以下所有 */
.thinking-indicator { ... }
.think-dot { ... }
@keyframes dot-bounce { ... }
```

### 验证
- 发送消息 → 在思考状态时，三个圆点应正常跳动
- `vue-tsc --noEmit` 无错误

---

## 🟡 修复 5: manifest.json 迁移 + index.html 修正

### 问题
- `manifest.json` 创建在了 `static/manifest.json`（后端静态目录）
- 应位于 `web-vue/public/manifest.json`（Vite 静态根目录）
- `index.html` 中的 `<link rel="manifest" href="/static/manifest.json">` 指向错误路径

### 修复步骤

#### 步骤 A: 移动文件
```bash
# 在项目根目录 D:\go web 01\scenic-guide\ 执行
mv static/manifest.json web-vue/public/manifest.json
```

#### 步骤 B: 修改 `web-vue/index.html` 第 11 行

当前：
```html
<link rel="manifest" href="/static/manifest.json" />
```

改为：
```html
<link rel="manifest" href="/manifest.json" />
```

> 原因: Vite 的 `public/` 目录直接映射到构建输出的根路径。`public/manifest.json` → 访问路径为 `/manifest.json`。

### 验证
- 确认 `web-vue/public/manifest.json` 存在
- 确认 `static/manifest.json` 已删除
- `vite build` 成功后，`web-vue/dist/manifest.json` 应该存在
- 浏览器 DevTools → Application → Manifest 应能读取到

---

## 🟡 修复 6: resize/scroll 事件节流

### 问题
计划要求 resize 和 scroll 使用 `requestAnimationFrame` 节流，但当前：
- Live2DStage.vue 的 resize handler 直接绑定 `syncLive2DLayout`（第 212 行）
- DigitalHumanView.vue 的 message-list 无 scroll 节流

### 修复步骤

#### 步骤 A: Live2DStage.vue resize rAF 节流（约第 212 行）

在 `<script setup>` 中添加 rAF 包装（约第 26 行附近，其他 let 变量之后）：

```ts
let resizeRafId = 0;
```

修改 `loadLive2DModel` 中的 resize 绑定（第 212 行）：

当前：
```ts
window.addEventListener('resize', syncLive2DLayout);
```

改为：
```ts
window.addEventListener('resize', onResize);
```

添加节流函数（放在 `syncLive2DLayout` 函数附近）：
```ts
function onResize() {
  if (resizeRafId) return;
  resizeRafId = requestAnimationFrame(() => {
    resizeRafId = 0;
    syncLive2DLayout();
  });
}
```

修改 `onUnmounted`（第 574 行）：
```ts
// 将 window.removeEventListener('resize', syncLive2DLayout);
// 改为：
window.removeEventListener('resize', onResize);
window.cancelAnimationFrame(resizeRafId);
```

#### 步骤 B: DigitalHumanView.vue 的 message-list scroll 被动监听

找到 message-list 的 div（第 806 行）：
```html
<div v-if="!showSearch || !searchQuery" class="message-list">
```

由于 Vue 模板中 `@scroll.passive` 可以直接写，改为：
```html
<div v-if="!showSearch || !searchQuery" class="message-list" @scroll.passive>
```

> 说明: `.passive` 修饰符告诉浏览器该事件不会调用 `preventDefault()`，浏览器可以立即滚动而不等待 JS 执行。这是移动端滚动性能的最佳实践。

### 验证
- 快速 resize 浏览器窗口 → Live2D 不应出现明显闪烁或卡顿
- 移动端滚动消息列表 → 应流畅无卡顿

---

## 最终验证清单

完成所有 6 项修复后，运行以下验证：

```bash
# 前端类型检查
cd web-vue && npx vue-tsc --noEmit

# 前端构建
npx vite build

# 后端测试
cd .. && go test ./internal/handler/ -run TestDetect -v

# 后端构建
go build ./...
```

手动验证：
1. [ ] 打开数字人页面 → `connecting` 状态有差异化动作
2. [ ] 发送消息 → `thinking` 时播放 special_02，`speaking` 时随机播放 mtn_02/03/04
3. [ ] 打断 → `interrupted` 时播放 special_03
4. [ ] Chrome DevTools 模拟 375px → 底部 Tab 可切换数字人/聊天
5. [ ] ThinkingIndicator 三个圆点在思考时正常跳动
6. [ ] `docs/ROADMAP.md` 包含 2026-06-11 变更记录
7. [ ] `vue-tsc --noEmit` 零错误
8. [ ] `vite build` 成功
9. [ ] `go test` 全部通过

---

## 关键文件路径汇总

```
scenic-guide/
├── docs/ROADMAP.md                                    # 修复 1
├── static/manifest.json                               # 修复 5: 移走
├── web-vue/
│   ├── index.html                                     # 修复 5: 修改 manifest href
│   ├── public/
│   │   └── manifest.json                              # 修复 5: 移到这里
│   └── src/
│       ├── components/
│       │   ├── Live2DStage.vue                        # 修复 2, 4, 6
│       │   └── ThinkingIndicator.vue                  # 修复 4: 新建
│       ├── views/
│       │   └── DigitalHumanView.vue                   # 修复 3, 6
│       └── locales/
│           ├── zh-CN.json                             # 修复 3: 新增 tab i18n
│           └── en-US.json                             # 修复 3: 新增 tab i18n
```

## 类型定义参考

以下类型已在 `web-vue/src/types/digitalHuman.ts` 中定义，可直接使用：

```ts
export type ConversationState = 'idle' | 'connecting' | 'listening' | 'thinking' | 'speaking' | 'interrupted' | 'error';
export type Live2DExpression = 'neutral' | 'happy' | 'sad' | 'angry' | 'thinking' | 'surprised' | 'interrupted' | 'blush' | 'idle';
export type MotionGroup = 'Idle' | 'Tap' | 'FlickUp' | 'Flick3' | '';
export type EmotionToken = 'neutral' | 'joy' | 'sadness' | 'surprise' | 'anger' | 'fear' | 'disgust';
```

## 已安装依赖

- `marked` — Markdown 解析
- `dompurify` — XSS 防护
- `pixi.js` + `pixi-live2d-display/cubism4` — Live2D 渲染
- `vue-i18n` — 国际化

不需要安装新依赖即可完成以上修复。
