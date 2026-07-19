import fs from 'node:fs/promises';
import path from 'node:path';
import { createRequire } from 'node:module';

const projectRoot = path.resolve(process.cwd());
const requireFromWebVue = createRequire(path.join(projectRoot, 'web-vue', 'package.json'));
const { chromium } = requireFromWebVue('playwright');

const accentByScene = {
  '01-problem': '#f0bd61',
  '02-visitor-card': '#73d9ff',
  '04-ai-card': '#8ee6c0',
  '07-operations-card': '#8eaaff',
  '11-proof-close': '#f0bd61',
};

const labelByScene = {
  '01-problem': 'SCENIC INTELLIGENCE / 01',
  '02-visitor-card': 'VISITOR EXPERIENCE / 02',
  '04-ai-card': 'RETRIEVAL AUGMENTED ANSWER / 03',
  '07-operations-card': 'OPERATIONS LOOP / 04',
  '11-proof-close': 'PROOF OF WORK / 05',
};

function escapeHtml(value) {
  return String(value)
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#39;');
}

function cardHtml(scene) {
  const accent = accentByScene[scene.id] || '#8ee6c0';
  const label = labelByScene[scene.id] || 'SCENIC GUIDE / DEMO';
  return `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8" />
  <style>
    :root { color-scheme: dark; }
    * { box-sizing: border-box; }
    html, body { width: 1920px; height: 1080px; margin: 0; overflow: hidden; }
    body {
      color: #edf7fb;
      background-color: #07131c;
      background-image:
        linear-gradient(rgba(100, 190, 218, 0.065) 1px, transparent 1px),
        linear-gradient(90deg, rgba(100, 190, 218, 0.065) 1px, transparent 1px);
      background-size: 64px 64px;
      font-family: "Microsoft YaHei", "Noto Sans CJK SC", Arial, sans-serif;
    }
    .frame { position: relative; width: 100%; height: 100%; padding: 92px 118px; }
    .frame::before { content: ""; position: absolute; inset: 54px; border: 1px solid rgba(154, 225, 245, .18); pointer-events: none; }
    .frame::after { content: ""; position: absolute; left: 118px; right: 118px; bottom: 70px; height: 1px; background: linear-gradient(90deg, ${accent}, transparent 72%); opacity: .7; }
    .copy { position: relative; z-index: 2; width: 42%; padding-top: 36px; }
    .label { color: ${accent}; font-size: 22px; letter-spacing: 3px; line-height: 1.2; }
    h1 { margin: 42px 0 26px; max-width: 720px; font-size: 72px; line-height: 1.16; letter-spacing: 0; font-weight: 700; }
    p { max-width: 640px; margin: 0; color: rgba(237, 247, 251, .72); font-size: 28px; line-height: 1.65; }
    .index { position: absolute; right: 125px; top: 90px; color: rgba(237, 247, 251, .42); font-size: 18px; letter-spacing: 2px; }
    .visual { position: absolute; z-index: 1; top: 185px; right: 154px; width: 890px; height: 665px; }
    .halo { position: absolute; width: 580px; height: 580px; right: 100px; top: 36px; border: 1px solid color-mix(in srgb, ${accent} 52%, transparent); border-radius: 50%; opacity: .45; }
    .halo::before, .halo::after { content: ""; position: absolute; border: 1px solid rgba(115, 217, 255, .28); border-radius: 50%; }
    .halo::before { inset: 62px; }
    .halo::after { inset: 128px; }
    .trace { position: absolute; height: 2px; transform-origin: left center; background: linear-gradient(90deg, ${accent}, transparent); opacity: .85; }
    .trace.one { left: 58px; top: 420px; width: 560px; transform: rotate(-25deg); }
    .trace.two { left: 164px; top: 532px; width: 560px; transform: rotate(-11deg); }
    .trace.three { left: 320px; top: 122px; width: 460px; transform: rotate(39deg); }
    .node { position: absolute; width: 20px; height: 20px; border: 3px solid ${accent}; background: #07131c; box-shadow: 0 0 0 7px rgba(115, 217, 255, .12); }
    .node.a { left: 48px; top: 411px; }
    .node.b { left: 310px; top: 290px; }
    .node.c { right: 82px; top: 160px; }
    .node.d { right: 195px; bottom: 50px; }
    .metric { position: absolute; right: 26px; bottom: 10px; width: 246px; padding: 16px 18px; border-left: 3px solid ${accent}; background: rgba(11, 29, 40, .78); }
    .metric strong { display: block; color: ${accent}; font-size: 34px; line-height: 1; }
    .metric span { display: block; margin-top: 8px; color: rgba(237,247,251,.62); font-size: 16px; letter-spacing: 1px; }
  </style>
</head>
<body>
  <main class="frame">
    <div class="index">JUDGE DEMO · ${escapeHtml(scene.id)}</div>
    <section class="copy">
      <div class="label">${escapeHtml(label)}</div>
      <h1>${escapeHtml(scene.subtitle)}</h1>
      <p>${escapeHtml(scene.claimBoundary)}</p>
    </section>
    <section class="visual" aria-hidden="true">
      <div class="halo"></div>
      <div class="trace one"></div><div class="trace two"></div><div class="trace three"></div>
      <div class="node a"></div><div class="node b"></div><div class="node c"></div><div class="node d"></div>
      <div class="metric"><strong>${String(scene.durationSec).padStart(2, '0')}s</strong><span>SCENE DURATION / VERIFIED COPY</span></div>
    </section>
  </main>
</body>
</html>`;
}

export async function renderCards({ manifestPath, outputDir }) {
  const manifest = JSON.parse(await fs.readFile(manifestPath, 'utf8'));
  const cardScenes = manifest.scenes.filter(scene => scene.kind === 'card');
  await fs.mkdir(outputDir, { recursive: true });
  const browser = await chromium.launch({ headless: true, args: ['--disable-gpu'] });
  const page = await browser.newPage({ viewport: { width: 1920, height: 1080 }, deviceScaleFactor: 1 });
  const rendered = [];
  try {
    for (const scene of cardScenes) {
      await page.setContent(cardHtml(scene), { waitUntil: 'load' });
      const outputPath = path.join(outputDir, scene.asset);
      await page.screenshot({ path: outputPath, type: 'png' });
      rendered.push(outputPath);
    }
  } finally {
    await page.close();
    await browser.close();
  }
  return rendered;
}

if (process.argv[1] && import.meta.url === new URL(`file://${process.argv[1].replaceAll('\\', '/')}`).href) {
  const manifestPath = process.argv[2] || 'scripts/judge-video/manifest.json';
  const outputDir = process.argv[3] || 'tmp/video-work/cards';
  const rendered = await renderCards({ manifestPath, outputDir });
  console.log(JSON.stringify({ rendered }, null, 2));
}
