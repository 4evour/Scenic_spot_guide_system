import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const viewPath = path.join(root, 'web-vue/src/views/LoginView.vue');
const zhPath = path.join(root, 'web-vue/src/locales/zh-CN.json');
const enPath = path.join(root, 'web-vue/src/locales/en-US.json');

const source = fs.readFileSync(viewPath, 'utf8');
const zhLocale = JSON.parse(fs.readFileSync(zhPath, 'utf8'));
const enLocale = JSON.parse(fs.readFileSync(enPath, 'utf8'));

const hardcodedPatterns = [
  /'已以游客身份登录'/,
  /'游客登录失败，请稍后重试'/,
  /'网络错误'/,
  />\s*🏖️ 以游客身份继续\s*</,
];

const requiredKeys = [
  'login.modeLogin',
  'login.modeRegister',
  'login.email',
  'login.emailPlaceholder',
  'login.registerSubmit',
  'login.registerHint',
  'login.registerSuccess',
  'login.registerFailed',
  'login.passwordPolicy',
  'login.passwordPolicyInvalid',
  'login.guestContinue',
  'login.guestSuccess',
  'login.guestFailed',
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
    failures.push(`LoginView.vue still has hardcoded visitor login text matching ${pattern}`);
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
  console.error('Login i18n check failed:');
  for (const failure of failures) {
    console.error(`- ${failure}`);
  }
  process.exit(1);
}

console.log('Login i18n check passed.');
