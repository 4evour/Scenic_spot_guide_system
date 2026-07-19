import test from 'node:test';
import assert from 'node:assert/strict';
import { validateManifest } from './validate-manifest.mjs';

test('rejects a manifest outside the five-minute duration window', () => {
  const result = validateManifest({
    version: 1,
    canvas: { width: 1280, height: 720 },
    scenes: [{ id: 'one', kind: 'page', durationSec: 1, route: '/map', voiceSegment: 'one', narration: 'x', subtitle: 'x', claimBoundary: 'tested' }],
    outputs: { withBgm: 'x.mp4', withoutBgm: 'y.mp4' },
  });
  assert.equal(result.valid, false);
  assert.match(result.errors.join('\n'), /285-315/);
});

test('rejects duplicate scene ids and forbidden secret-like fields', () => {
  const result = validateManifest({
    version: 1,
    canvas: { width: 1280, height: 720 },
    scenes: [
      { id: 'dup', kind: 'page', durationSec: 150, route: '/map', voiceSegment: 'x', narration: 'x', subtitle: 'x', claimBoundary: 'tested' },
      { id: 'dup', kind: 'page', durationSec: 150, route: '/dashboard', voiceSegment: 'x', narration: 'x', subtitle: 'x', claimBoundary: 'tested', api_key: 'secret' },
    ],
    outputs: { withBgm: 'x.mp4', withoutBgm: 'y.mp4' },
  });
  assert.equal(result.valid, false);
  assert.match(result.errors.join('\n'), /duplicate|sensitive/i);
});

test('rejects reused voice segments that leave page scenes under-narrated', () => {
  const result = validateManifest({
    version: 1,
    canvas: { width: 1280, height: 720 },
    scenes: [
      { id: 'card', kind: 'card', durationSec: 10, asset: 'card.png', voiceSegment: 'shared', narration: '完整的引入旁白文案。', subtitle: 'intro', claimBoundary: 'tested' },
      { id: 'page', kind: 'page', durationSec: 290, route: '/page', voiceSegment: 'shared', narration: '页面旁白。', subtitle: 'page', claimBoundary: 'tested' },
    ],
    outputs: { withBgm: 'x.mp4', withoutBgm: 'y.mp4' },
  });
  assert.equal(result.valid, false);
  assert.match(result.errors.join('\n'), /voice segment|narration coverage/i);
});

test('accepts the approved 300-second production manifest', async () => {
  const fs = await import('node:fs/promises');
  const manifest = JSON.parse(await fs.readFile(new URL('./manifest.json', import.meta.url), 'utf8'));
  const result = validateManifest(manifest);
  assert.equal(result.valid, true, result.errors.join('\n'));
  assert.equal(result.totalSeconds, 300);
});
