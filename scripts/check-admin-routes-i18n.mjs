import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const viewPath = path.join(root, 'web-vue/src/views/AdminRoutes.vue');
const zhPath = path.join(root, 'web-vue/src/locales/zh-CN.json');
const enPath = path.join(root, 'web-vue/src/locales/en-US.json');

const source = fs.readFileSync(viewPath, 'utf8');
const zhLocale = JSON.parse(fs.readFileSync(zhPath, 'utf8'));
const enLocale = JSON.parse(fs.readFileSync(enPath, 'utf8'));

const hardcodedPatterns = [
  /'轻松'/,
  /'中等'/,
  /'挑战'/,
  /'小时'/,
  /'分钟'/,
  /'请输入路线名称'/,
  /'请输入时长'/,
  /'时长至少为 1 分钟'/,
  /'请选择难度'/,
  /'评分范围 0-5'/,
  /title: '名称'/,
  /title: '描述'/,
  /title: '景点数'/,
  /title: '时长'/,
  /title: '难度'/,
  /title: '评分'/,
  /title: '操作'/,
  /'编辑'/,
  /'删除'/,
  />\s*路线管理\s*</,
  />\s*新增路线\s*</,
  /'编辑路线'/,
  /'新增路线'/,
  /label="名称"/,
  /placeholder="请输入路线名称"/,
  /label="描述"/,
  /placeholder="请输入路线描述"/,
  /label="景点列表"/,
  /placeholder="景点名称用逗号分隔，如：西湖,断桥,雷峰塔"/,
  /label="时长\(分钟\)"/,
  /placeholder="请输入时长"/,
  /label="难度"/,
  /placeholder="请选择难度"/,
  /label="评分"/,
  /placeholder="请输入评分"/,
  />\s*取消\s*</,
  />\s*更新\s*</,
  />\s*创建\s*</,
];

const requiredKeys = [
  'adminRoutes.title',
  'adminRoutes.actions.create',
  'adminRoutes.actions.edit',
  'adminRoutes.actions.delete',
  'adminRoutes.actions.cancel',
  'adminRoutes.actions.update',
  'adminRoutes.drawer.createTitle',
  'adminRoutes.drawer.editTitle',
  'adminRoutes.columns.name',
  'adminRoutes.columns.description',
  'adminRoutes.columns.spotCount',
  'adminRoutes.columns.duration',
  'adminRoutes.columns.difficulty',
  'adminRoutes.columns.rating',
  'adminRoutes.columns.actions',
  'adminRoutes.difficulty.easy',
  'adminRoutes.difficulty.medium',
  'adminRoutes.difficulty.hard',
  'adminRoutes.form.name',
  'adminRoutes.form.description',
  'adminRoutes.form.spots',
  'adminRoutes.form.duration',
  'adminRoutes.form.difficulty',
  'adminRoutes.form.rating',
  'adminRoutes.placeholders.name',
  'adminRoutes.placeholders.description',
  'adminRoutes.placeholders.spots',
  'adminRoutes.placeholders.duration',
  'adminRoutes.placeholders.difficulty',
  'adminRoutes.placeholders.rating',
  'adminRoutes.validation.nameRequired',
  'adminRoutes.validation.durationRequired',
  'adminRoutes.validation.durationMin',
  'adminRoutes.validation.difficultyRequired',
  'adminRoutes.validation.ratingRange',
  'adminRoutes.units.hour',
  'adminRoutes.units.minute',
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
    failures.push(`AdminRoutes.vue still has hardcoded Chinese UI text matching ${pattern}`);
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
  console.error('Admin routes i18n check failed:');
  for (const failure of failures) {
    console.error(`- ${failure}`);
  }
  process.exit(1);
}

console.log('Admin routes i18n check passed.');
