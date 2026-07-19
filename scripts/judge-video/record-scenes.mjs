import fs from 'node:fs/promises';
import path from 'node:path';
import { createRequire } from 'node:module';

const projectRoot = path.resolve(process.cwd());
const requireFromWebVue = createRequire(path.join(projectRoot, 'web-vue', 'package.json'));
const { chromium } = requireFromWebVue('playwright');

const BASE_URL = process.env.SCENIC_DEMO_BASE_URL || 'http://127.0.0.1:8080';
const ADMIN_USER = process.env.SCENIC_DEMO_ADMIN_USER || 'admin';
const ADMIN_PASSWORD = process.env.SCENIC_DEMO_ADMIN_PASSWORD || '';
const VIEWPORT = { width: 1280, height: 720 };

function sleep(ms) {
  return new Promise(resolve => setTimeout(resolve, ms));
}

async function checkService(baseUrl) {
  const response = await fetch(`${baseUrl}/health`);
  if (!response.ok) throw new Error(`health check failed: ${response.status}`);
}

async function login(baseUrl) {
  if (!ADMIN_PASSWORD) throw new Error('SCENIC_DEMO_ADMIN_PASSWORD is required');
  const response = await fetch(`${baseUrl}/api/v1/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username: ADMIN_USER, password: ADMIN_PASSWORD }),
  });
  if (!response.ok) throw new Error(`login failed: ${response.status}`);
  const payload = await response.json();
  if (!payload || payload.code !== 0) throw new Error('login returned a non-zero code');
  const setCookie = response.headers.get('set-cookie') || '';
  const cookiePair = setCookie.split(';')[0];
  const [name, value] = cookiePair.split('=');
  if (name !== 'auth_token' || !value) throw new Error('login response did not include auth_token');
  return value;
}

async function stabilize(page) {
  await page.waitForLoadState('domcontentloaded').catch(() => {});
  await page.waitForLoadState('networkidle', { timeout: 5000 }).catch(() => {});
  await sleep(1200);
}

async function overlay(page, scene) {
  await page.evaluate(({ sceneId, subtitle, boundary }) => {
    const id = 'judge-demo-overlay';
    let root = document.getElementById(id);
    if (!root) {
      root = document.createElement('div');
      root.id = id;
      root.innerHTML = '<div class="kicker"></div><div class="main"></div><div class="note"></div>';
      document.body.appendChild(root);
      const style = document.createElement('style');
      style.textContent = `
        #judge-demo-overlay { position: fixed; z-index: 2147483647; left: 50%; bottom: 24px; transform: translateX(-50%); width: min(1060px, calc(100vw - 80px)); padding: 12px 22px 14px; border: 1px solid rgba(142, 230, 192, .3); border-radius: 10px; background: rgba(4, 15, 23, .88); color: #eef7fb; font-family: "Microsoft YaHei", "Noto Sans CJK SC", Arial, sans-serif; text-align: center; pointer-events: none; box-shadow: 0 18px 52px rgba(0,0,0,.32); }
        #judge-demo-overlay .kicker { color: #8ee6c0; font-size: 12px; letter-spacing: 2px; }
        #judge-demo-overlay .main { margin-top: 3px; font-size: 24px; line-height: 1.35; font-weight: 700; }
        #judge-demo-overlay .note { margin-top: 3px; color: rgba(238,247,251,.72); font-size: 14px; line-height: 1.35; }
      `;
      document.head.appendChild(style);
    }
    root.querySelector('.kicker').textContent = `JUDGE DEMO / ${sceneId}`;
    root.querySelector('.main').textContent = subtitle;
    root.querySelector('.note').textContent = boundary;
  }, { sceneId: scene.id, subtitle: scene.subtitle, boundary: scene.claimBoundary });
}

async function fillFirstVisibleInput(page, value) {
  const inputs = page.locator('input');
  const count = await inputs.count();
  for (let index = 0; index < count; index += 1) {
    const input = inputs.nth(index);
    if (await input.isVisible().catch(() => false)) {
      await input.fill(value).catch(() => {});
      await input.press('Enter').catch(() => {});
      return true;
    }
  }
  return false;
}

async function performSceneAction(page, sceneId) {
  if (sceneId === '03-visitor-map') {
    await page.mouse.move(180, 170);
    await page.mouse.move(510, 315, { steps: 8 });
    await fillFirstVisibleInput(page, '灵山大佛有多高');
  } else if (sceneId === '05-ai-rag') {
    await fillFirstVisibleInput(page, '灵山大佛有多高');
    await sleep(4500);
    await fillFirstVisibleInput(page, '它什么时候开放');
  } else if (sceneId === '06-digital-human') {
    await page.mouse.move(260, 220);
    await page.mouse.move(820, 380, { steps: 10 });
    await sleep(1200);
  } else if (sceneId === '08-dashboard') {
    await page.mouse.move(280, 170);
    await page.mouse.wheel(0, 360);
    await sleep(1400);
    await page.mouse.wheel(0, -240);
  } else if (sceneId === '09-admin-knowledge') {
    await fillFirstVisibleInput(page, '灵山');
    await sleep(1400);
  } else if (sceneId === '10-admin-queries') {
    await page.mouse.move(300, 190);
    await page.mouse.wheel(0, 280);
    await sleep(1400);
  }
}

export async function recordScenes({ manifestPath, outputDir, baseUrl = BASE_URL }) {
  const manifest = JSON.parse(await fs.readFile(manifestPath, 'utf8'));
  const pageScenes = manifest.scenes.filter(scene => scene.kind === 'page');
  await fs.mkdir(outputDir, { recursive: true });
  await checkService(baseUrl);
  const cookieValue = await login(baseUrl);
  const browser = await chromium.launch({ headless: true, args: ['--autoplay-policy=no-user-gesture-required', '--disable-gpu'] });
  const results = [];
  try {
    for (const scene of pageScenes) {
      if (!/^[a-z0-9-]+$/.test(scene.id)) throw new Error(`invalid scene id: ${scene.id}`);
      const tempDir = path.join(outputDir, `.recording-${scene.id}`);
      await fs.rm(tempDir, { recursive: true, force: true });
      await fs.mkdir(tempDir, { recursive: true });
      const context = await browser.newContext({
        viewport: VIEWPORT,
        deviceScaleFactor: 1,
        locale: 'zh-CN',
        recordVideo: { dir: tempDir, size: VIEWPORT },
      });
      await context.addCookies([{ name: 'auth_token', value: cookieValue, url: new URL(baseUrl).origin, httpOnly: true, sameSite: 'Strict' }]);
      const page = await context.newPage();
      page.setDefaultTimeout(8000);
      const errors = [];
      page.on('pageerror', error => errors.push(error.message.slice(0, 160)));
      page.on('console', message => { if (message.type() === 'error') errors.push(message.text().slice(0, 160)); });
      const started = Date.now();
      await page.goto(`${baseUrl}${scene.route}`, { waitUntil: 'domcontentloaded' });
      await stabilize(page);
      await overlay(page, scene);
      await performSceneAction(page, scene.id);
      const remaining = Math.max(2500, scene.durationSec * 1000 - (Date.now() - started));
      await sleep(remaining);
      const videoPath = await page.video().path();
      await page.close();
      await context.close();
      const outputPath = path.join(outputDir, `${scene.id}.webm`);
      await fs.copyFile(videoPath, outputPath);
      await fs.rm(tempDir, { recursive: true, force: true });
      const stat = await fs.stat(outputPath);
      const durationSec = Number(((Date.now() - started) / 1000).toFixed(3));
      results.push({ id: scene.id, durationSec, sizeBytes: stat.size, pageErrors: errors.slice(0, 3) });
    }
  } finally {
    await browser.close();
  }
  await fs.writeFile(path.join(outputDir, 'recording-result.json'), JSON.stringify({ baseUrl, scenes: results }, null, 2));
  return results;
}

if (process.argv[1] && import.meta.url === new URL(`file://${process.argv[1].replaceAll('\\', '/')}`).href) {
  const manifestPath = process.argv[2] || 'scripts/judge-video/manifest.json';
  const outputDir = process.argv[3] || 'tmp/video-work/scenes';
  const results = await recordScenes({ manifestPath, outputDir });
  console.log(JSON.stringify({ scenes: results }, null, 2));
}
