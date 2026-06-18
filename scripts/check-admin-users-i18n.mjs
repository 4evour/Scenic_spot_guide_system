import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const viewPath = path.join(root, 'web-vue/src/views/AdminUsers.vue');
const zhPath = path.join(root, 'web-vue/src/locales/zh-CN.json');
const enPath = path.join(root, 'web-vue/src/locales/en-US.json');

const source = fs.readFileSync(viewPath, 'utf8');
const zhLocale = JSON.parse(fs.readFileSync(zhPath, 'utf8'));
const enLocale = JSON.parse(fs.readFileSync(enPath, 'utf8'));

const hardcodedPatterns = [
  /'管理员'/,
  /'游客'/,
  /'请输入密码'/,
  /'请输入用户名'/,
  /'用户名长度为 2-32 个字符'/,
  /'请输入有效的邮箱地址'/,
  /'密码需 8-128 位且包含大小写字母和数字'/,
  /'请选择角色'/,
  /'编辑用户'/,
  /'新增用户'/,
  /title: '用户名'/,
  /title: '邮箱'/,
  /title: '角色'/,
  /title: '创建时间'/,
  /title: '操作'/,
  /'编辑'/,
  /'删除'/,
  /toLocaleString\('zh-CN'/,
  />\s*用户管理\s*</,
  />\s*管理系统的用户账号和权限\s*</,
  />\s*新增用户\s*</,
  /用户管理 API 尚未就绪/,
  /API 未就绪，无法加载用户列表/,
  /label="用户名"/,
  /label="邮箱"/,
  /label="密码"/,
  /label="角色"/,
  /placeholder="请输入用户名"/,
  /placeholder="请输入邮箱地址"/,
  /'留空则不修改密码'/,
  /placeholder="请选择角色"/,
  />\s*取消\s*</,
  />\s*保存\s*</,
  />\s*创建\s*</,
];

const requiredKeys = [
  'adminUsers.title',
  'adminUsers.subtitle',
  'adminUsers.actions.create',
  'adminUsers.actions.edit',
  'adminUsers.actions.delete',
  'adminUsers.actions.cancel',
  'adminUsers.actions.save',
  'adminUsers.columns.username',
  'adminUsers.columns.email',
  'adminUsers.columns.role',
  'adminUsers.columns.createdAt',
  'adminUsers.columns.actions',
  'adminUsers.roles.admin',
  'adminUsers.roles.visitor',
  'adminUsers.drawer.createTitle',
  'adminUsers.drawer.editTitle',
  'adminUsers.form.username',
  'adminUsers.form.email',
  'adminUsers.form.password',
  'adminUsers.form.role',
  'adminUsers.placeholders.username',
  'adminUsers.placeholders.email',
  'adminUsers.placeholders.password',
  'adminUsers.placeholders.passwordEdit',
  'adminUsers.placeholders.role',
  'adminUsers.validation.usernameRequired',
  'adminUsers.validation.usernameLength',
  'adminUsers.validation.emailInvalid',
  'adminUsers.validation.passwordRequired',
  'adminUsers.validation.passwordPolicy',
  'adminUsers.validation.roleRequired',
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
    failures.push(`AdminUsers.vue still has hardcoded or stale UI text matching ${pattern}`);
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
  console.error('Admin users i18n check failed:');
  for (const failure of failures) {
    console.error(`- ${failure}`);
  }
  process.exit(1);
}

console.log('Admin users i18n check passed.');
