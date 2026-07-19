# 评委版项目演示视频 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 生成一条 285-315 秒、1280x720 的评委版景区导览系统演示视频，以真实页面操作为主体，插入统一科技感解释卡，使用沉稳男声配音并输出带/不带背景音乐两个版本。

**Architecture:** 用一个结构化场景清单描述页面路径、解释卡、旁白和目标时长；Playwright 按清单录制独立真实页面片段，AI 只生成无文字的解释卡背景，Playwright/HTML 统一叠加中文文字，Edge TTS 生成分段 MP3，FFmpeg 按清单合成最终视频。业务页面、API 和数据库不为视频改动。

**Tech Stack:** Node.js 20+、Playwright（复用 `web-vue` 依赖）、Microsoft Edge TTS CLI（`Open-LLM-VTuber/.venv/Scripts/edge-tts.exe`）、FFmpeg 8.x（libx264/aac/drawtext/loudnorm）、PowerShell、PNG/MP3/MP4。

## Global Constraints

- 受众是项目评委，叙事顺序必须是问题 -> 真实产品 -> 技术证据 -> 闭环收束。
- 真实页面录屏占主体，解释卡每段 8-12 秒；不使用假接口或预置截图冒充运行结果。
- 最终画面 1280x720，时长 285-315 秒；中文标题、正文和字幕必须由统一模板生成。
- AI 图片只生成无文字背景；所有中文文字在后期 HTML/Playwright 中叠加。
- 配音使用 Edge TTS 白名单中的沉稳男声 `male_yunjian` 或 `male_yunyang`，不可沿用默认女声。
- 不把 Open-LLM-VTuber、Live2D、Edge TTS 或其他外部框架描述为从零自研。
- 不读取、打印、提交 `.env`、Cookie、数据库、API Key 或完整运行日志。
- 所有项目文件变更都在根目录 `D:\go web 01\CHANGELOG.md` 追加时间、内容、原因和影响范围。

## 文件结构

- Create: `scenic-guide/docs/video/judge-demo-script.md` — 六段旁白、字幕、画面动作和口径边界。
- Create: `scenic-guide/scripts/judge-video/manifest.json` — 可机器读取的场景、资源、时长和合成顺序。
- Create: `scenic-guide/scripts/judge-video/validate-manifest.mjs` — 校验清单结构、总时长、资源唯一性和敏感字段。
- Create: `scenic-guide/scripts/judge-video/validate-manifest.test.mjs` — 清单校验的 Node 内置测试。
- Create: `scenic-guide/scripts/judge-video/render-cards.mjs` — 读取解释卡背景和清单，输出统一版式 PNG。
- Create: `scenic-guide/scripts/judge-video/record-scenes.mjs` — 登录本地演示服务并输出独立真实页面片段。
- Create: `scenic-guide/scripts/judge-video/generate-voice.ps1` — 用 Edge TTS 生成六段男声 MP3，并做响度预处理。
- Create: `scenic-guide/scripts/judge-video/compose.ps1` — 按 manifest 拼接卡片、录屏、配音、字幕和可选 BGM。
- Create: `scenic-guide/docs/assets/judge-video/backgrounds/` — 五张无文字 AI 背景位图；若资源已在本地则不覆盖。
- Create: `scenic-guide/tmp/video-work/` — 录屏、音频和中间片段，受 `.gitignore` 保护。
- Create: `scenic-guide/output/video/judge-demo-*.mp4` — 最终成片和无 BGM 版本，受 `.gitignore` 保护。
- Modify: `D:\go web 01\CHANGELOG.md` — 记录每次视频制作文件变更。

### Task 1: 固化旁白、字幕和场景清单

**Files:**
- Create: `docs/video/judge-demo-script.md`
- Create: `scripts/judge-video/manifest.json`
- Create: `scripts/judge-video/validate-manifest.mjs`
- Create: `scripts/judge-video/validate-manifest.test.mjs`

**Interfaces:**
- `manifest.json` 顶层字段：`version`, `canvas`, `scenes`, `outputs`。
- 每个 scene 字段：`id`, `kind` (`card|page`), `durationSec`, `asset`, `narration`, `subtitle`, `route`（page 必填）、`claimBoundary`。
- `validateManifest(manifest, { rootDir })` 返回 `{ valid: boolean, errors: string[] }`；不抛出用户输入错误。

- [ ] **Step 1: 写失败测试**

```js
import test from 'node:test';
import assert from 'node:assert/strict';
import { validateManifest } from './validate-manifest.mjs';

test('rejects a manifest outside the five-minute duration window', () => {
  const result = validateManifest({
    version: 1,
    canvas: { width: 1280, height: 720 },
    scenes: [{ id: 'one', kind: 'page', durationSec: 1, route: '/map', narration: 'x', subtitle: 'x', claimBoundary: 'tested' }],
    outputs: { withBgm: 'x.mp4', withoutBgm: 'y.mp4' },
  }, { rootDir: process.cwd() });
  assert.equal(result.valid, false);
  assert.match(result.errors.join('\n'), /285-315/);
});

test('rejects duplicate scene ids and forbidden secret-like fields', () => {
  const result = validateManifest({
    version: 1,
    canvas: { width: 1280, height: 720 },
    scenes: [
      { id: 'dup', kind: 'page', durationSec: 150, route: '/map', narration: 'x', subtitle: 'x', claimBoundary: 'tested' },
      { id: 'dup', kind: 'page', durationSec: 150, route: '/dashboard', narration: 'x', subtitle: 'x', claimBoundary: 'tested', api_key: 'secret' },
    ],
    outputs: { withBgm: 'x.mp4', withoutBgm: 'y.mp4' },
  }, { rootDir: process.cwd() });
  assert.equal(result.valid, false);
  assert.match(result.errors.join('\n'), /duplicate|sensitive/i);
});
```

- [ ] **Step 2: 运行测试确认失败**

Run: `node --test scripts/judge-video/validate-manifest.test.mjs`

Expected: FAIL because `validate-manifest.mjs` does not exist.

- [ ] **Step 3: 实现最小校验器**

```js
const MIN_SECONDS = 285;
const MAX_SECONDS = 315;
const SECRET_KEYS = /(?:api[_-]?key|token|cookie|password|secret)/i;

export function validateManifest(manifest, { rootDir = process.cwd() } = {}) {
  const errors = [];
  if (manifest?.canvas?.width !== 1280 || manifest?.canvas?.height !== 720) errors.push('canvas must be 1280x720');
  if (!Array.isArray(manifest?.scenes) || manifest.scenes.length === 0) errors.push('scenes are required');
  const ids = new Set();
  let total = 0;
  for (const scene of manifest.scenes || []) {
    if (!scene.id || ids.has(scene.id)) errors.push(`duplicate or missing scene id: ${scene.id || '<empty>'}`);
    ids.add(scene.id);
    if (!['card', 'page'].includes(scene.kind)) errors.push(`invalid kind for ${scene.id}`);
    if (!Number.isFinite(scene.durationSec) || scene.durationSec <= 0) errors.push(`invalid duration for ${scene.id}`);
    if (!scene.narration || !scene.subtitle || !scene.claimBoundary) errors.push(`copy fields missing for ${scene.id}`);
    if (scene.kind === 'page' && !scene.route) errors.push(`route missing for ${scene.id}`);
    if (scene.kind === 'card' && !scene.asset) errors.push(`asset missing for ${scene.id}`);
    total += Number(scene.durationSec) || 0;
    if (JSON.stringify(scene).match(SECRET_KEYS)) errors.push(`sensitive field detected in ${scene.id}`);
  }
  if (total < MIN_SECONDS || total > MAX_SECONDS) errors.push(`total duration must be 285-315 seconds, got ${total}`);
  if (!manifest?.outputs?.withBgm || !manifest?.outputs?.withoutBgm) errors.push('both output paths are required');
  return { valid: errors.length === 0, errors, totalSeconds: total, rootDir };
}

if (import.meta.url === `file://${process.argv[1]}`) {
  const fs = await import('node:fs');
  const manifest = JSON.parse(fs.readFileSync(process.argv[2] || 'scripts/judge-video/manifest.json', 'utf8'));
  const result = validateManifest(manifest);
  if (!result.valid) { console.error(result.errors.join('\n')); process.exit(1); }
  console.log(`manifest valid: ${result.totalSeconds}s`);
}
```

- [ ] **Step 4: 编写真实旁白和 manifest**

Use six exact narration blocks in `docs/video/judge-demo-script.md` and reference them from `manifest.json`:

```text
01 前景：景区导览真正难的，不是把页面做出来，而是在游客需要答案的现场，把分散的信息变成可信、及时、可执行的下一步。
02 游客端：游客从地图开始，查看景点详情，选择路线，并通过二维码进入导览。这里展示的不是静态宣传页，而是从接口到页面状态都能实际跑起来的游客入口。
03 AI：当游客直接提问，系统先从景区知识库检索相关内容，再以流式方式返回回答。继续追问时，系统保留当前会话主题，把“它有多高”这类问题承接为明确的景点问题。
04 数字人：数字人部分聚焦协议适配和前端二开：文本回答、连接状态、打断控制和外部服务不可用时的 Go RAG 降级，都在同一条体验里保持可用。外部框架的能力边界会被明确标注。
05 运营：游客侧产生的问题、反馈和热点，回到管理端的数据看板、知识库和游客问题处理流程。运营人员可以维护景点、路线和内容，让下一次导览继续变得更准确。
06 收束：最终形成的是一条可验证的闭环：游客获得导览，AI 提供检索增强回答，运营侧持续维护内容和反馈。项目重点不是堆叠名词，而是把真实页面、接口链路和可复现的评估边界放在一起。
```

Set the manifest durations to sum to 300 seconds, with 5 card scenes of 10 seconds each and page scenes carrying the remaining time. Set page routes to `/app#/map`, `/digital-human#/digital-human`, `/dashboard#/dashboard`, and `/admin#/admin`; set `claimBoundary` to the exact capability boundary shown in the script.

- [ ] **Step 5: 运行校验测试**

Run: `node --test scripts/judge-video/validate-manifest.test.mjs && node scripts/judge-video/validate-manifest.mjs scripts/judge-video/manifest.json`

Expected: PASS and `manifest valid: 300s`.

- [ ] **Step 6: Commit**

```powershell
git add docs/video/judge-demo-script.md scripts/judge-video/manifest.json scripts/judge-video/validate-manifest.mjs scripts/judge-video/validate-manifest.test.mjs
git commit -m "docs: define judge demo video scenes"
```

### Task 2: 生成并渲染无文字解释卡

**Files:**
- Create: `docs/assets/judge-video/backgrounds/01-problem.png` through `05-proof.png` using the built-in image generation tool.
- Create: `scripts/judge-video/render-cards.mjs`
- Create: `tmp/video-work/cards/01-problem.png` through `05-proof.png`

**Interfaces:**
- `renderCards({ manifestPath, backgroundDir, outputDir })` returns an array of rendered PNG paths.
- The renderer reads `scene.asset`, applies the same font stack (`Microsoft YaHei`, `Noto Sans CJK SC`, `Arial`), and renders `scene.subtitle` plus a short section label; it never trusts text embedded in the bitmap.

- [ ] **Step 1: Generate five backgrounds with `image_gen`**

Use one built-in call per asset. Prompts must request 1920x1080 landscape, no words, no logos, no watermark, clean negative space on the left, and the dark data-cockpit palette. Use these subjects:

```text
01 problem: abstract scenic information fragments converging into one calm illuminated path, dark navy data cockpit, restrained cyan grid, amber signal, clean left negative space, no text.
02 visitor: elegant topographic route lines over a misty mountain silhouette, dark navy and teal, one highlighted route, clean left negative space, no text.
03 ai: abstract retrieval pipeline with three connected nodes and a bright answer beam, dark navy, cyan and amber accents, clean left negative space, no text.
04 operations: calm operations dashboard atmosphere with linked data points and a scenic contour line, dark navy with mint and amber evidence markers, clean left negative space, no text.
05 proof: minimal convergence of visitor map, knowledge nodes and operations signal into one bright center, dark navy, mint and blue evidence accents, clean left negative space, no text.
```

Inspect every generated image with `view_image`; reject any image containing accidental letters, watermarks, clutter behind the title area, or a dominant decorative orb.

- [ ] **Step 2: Write the card renderer**

Create a Playwright-based renderer that loads a local HTML template at 1920x1080, sets the background image, overlays the scene number/title/subtitle from the manifest, and saves a PNG screenshot. The title area must reserve 42% width and use fixed line-height so text cannot overlap the background or viewport edges.

- [ ] **Step 3: Render and inspect cards**

Run: `node scripts/judge-video/render-cards.mjs scripts/judge-video/manifest.json`

Expected: five PNGs in `tmp/video-work/cards/`; inspect a contact sheet and confirm consistent font, margins, contrast and no clipping.

- [ ] **Step 4: Commit reusable backgrounds and renderer**

```powershell
git add docs/assets/judge-video/backgrounds scripts/judge-video/render-cards.mjs
git commit -m "feat: add judge demo transition cards"
```

### Task 3: Record independent real-page scenes

**Files:**
- Create: `scripts/judge-video/record-scenes.mjs`
- Create: `tmp/video-work/scenes/*.webm`

**Interfaces:**
- `recordScenes({ baseUrl, outputDir, manifestPath })` checks `/health`, logs in with local demo credentials from environment, and writes one WebM per page scene.
- The recorder never prints the password or cookie and fails if `/health` or login is unsuccessful.

- [ ] **Step 1: Add a recorder smoke test**

Use a local HTTP fixture in `scripts/judge-video/record-scenes.test.mjs` to assert that a non-200 health response exits with a non-zero status and that scene filenames are derived only from manifest ids. Do not launch a real browser in this unit test.

- [ ] **Step 2: Implement the recorder**

Reuse the existing `scripts/record_lingshan_demo.js` patterns: Playwright Chromium, 1280x720 viewport, `zh-CN` locale, cookie-based login, `page.video().path()`, page-error logging without sensitive values, and 1-2 second pauses after actions. Record these page scenes:

```text
visitor-map: /app#/map; map, spot list/detail, route selection, QR entry.
ai-rag: /digital-human#/digital-human; text question, streaming answer, follow-up, feedback.
dashboard: /dashboard#/dashboard; KPI cards, trends, hot questions, satisfaction.
admin-knowledge: /admin#/admin/knowledge; knowledge list, search/filter, item detail.
admin-queries: /admin#/admin/queries; unanswered question, reply/edit state, processed state.
digital-human: /digital-human#/digital-human; Live2D/connection state and tested fallback only.
```

The recorder must keep page scenes as independent WebM files and write a JSON result containing duration and file size only.

- [ ] **Step 3: Run against the local demo environment**

Run: `powershell -NoProfile -ExecutionPolicy Bypass -File scripts/start-local.ps1 -Restart -NoBrowser` and then `node scripts/judge-video/record-scenes.mjs scripts/judge-video/manifest.json`.

Expected: service health passes, six WebM scenes are created, no scene is smaller than 1 MB, and the result contains no cookie/password fields.

- [ ] **Step 4: Inspect scene frames**

Run FFmpeg thumbnail extraction for each WebM and inspect the first, middle and last frame. Re-record any scene with black frames, layout overflow, console error visible in the UI, or claims not supported by the actual state.

- [ ] **Step 5: Commit recorder source only**

```powershell
git add scripts/judge-video/record-scenes.mjs scripts/judge-video/record-scenes.test.mjs
git commit -m "feat: record judge demo page scenes"
```

### Task 4: Generate and normalize the沉稳男声 narration

**Files:**
- Create: `scripts/judge-video/generate-voice.ps1`
- Create: `tmp/video-work/voice/01-intro.mp3` through `06-close.mp3`

**Interfaces:**
- PowerShell parameters: `-ScriptPath`, `-OutputDir`, `-Voice` (default `zh-CN-YunjianNeural`), `-Rate` (default `-8%`).
- The script uses `Open-LLM-VTuber/.venv/Scripts/edge-tts.exe` and writes a manifest containing voice name, rate, duration and file size, never source secrets.

- [ ] **Step 1: Validate the local Edge TTS executable**

Run: `& ..\Open-LLM-VTuber\.venv\Scripts\edge-tts.exe --list-voices | Select-String 'zh-CN-YunjianNeural|zh-CN-YunyangNeural'`

Expected: at least one whitelisted male voice is listed; otherwise stop and report the missing dependency.

- [ ] **Step 2: Generate six narration files**

Read the exact text from `docs/video/judge-demo-script.md`, call Edge TTS once per segment, then run FFmpeg `loudnorm=I=-16:TP=-1.5:LRA=11` and trim only leading/trailing silence. Use `zh-CN-YunjianNeural` first because it is the explicit male narrator mapping in the project; use `zh-CN-YunyangNeural` only if Yunjian fails.

- [ ] **Step 3: Verify audio output**

Run: `ffprobe -v error -show_entries stream=codec_name,sample_rate,channels:format=duration -of json tmp/video-work/voice/*.mp3`

Expected: every file has an audio stream, mono or stereo channels, a positive duration, and no zero-byte output. Listen to the first 15 seconds of the intro and one transition segment; reject obvious robotic artifacts, clipped syllables or excessive speed before composing.

- [ ] **Step 4: Commit voice generation script**

```powershell
git add scripts/judge-video/generate-voice.ps1
git commit -m "feat: add judge demo narration generation"
```

### Task 5: Compose two final MP4s

**Files:**
- Create: `scripts/judge-video/compose.ps1`
- Create: `tmp/video-work/timeline.txt`
- Create: `output/video/judge-demo-with-bgm.mp4`
- Create: `output/video/judge-demo-no-bgm.mp4`

**Interfaces:**
- PowerShell parameters: `-ManifestPath`, `-WorkDir`, `-OutputDir`, `-WithBgm`.
- The composer validates the manifest before running FFmpeg, concatenates scene video and card PNG segments at 30 fps, muxes the matching narration, applies `loudnorm`, and writes `judge-demo-with-bgm.mp4` plus `judge-demo-no-bgm.mp4`.

- [ ] **Step 1: Build a deterministic timeline**

For each manifest scene, write a concat entry pointing to either the rendered card PNG or the recorded WebM. Use the scene `durationSec` as the card loop duration; use the actual page clip duration and update only the manifest duration after inspection. Fail if the measured total is outside 285-315 seconds.

- [ ] **Step 2: Compose the no-BGM version first**

Run: `powershell -NoProfile -ExecutionPolicy Bypass -File scripts/judge-video/compose.ps1 -ManifestPath scripts/judge-video/manifest.json -WorkDir tmp/video-work -OutputDir output/video`

Expected: `output/video/judge-demo-no-bgm.mp4` exists with H.264 video, AAC audio, 1280x720 dimensions and 285-315 seconds duration.

- [ ] **Step 3: Add an original low-level ambient bed for the BGM version**

Create a short procedural ambient bed with FFmpeg sine sources and a slow volume envelope; keep it below `-30 dB` and duck it under narration. Do not download an unlicensed track or embed third-party music without a license file.

- [ ] **Step 4: Compose the BGM version**

Run the same composer with `-WithBgm`.

Expected: `output/video/judge-demo-with-bgm.mp4` exists, voice remains intelligible, and BGM is not audible over page feedback or digital-human audio.

- [ ] **Step 5: Commit composer source**

```powershell
git add scripts/judge-video/compose.ps1
git commit -m "feat: compose judge demo video outputs"
```

### Task 6: Verify, document and update the knowledge graph

**Files:**
- Modify: `D:\go web 01\CHANGELOG.md`
- Create: `scenic-guide/output/video/judge-demo-contact-sheet.png` (ignored output)

- [ ] **Step 1: Run media verification**

Run:

```powershell
ffprobe -v error -show_entries format=duration:stream=codec_name,width,height -of json output/video/judge-demo-no-bgm.mp4
ffprobe -v error -show_entries format=duration:stream=codec_name,width,height -of json output/video/judge-demo-with-bgm.mp4
ffmpeg -y -i output/video/judge-demo-no-bgm.mp4 -vf "fps=1/30,scale=320:-1,tile=4x3" -frames:v 1 output/video/judge-demo-contact-sheet.png
```

Expected: both files are 1280x720 H.264/AAC videos in the duration window; contact sheet shows all major page and transition states without black frames.

- [ ] **Step 2: Run source checks**

Run: `node --test scripts/judge-video/*.test.mjs`, `npm.cmd --prefix web-vue run check`, and `git diff --check`.

Expected: media scripts pass their tests, existing Vue type checks remain green, and no whitespace errors are introduced.

- [ ] **Step 3: Append CHANGELOG entry**

Record exact generated outputs, scripts and verification commands in `D:\go web 01\CHANGELOG.md`; do not include credentials, cookies, or logs.

- [ ] **Step 4: Update codebase graph**

The codebase-memory graph tools were unavailable in this session. Before claiming completion, retry discovery for `index_repository`; if still unavailable, report the failure explicitly instead of claiming synchronization.

- [ ] **Step 5: Final review**

Check that only the intended source files are staged, user changes remain untouched, both MP4s open, and the final response names actual files and commands run.

## Execution Order

Complete Tasks 1-6 in order. Tasks 2-4 may use separate working directories under `tmp/video-work`, but composition (Task 5) must wait until all scene clips, card PNGs and voice files have been inspected. Do not modify `scenic-guide/static/vue-app/` or any generated frontend bundle.
