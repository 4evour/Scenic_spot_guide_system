import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const viewPath = path.join(root, 'web-vue/src/views/AdminContent.vue');
const zhPath = path.join(root, 'web-vue/src/locales/zh-CN.json');
const enPath = path.join(root, 'web-vue/src/locales/en-US.json');

const source = fs.readFileSync(viewPath, 'utf8');
const zhLocale = JSON.parse(fs.readFileSync(zhPath, 'utf8'));
const enLocale = JSON.parse(fs.readFileSync(enPath, 'utf8'));

const hardcodedPatterns = [
  /label:\s*'讲解词'/,
  /label:\s*'文史资料'/,
  /label:\s*'服务信息'/,
  /message:\s*'请输入标题'/,
  /message:\s*'请选择类型'/,
  /message:\s*'请输入内容'/,
  /title:\s*'标题'/,
  /title:\s*'类型'/,
  /title:\s*'内容预览'/,
  /title:\s*'关联景点ID'/,
  /title:\s*'音频'/,
  /title:\s*'操作'/,
  /row\.audio_url \? '有' : '无'/,
  /default:\s*\(\)\s*=>\s*'编辑'/,
  /default:\s*\(\)\s*=>\s*'删除'/,
  />\s*讲解内容管理\s*</,
  />\s*\+\s*新增内容\s*</,
  /'编辑内容'/,
  /'新增内容'/,
  /label="标题"/,
  /placeholder="请输入标题"/,
  /label="类型"/,
  /placeholder="请选择类型"/,
  /label="关联景点ID"/,
  /placeholder="请输入景点ID"/,
  /label="内容"/,
  /placeholder="请输入讲解内容"/,
  /label="音频URL"/,
  /placeholder="可选，音频文件链接"/,
  />\s*取消\s*</,
  />\s*保存\s*</,
  />\s*创建\s*</,
];

const requiredKeys = [
  'adminContent.title',
  'adminContent.actions.create',
  'adminContent.actions.edit',
  'adminContent.actions.delete',
  'adminContent.actions.cancel',
  'adminContent.actions.save',
  'adminContent.actions.submitCreate',
  'adminContent.drawer.createTitle',
  'adminContent.drawer.editTitle',
  'adminContent.columns.title',
  'adminContent.columns.type',
  'adminContent.columns.preview',
  'adminContent.columns.spotID',
  'adminContent.columns.audio',
  'adminContent.columns.actions',
  'adminContent.contentTypes.guide',
  'adminContent.contentTypes.faq',
  'adminContent.contentTypes.history',
  'adminContent.contentTypes.service',
  'adminContent.audio.available',
  'adminContent.audio.missing',
  'adminContent.form.title',
  'adminContent.form.type',
  'adminContent.form.spotID',
  'adminContent.form.content',
  'adminContent.form.audioURL',
  'adminContent.placeholders.title',
  'adminContent.placeholders.type',
  'adminContent.placeholders.spotID',
  'adminContent.placeholders.content',
  'adminContent.placeholders.audioURL',
  'adminContent.validation.titleRequired',
  'adminContent.validation.typeRequired',
  'adminContent.validation.contentRequired',
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
    failures.push(`AdminContent.vue still has hardcoded Chinese UI text matching ${pattern}`);
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
  console.error('Admin content i18n check failed:');
  for (const failure of failures) {
    console.error(`- ${failure}`);
  }
  process.exit(1);
}

console.log('Admin content i18n check passed.');
