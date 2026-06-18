import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const viewPath = path.join(root, 'web-vue/src/views/AdminQRCode.vue');
const zhPath = path.join(root, 'web-vue/src/locales/zh-CN.json');
const enPath = path.join(root, 'web-vue/src/locales/en-US.json');

const source = fs.readFileSync(viewPath, 'utf8');
const zh = fs.readFileSync(zhPath, 'utf8');
const en = fs.readFileSync(enPath, 'utf8');

const hardcodedPatterns = [
  /'二维码数据加载失败'/,
  /'批量生成失败'/,
  /'扫码链接已复制。'/,
  /'复制失败，请手动复制链接。'/,
  /'二维码 ID 最多 100 个字符。'/,
  /'二维码配置已保存。'/,
  /'二维码配置保存失败'/,
  /'景点'/,
  /'分类'/,
  /'状态'/,
  /'已启用'/,
  /'未启用'/,
  /'操作'/,
  /'编辑'/,
  /'复制链接'/,
  />\s*二维码管理\s*</,
  />\s*生成、复制和下载景点扫码导览二维码。\s*</,
  />\s*刷新\s*</,
  />\s*批量生成\s*</,
  /description="暂无景点二维码配置"/,
  /label="启用扫码"/,
  /label="讲解词"/,
  /placeholder="留空时由后端根据景点信息生成讲解"/,
  />\s*取消\s*</,
  />\s*保存\s*</,
];

const requiredKeys = [
  'adminQRCode.title',
  'adminQRCode.subtitle',
  'adminQRCode.columns.spot',
  'adminQRCode.columns.category',
  'adminQRCode.columns.status',
  'adminQRCode.status.enabled',
  'adminQRCode.status.disabled',
  'adminQRCode.actions.copyLink',
  'adminQRCode.messages.loadFailed',
  'adminQRCode.messages.saveSuccess',
];

const failures = [];
for (const pattern of hardcodedPatterns) {
  if (pattern.test(source)) {
    failures.push(`AdminQRCode.vue still has hardcoded Chinese UI text matching ${pattern}`);
  }
}

for (const key of requiredKeys) {
  const leafKey = key.split('.').at(-1);
  const keyPattern = new RegExp(`"${leafKey}"\\s*:`);
  if (!zh.includes('"adminQRCode"') || !keyPattern.test(zh)) {
    failures.push(`zh-CN locale is missing ${key}`);
  }
  if (!en.includes('"adminQRCode"') || !keyPattern.test(en)) {
    failures.push(`en-US locale is missing ${key}`);
  }
}

if (failures.length > 0) {
  console.error('Admin QR code i18n check failed:');
  for (const failure of failures) {
    console.error(`- ${failure}`);
  }
  process.exit(1);
}

console.log('Admin QR code i18n check passed.');
