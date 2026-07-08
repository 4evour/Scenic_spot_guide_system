import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const dashboardPath = path.join(root, 'web-vue', 'src', 'views', 'DashboardView.vue');
const visualizationPath = path.join(root, 'web-vue', 'src', 'constants', 'scenicVisualization.ts');
const source = fs.readFileSync(dashboardPath, 'utf8');
const visualizationSource = fs.readFileSync(visualizationPath, 'utf8');

const disallowedPatterns = [
  {
    name: 'dashboard_fallback_import',
    pattern: /DASHBOARD_FALLBACK/,
    source,
    reason: '运营大屏不能把演示兜底数据当作真实运营数据展示。',
  },
  {
    name: 'fake_knowledge_total',
    pattern: /knowledgeStats\.total_count\s*\|\|\s*\d+/,
    source,
    reason: '知识库条目为 0 时必须显示真实 0，不能回退到硬编码数量。',
  },
  {
    name: 'fake_accuracy',
    pattern: /accuracy\s*[:?]\s*(matched\?\.)?accuracy\s*\|\|\s*\d+/,
    source,
    reason: '热门问题准确率没有真实后端来源时不能硬编码展示。',
  },
  {
    name: 'dashboard_or_report_fallback_constant',
    pattern: /export const (DASHBOARD|REPORT)_FALLBACK/,
    source: visualizationSource,
    reason: '产品源码中不保留未使用的运营/报告演示兜底常量。',
  },
];

const failures = disallowedPatterns.filter(item => item.pattern.test(item.source));

if (failures.length > 0) {
  console.error('Dashboard data boundary check failed:');
  for (const failure of failures) {
    console.error(`- ${failure.name}: ${failure.reason}`);
  }
  process.exit(1);
}

console.log('Dashboard data boundary check passed.');
