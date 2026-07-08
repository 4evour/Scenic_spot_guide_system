import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const viewPath = path.join(root, 'web-vue/src/views/AdminSpots.vue');
const zhPath = path.join(root, 'web-vue/src/locales/zh-CN.json');
const enPath = path.join(root, 'web-vue/src/locales/en-US.json');

const source = fs.readFileSync(viewPath, 'utf8');
const zhLocale = JSON.parse(fs.readFileSync(zhPath, 'utf8'));
const enLocale = JSON.parse(fs.readFileSync(enPath, 'utf8'));

const hardcodedPatterns = [
  /label:\s*'核心景点'/,
  /label:\s*'演艺体验'/,
  /label:\s*'文化建筑'/,
  /label:\s*'服务设施'/,
  /message:\s*'请输入景点名称'/,
  /message:\s*'请输入位置信息'/,
  /message:\s*'请选择分类'/,
  /message:\s*'评分范围为 0-5'/,
  /title:\s*'名称'/,
  /title:\s*'分类'/,
  /title:\s*'位置'/,
  /title:\s*'评分'/,
  /title:\s*'价格'/,
  /title:\s*'二维码'/,
  /title:\s*'电子围栏'/,
  /title:\s*'操作'/,
  /:\s*'免费'/,
  /'未配置'/,
  /'未启用'/,
  /default:\s*\(\)\s*=>\s*'编辑'/,
  /default:\s*\(\)\s*=>\s*'删除'/,
  />\s*景点管理\s*</,
  />\s*管理景区内的所有景点信息、分类与位置\s*</,
  />\s*新增景点\s*</,
  /'编辑景点'/,
  /'新增景点'/,
  /label="名称"/,
  /placeholder="请输入景点名称"/,
  /label="描述"/,
  /placeholder="请输入景点描述"/,
  /label="位置"/,
  /placeholder="请输入位置信息"/,
  /label="分类"/,
  /placeholder="请选择分类"/,
  /label="评分"/,
  /label="价格"/,
  /placeholder="0 为免费"/,
  /label="图片链接"/,
  /placeholder="可选，图片 URL"/,
  /label="经度"/,
  /placeholder="经度"/,
  /label="纬度"/,
  /placeholder="纬度"/,
  /label="排序"/,
  /placeholder="数值越小越靠前"/,
  /label="二维码 ID"/,
  /placeholder="留空则自动生成，如 SPOT-0001"/,
  /label="开场白"/,
  /placeholder="扫码后数字人自动说的开场白/,
  /label="启用扫码"/,
  />\s*游客可扫码触发讲解\s*</,
  />\s*扫码功能已关闭\s*</,
  /label="到点讲解"/,
  />\s*游客到达附近自动触发\s*</,
  />\s*电子围栏已关闭\s*</,
  /label="半径\(m\)"/,
  /label="冷却\(分\)"/,
  /label="触发文案"/,
  /placeholder="到达该景点时优先播报/,
  />\s*取消\s*</,
  />\s*保存修改\s*</,
  />\s*确认新增\s*</,
];

const requiredKeys = [
  'adminSpots.title',
  'adminSpots.subtitle',
  'adminSpots.actions.create',
  'adminSpots.actions.edit',
  'adminSpots.actions.delete',
  'adminSpots.actions.cancel',
  'adminSpots.actions.save',
  'adminSpots.actions.submitCreate',
  'adminSpots.drawer.createTitle',
  'adminSpots.drawer.editTitle',
  'adminSpots.columns.name',
  'adminSpots.columns.category',
  'adminSpots.columns.location',
  'adminSpots.columns.rating',
  'adminSpots.columns.price',
  'adminSpots.columns.qrCode',
  'adminSpots.columns.geofence',
  'adminSpots.columns.actions',
  'adminSpots.categories.core',
  'adminSpots.categories.performance',
  'adminSpots.categories.culture',
  'adminSpots.categories.service',
  'adminSpots.price.free',
  'adminSpots.status.notConfigured',
  'adminSpots.status.disabled',
  'adminSpots.form.name',
  'adminSpots.form.description',
  'adminSpots.form.location',
  'adminSpots.form.category',
  'adminSpots.form.rating',
  'adminSpots.form.price',
  'adminSpots.form.imageURL',
  'adminSpots.form.longitude',
  'adminSpots.form.latitude',
  'adminSpots.form.sortOrder',
  'adminSpots.form.qrCode',
  'adminSpots.form.qrIntro',
  'adminSpots.form.qrEnabled',
  'adminSpots.form.geofenceEnabled',
  'adminSpots.form.geofenceRadius',
  'adminSpots.form.geofenceCooldown',
  'adminSpots.form.geofenceIntro',
  'adminSpots.placeholders.name',
  'adminSpots.placeholders.description',
  'adminSpots.placeholders.location',
  'adminSpots.placeholders.category',
  'adminSpots.placeholders.price',
  'adminSpots.placeholders.imageURL',
  'adminSpots.placeholders.longitude',
  'adminSpots.placeholders.latitude',
  'adminSpots.placeholders.sortOrder',
  'adminSpots.placeholders.qrCode',
  'adminSpots.placeholders.qrIntro',
  'adminSpots.placeholders.geofenceIntro',
  'adminSpots.validation.nameRequired',
  'adminSpots.validation.locationRequired',
  'adminSpots.validation.categoryRequired',
  'adminSpots.validation.ratingRange',
  'adminSpots.switches.qrEnabled',
  'adminSpots.switches.qrDisabled',
  'adminSpots.switches.geofenceEnabled',
  'adminSpots.switches.geofenceDisabled',
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
    failures.push(`AdminSpots.vue still has hardcoded Chinese UI text matching ${pattern}`);
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
  console.error('Admin spots i18n check failed:');
  for (const failure of failures) {
    console.error(`- ${failure}`);
  }
  process.exit(1);
}

console.log('Admin spots i18n check passed.');
