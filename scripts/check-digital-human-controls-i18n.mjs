import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const viewPath = path.join(root, 'web-vue/src/views/DigitalHumanView.vue');
const zhPath = path.join(root, 'web-vue/src/locales/zh-CN.json');
const enPath = path.join(root, 'web-vue/src/locales/en-US.json');

const source = fs.readFileSync(viewPath, 'utf8');
const zhLocale = JSON.parse(fs.readFileSync(zhPath, 'utf8'));
const enLocale = JSON.parse(fs.readFileSync(enPath, 'utf8'));

const hardcodedPatterns = [
  /ref\('点击“启用声音”后，我会朗读回答并驱动口型。'\)/,
  /audioNotice\.value = '正在播放语音，口型会跟随音频变化。'/,
  /audioNotice\.value = '声音已启用。'/,
  /'请先点击“启用声音”，之后我会朗读回答并驱动口型。'/,
  /'语音合成暂时不可用，已切换为浏览器朗读。'/,
  /`语音合成暂时不可用，已切换为浏览器朗读：\$\{err\.message\}`/,
  /audioNotice\.value = '声音已启用。后续回答会自动朗读，口型会跟随音频或文字朗读节奏。'/,
  /aria-label="数字人形象选择"/,
  />\s*景区指定\s*</,
  /\?\s*'播放中'\s*:\s*audioStatus === 'ready'\s*\?\s*'声音已启用'\s*:\s*'启用声音'/,
  /\?\s*'到点讲解已开'\s*:\s*'到点讲解'/,
  /\?\s*'退出老年模式'\s*:\s*'老年模式'/,
  /title="拖动调整聊天框宽度"/,
];

const requiredKeys = [
  'dh.audio.initialNotice',
  'dh.audio.playingNotice',
  'dh.audio.readyNotice',
  'dh.audio.lockedNotice',
  'dh.audio.ttsFallback',
  'dh.audio.ttsFallbackWithMessage',
  'dh.audio.enabledNotice',
  'dh.controls.soundPlaying',
  'dh.controls.soundReady',
  'dh.controls.soundEnable',
  'dh.controls.autoGuideOn',
  'dh.controls.autoGuideOff',
  'dh.controls.seniorMode',
  'dh.controls.exitSeniorMode',
  'dh.controls.chatResizeTitle',
  'dh.avatar.ariaLabel',
  'dh.avatar.lockedLabel',
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
    failures.push(`DigitalHumanView.vue still has hardcoded control text matching ${pattern}`);
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
  console.error('Digital human controls i18n check failed:');
  for (const failure of failures) {
    console.error(`- ${failure}`);
  }
  process.exit(1);
}

console.log('Digital human controls i18n check passed.');
