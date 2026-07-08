import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const viewPath = path.join(root, 'web-vue/src/views/MapView.vue');
const zhPath = path.join(root, 'web-vue/src/locales/zh-CN.json');
const enPath = path.join(root, 'web-vue/src/locales/en-US.json');

const source = fs.readFileSync(viewPath, 'utf8');
const zhLocale = JSON.parse(fs.readFileSync(zhPath, 'utf8'));
const enLocale = JSON.parse(fs.readFileSync(enPath, 'utf8'));

const hardcodedPatterns = [
  /return '地图就绪'/,
  /return '离线示意图'/,
  /'自动导览已开启，等待浏览器定位授权；未授权时仍可点击景点查看离线路线。'/,
  /'GPS 信号弱，已切换离线景点列表；后台可标注梵宫等弱信号区域用于现场疏导。'/,
  /`AR 指引已就绪，当前位置精度约 \$\{Math\.round\(currentPosition\.value\.accuracy\)\}m。`/,
  /'开启到点讲解后，可基于定位显示方向提示；未授权时保留离线景点选择。'/,
  /'已请求定位权限，离线示意图会保留景点和路线。'/,
  /'自动导览已开启，靠近景点后会自动讲解。'/,
  /'自动导览已关闭。'/,
  /'老年模式已开启，文字和按钮会放大。'/,
  /'老年模式已关闭。'/,
  />\s*\{\{\s*seniorModeEnabled\s*\?\s*'退出老年模式'\s*:\s*'老年模式'\s*\}\}\s*</,
  /\{\{\s*geoError\s*\?\s*'离线导览模式'\s*:\s*'AR 导航提示'\s*\}\}/,
];

const requiredKeys = [
  'map.ready',
  'map.offlineMap',
  'map.seniorMode',
  'map.exitSeniorMode',
  'map.ar.autoGuideWaiting',
  'map.ar.gpsWeakFallback',
  'map.ar.ready',
  'map.ar.idle',
  'map.ar.offlineMode',
  'map.ar.navigationHint',
  'map.messages.locateFallback',
  'map.messages.autoGuideStarted',
  'map.messages.autoGuideStopped',
  'map.messages.seniorModeEnabled',
  'map.messages.seniorModeDisabled',
];

function hasKey(locale, key) {
  return key.split('.').reduce((current, part) => {
    if (current && Object.prototype.hasOwnProperty.call(current, part)) {
      return current[part];
    }
    return undefined;
  }, locale) !== undefined;
}

const failures = [];

for (const pattern of hardcodedPatterns) {
  if (pattern.test(source)) {
    failures.push(`MapView.vue still has hardcoded map guide text matching ${pattern}`);
  }
}

for (const key of requiredKeys) {
  if (!hasKey(zhLocale, key)) {
    failures.push(`zh-CN locale is missing ${key}`);
  }
  if (!hasKey(enLocale, key)) {
    failures.push(`en-US locale is missing ${key}`);
  }
}

if (failures.length > 0) {
  console.error('Map guide i18n check failed:');
  for (const failure of failures) {
    console.error(`- ${failure}`);
  }
  process.exit(1);
}

console.log('Map guide i18n check passed.');
