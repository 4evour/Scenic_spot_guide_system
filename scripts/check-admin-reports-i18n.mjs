import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const viewPath = path.join(root, 'web-vue/src/views/AdminReports.vue');
const zhPath = path.join(root, 'web-vue/src/locales/zh-CN.json');
const enPath = path.join(root, 'web-vue/src/locales/en-US.json');

const source = fs.readFileSync(viewPath, 'utf8');
const zh = fs.readFileSync(zhPath, 'utf8');
const en = fs.readFileSync(enPath, 'utf8');
const zhLocale = JSON.parse(zh);
const enLocale = JSON.parse(en);

const hardcodedPatterns = [
  /'近 7 天'/,
  /'近 30 天'/,
  /'感受度报告加载失败'/,
  />\s*游客感受度报告\s*</,
  />\s*7天\s*</,
  />\s*30天\s*</,
  />\s*刷新报告\s*</,
  /label="交互记录"/,
  /label="满意倾向"/,
  /label="负面占比"/,
  /label="高峰时段"/,
  />\s*游客关注点分析\s*</,
  />\s*正在生成报告\.\.\.\s*</,
  />\s*暂无真实词云数据\s*</,
  />\s*情绪分布\s*</,
  />\s*情感趋势\s*</,
  />\s*负面反馈原因下钻\s*</,
  />\s*人群画像与路线匹配\s*</,
  />\s*路线点击率\/满意度\s*</,
  />\s*热门时段\s*</,
  />\s*自动化改进建议\s*</,
  />\s*暂无真实交互数据，暂不能生成改进建议。\s*</,
];

const requiredKeys = [
  'adminReports.title',
  'adminReports.subtitle',
  'adminReports.periods.7d',
  'adminReports.periods.30d',
  'adminReports.periodButtons.7d',
  'adminReports.periodButtons.30d',
  'adminReports.actions.refresh',
  'adminReports.kpis.interactions',
  'adminReports.kpis.satisfaction',
  'adminReports.kpis.negative',
  'adminReports.kpis.peakHour',
  'adminReports.kpiNotes.periodTotal',
  'adminReports.kpiNotes.positiveEmotion',
  'adminReports.kpiNotes.needsReview',
  'adminReports.kpiNotes.topConcern',
  'adminReports.sections.attention',
  'adminReports.sections.emotionDistribution',
  'adminReports.sections.emotionTrend',
  'adminReports.sections.negativeReasons',
  'adminReports.sections.audienceProfiles',
  'adminReports.sections.routeSatisfaction',
  'adminReports.sections.peakHours',
  'adminReports.sections.suggestions',
  'adminReports.messages.loadFailed',
  'adminReports.messages.generating',
  'adminReports.empty.wordCloud',
  'adminReports.empty.audienceProfiles',
  'adminReports.empty.routeSatisfaction',
  'adminReports.empty.suggestions',
  'adminReports.labels.satisfactionPercent',
  'adminReports.labels.clickRatePercent',
  'adminReports.emotions.positive',
  'adminReports.emotions.neutral',
  'adminReports.emotions.negative',
];

const failures = [];
for (const pattern of hardcodedPatterns) {
  if (pattern.test(source)) {
    failures.push(`AdminReports.vue still has hardcoded Chinese UI text matching ${pattern}`);
  }
}

function hasKey(locale, key) {
  return key.split('.').reduce((current, part) => {
    if (current && Object.prototype.hasOwnProperty.call(current, part)) {
      return current[part];
    }
    return undefined;
  }, locale) !== undefined;
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
  console.error('Admin reports i18n check failed:');
  for (const failure of failures) {
    console.error(`- ${failure}`);
  }
  process.exit(1);
}

console.log('Admin reports i18n check passed.');
