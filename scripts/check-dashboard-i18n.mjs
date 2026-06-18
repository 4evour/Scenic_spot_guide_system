import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const viewPath = path.join(root, 'web-vue/src/views/DashboardView.vue');
const zhPath = path.join(root, 'web-vue/src/locales/zh-CN.json');
const enPath = path.join(root, 'web-vue/src/locales/en-US.json');

const source = fs.readFileSync(viewPath, 'utf8');
const zh = JSON.parse(fs.readFileSync(zhPath, 'utf8'));
const en = JSON.parse(fs.readFileSync(enPath, 'utf8'));

const hardcodedPatterns = [
  /数据大屏/,
  /实时运营数据概览/,
  /加载失败/,
  /今日服务人次/,
  /本周问答次数/,
  /用户满意度/,
  /平均响应延迟/,
  /知识库条目/,
  /热门景点 TOP5/,
  /热门问答云图/,
  /实时人流热力/,
  /演出\/活动状态/,
  /数字人终端状态/,
  /知识库运营/,
  /24 小时流量趋势/,
  /关注点分布/,
  /7 日满意度趋势/,
  /热门问题 Top 10/,
  /最近对话/,
  /暂无/,
];

const requiredPaths = [
  'dashboard.title',
  'dashboard.subtitle',
  'dashboard.kpis.todayVisitors',
  'dashboard.kpis.weeklyChats',
  'dashboard.kpis.satisfaction',
  'dashboard.kpis.avgResponseTime',
  'dashboard.kpis.knowledgeItems',
  'dashboard.sections.hotSpots',
  'dashboard.sections.questionCloud',
  'dashboard.sections.crowdHeat',
  'dashboard.sections.activityStatus',
  'dashboard.sections.terminalStatus',
  'dashboard.sections.knowledgeOps',
  'dashboard.sections.hourlyTrend',
  'dashboard.sections.categoryDistribution',
  'dashboard.sections.satisfactionTrend',
  'dashboard.sections.topQuestions',
  'dashboard.sections.recentConversations',
  'dashboard.empty.hotSpots',
  'dashboard.empty.questionCloud',
  'dashboard.empty.crowdHeat',
  'dashboard.empty.activityStatus',
  'dashboard.empty.terminalStatus',
  'dashboard.empty.knowledgeGaps',
  'dashboard.empty.satisfactionTrend',
  'dashboard.empty.topQuestions',
  'dashboard.empty.recentConversations',
  'dashboard.columns.time',
  'dashboard.columns.visitorQuery',
  'dashboard.columns.aiResponse',
  'dashboard.columns.emotion',
  'dashboard.columns.hotQuestion',
  'dashboard.columns.count',
  'dashboard.emotions.joy',
  'dashboard.emotions.surprise',
  'dashboard.emotions.neutral',
  'dashboard.emotions.sadness',
  'dashboard.emotions.fear',
];

function getPath(obj, dottedPath) {
  return dottedPath.split('.').reduce((current, key) => {
    if (!current || typeof current !== 'object') return undefined;
    return current[key];
  }, obj);
}

const failures = [];

for (const pattern of hardcodedPatterns) {
  if (pattern.test(source)) {
    failures.push(`DashboardView.vue still has hardcoded Chinese UI text matching ${pattern}`);
  }
}

for (const key of requiredPaths) {
  if (typeof getPath(zh, key) !== 'string') {
    failures.push(`zh-CN locale is missing ${key}`);
  }
  if (typeof getPath(en, key) !== 'string') {
    failures.push(`en-US locale is missing ${key}`);
  }
}

if (failures.length > 0) {
  console.error('Dashboard i18n check failed:');
  for (const failure of failures) {
    console.error(`- ${failure}`);
  }
  process.exit(1);
}

console.log('Dashboard i18n check passed.');
