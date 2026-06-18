import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const compose = fs.readFileSync(path.join(root, 'docker-compose.yml'), 'utf8');
const dockerfile = fs.readFileSync(path.join(root, 'Dockerfile'), 'utf8');

const failures = [];

if (!/scenic-guide:[\s\S]*?healthcheck:/m.test(compose)) {
  failures.push('docker-compose.yml scenic-guide service is missing an application healthcheck.');
}

if (!/test:\s*\[[^\]]*\/health[^\]]*\]/m.test(compose)) {
  failures.push('docker-compose.yml scenic-guide healthcheck must probe the /health endpoint.');
}

if (!/wget|curl/.test(dockerfile)) {
  failures.push('Dockerfile runtime image must include a lightweight HTTP client for healthcheck probes.');
}

if (failures.length > 0) {
  console.error('Compose healthcheck check failed:');
  for (const failure of failures) {
    console.error(`- ${failure}`);
  }
  process.exit(1);
}

console.log('Compose healthcheck check passed.');
