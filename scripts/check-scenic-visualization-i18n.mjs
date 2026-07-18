import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const scenicPath = path.join(root, 'web-vue/src/constants/scenicVisualization.ts');
const mapPath = path.join(root, 'web-vue/src/views/MapView.vue');
const dhPath = path.join(root, 'web-vue/src/views/DigitalHumanView.vue');
const zhPath = path.join(root, 'web-vue/src/locales/zh-CN.json');
const enPath = path.join(root, 'web-vue/src/locales/en-US.json');

const scenicSource = fs.readFileSync(scenicPath, 'utf8');
const mapSource = fs.readFileSync(mapPath, 'utf8');
const dhSource = fs.readFileSync(dhPath, 'utf8');
const zhLocale = JSON.parse(fs.readFileSync(zhPath, 'utf8'));
const enLocale = JSON.parse(fs.readFileSync(enPath, 'utf8'));

const requiredSourcePatterns = [
  ['scenicVisualization.ts', /export function localizeScenicSpots/],
  ['scenicVisualization.ts', /export function localizeScenicRoutes/],
  ['scenicVisualization.ts', /export function localizeServiceReminders/],
  ['MapView.vue', /localizeScenicSpots\(locale\.value\)/],
  ['MapView.vue', /localizeScenicRoutes\(locale\.value\)/],
  ['MapView.vue', /localizeServiceReminders\(locale\.value\)/],
  ['DigitalHumanView.vue', /buildGuideInsight\(displayText, locale\.value\)/],
  ['DigitalHumanView.vue', /buildGuideInsight\(activeAssistantText, locale\.value\)/],
];

const forbiddenMapPatterns = [
  /SCENIC_SPOTS/,
  /SCENIC_ROUTES/,
  /SERVICE_REMINDERS/,
  /`景点\$\{i \+ 1\}`/,
  /'加载失败'/,
  /\|\|\s*'灵山胜境'/,
  /\|\|\s*'文化休憩'/,
];

const requiredKeys = [
  'map.spotFallbackName',
  'map.defaultArea',
  'map.defaultCategory',
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

const sources = {
  'scenicVisualization.ts': scenicSource,
  'MapView.vue': mapSource,
  'DigitalHumanView.vue': dhSource,
};

for (const [file, pattern] of requiredSourcePatterns) {
  if (!pattern.test(sources[file])) {
    failures.push(`${file} is missing expected scenic visualization i18n pattern ${pattern}`);
  }
}

for (const pattern of forbiddenMapPatterns) {
  if (pattern.test(mapSource)) {
    failures.push(`MapView.vue still directly uses non-localized scenic data matching ${pattern}`);
  }
}

const literalTranslationCount = (scenicSource.match(/translations:\s*\{\s*'en-US':/g) || []).length;
const generatedRouteSpotCount = (scenicSource.match(/^\s+routeOnlySpot\(/gm) || []).length;
const spotTranslationCount = literalTranslationCount + generatedRouteSpotCount;
if (spotTranslationCount < 14) {
  failures.push(`scenicVisualization.ts expected English translations for spots, routes, and reminders; found ${spotTranslationCount}`);
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
  console.error('Scenic visualization i18n check failed:');
  for (const failure of failures) {
    console.error(`- ${failure}`);
  }
  process.exit(1);
}

console.log('Scenic visualization i18n check passed.');
