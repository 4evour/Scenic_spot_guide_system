import fs from 'node:fs';

const MIN_SECONDS = 285;
const MAX_SECONDS = 315;
const SECRET_KEYS = /(?:api[_-]?key|token|cookie|password|secret)/i;

export function validateManifest(manifest, { rootDir = process.cwd() } = {}) {
  const errors = [];
  if (manifest?.canvas?.width !== 1280 || manifest?.canvas?.height !== 720) errors.push('canvas must be 1280x720');
  if (!Array.isArray(manifest?.scenes) || manifest.scenes.length === 0) errors.push('scenes are required');
  const ids = new Set();
  const voiceSegments = new Set();
  let total = 0;
  for (const scene of manifest.scenes || []) {
    if (!scene.id || ids.has(scene.id)) errors.push(`duplicate or missing scene id: ${scene.id || '<empty>'}`);
    ids.add(scene.id);
    if (!['card', 'page'].includes(scene.kind)) errors.push(`invalid kind for ${scene.id}`);
    if (!Number.isFinite(scene.durationSec) || scene.durationSec <= 0) errors.push(`invalid duration for ${scene.id}`);
    if (!scene.narration || !scene.subtitle || !scene.claimBoundary || !scene.voiceSegment) errors.push(`copy fields missing for ${scene.id}`);
    if (scene.voiceSegment && voiceSegments.has(scene.voiceSegment)) errors.push(`voice segment reused: ${scene.voiceSegment}`);
    voiceSegments.add(scene.voiceSegment);
    const minimumNarrationChars = Math.ceil((Number(scene.durationSec) || 0) * (scene.kind === 'page' ? 2.8 : 2.2));
    if (String(scene.narration || '').replace(/\s/g, '').length < minimumNarrationChars) errors.push(`narration coverage too short for ${scene.id}`);
    if (scene.kind === 'page' && !scene.route) errors.push(`route missing for ${scene.id}`);
    if (scene.kind === 'card' && !scene.asset) errors.push(`asset missing for ${scene.id}`);
    total += Number(scene.durationSec) || 0;
    if (JSON.stringify(scene).match(SECRET_KEYS)) errors.push(`sensitive field detected in ${scene.id}`);
  }
  if (total < MIN_SECONDS || total > MAX_SECONDS) errors.push(`total duration must be 285-315 seconds, got ${total}`);
  if (!manifest?.outputs?.withBgm || !manifest?.outputs?.withoutBgm) errors.push('both output paths are required');
  return { valid: errors.length === 0, errors, totalSeconds: total, rootDir };
}

if (process.argv[1] && import.meta.url === new URL(`file://${process.argv[1].replaceAll('\\', '/')}`).href) {
  const manifestPath = process.argv[2] || 'scripts/judge-video/manifest.json';
  const manifest = JSON.parse(fs.readFileSync(manifestPath, 'utf8'));
  const result = validateManifest(manifest, { rootDir: process.cwd() });
  if (!result.valid) {
    console.error(result.errors.join('\n'));
    process.exit(1);
  }
  console.log(`manifest valid: ${result.totalSeconds}s`);
}
