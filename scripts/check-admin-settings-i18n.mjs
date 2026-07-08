import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const viewPath = path.join(root, 'web-vue/src/views/AdminSettings.vue');
const zhPath = path.join(root, 'web-vue/src/locales/zh-CN.json');
const enPath = path.join(root, 'web-vue/src/locales/en-US.json');

const source = fs.readFileSync(viewPath, 'utf8');
const zhLocale = JSON.parse(fs.readFileSync(zhPath, 'utf8'));
const enLocale = JSON.parse(fs.readFileSync(enPath, 'utf8'));

const hardcodedPatterns = [
  /'请输入景区名称'/,
  /'请输入数据保留天数'/,
  /'1-365天'/,
  /'系统设置加载失败'/,
  /'系统设置已保存'/,
  /'保存失败'/,
  /title="系统设置"/,
  />\s*基本信息\s*</,
  /label="景区名称"/,
  /placeholder="请输入景区名称"/,
  /label="景区简介"/,
  /placeholder="请输入景区简介"/,
  /label="服务热线"/,
  /placeholder="请输入服务热线"/,
  />\s*系统功能\s*</,
  /label="启用用户登录"/,
  /label="启用语音服务"/,
  /label="启用游客感受度分析"/,
  />\s*数据管理\s*</,
  /label="数据保留天数"/,
  /label="备份频率"/,
  /placeholder="请选择备份频率"/,
  />\s*保存设置\s*</,
  />\s*重置\s*</,
];

const requiredKeys = [
  'adminSettings.title',
  'adminSettings.sections.basic',
  'adminSettings.sections.features',
  'adminSettings.sections.data',
  'adminSettings.form.scenicName',
  'adminSettings.form.scenicDesc',
  'adminSettings.form.serviceHotline',
  'adminSettings.form.enableLogin',
  'adminSettings.form.enableVoice',
  'adminSettings.form.enableFilter',
  'adminSettings.form.dataRetention',
  'adminSettings.form.backupFrequency',
  'adminSettings.placeholders.scenicName',
  'adminSettings.placeholders.scenicDesc',
  'adminSettings.placeholders.serviceHotline',
  'adminSettings.placeholders.backupFrequency',
  'adminSettings.backupFrequency.daily',
  'adminSettings.backupFrequency.weekly',
  'adminSettings.backupFrequency.monthly',
  'adminSettings.backupFrequency.manual',
  'adminSettings.actions.save',
  'adminSettings.actions.reset',
  'adminSettings.validation.scenicNameRequired',
  'adminSettings.validation.dataRetentionRequired',
  'adminSettings.validation.dataRetentionRange',
  'adminSettings.messages.loadFailed',
  'adminSettings.messages.saveSuccess',
  'adminSettings.messages.saveFailed',
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
    failures.push(`AdminSettings.vue still has hardcoded Chinese UI text matching ${pattern}`);
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
  console.error('Admin settings i18n check failed:');
  for (const failure of failures) {
    console.error(`- ${failure}`);
  }
  process.exit(1);
}

console.log('Admin settings i18n check passed.');
