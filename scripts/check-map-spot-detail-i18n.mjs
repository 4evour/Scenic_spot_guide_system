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
  />\s*建筑参数\s*</,
  />\s*文化内涵\s*</,
  />\s*游玩亮点\s*</,
  />\s*开放\/演出\s*</,
  /'\s*暂无参数\s*'/,
  /'\s*暂无说明\s*'/,
  /'\s*暂无亮点\s*'/,
  /'\s*随景区开放\s*'/,
  />\s*弱信号\s*</,
];

const requiredKeys = [
  'map.spotDetail.parameters',
  'map.spotDetail.culture',
  'map.spotDetail.highlights',
  'map.spotDetail.openInfo',
  'map.spotDetail.noParameters',
  'map.spotDetail.noDescription',
  'map.spotDetail.noHighlights',
  'map.spotDetail.defaultOpenInfo',
  'map.spotDetail.weakSignal',
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
    failures.push(`MapView.vue still has hardcoded spot detail text matching ${pattern}`);
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
  console.error('Map spot detail i18n check failed:');
  for (const failure of failures) {
    console.error(`- ${failure}`);
  }
  process.exit(1);
}

console.log('Map spot detail i18n check passed.');
