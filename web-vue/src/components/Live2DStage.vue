<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue';
import type { ConversationState, PixiAppLike, Live2DModelLike } from '../types/digitalHuman';

const props = defineProps<{
  state: ConversationState;
  mouthOpen: number;
  expression: 'neutral' | 'happy' | 'thinking' | 'surprised' | 'interrupted';
}>();

const canvas = ref<HTMLCanvasElement | null>(null);
const live2dHost = ref<HTMLDivElement | null>(null);
const live2dLoaded = ref(false);
const live2dError = ref('');

let frame = 0;
let raf = 0;
let pixiApp: PixiAppLike | null = null;
let live2dModel: Live2DModelLike | null = null;
let resizeObserver: ResizeObserver | null = null;
let appliedExpression = '';
let expressionSeq = 0;

const lipSyncParameterIds = ['ParamA'];

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

  ctx.fillStyle = '#183339';
  const eyeY = -88 + Math.sin(t * 2) * 1.5;
  const blink = Math.sin(t * 0.65) > 0.97 ? 2 : 13;
  ctx.fillRect(-40, eyeY, 20, blink);
  ctx.fillRect(20, eyeY, 20, blink);

  ctx.strokeStyle = props.expression === 'happy' ? '#26746f' : props.expression === 'interrupted' ? '#9b2f3c' : '#142c31';
  ctx.lineWidth = 5;
  ctx.beginPath();
  const mouth = props.state === 'speaking' ? 4 + props.mouthOpen * 28 : 4;
  ctx.ellipse(0, -46, 15, mouth, 0, 0, Math.PI * 2);
  ctx.stroke();

  ctx.fillStyle = props.expression === 'interrupted' ? 'rgba(255,139,139,.9)' : 'rgba(82,240,238,.95)';
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
    live2dHost.value.appendChild(pixiApp.view);

    live2dModel = await Live2DModel.from('/static/live2d-models/mao_pro/runtime/mao_pro.model3.json') as unknown as Live2DModelLike;
    live2dModel!.autoUpdate = false;
    live2dModel!.anchor.set(0.5, 0.5);
    syncLive2DLayout();
    pixiApp!.stage.addChild(live2dModel!);
    pixiApp!.ticker.add(syncLive2DFrame);
    live2dModel!.internalModel?.on?.('beforeModelUpdate', applyLipSyncParameters);
    live2dLoaded.value = true;
    live2dError.value = '';
    cancelAnimationFrame(raf);
    window.addEventListener('resize', syncLive2DLayout);
    resizeObserver = new ResizeObserver(syncLive2DLayout);
    resizeObserver.observe(live2dHost.value!);

    live2dModel!.motion?.('Idle');
    applyLive2DState();
  } catch (error) {
    live2dLoaded.value = false;
    live2dError.value = 'Live2D SDK 未就绪，已启用备用动效预览';
    console.warn('Live2D SDK unavailable, fallback avatar is active.', error);
  }
}

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
}

function applyExpression() {
  if (!live2dModel || typeof live2dModel.expression !== 'function') return;
  const map = {
    neutral: 'exp_02',
    happy: 'exp_01',
    thinking: 'exp_04',
    surprised: 'exp_05',
    interrupted: 'exp_07',
  } as const;
  const expression = map[props.expression];
  if (expression === appliedExpression) return;
  appliedExpression = expression;
  const seq = ++expressionSeq;
  void live2dModel.expression(expression).then((applied: boolean) => {
    if (!applied && seq === expressionSeq) {
      appliedExpression = '';
      console.warn(`Live2D expression "${expression}" was not applied.`);
    }
  }).catch((error: unknown) => {
    if (seq === expressionSeq) appliedExpression = '';
    console.warn(`Live2D expression "${expression}" failed.`, error);
  });
}

function syncLive2DFrame() {
  if (!live2dModel) return;
  live2dModel.update?.(pixiApp?.ticker?.deltaMS ?? 16.7);
  applyExpression();
}

function applyLipSyncParameters() {
  if (!live2dModel) return;
  const mouthValue = props.state === 'speaking' ? Math.min(Math.max(props.mouthOpen, 0), 1) : 0;
  const coreModel = live2dModel.internalModel?.coreModel;
  if (coreModel?.setParameterValueById) {
    lipSyncParameterIds.forEach(id => coreModel.setParameterValueById(id, mouthValue, 1));
  }
}

function applyLive2DState() {
  applyLipSyncParameters();
  applyExpression();
}

function loop() {
  if (live2dLoaded.value) return;
  frame += 1;
  drawFallback();
  raf = requestAnimationFrame(loop);
}

onMounted(() => {
  loop();
  void loadLive2DModel();
});

onUnmounted(() => {
  cancelAnimationFrame(raf);
  window.removeEventListener('resize', syncLive2DLayout);
  resizeObserver?.disconnect();
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
