const fs = require('fs');
const path = require('path');
const { pathToFileURL } = require('url');
const { createRequire } = require('module');

const PROJECT_ROOT = path.resolve(__dirname, '..');
const requireFromWebVue = createRequire(path.join(PROJECT_ROOT, 'web-vue', 'package.json'));
const { chromium } = requireFromWebVue('playwright');
const BASE_URL = process.env.SCENIC_DEMO_BASE_URL || 'http://127.0.0.1:8080';
const CHROME_PATH = process.env.CHROME_PATH || '';
const OUTPUT_DIR = process.env.SCENIC_DEMO_OUTPUT_DIR || path.join(PROJECT_ROOT, 'tmp', 'demo-video');
const TEMP_DIR = path.join(OUTPUT_DIR, '.recording-temp');
const OUTPUT_WEBM = path.join(OUTPUT_DIR, 'lingshan-demo-subtitled.webm');

const VIEWPORT = { width: 1280, height: 720 };
const ADMIN_USER = process.env.SCENIC_DEMO_ADMIN_USER || 'admin';
const ADMIN_PASSWORD = process.env.SCENIC_DEMO_ADMIN_PASSWORD || 'DemoAdmin123456';

function sleep(ms) {
  return new Promise(resolve => setTimeout(resolve, ms));
}

async function checkService() {
  const response = await fetch(`${BASE_URL}/health`);
  if (!response.ok) {
    throw new Error(`health check failed: ${response.status}`);
  }
}

async function login() {
  const response = await fetch(`${BASE_URL}/api/v1/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username: ADMIN_USER, password: ADMIN_PASSWORD }),
  });
  if (!response.ok) {
    throw new Error(`login failed: ${response.status}`);
  }
  const payload = await response.json();
  const token = payload && payload.data && payload.data.token;
  if (!token) {
    throw new Error('login response did not include token');
  }
  return token;
}

function cleanupOutput() {
  fs.mkdirSync(OUTPUT_DIR, { recursive: true });
  fs.rmSync(TEMP_DIR, { recursive: true, force: true });
  fs.mkdirSync(TEMP_DIR, { recursive: true });
  fs.rmSync(OUTPUT_WEBM, { force: true });
}

async function ensureSubtitle(page, text, note = '') {
  await page.evaluate(
    ({ text, note }) => {
      const rootId = 'codex-demo-subtitle';
      let root = document.getElementById(rootId);
      if (!root) {
        root = document.createElement('div');
        root.id = rootId;
        root.innerHTML = `
          <div class="codex-demo-kicker">LingShan Scenic Guide Portfolio Demo</div>
          <div class="codex-demo-main"></div>
          <div class="codex-demo-note"></div>
        `;
        document.body.appendChild(root);
      }

      const styleId = 'codex-demo-subtitle-style';
      if (!document.getElementById(styleId)) {
        const style = document.createElement('style');
        style.id = styleId;
        style.textContent = `
          #codex-demo-subtitle {
            position: fixed;
            left: 50%;
            bottom: 26px;
            transform: translateX(-50%);
            z-index: 2147483647;
            width: min(1060px, calc(100vw - 96px));
            padding: 14px 22px 15px;
            color: #fff;
            background: rgba(8, 13, 24, 0.82);
            border: 1px solid rgba(255,255,255,0.16);
            box-shadow: 0 18px 52px rgba(0,0,0,0.35);
            border-radius: 12px;
            font-family: "Microsoft YaHei", "PingFang SC", "Noto Sans CJK SC", Arial, sans-serif;
            text-align: center;
            pointer-events: none;
            backdrop-filter: blur(12px);
          }
          #codex-demo-subtitle .codex-demo-kicker {
            margin-bottom: 4px;
            color: #8be9fd;
            font-size: 13px;
            line-height: 1.2;
            letter-spacing: 0;
          }
          #codex-demo-subtitle .codex-demo-main {
            font-size: 26px;
            line-height: 1.35;
            font-weight: 700;
            letter-spacing: 0;
          }
          #codex-demo-subtitle .codex-demo-note {
            margin-top: 4px;
            color: rgba(255,255,255,0.78);
            font-size: 16px;
            line-height: 1.35;
            letter-spacing: 0;
          }
        `;
        document.head.appendChild(style);
      }

      root.querySelector('.codex-demo-main').textContent = text;
      root.querySelector('.codex-demo-note').textContent = note || '';
    },
    { text, note },
  );
}

async function stabilize(page) {
  await page.waitForLoadState('domcontentloaded').catch(() => {});
  await page.waitForLoadState('networkidle', { timeout: 5000 }).catch(() => {});
  await sleep(900);
}

async function gotoScene(page, target, subtitle, note, holdMs = 9000) {
  if (target.startsWith('data:') || target.startsWith('file:')) {
    await page.goto(target, { waitUntil: 'domcontentloaded' });
  } else {
    await page.goto(`${BASE_URL}${target}`, { waitUntil: 'domcontentloaded' });
  }
  await stabilize(page);
  await ensureSubtitle(page, subtitle, note);
  await sleep(holdMs);
}

async function clickIfVisible(page, selector) {
  const locator = page.locator(selector).first();
  if (await locator.count()) {
    try {
      await locator.click({ timeout: 1200 });
      await sleep(900);
      return true;
    } catch {
      return false;
    }
  }
  return false;
}

async function scrollPage(page, amount = 420, steps = 2) {
  for (let i = 0; i < steps; i += 1) {
    await page.mouse.wheel(0, amount);
    await sleep(900);
  }
}

function ragReportUrl() {
  const html = `
<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>灵山 RAG 评估边界</title>
  <style>
    :root { color-scheme: dark; }
    body {
      margin: 0;
      min-height: 100vh;
      color: #eef6ff;
      background:
        radial-gradient(circle at 18% 20%, rgba(82,240,238,.18), transparent 32%),
        linear-gradient(135deg, #101827, #172335 52%, #101522);
      font-family: "Microsoft YaHei", "PingFang SC", "Noto Sans CJK SC", Arial, sans-serif;
    }
    main {
      max-width: 1100px;
      margin: 0 auto;
      padding: 54px 44px 152px;
    }
    .eyebrow {
      color: #8be9fd;
      font-size: 18px;
      margin: 0 0 14px;
    }
    h1 {
      margin: 0 0 16px;
      font-size: 44px;
      line-height: 1.16;
      letter-spacing: 0;
    }
    .lead {
      max-width: 900px;
      margin: 0 0 30px;
      color: rgba(238,246,255,.82);
      font-size: 21px;
      line-height: 1.75;
    }
    .grid {
      display: grid;
      grid-template-columns: repeat(4, 1fr);
      gap: 14px;
      margin: 30px 0;
    }
    .metric, .panel {
      border: 1px solid rgba(255,255,255,.13);
      background: rgba(255,255,255,.075);
      border-radius: 10px;
      box-shadow: 0 18px 60px rgba(0,0,0,.2);
    }
    .metric { padding: 18px; }
    .metric strong {
      display: block;
      margin-bottom: 8px;
      color: #f4c765;
      font-size: 34px;
      line-height: 1;
    }
    .metric span {
      color: rgba(238,246,255,.76);
      font-size: 16px;
    }
    .panel {
      margin-top: 18px;
      padding: 22px 26px;
    }
    h2 { margin: 0 0 14px; font-size: 24px; }
    ul { margin: 0; padding-left: 22px; }
    li { margin: 10px 0; color: rgba(238,246,255,.84); font-size: 18px; line-height: 1.55; }
    code {
      padding: 2px 7px;
      border-radius: 6px;
      color: #8be9fd;
      background: rgba(0,0,0,.28);
      font-family: Consolas, monospace;
    }
  </style>
</head>
<body>
  <main>
    <p class="eyebrow">RAG Evaluation & Reproducibility</p>
    <h1>把演示项目说清楚：可运行、可复现，也清楚标注数据边界</h1>
    <p class="lead">
      项目保留 32 个演示知识切片和 5 条 smoke test 问答，同时增加 3000/300 合成闭集实验，用来验证检索链路、指标脚本和降级策略。
    </p>
    <section class="grid">
      <article class="metric"><strong>32 / 5</strong><span>基础演示集与 smoke test</span></article>
      <article class="metric"><strong>3000 / 300</strong><span>合成闭集规模实验</span></article>
      <article class="metric"><strong>Recall@8 85.5%</strong><span>122 切片 / 203 问答真实资料评估</span></article>
      <article class="metric"><strong>p95 约 10ms</strong><span>纯检索、本地并发 bench</span></article>
    </section>
    <section class="panel">
      <h2>面试中主动讲清楚的边界</h2>
      <ul>
        <li>3000/300 是合成闭集实验，不代表真实开放域召回率。</li>
        <li>有 Embedding Key 时走向量检索；无 Key 时可用 BM25 / 词面检索复现基础流程。</li>
        <li>PostgreSQL 是主配置口径，SQLite 只用于本地开发和轻量测试。</li>
        <li>数字人部分聚焦 OpenAI 兼容接口、SSE、WebSocket 代理和前端二开，不包装成完整自研数字人框架。</li>
      </ul>
    </section>
  </main>
</body>
</html>`;
  return `data:text/html;charset=utf-8,${encodeURIComponent(html)}`;
}

async function record() {
  cleanupOutput();
  await checkService();
  const token = await login();

  const launchOptions = {
    headless: true,
    args: [
      '--autoplay-policy=no-user-gesture-required',
      '--disable-dev-shm-usage',
      '--disable-gpu',
      '--font-render-hinting=medium',
      '--hide-scrollbars',
    ],
  };
  if (CHROME_PATH) {
    launchOptions.executablePath = CHROME_PATH;
  }
  const browser = await chromium.launch(launchOptions);

  const context = await browser.newContext({
    viewport: VIEWPORT,
    deviceScaleFactor: 1,
    recordVideo: { dir: TEMP_DIR, size: VIEWPORT },
    locale: 'zh-CN',
  });

  await context.addInitScript(authToken => {
    window.localStorage.setItem('authToken', authToken);
  }, token);

  const page = await context.newPage();
  page.setDefaultTimeout(8000);
  page.on('pageerror', error => console.warn('[pageerror]', error.message));
  page.on('console', message => {
    if (message.type() === 'error') console.warn('[console]', message.text());
  });

  await gotoScene(
    page,
    '/',
    '灵山胜境智能导览系统：Go + Vue + RAG 的作品集实验项目',
    '展示游客问答、知识库维护、地图导览、运营看板和数字人联调的核心链路。',
    10500,
  );

  await clickIfVisible(page, '#chatInput, input[placeholder*="问题"], textarea');
  await ensureSubtitle(
    page,
    '游客端重点不是“堆 AI 名词”，而是让问答入口、景点数据和导览流程能真实跑起来',
    '无外部 Key 时仍可用 BM25 / 词面检索复现基础问答流程。',
  );
  await sleep(9000);

  await gotoScene(
    page,
    '/dashboard#/dashboard',
    '数据看板展示运营统计、热门问答、满意度趋势和响应时间分布',
    '脚本会先登录本地演示账号，避免录到未授权页面。',
    11500,
  );
  await scrollPage(page, 360, 2);
  await ensureSubtitle(
    page,
    '这些图表用于说明后台观测能力：能看趋势、看热点、看响应效率',
    '它是作品集演示数据，不包装成真实商业运营数据。',
  );
  await sleep(8000);

  await gotoScene(
    page,
    '/admin#/admin',
    '管理后台覆盖知识库维护、多格式导入、数字人形象和系统设置入口',
    'PostgreSQL 是主数据库配置；SQLite 只作为本地开发和轻量测试方案。',
    11500,
  );
  await clickIfVisible(page, 'text=数字人形象');
  await ensureSubtitle(
    page,
    '后台更适合面试展示工程闭环：接口、权限、数据模型和可维护配置',
    '相比“完整上线系统”，这里更准确的说法是可运行的课程与作品集项目。',
  );
  await sleep(8500);

  await gotoScene(
    page,
    '/app#/map',
    '游客地图从景点接口读取数据，并提供当前位置、下一站和路线导览入口',
    '接口不可用时才回退前端演示点位，便于本地展示不中断。',
    12000,
  );
  await clickIfVisible(page, '.route-option, .next-list button');
  await sleep(2200);
  await ensureSubtitle(
    page,
    '地图页体现的是后端数据到前端导览 UI 的连接能力',
    '这一段适合在面试里展开讲接口设计、字段归一化和降级策略。',
  );
  await sleep(8000);

  await gotoScene(
    page,
    '/digital-human#/digital-human',
    '数字人导览聚焦协议适配：OpenAI 兼容接口、SSE、WebSocket 和前端二开',
    '这里接入 Open-LLM-VTuber，不暗示完整自研数字人框架。',
    12000,
  );
  await clickIfVisible(page, 'text=历史路线');
  await sleep(2600);
  await ensureSubtitle(
    page,
    '数字人页面保留离线演示模式，也支持连接后端服务进行文本、音频和打断控制',
    '面试时建议主动说明哪些是自研链路，哪些是开源项目协议适配。',
  );
  await sleep(9000);

  await gotoScene(
    page,
    ragReportUrl(),
    'RAG 评估强调可复现：32/5 smoke test，加 3000/300 合成闭集实验',
    'Recall@8、MRR、关键词覆盖率和 p95 都必须带上测试环境与数据边界。',
    13500,
  );
  await scrollPage(page, 320, 2);
  await ensureSubtitle(
    page,
    '3000/300 是合成闭集实验，不代表真实开放域召回率',
    '它的价值是证明检索链路、评估脚本和降级策略能够被复现和追问。',
  );
  await sleep(11000);

  await ensureSubtitle(
    page,
    '最终定位：可信、可运行、可解释的后端 + AI 应用作品集项目',
    '投递实习时，重点讲清技术选择、数据边界、复现方式和下一步改进。',
  );
  await sleep(10000);

  const video = page.video();
  await page.close();
  await context.close();
  await browser.close();

  const rawVideo = await video.path();
  fs.copyFileSync(rawVideo, OUTPUT_WEBM);
  fs.rmSync(TEMP_DIR, { recursive: true, force: true });

  const stat = fs.statSync(OUTPUT_WEBM);
  if (stat.size < 1024 * 1024) {
    throw new Error(`recorded video is too small: ${stat.size} bytes`);
  }

  console.log(JSON.stringify({
    webm: OUTPUT_WEBM,
    sizeBytes: stat.size,
    projectRoot: PROJECT_ROOT,
  }, null, 2));
}

record().catch(error => {
  console.error(error);
  process.exit(1);
});
