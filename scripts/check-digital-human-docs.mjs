import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');

const files = [
  'docs/digital-human-integration.md',
  'docs/digital-human-runbook.md',
  'docs/digital-human-production-check.md',
];

const contents = new Map(
  files.map(file => [file, fs.readFileSync(path.join(root, file), 'utf8')]),
);

const failures = [];

function assertAbsent(file, pattern, reason) {
  const content = contents.get(file);
  if (pattern.test(content)) {
    failures.push(`${file}: ${reason}`);
  }
}

function assertPresent(file, pattern, reason) {
  const content = contents.get(file);
  if (!pattern.test(content)) {
    failures.push(`${file}: ${reason}`);
  }
}

const runbook = contents.get('docs/digital-human-runbook.md');
const fencedBlockPattern = /```(?:bash|sh|powershell)?\r?\n([\s\S]*?)```/g;
for (const match of runbook.matchAll(fencedBlockPattern)) {
  const block = match[1];
  const protectedDhPost =
    /curl\s+-X\s+POST/.test(block) &&
    /http:\/\/localhost:8080\/api\/v1\/dh\/(?:session\/create|chat\/text|chat\/voice-transcript|feedback)/.test(block);
  if (protectedDhPost && (!/-b\s+cookies\.txt/.test(block) || !/X-CSRF-Token/.test(block))) {
    failures.push(
      'docs/digital-human-runbook.md: protected digital-human POST examples must include the login Cookie and CSRF header.',
    );
  }
}

assertAbsent(
  'docs/digital-human-runbook.md',
  /GO_DH_(?:SESSION|TEXT|VOICE|FEEDBACK)_API|配置文件路径：`configs\/digital_human\.yaml`/,
  'the runbook must not reference a non-existent configs/digital_human.yaml file.',
);

assertAbsent(
  'docs/digital-human-integration.md',
  /配置文件路径：`configs\/digital_human\.yaml`/,
  'the integration doc must not reference a non-existent configs/digital_human.yaml file.',
);

assertAbsent(
  'docs/digital-human-production-check.md',
  /打开数字人页面：\s*```text\s*http:\/\/127\.0\.0\.1:12393\//m,
  'production check must use the Go-hosted Vue digital-human page as the main entry.',
);

assertPresent(
  'docs/digital-human-runbook.md',
  /auth_token.*csrf_token/s,
  'the runbook must document that protected digital-human POST APIs require auth_token and csrf_token cookies.',
);

assertPresent(
  'docs/digital-human-integration.md',
  /Vue.*\/digital-human/s,
  'the integration doc must describe the Go-hosted Vue digital-human page as the main frontend path.',
);

assertPresent(
  'docs/digital-human-production-check.md',
  /http:\/\/127\.0\.0\.1:8080\/digital-human#\/digital-human/,
  'production check must include the current Go-hosted Vue digital-human URL.',
);

if (failures.length > 0) {
  console.error('Digital human documentation check failed:');
  for (const failure of failures) {
    console.error(`- ${failure}`);
  }
  process.exit(1);
}

console.log('Digital human documentation check passed.');
