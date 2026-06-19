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
  /\|\|\s*'请填写用户名和密码'/,
  /\|\|\s*'密码至少6位'/,
  /\|\|\s*'升级失败，用户名可能已被占用'/,
  /\|\|\s*'升级失败'/,
  /\|\|\s*'注册正式账号'/,
  /\|\|\s*'升级后可保存所有对话记录，跨设备同步。'/,
  /\|\|\s*'用户名'/,
  /\|\|\s*'密码（至少6位）'/,
  /\|\|\s*'邮箱（可选）'/,
  /\|\|\s*'确认注册'/,
  /upgradeLoading\s*\?\s*'\.\.\.'/,
];

const requiredKeys = [
  'auth.upgradeTitle',
  'auth.upgradeHint',
  'auth.upgradeButton',
  'auth.upgradeLoading',
  'auth.usernamePlaceholder',
  'auth.passwordPlaceholder',
  'auth.emailPlaceholder',
  'auth.usernamePasswordRequired',
  'auth.passwordTooShort',
  'auth.upgradeFailed',
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
    failures.push(`DigitalHumanView.vue still has hardcoded upgrade text matching ${pattern}`);
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
  console.error('Digital human upgrade i18n check failed:');
  for (const failure of failures) {
    console.error(`- ${failure}`);
  }
  process.exit(1);
}

console.log('Digital human upgrade i18n check passed.');
