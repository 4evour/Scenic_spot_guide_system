import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const viewPath = path.join(root, 'web-vue/src/views/AdminKnowledge.vue');
const typePath = path.join(root, 'web-vue/src/types/admin.ts');
const packagePath = path.join(root, 'web-vue/package.json');
const makefilePath = path.join(root, 'Makefile');

const view = fs.readFileSync(viewPath, 'utf8');
const types = fs.readFileSync(typePath, 'utf8');
const pkg = fs.readFileSync(packagePath, 'utf8');
const makefile = fs.readFileSync(makefilePath, 'utf8');

const failures = [];

const requiredViewSnippets = [
  '/admin/insights/analyses?page=1&page_size=5',
  'insightAnalyses',
  'loadInsightAnalyses',
  'adminKnowledge.analysis.title',
  'adminKnowledge.analysis.satisfaction',
  'adminKnowledge.analysis.attentionPoint',
  'adminKnowledge.analysis.negativeReason',
];

for (const snippet of requiredViewSnippets) {
  if (!view.includes(snippet)) {
    failures.push(`AdminKnowledge.vue is missing ${snippet}`);
  }
}

if (!types.includes('export type VisitorInsightAnalysis')) {
  failures.push('admin types are missing VisitorInsightAnalysis');
}

if (!pkg.includes('"check:admin-knowledge-insights"')) {
  failures.push('web-vue/package.json is missing check:admin-knowledge-insights');
}

if (!makefile.includes('npm run check:admin-knowledge-insights')) {
  failures.push('Makefile frontend-contracts is missing check:admin-knowledge-insights');
}

if (failures.length > 0) {
  console.error('Admin knowledge insights check failed:');
  for (const failure of failures) {
    console.error(`- ${failure}`);
  }
  process.exit(1);
}

console.log('Admin knowledge insights check passed.');
