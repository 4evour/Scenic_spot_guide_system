import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const files = [
  'web-vue/src/services/api.ts',
  'web-vue/src/services/audioPlayback.ts',
  'web-vue/src/services/vtuberSocket.ts',
];
const zhPath = path.join(root, 'web-vue/src/locales/zh-CN.json');
const enPath = path.join(root, 'web-vue/src/locales/en-US.json');

const sources = new Map(
  files.map(file => [file, fs.readFileSync(path.join(root, file), 'utf8')]),
);
const zhLocale = JSON.parse(fs.readFileSync(zhPath, 'utf8'));
const enLocale = JSON.parse(fs.readFileSync(enPath, 'utf8'));

const hardcodedPatterns = [
  ['web-vue/src/services/api.ts', /'未登录或登录已过期'/],
  ['web-vue/src/services/api.ts', /接口返回非 JSON 响应/],
  ['web-vue/src/services/api.ts', /`请求失败 \(\$\{response\.status\}\)`/],
  ['web-vue/src/services/audioPlayback.ts', /'语音合成没有返回音频，已切换为浏览器朗读。'/],
  ['web-vue/src/services/audioPlayback.ts', /utterance\.lang = 'zh-CN'/],
  ['web-vue/src/services/audioPlayback.ts', /'浏览器朗读被阻止，请点击“启用声音”后重试。'/],
  ['web-vue/src/services/audioPlayback.ts', /'浏览器未允许播放声音，请检查站点声音权限。'/],
  ['web-vue/src/services/audioPlayback.ts', /'浏览器阻止了自动播放，请点击“启用声音”。'/],
  ['web-vue/src/services/audioPlayback.ts', /'音频播放失败，已尝试播放下一段。'/],
  ['web-vue/src/services/vtuberSocket.ts', /'数字人语音服务未连接。文字聊天仍可正常使用，如需语音功能请启动 Open-LLM-VTuber 服务。'/],
  ['web-vue/src/services/vtuberSocket.ts', /'收到无法解析的数字人消息。'/],
  ['web-vue/src/services/vtuberSocket.ts', /'数字人语音服务重连失败，请检查服务是否启动。'/],
];

const requiredKeys = [
  'api.unauthorizedExpired',
  'api.nonJsonResponse',
  'api.requestFailed',
  'dh.audio.emptyStreamFallback',
  'dh.audio.browserSpeechBlocked',
  'dh.audio.soundPermissionDenied',
  'dh.audio.autoplayBlocked',
  'dh.audio.playbackFailedNext',
  'dh.socket.notConnected',
  'dh.socket.invalidMessage',
  'dh.socket.reconnectFailed',
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

for (const [file, pattern] of hardcodedPatterns) {
  if (pattern.test(sources.get(file))) {
    failures.push(`${file} still has hardcoded service text matching ${pattern}`);
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
  console.error('Service i18n check failed:');
  for (const failure of failures) {
    console.error(`- ${failure}`);
  }
  process.exit(1);
}

console.log('Service i18n check passed.');
