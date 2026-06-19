import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const viewPath = path.join(root, 'web-vue/src/views/DigitalHumanView.vue');
const zhPath = path.join(root, 'web-vue/src/locales/zh-CN.json');
const enPath = path.join(root, 'web-vue/src/locales/en-US.json');

const source = fs.readFileSync(viewPath, 'utf8');
const zhLocale = JSON.parse(fs.readFileSync(zhPath, 'utf8'));
const enLocale = JSON.parse(fs.readFileSync(enPath, 'utf8'));

const hardcodedPatterns = [
  /toLocaleTimeString\('zh-CN'/,
  /toLocaleString\('zh-CN'/,
  /'数字人偏好保存失败'/,
  /`景点\$\{index \+ 1\}`/,
  /`欢迎来到\$\{spot\.name\}/,
  /`已到达\$\{spot\.name\}/,
  /query: '这条路线有哪些主要景点？'/,
  /query: '需要多长时间走完？'/,
  /query: '能讲讲这里的历史故事吗？'/,
  /\$\{entity\}的详细介绍/,
  /能详细介绍一下\$\{entity\}/,
  /sessionTitle: '当前会话'/,
  /你戳了戳/,
  /t\('dh\.fallbackGeneric'\)\s*\|\|/,
];

const requiredKeys = [
  'dh.spotFallbackName',
  'dh.autoGuideIntro',
  'dh.autoGuideTriggered',
  'dh.currentSession',
  'dh.quickAsk.routeDetailQuery',
  'dh.quickAsk.routeTimeQuery',
  'dh.quickAsk.historyDetailQuery',
  'dh.quickAsk.entityDetailLabel',
  'dh.quickAsk.entityDetailQuery',
  'dh.avatar.fallbackName',
  'dh.avatar.saveFailed',
  'dh.avatar.poked',
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
    failures.push(`DigitalHumanView.vue still has runtime hardcoded text matching ${pattern}`);
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
  console.error('Digital human runtime i18n check failed:');
  for (const failure of failures) {
    console.error(`- ${failure}`);
  }
  process.exit(1);
}

console.log('Digital human runtime i18n check passed.');
