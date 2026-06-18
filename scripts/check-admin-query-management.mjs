import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');

function read(relativePath) {
  return fs.readFileSync(path.join(root, relativePath), 'utf8');
}

const checks = [
  {
    name: 'admin_query_view',
    ok: fs.existsSync(path.join(root, 'web-vue/src/views/AdminQueries.vue')),
    reason: '游客问题后端管理接口必须有 Vue 管理端页面承接。',
  },
  {
    name: 'admin_query_route',
    ok: /admin\/queries/.test(read('web-vue/src/router/index.ts')),
    reason: '游客问题管理页面必须注册管理员路由。',
  },
  {
    name: 'admin_query_sidebar',
    ok: /admin-queries/.test(read('web-vue/src/layout/GlobalSider.vue')),
    reason: '游客问题管理页面必须有侧边栏入口。',
  },
  {
    name: 'admin_query_i18n_zh',
    ok: /"queries"\s*:\s*"游客问题"/.test(read('web-vue/src/locales/zh-CN.json')),
    reason: '中文导航必须包含游客问题入口文案。',
  },
  {
    name: 'admin_query_i18n_en',
    ok: /"queries"\s*:\s*"Visitor Questions"/.test(read('web-vue/src/locales/en-US.json')),
    reason: '英文导航必须包含游客问题入口文案。',
  },
  {
    name: 'admin_query_docs',
    ok: /游客问题处理/.test(read('README.md')) && /游客问题/.test(read('PROJECT_OVERVIEW.md')),
    reason: '项目说明必须同步记录游客问题管理闭环。',
  },
];

const failures = checks.filter(check => !check.ok);

if (failures.length > 0) {
  console.error('Admin query management check failed:');
  for (const failure of failures) {
    console.error(`- ${failure.name}: ${failure.reason}`);
  }
  process.exit(1);
}

console.log('Admin query management check passed.');
