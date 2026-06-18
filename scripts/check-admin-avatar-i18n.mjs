import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const viewPath = path.join(root, 'web-vue/src/views/AdminAvatar.vue');
const zhPath = path.join(root, 'web-vue/src/locales/zh-CN.json');
const enPath = path.join(root, 'web-vue/src/locales/en-US.json');

const source = fs.readFileSync(viewPath, 'utf8');
const zh = fs.readFileSync(zhPath, 'utf8');
const en = fs.readFileSync(enPath, 'utf8');

const hardcodedPatterns = [
  /'数字人配置加载失败'/,
  /'数字人形象配置已保存。'/,
  /'保存失败'/,
  /'请输入数字人名称'/,
  /title="数字人预览"/,
  /title="形象与声音设定"/,
  />\s*默认形象\s*</,
  />\s*服装\s*</,
  />\s*声音\s*</,
  />\s*语气\s*</,
  /label="语速"/,
  /label="音量"/,
  /label="表情强度"/,
  />\s*当前方案：/,
  />\s*形象设定\s*</,
  /label="默认数字人"/,
  /label="游客切换"/,
  />\s*允许\s*</,
  />\s*限制为默认\s*</,
  /label="数字人名称"/,
  /placeholder="请输入数字人名称"/,
  /label="外观定位"/,
  /label="服装风格"/,
  /label="主视觉颜色"/,
  /label="景区文化主题"/,
  /label="欢迎语"/,
  />\s*声音与表达\s*</,
  /label="讲解声音"/,
  /label="讲解语气"/,
  /label="默认表情"/,
  />\s*保存配置\s*</,
  />\s*重新加载\s*</,
];

const requiredKeys = [
  'adminAvatar.previewTitle',
  'adminAvatar.formTitle',
  'adminAvatar.summary.defaultAvatar',
  'adminAvatar.form.name',
  'adminAvatar.actions.save',
  'adminAvatar.messages.loadFailed',
  'adminAvatar.messages.saveSuccess',
];

const failures = [];
for (const pattern of hardcodedPatterns) {
  if (pattern.test(source)) {
    failures.push(`AdminAvatar.vue still has hardcoded Chinese UI text matching ${pattern}`);
  }
}

for (const key of requiredKeys) {
  const leafKey = key.split('.').at(-1);
  const keyPattern = new RegExp(`"${leafKey}"\\s*:`);
  if (!zh.includes('"adminAvatar"') || !keyPattern.test(zh)) {
    failures.push(`zh-CN locale is missing ${key}`);
  }
  if (!en.includes('"adminAvatar"') || !keyPattern.test(en)) {
    failures.push(`en-US locale is missing ${key}`);
  }
}

if (failures.length > 0) {
  console.error('Admin avatar i18n check failed:');
  for (const failure of failures) {
    console.error(`- ${failure}`);
  }
  process.exit(1);
}

console.log('Admin avatar i18n check passed.');
