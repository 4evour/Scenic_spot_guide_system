import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const componentPath = path.join(root, 'web-vue/src/components/Live2DStage.vue');
const zhPath = path.join(root, 'web-vue/src/locales/zh-CN.json');
const enPath = path.join(root, 'web-vue/src/locales/en-US.json');

const source = fs.readFileSync(componentPath, 'utf8');
const zhLocale = JSON.parse(fs.readFileSync(zhPath, 'utf8'));
const enLocale = JSON.parse(fs.readFileSync(enPath, 'utf8'));

const hardcodedPatterns = [
  /return '讲解中'/,
  /return '思考中'/,
  /return '聆听中'/,
  /return '已打断'/,
  /return '连接中'/,
  /return '异常'/,
  /return '待命'/,
  /'Live2D SDK 未就绪，已启用备用动效预览'/,
  /'Live2D 模型已接入，表情与口型由前端状态驱动。'/,
  /'正在加载 Live2D 模型\.\.\.'/,
];

const requiredKeys = [
  'live2dStage.status.speaking',
  'live2dStage.status.thinking',
  'live2dStage.status.listening',
  'live2dStage.status.interrupted',
  'live2dStage.status.connecting',
  'live2dStage.status.error',
  'live2dStage.status.idle',
  'live2dStage.sdkFallback',
  'live2dStage.readyNote',
  'live2dStage.loadingNote',
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
    failures.push(`Live2DStage.vue still has hardcoded Live2D stage text matching ${pattern}`);
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
  console.error('Live2D stage i18n check failed:');
  for (const failure of failures) {
    console.error(`- ${failure}`);
  }
  process.exit(1);
}

console.log('Live2D stage i18n check passed.');
