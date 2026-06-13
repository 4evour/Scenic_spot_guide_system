<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue';
import ThinkingIndicator from './ThinkingIndicator.vue';
import type { ConversationState, Live2DExpression, PixiAppLike, Live2DModelLike, MotionGroup } from '../types/digitalHuman';

const props = defineProps<{
  state: ConversationState;
  mouthOpen: number;
  expression: Live2DExpression;
}>();

const emit = defineEmits<{
  (e: 'head-click'): void;
  (e: 'body-click'): void;
}>();

const canvas = ref<HTMLCanvasElement | null>(null);
const live2dHost = ref<HTMLDivElement | null>(null);
const live2dLoaded = ref(false);
const live2dError = ref('');
const isMobile = ref(false);

let frame = 0;
let raf = 0;
let pixiApp: PixiAppLike | null = null;
let live2dModel: Live2DModelLike | null = null;
let resizeObserver: ResizeObserver | null = null;
let appliedExpression = '';
let expressionSeq = 0;
let idleMotionTimer = 0;
let currentMotionGroup: MotionGroup | null = null;
let smoothedMouthOpen = 0;
let resizeRafId = 0;

// --- lip-sync parameter IDs (read from model3.json Groups) ---
const lipSyncParameterIds = ['ParamA'];
let modelLipSyncIds: string[] = ['ParamA'];

// --- expression class ---
const expressionClass = computed(() => `expr-${props.expression}`);

const statusText = computed(() => {
  if (props.state === 'speaking') return '讲解中';
  if (props.state === 'thinking') return '思考中';
  if (props.state === 'listening') return '聆听中';
  if (props.state === 'interrupted') return '已打断';
  if (props.state === 'connecting') return '连接中';
  if (props.state === 'error') return '异常';
  return '待命';
});

// --- fallback drawing helpers ---
function roundedRect(ctx: CanvasRenderingContext2D, x: number, y: number, w: number, h: number, r: number) {
  const radius = Math.min(r, w / 2, h / 2);
  ctx.beginPath();
  ctx.moveTo(x + radius, y);
  ctx.arcTo(x + w, y, x + w, y + h, radius);
  ctx.arcTo(x + w, y + h, x, y + h, radius);
  ctx.arcTo(x, y + h, x, y, radius);
  ctx.arcTo(x, y, x + w, y, radius);
  ctx.closePath();
}

function fallbackEyeColor(): string {
  switch (props.expression) {
    case 'angry': return '#c0392b';
    case 'sad': return '#5d6d7e';
    case 'blush': return '#e8a0bf';
    default: return '#183339';
  }
}

function fallbackMouthColor(): string {
  switch (props.expression) {
    case 'happy': return '#26746f';
    case 'angry': return '#9b2f3c';
    case 'sad': return '#5b6c7a';
    case 'blush': return '#c97a9a';
    case 'interrupted': return '#9b2f3c';
    default: return '#142c31';
  }
}

function fallbackBodyColor(): string {
  switch (props.expression) {
    case 'interrupted': return 'rgba(255,139,139,.9)';
    case 'angry': return 'rgba(255,139,100,.9)';
    case 'sad': return 'rgba(140,160,180,.9)';
    case 'blush': return 'rgba(240,180,200,.9)';
    default: return 'rgba(82,240,238,.95)';
  }
}

function drawFallback() {
  if (live2dLoaded.value) return;
  const el = canvas.value;
  if (!el) return;
  const ctx = el.getContext('2d');
  if (!ctx) return;

  const dpr = window.devicePixelRatio || 1;
  const rect = el.getBoundingClientRect();
  el.width = Math.max(1, Math.floor(rect.width * dpr));
  el.height = Math.max(1, Math.floor(rect.height * dpr));
  ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
  ctx.clearRect(0, 0, rect.width, rect.height);

  const t = frame / 60;
  const cx = rect.width / 2;
  const cy = rect.height * 0.5 + Math.sin(t * 1.8) * 7;
  const glow = ctx.createRadialGradient(cx, cy, 50, cx, cy, 260);
  glow.addColorStop(0, 'rgba(82,240,238,.24)');
  glow.addColorStop(1, 'rgba(82,240,238,0)');
  ctx.fillStyle = glow;
  ctx.fillRect(0, 0, rect.width, rect.height);

  ctx.save();
  ctx.translate(cx, cy);
  ctx.rotate(Math.sin(t * 0.8) * 0.035);

  ctx.fillStyle = '#f2ffff';
  ctx.beginPath();
  ctx.ellipse(0, -74, 82, 92, 0, 0, Math.PI * 2);
  ctx.fill();

  ctx.fillStyle = fallbackEyeColor();
  const eyeY = -88 + Math.sin(t * 2) * 1.5;
  const blink = Math.sin(t * 0.65) > 0.97 ? 2 : 13;
  // Angry expression: angled eyebrows
  if (props.expression === 'angry') {
    ctx.fillRect(-40, eyeY - 3, 20, blink);
    ctx.fillRect(20, eyeY - 3, 20, blink);
  } else {
    ctx.fillRect(-40, eyeY, 20, blink);
    ctx.fillRect(20, eyeY, 20, blink);
  }

  ctx.strokeStyle = fallbackMouthColor();
  ctx.lineWidth = 5;
  ctx.beginPath();
  const mouth = props.state === 'speaking' ? 4 + props.mouthOpen * 28 : 4;
  if (props.expression === 'sad') {
    // Sad mouth: inverted smile
    ctx.ellipse(0, -40, 15, mouth, 0, Math.PI, 0);
  } else {
    ctx.ellipse(0, -46, 15, mouth, 0, 0, Math.PI * 2);
  }
  ctx.stroke();

  // Blush cheeks
  if (props.expression === 'blush') {
    ctx.fillStyle = 'rgba(255,150,180,.4)';
    ctx.beginPath();
    ctx.ellipse(-30, -60, 12, 8, 0, 0, Math.PI * 2);
    ctx.ellipse(30, -60, 12, 8, 0, 0, Math.PI * 2);
    ctx.fill();
  }

  ctx.fillStyle = fallbackBodyColor();
  roundedRect(ctx, -92, 26, 184, 170, 54);
  ctx.fill();

  ctx.fillStyle = 'rgba(244,199,101,.94)';
  ctx.beginPath();
  ctx.moveTo(-52, 58);
  ctx.lineTo(0, 120);
  ctx.lineTo(52, 58);
  ctx.lineTo(78, 196);
  ctx.lineTo(-78, 196);
  ctx.closePath();
  ctx.fill();
  ctx.restore();
}

// --- Live2D model loading ---
async function loadLive2DModel() {
  if (!live2dHost.value) return;
  try {
    const PIXI = await import('pixi.js');
    const { Live2DModel } = await import('pixi-live2d-display/cubism4');
    window.PIXI = PIXI;

    pixiApp = new PIXI.Application({
      resizeTo: live2dHost.value,
      transparent: true,
      antialias: true,
      autoStart: true,
    }) as unknown as PixiAppLike;

    // Set FPS cap for mobile
    isMobile.value = window.innerWidth < 768;
    if (pixiApp.ticker) {
      pixiApp.ticker.maxFPS = isMobile.value ? 30 : 60;
    }

    live2dHost.value.appendChild(pixiApp.view);

    const modelUrl = '/static/live2d-models/mao_pro/runtime/mao_pro.model3.json';
    live2dModel = await Live2DModel.from(modelUrl) as unknown as Live2DModelLike;
    live2dModel!.autoUpdate = false;
    live2dModel!.anchor.set(0.5, 0.5);
    syncLive2DLayout();

    // Parse LipSync params from model config
    await detectLipSyncParams();

    pixiApp!.stage.addChild(live2dModel!);
    pixiApp!.ticker.add(syncLive2DFrame);
    live2dModel!.internalModel?.on?.('beforeModelUpdate', applyLipSyncParameters);
    live2dLoaded.value = true;
    live2dError.value = '';

    cancelAnimationFrame(raf);
    window.addEventListener('resize', syncLive2DLayout);
    resizeObserver = new ResizeObserver(syncLive2DLayout);
    resizeObserver.observe(live2dHost.value!);

    // Register hit area handlers
    registerHitAreas();
    registerTouchGestures();

    // Start idle motion and initial expression
    playMotion('Idle');
    startIdleMotionCycle();
    applyLive2DState();
  } catch (error) {
    live2dLoaded.value = false;
    live2dError.value = 'Live2D SDK 未就绪，已启用备用动效预览';
    console.warn('Live2D SDK unavailable, fallback avatar is active.', error);
  }
}

// --- Lip-Sync parameter detection ---
async function detectLipSyncParams() {
  try {
    const resp = await fetch('/static/live2d-models/mao_pro/runtime/mao_pro.model3.json');
    const modelJson = await resp.json();
    const lipGroup = (modelJson.Groups || []).find(
      (g: { Name: string }) => g.Name === 'LipSync',
    );
    if (lipGroup && lipGroup.Ids && lipGroup.Ids.length > 0) {
      modelLipSyncIds = lipGroup.Ids;
    }
  } catch {
    // Fallback to default ParamA
    modelLipSyncIds = ['ParamA'];
  }
}

// --- Layout ---
function syncLive2DLayout() {
  if (!live2dHost.value || !live2dModel) return;
  const width = live2dHost.value.clientWidth;
  const height = live2dHost.value.clientHeight;
  const bounds = live2dModel.getLocalBounds?.();
  const naturalWidth = Math.max(bounds?.width || 0, 1);
  const naturalHeight = Math.max(bounds?.height || 0, 1);
  const fitScale = Math.min((width * 0.5) / naturalWidth, (height * 0.82) / naturalHeight);
  const scale = Math.min(Math.max(fitScale, 0.08), 0.145);

  live2dModel.scale.set(scale);
  live2dModel.x = width / 2;
  live2dModel.y = height * 0.54;
  isMobile.value = width < 768;
  if (pixiApp?.ticker) {
    pixiApp.ticker.maxFPS = isMobile.value ? 30 : 60;
  }
}

function onResize() {
  if (resizeRafId) return;
  resizeRafId = requestAnimationFrame(() => {
    resizeRafId = 0;
    syncLive2DLayout();
  });
}

// --- Expression mapping (8 expressions) ---
const expressionMap: Record<Live2DExpression, string> = {
  happy: 'exp_01',
  neutral: 'exp_02',
  angry: 'exp_03',
  thinking: 'exp_04',
  surprised: 'exp_05',
  sad: 'exp_06',
  interrupted: 'exp_07',
  blush: 'exp_08',
  idle: '', // No expression
};

function applyExpression() {
  if (!live2dModel || typeof live2dModel.expression !== 'function') return;
  const target = expressionMap[props.expression] || expressionMap.neutral;
  if (target === appliedExpression) return;
  appliedExpression = target;
  const seq = ++expressionSeq;
  void live2dModel.expression(target).then((applied: boolean) => {
    if (!applied && seq === expressionSeq) {
      appliedExpression = '';
      console.warn(`Live2D expression "${target}" was not applied.`);
    }
  }).catch((error: unknown) => {
    if (seq === expressionSeq) appliedExpression = '';
    console.warn(`Live2D expression "${target}" failed.`, error);
  });
}

// --- Motion system ---
// mao_pro model3.json Motion groups:
//   "Idle" group: [mtn_01] — idle/waiting animation
//   "" group: [mtn_02(0), mtn_03(1), mtn_04(2), special_01(3), special_02(4), special_03(5)]
const STATE_MOTION_INDEX: Record<ConversationState, number | null> = {
  idle: null,         // Uses Idle group
  connecting: null,    // Uses Idle group (mtn_01)
  listening: 3,       // special_01 — attentive listening
  thinking: 4,        // special_02 — thinking pose
  speaking: -1,       // -1 = random from speaking pool
  interrupted: 5,     // special_03 — surprised/interrupted
  error: null,        // Uses Idle group
};

const SPEAKING_MOTION_POOL = [0, 1, 2]; // mtn_02, mtn_03, mtn_04

const MOTION_NAMES: Record<string, string> = {
  Idle: 'Idle',
  Tap: '',
};

function playMotion(group: MotionGroup, index?: number) {
  if (!live2dModel || typeof live2dModel.motion !== 'function') return;
  const groupName = MOTION_NAMES[group] ?? group;
  if (!groupName && groupName !== '') return;
  try {
    if (index !== undefined) {
      live2dModel.motion(groupName, index);
    } else {
      live2dModel.motion(groupName);
    }
    currentMotionGroup = group;
  } catch (e) {
    console.warn('Live2D motion failed:', e);
  }
}

function playStateMotion(state: ConversationState) {
  const motionIndex = STATE_MOTION_INDEX[state];
  if (motionIndex === null) {
    // Use Idle group
    playMotion('Idle');
  } else if (motionIndex === -1) {
    // Random from speaking pool
    const idx = SPEAKING_MOTION_POOL[Math.floor(Math.random() * SPEAKING_MOTION_POOL.length)];
    playMotion('', idx);
  } else {
    playMotion('', motionIndex);
  }
}

function getRandomMotionIndex(): number {
  // Random from speaking pool for idle variations
  return SPEAKING_MOTION_POOL[Math.floor(Math.random() * SPEAKING_MOTION_POOL.length)];
}

function startIdleMotionCycle() {
  window.clearInterval(idleMotionTimer);
  idleMotionTimer = window.setInterval(() => {
    if (props.state === 'idle' && live2dLoaded.value) {
      // Random idle variation: sometimes play a speaking-pool motion
      if (Math.random() < 0.3) {
        playStateMotion('speaking');
      }
    }
  }, 10000 + Math.random() * 5000); // 10-15 seconds
}

function syncLive2DFrame() {
  if (!live2dModel) return;
  live2dModel.update?.(pixiApp?.ticker?.deltaMS ?? 16.7);
  applyExpression();
  applyLipSyncParameters();
}

// --- Lip-sync with lerp smoothing ---
function applyLipSyncParameters() {
  if (!live2dModel) return;
  const targetMouth = props.state === 'speaking'
    ? Math.min(Math.max(props.mouthOpen * 1.2, 0.15), 1)
    : 0;

  // Smooth lerp to avoid jitter
  const lerpFactor = 0.3;
  smoothedMouthOpen = smoothedMouthOpen + (targetMouth - smoothedMouthOpen) * lerpFactor;

  const coreModel = live2dModel.internalModel?.coreModel;
  if (coreModel?.setParameterValueById) {
    modelLipSyncIds.forEach(id => {
      coreModel.setParameterValueById(id, smoothedMouthOpen, 1);
    });
  }
}

// --- Hit area interaction ---
function registerHitAreas() {
  if (!pixiApp || !live2dModel) return;

  const view = pixiApp.view;
  view.style.pointerEvents = 'auto';
  view.style.cursor = 'pointer';

  const clickHandler = (e: MouseEvent) => {
    if (!live2dModel || typeof live2dModel.hitTest !== 'function') return;
    const rect = view.getBoundingClientRect();
    const x = (e.clientX - rect.left) * (view.width / rect.width);
    const y = (e.clientY - rect.top) * (view.height / rect.height);
    const hits = live2dModel.hitTest(x, y);

    if (hits.includes('HitAreaHead')) {
      createClickRipple(e.clientX, e.clientY);
      emit('head-click');
    } else if (hits.includes('HitAreaBody')) {
      createClickRipple(e.clientX, e.clientY);
      emit('body-click');
    }
  };

  view.addEventListener('click', clickHandler);

  // Store for cleanup
  const cleanup = () => view.removeEventListener('click', clickHandler);
  if (live2dModel) (live2dModel as Record<string, unknown>)._cleanupHitArea = cleanup;
}

// --- Touch gestures (pinch-to-zoom + single-finger drag) ---
let touchCleanup: (() => void) | null = null;

function registerTouchGestures() {
  if (!pixiApp || !live2dModel) return;
  const host = live2dHost.value;
  if (!host) return;

  let initialPinchDist = 0;
  let initialScale = 0;
  let dragStartX = 0;
  let dragStartY = 0;
  let modelStartX = 0;
  let modelStartY = 0;
  let isDragging = false;
  let dragMoved = false;
  let bounceTimer = 0;

  const SCALE_MIN = 0.05;
  const SCALE_MAX = 0.25;

  function getTouchDist(touches: TouchList): number {
    if (touches.length < 2) return 0;
    const dx = touches[0].clientX - touches[1].clientX;
    const dy = touches[0].clientY - touches[1].clientY;
    return Math.sqrt(dx * dx + dy * dy);
  }

  function animateBounceBack() {
    if (!live2dModel) return;
    const startX = live2dModel.x;
    const startY = live2dModel.y;
    const targetX = host!.clientWidth / 2;
    const targetY = host!.clientHeight * 0.54;
    const duration = 300;
    const start = performance.now();

    function step(now: number) {
      if (!live2dModel) return;
      const elapsed = now - start;
      const t = Math.min(elapsed / duration, 1);
      // ease-out cubic
      const ease = 1 - Math.pow(1 - t, 3);
      live2dModel.x = startX + (targetX - startX) * ease;
      live2dModel.y = startY + (targetY - startY) * ease;
      if (t < 1) {
        bounceTimer = window.requestAnimationFrame(step);
      }
    }
    window.cancelAnimationFrame(bounceTimer);
    bounceTimer = window.requestAnimationFrame(step);
  }

  const touchStartHandler = (e: TouchEvent) => {
    if (!live2dModel) return;
    if (e.touches.length === 2) {
      // Pinch start
      initialPinchDist = getTouchDist(e.touches);
      initialScale = live2dModel.scale.get();
      isDragging = false;
    } else if (e.touches.length === 1) {
      // Drag start
      window.cancelAnimationFrame(bounceTimer);
      dragStartX = e.touches[0].clientX;
      dragStartY = e.touches[0].clientY;
      modelStartX = live2dModel.x;
      modelStartY = live2dModel.y;
      isDragging = true;
      dragMoved = false;
    }
  };

  const touchMoveHandler = (e: TouchEvent) => {
    if (!live2dModel) return;
    if (e.touches.length === 2 && initialPinchDist > 0) {
      // Pinch zoom
      const dist = getTouchDist(e.touches);
      const ratio = dist / initialPinchDist;
      const newScale = Math.min(SCALE_MAX, Math.max(SCALE_MIN, initialScale * ratio));
      live2dModel.scale.set(newScale);
      isDragging = false;
    } else if (e.touches.length === 1 && isDragging) {
      // Single finger drag
      const dx = e.touches[0].clientX - dragStartX;
      const dy = e.touches[0].clientY - dragStartY;
      if (Math.abs(dx) > 3 || Math.abs(dy) > 3) dragMoved = true;
      live2dModel.x = modelStartX + dx;
      live2dModel.y = modelStartY + dy;
    }
  };

  const touchEndHandler = () => {
    if (isDragging && dragMoved) {
      // Animate bounce back to center
      animateBounceBack();
    }
    isDragging = false;
    initialPinchDist = 0;
  };

  host.addEventListener('touchstart', touchStartHandler, { passive: false });
  host.addEventListener('touchmove', touchMoveHandler, { passive: false });
  host.addEventListener('touchend', touchEndHandler);

  touchCleanup = () => {
    host.removeEventListener('touchstart', touchStartHandler);
    host.removeEventListener('touchmove', touchMoveHandler);
    host.removeEventListener('touchend', touchEndHandler);
    window.cancelAnimationFrame(bounceTimer);
  };
}

function createClickRipple(clientX: number, clientY: number) {
  const ripple = document.createElement('div');
  ripple.className = 'click-ripple';
  ripple.style.cssText = `
    position: fixed;
    left: ${clientX - 12}px;
    top: ${clientY - 12}px;
    width: 24px;
    height: 24px;
    border-radius: 50%;
    background: radial-gradient(circle, rgba(82,240,238,.6), transparent);
    pointer-events: none;
    z-index: 9999;
    animation: ripple-out 0.5s ease-out forwards;
  `;
  document.body.appendChild(ripple);
  ripple.addEventListener('animationend', () => ripple.remove());
}

function applyLive2DState() {
  applyLipSyncParameters();
  applyExpression();
}

// --- Fallback animation loop ---
function loop() {
  if (live2dLoaded.value) return;
  frame += 1;
  drawFallback();
  raf = requestAnimationFrame(loop);
}

// --- Watch state changes for motion ---
watch(() => props.state, (newState, oldState) => {
  if (!live2dLoaded.value) return;
  if (newState === oldState) return;

  // Play state-specific motion for all transitions
  playStateMotion(newState);
});

onMounted(() => {
  loop();
  void loadLive2DModel();
});

onUnmounted(() => {
  window.clearInterval(idleMotionTimer);
  cancelAnimationFrame(raf);
  window.removeEventListener('resize', onResize);
  window.cancelAnimationFrame(resizeRafId);
  resizeObserver?.disconnect();
  // Cleanup touch gestures
  touchCleanup?.();
  // Cleanup hit area handler
  const model = live2dModel as Record<string, unknown> | null;
  if (model?._cleanupHitArea) (model._cleanupHitArea as () => void)();
  pixiApp?.ticker?.remove?.(syncLive2DFrame);
  live2dModel?.internalModel?.off?.('beforeModelUpdate', applyLipSyncParameters);
  pixiApp?.destroy?.(true);
});

watch(() => [props.state, props.mouthOpen, props.expression], () => {
  drawFallback();
  applyLive2DState();
});
</script>

<template>
  <section class="live2d-stage" :class="[expressionClass, `state-${state}`]">
    <div class="stage-grid" />
    <div ref="live2dHost" class="live2d-host" :class="{ loaded: live2dLoaded }" />
    <canvas ref="canvas" class="live2d-canvas" :class="{ hidden: live2dLoaded }" />

    <ThinkingIndicator :visible="state === 'thinking' && live2dLoaded" />

    <div class="model-status">
      <span class="status-pulse" />
      <strong>{{ statusText }}</strong>
    </div>

    <div class="live2d-note">
      {{ live2dLoaded ? 'Live2D 模型已接入，表情与口型由前端状态驱动。' : live2dError || '正在加载 Live2D 模型...' }}
    </div>
  </section>
</template>

<style scoped>
.live2d-stage {
  position: relative;
  width: 100%;
  height: 100%;
  min-height: 300px;
  overflow: hidden;
}

.stage-grid {
  position: absolute;
  inset: 0;
  background-image:
    linear-gradient(rgba(82, 240, 238, 0.03) 1px, transparent 1px),
    linear-gradient(90deg, rgba(82, 240, 238, 0.03) 1px, transparent 1px);
  background-size: 40px 40px;
  pointer-events: none;
}

.live2d-host {
  position: absolute;
  inset: 0;
  z-index: 1;
}
.live2d-host :deep(canvas) {
  display: block;
  width: 100% !important;
  height: 100% !important;
}

.live2d-canvas {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  z-index: 0;
}
.live2d-canvas.hidden {
  display: none;
}


/* --- Click ripple --- */
:global(.click-ripple) {
  position: fixed;
  pointer-events: none;
  z-index: 9999;
}
@keyframes ripple-out {
  0% { transform: scale(0.5); opacity: 1; }
  100% { transform: scale(4); opacity: 0; }
}

.model-status {
  position: absolute;
  bottom: 16px;
  left: 16px;
  display: flex;
  align-items: center;
  gap: 8px;
  z-index: 2;
  font-size: 12px;
  color: rgba(255, 255, 255, 0.5);
}
.status-pulse {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #666;
}
.state-speaking .status-pulse { background: var(--sg-jade-bright, #63e2b7); animation: pulse 1s infinite; }
.state-thinking .status-pulse { background: var(--sg-gold, #f4c765); animation: pulse 1.5s infinite; }
.state-listening .status-pulse { background: var(--sg-cyan, #52f0ee); }
.state-error .status-pulse { background: var(--sg-red-bright, #e88080); }

.live2d-note {
  position: absolute;
  bottom: 16px;
  right: 16px;
  font-size: 11px;
  color: rgba(255, 255, 255, 0.25);
  z-index: 2;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.4; }
}
</style>
