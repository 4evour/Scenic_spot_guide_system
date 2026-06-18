import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const viewPath = path.join(root, 'web-vue/src/views/AdminKnowledge.vue');
const zhPath = path.join(root, 'web-vue/src/locales/zh-CN.json');
const enPath = path.join(root, 'web-vue/src/locales/en-US.json');

const source = fs.readFileSync(viewPath, 'utf8');
const zhLocale = JSON.parse(fs.readFileSync(zhPath, 'utf8'));
const enLocale = JSON.parse(fs.readFileSync(enPath, 'utf8'));

const hardcodedPatterns = [
  /label:\s*'全部分类'/,
  /label:\s*'官方资料'/,
  /label:\s*'政府资料'/,
  /label:\s*'景区概况'/,
  /label:\s*'景点介绍'/,
  /label:\s*'实时边界'/,
  /label:\s*'全部景点分类'/,
  /message:\s*'请输入知识内容'/,
  /toLocaleDateString\('zh-CN'\)/,
  /toLocaleString\('zh-CN'/,
  /`景点\$\{/,
  /'知识库加载失败'/,
  /'知识条目已更新，数字人检索缓存已刷新。'/,
  /'知识条目已加入数字人知识库。'/,
  /'保存失败'/,
  /title:\s*'确认删除'/,
  /positiveText:\s*'删除'/,
  /negativeText:\s*'取消'/,
  /'知识条目已删除。'/,
  /'删除失败'/,
  /'请先选择知识文档。'/,
  /`上传完成，已导入/,
  /'上传失败'/,
  /'请输入需要分析的会话 ID。'/,
  /'AI 分析完成，已生成知识候选。'/,
  /'AI 分析失败'/,
  /'AI 分析记录加载失败'/,
  /'知识候选加载失败'/,
  /'候选知识已入库。'/,
  /'入库失败'/,
  /'候选知识已拒绝。'/,
  /'拒绝失败'/,
  /label="标题"/,
  /placeholder="例如：九龙灌浴讲解词"/,
  /label="分类"/,
  /label="来源"/,
  /placeholder="admin \/ 文件名 \/ 景点名称"/,
  /label="知识内容"/,
  /placeholder="填写讲解词、历史背景、FAQ 问答或运营说明"/,
  />\s*保存更新\s*</,
  />\s*加入知识库\s*</,
  />\s*清空\s*</,
  /title="上传知识文档"/,
  />\s*上传并导入\s*</,
  /JSONL\/JSON 需包含/,
  /placeholder="搜索景点、讲解词、FAQ、来源\.\.\."/,
  />\s*查询\s*</,
  />\s*刷新\s*</,
  /description="暂无知识条目"/,
  /来源：/,
  /title="AI 分析记录"/,
  /placeholder="输入会话 ID 生成候选"/,
  />\s*AI 分析会话\s*</,
  />\s*刷新记录\s*</,
  /description="暂无 AI 分析记录"/,
  /满意度/,
  /暂无分析摘要/,
  /关注点：/,
  /负面原因：/,
  /title="AI 知识候选"/,
  /最近分析：/,
  />\s*刷新候选\s*</,
  /description="暂无待审核知识候选"/,
  /会话：/,
  />\s*入库\s*</,
  />\s*拒绝\s*</,
];

const requiredKeys = [
  'adminKnowledge.categories.all',
  'adminKnowledge.categories.guide',
  'adminKnowledge.categories.history',
  'adminKnowledge.categories.faq',
  'adminKnowledge.categories.route',
  'adminKnowledge.categories.service',
  'adminKnowledge.categories.ticket',
  'adminKnowledge.categories.official',
  'adminKnowledge.categories.government',
  'adminKnowledge.categories.overview',
  'adminKnowledge.categories.spot',
  'adminKnowledge.categories.boundary',
  'adminKnowledge.categories.uncategorized',
  'adminKnowledge.spotCategories.all',
  'adminKnowledge.spotCategories.core',
  'adminKnowledge.spotCategories.performance',
  'adminKnowledge.spotCategories.culture',
  'adminKnowledge.spotCategories.service',
  'adminKnowledge.spots.all',
  'adminKnowledge.spots.fallback',
  'adminKnowledge.validation.contentRequired',
  'adminKnowledge.kpis.items.label',
  'adminKnowledge.kpis.items.note',
  'adminKnowledge.kpis.current.label',
  'adminKnowledge.kpis.current.note',
  'adminKnowledge.kpis.formats.label',
  'adminKnowledge.kpis.formats.note',
  'adminKnowledge.kpis.cache.label',
  'adminKnowledge.kpis.cache.value',
  'adminKnowledge.kpis.cache.note',
  'adminKnowledge.editor.createTitle',
  'adminKnowledge.editor.editTitle',
  'adminKnowledge.form.title',
  'adminKnowledge.form.category',
  'adminKnowledge.form.source',
  'adminKnowledge.form.content',
  'adminKnowledge.placeholders.title',
  'adminKnowledge.placeholders.source',
  'adminKnowledge.placeholders.content',
  'adminKnowledge.actions.save',
  'adminKnowledge.actions.add',
  'adminKnowledge.actions.clear',
  'adminKnowledge.actions.search',
  'adminKnowledge.actions.refresh',
  'adminKnowledge.actions.edit',
  'adminKnowledge.actions.delete',
  'adminKnowledge.actions.cancel',
  'adminKnowledge.actions.approve',
  'adminKnowledge.actions.reject',
  'adminKnowledge.upload.title',
  'adminKnowledge.upload.submit',
  'adminKnowledge.upload.hint',
  'adminKnowledge.filters.searchPlaceholder',
  'adminKnowledge.empty.knowledge',
  'adminKnowledge.empty.analyses',
  'adminKnowledge.empty.candidates',
  'adminKnowledge.labels.source',
  'adminKnowledge.labels.session',
  'adminKnowledge.analysis.title',
  'adminKnowledge.analysis.sessionPlaceholder',
  'adminKnowledge.analysis.analyze',
  'adminKnowledge.analysis.refresh',
  'adminKnowledge.analysis.satisfaction',
  'adminKnowledge.analysis.emptySummary',
  'adminKnowledge.analysis.attentionPoint',
  'adminKnowledge.analysis.negativeReason',
  'adminKnowledge.candidates.title',
  'adminKnowledge.candidates.latestAnalysis',
  'adminKnowledge.candidates.refresh',
  'adminKnowledge.dialog.deleteTitle',
  'adminKnowledge.dialog.deleteContent',
  'adminKnowledge.messages.loadFailed',
  'adminKnowledge.messages.updateSuccess',
  'adminKnowledge.messages.createSuccess',
  'adminKnowledge.messages.saveFailed',
  'adminKnowledge.messages.deleteSuccess',
  'adminKnowledge.messages.deleteFailed',
  'adminKnowledge.messages.chooseFile',
  'adminKnowledge.messages.uploadSuccess',
  'adminKnowledge.messages.uploadFailed',
  'adminKnowledge.messages.sessionRequired',
  'adminKnowledge.messages.analyzeSuccess',
  'adminKnowledge.messages.analyzeFailed',
  'adminKnowledge.messages.analysesLoadFailed',
  'adminKnowledge.messages.candidatesLoadFailed',
  'adminKnowledge.messages.approveSuccess',
  'adminKnowledge.messages.approveFailed',
  'adminKnowledge.messages.rejectSuccess',
  'adminKnowledge.messages.rejectFailed',
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
    failures.push(`AdminKnowledge.vue still has hardcoded Chinese UI text matching ${pattern}`);
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
  console.error('Admin knowledge i18n check failed:');
  for (const failure of failures) {
    console.error(`- ${failure}`);
  }
  process.exit(1);
}

console.log('Admin knowledge i18n check passed.');
