import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const ignoredSegments = new Set(['.git', 'node_modules', 'data']);
const ignoredFiles = new Set(['configs/config.yaml', 'configs/digital_human.yaml']);
const textExtensions = new Set([
  '.go',
  '.js',
  '.ts',
  '.vue',
  '.md',
  '.yaml',
  '.yml',
  '.json',
  '.html',
  '.css',
  '.txt',
  '.toml',
  '.mod',
  '.sum',
]);

const patterns = [
  ['openai_or_deepseek_style_key', /sk-[A-Za-z0-9]{12,}/g],
  ['github_token', /gh[pousr]_[0-9A-Za-z_]{20,}/g],
  ['google_api_key', /AIza[0-9A-Za-z_-]{20,}/g],
  ['aws_access_key', /AKIA[0-9A-Z]{16}/g],
  ['private_key', /-----BEGIN (?:RSA |EC |OPENSSH |DSA )?PRIVATE KEY-----/g],
  ['bearer_token_literal', /Bearer\s+[A-Za-z0-9._-]{24,}/g],
];

function shouldIgnore(filePath) {
  const rel = path.relative(root, filePath).replaceAll(path.sep, '/');
  if (ignoredFiles.has(rel)) return true;
  return rel.split('/').some(segment => ignoredSegments.has(segment));
}

function collectFiles(dir, files = []) {
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const filePath = path.join(dir, entry.name);
    if (shouldIgnore(filePath)) continue;
    if (entry.isDirectory()) {
      collectFiles(filePath, files);
    } else if (textExtensions.has(path.extname(entry.name).toLowerCase())) {
      files.push(filePath);
    }
  }
  return files;
}

const failures = [];
for (const file of collectFiles(root)) {
  let text;
  try {
    text = fs.readFileSync(file, 'utf8');
  } catch {
    continue;
  }
  for (const [name, pattern] of patterns) {
    for (const match of text.matchAll(pattern)) {
      failures.push({
        file: path.relative(root, file).replaceAll(path.sep, '/'),
        line: text.slice(0, match.index).split(/\r?\n/).length,
        name,
      });
    }
  }
}

if (failures.length > 0) {
  console.error('Secret scan failed. High-confidence secret patterns were found:');
  for (const failure of failures.slice(0, 50)) {
    console.error(`- ${failure.file}:${failure.line} ${failure.name}`);
  }
  if (failures.length > 50) {
    console.error(`...and ${failures.length - 50} more`);
  }
  process.exit(1);
}

console.log(`Secret scan passed for ${collectFiles(root).length} files.`);
