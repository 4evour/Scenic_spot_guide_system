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
  /label:\s*'地标建筑'/,
  /label:\s*'体验景点'/,
  /label:\s*'休憩文化'/,
  /结构化导览/,
  /离线示意图/,
  /`评分 \$\{spot\.rating\}`/,
  /aria-label="景区路线图"/,
  /content:\s*"灵山胜境 \/ 拈花湾离线导览图"/,
  /visualTypeMeta\[spot\.visualType\]\.label/,
];

const requiredKeys = [
  'map.structuredGuide',
  'map.offlineDiagram',
  'map.rating',
  'map.offlineMapLabel',
  'map.offlineMapWatermark',
  'map.visualTypes.landmark',
  'map.visualTypes.experience',
  'map.visualTypes.culture',
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
    failures.push(`MapView.vue still has hardcoded spot list text matching ${pattern}`);
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
  console.error('Map spot list i18n check failed:');
  for (const failure of failures) {
    console.error(`- ${failure}`);
  }
  process.exit(1);
}

console.log('Map spot list i18n check passed.');
