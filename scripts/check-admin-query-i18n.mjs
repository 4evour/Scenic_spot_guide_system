import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const viewPath = path.join(root, 'web-vue/src/views/AdminQueries.vue');
const zhPath = path.join(root, 'web-vue/src/locales/zh-CN.json');
const enPath = path.join(root, 'web-vue/src/locales/en-US.json');

const source = fs.readFileSync(viewPath, 'utf8');
const zh = fs.readFileSync(zhPath, 'utf8');
const en = fs.readFileSync(enPath, 'utf8');

const hardcodedPatterns = [
  /'游客问题[^']*'/,
  /'未回答'/,
  /'已回答'/,
  /'全部'/,
  /'保存'/,
  /'删除'/,
  /'取消'/,
  /'刷新'/,
  /'处理'/,
  />\s*游客问题处理\s*</,
  /description="暂无游客问题"/,
  /title="处理游客问题"/,
  /label="游客问题"/,
  /label="关联景点 ID"/,
  /label="处理回复"/,
  /label="处理状态"/,
];

const requiredKeys = [
  'adminQueries.title',
  'adminQueries.subtitle',
  'adminQueries.filters.unanswered',
  'adminQueries.filters.all',
  'adminQueries.status.answered',
  'adminQueries.status.unanswered',
  'adminQueries.actions.process',
  'adminQueries.messages.loadFailed',
  'adminQueries.messages.saveSuccess',
  'adminQueries.messages.deleteSuccess',
];

const failures = [];
for (const pattern of hardcodedPatterns) {
  if (pattern.test(source)) {
    failures.push(`AdminQueries.vue still has hardcoded Chinese UI text matching ${pattern}`);
  }
}

for (const key of requiredKeys) {
  const leafKey = key.split('.').at(-1);
  const keyPattern = new RegExp(`"${leafKey}"\\s*:`);
  if (!zh.includes('"adminQueries"') || !keyPattern.test(zh)) {
    failures.push(`zh-CN locale is missing ${key}`);
  }
  if (!en.includes('"adminQueries"') || !keyPattern.test(en)) {
    failures.push(`en-US locale is missing ${key}`);
  }
}

if (failures.length > 0) {
  console.error('Admin query i18n check failed:');
  for (const failure of failures) {
    console.error(`- ${failure}`);
  }
  process.exit(1);
}

console.log('Admin query i18n check passed.');
