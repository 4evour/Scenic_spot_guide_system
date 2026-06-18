import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const extraRoots = [root, path.resolve(root, '..', 'PROJECT_OVERVIEW.md')];
const extensions = new Set([
  '.go',
  '.js',
  '.ts',
  '.vue',
  '.md',
  '.yaml',
  '.yml',
  '.html',
  '.css',
  '.json',
]);
const ignoredSegments = new Set([
  '.git',
  'node_modules',
  'data',
  'static',
]);
const ignoredRelativePrefixes = [
  'static/vue-app/assets/',
];
const mojibakePattern = /(?:[ÃÂ][\u0080-\uFFFF]|â[€€™€œ€“]|\b(?:鍙|鏃|娆|鎶|瀵|闂|鎺|寤|鏈|娉|鎻|璀|绯)[\u0080-\uFFFF])/;

function shouldIgnore(filePath) {
  const rel = path.relative(root, filePath).replaceAll(path.sep, '/');
  if (rel.startsWith('..')) {
    return false;
  }
  if (ignoredRelativePrefixes.some(prefix => rel.startsWith(prefix))) {
    return true;
  }
  return rel.split('/').some(segment => ignoredSegments.has(segment));
}

function collectFiles(target, files = []) {
  const stat = fs.statSync(target);
  if (stat.isFile()) {
    if (extensions.has(path.extname(target).toLowerCase()) && !shouldIgnore(target)) {
      files.push(target);
    }
    return files;
  }

  for (const entry of fs.readdirSync(target, { withFileTypes: true })) {
    const next = path.join(target, entry.name);
    if (shouldIgnore(next)) {
      continue;
    }
    if (entry.isDirectory()) {
      collectFiles(next, files);
    } else if (extensions.has(path.extname(entry.name).toLowerCase())) {
      files.push(next);
    }
  }
  return files;
}

const files = [];
for (const target of extraRoots) {
  if (fs.existsSync(target)) {
    collectFiles(target, files);
  }
}

const failures = [];
const controlCharacterFailures = [];
for (const file of files) {
  const text = fs.readFileSync(file, 'utf8');
  const lines = text.split(/\r?\n/);
  for (let index = 0; index < lines.length; index += 1) {
    const line = lines[index];
    if (line.includes('\uFFFD') || mojibakePattern.test(line)) {
      failures.push({
        file: path.relative(root, file).replaceAll(path.sep, '/'),
        line: index + 1,
        preview: line.trim().slice(0, 140),
      });
    }
    for (const char of line) {
      const code = char.charCodeAt(0);
      if ((code >= 0x00 && code <= 0x08) || code === 0x0b || code === 0x0c || (code >= 0x0e && code <= 0x1f)) {
        controlCharacterFailures.push({
          file: path.relative(root, file).replaceAll(path.sep, '/'),
          line: index + 1,
          code: `U+${code.toString(16).toUpperCase().padStart(4, '0')}`,
          preview: line.trim().slice(0, 140),
        });
        break;
      }
    }
  }
}

if (failures.length > 0 || controlCharacterFailures.length > 0) {
  console.error('Encoding check failed. Suspected replacement characters or mojibake were found:');
  for (const failure of failures.slice(0, 50)) {
    console.error(`- ${failure.file}:${failure.line} ${failure.preview}`);
  }
  for (const failure of controlCharacterFailures.slice(0, 50)) {
    console.error(`- ${failure.file}:${failure.line} ${failure.code} ${failure.preview}`);
  }
  if (failures.length > 50) {
    console.error(`...and ${failures.length - 50} more`);
  }
  if (controlCharacterFailures.length > 50) {
    console.error(`...and ${controlCharacterFailures.length - 50} more control character failures`);
  }
  process.exit(1);
}

console.log(`Encoding check passed for ${files.length} files.`);
