import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const composablePath = path.join(root, 'web-vue/src/composables/useGeolocation.ts');
const mapViewPath = path.join(root, 'web-vue/src/views/MapView.vue');
const digitalHumanViewPath = path.join(root, 'web-vue/src/views/DigitalHumanView.vue');
const zhPath = path.join(root, 'web-vue/src/locales/zh-CN.json');
const enPath = path.join(root, 'web-vue/src/locales/en-US.json');

const composable = fs.readFileSync(composablePath, 'utf8');
const mapView = fs.readFileSync(mapViewPath, 'utf8');
const digitalHumanView = fs.readFileSync(digitalHumanViewPath, 'utf8');
const zhLocale = JSON.parse(fs.readFileSync(zhPath, 'utf8'));
const enLocale = JSON.parse(fs.readFileSync(enPath, 'utf8'));

const hardcodedPatterns = [
  /'浏览器不支持地理位置'/,
  /'位置权限被拒绝，请在浏览器设置中允许定位'/,
  /'无法获取位置信息，请检查GPS信号'/,
  /'定位超时'/,
  /`定位失败: \$\{err\.message\}`/,
];

const requiredKeys = [
  'map.gpsDenied',
  'map.gpsUnavailable',
  'map.gpsTimeout',
  'map.gpsNotSupported',
  'map.gpsFailed',
];

const requiredCallPatterns = [
  { file: 'MapView.vue', source: mapView, pattern: /messages:\s*geolocationMessages/ },
  { file: 'DigitalHumanView.vue', source: digitalHumanView, pattern: /messages:\s*geolocationMessages/ },
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
  if (pattern.test(composable)) {
    failures.push(`useGeolocation.ts still has hardcoded geolocation text matching ${pattern}`);
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

for (const item of requiredCallPatterns) {
  if (!item.pattern.test(item.source)) {
    failures.push(`${item.file} does not pass geolocationMessages into useGeolocation`);
  }
}

if (failures.length > 0) {
  console.error('Geolocation i18n check failed:');
  for (const failure of failures) {
    console.error(`- ${failure}`);
  }
  process.exit(1);
}

console.log('Geolocation i18n check passed.');
