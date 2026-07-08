<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';
import { useSeniorMode } from '../composables/useSeniorMode';
import { apiFetch } from '../services/api';

const { t } = useI18n();
const router = useRouter();
const { seniorModeEnabled, toggleSeniorMode } = useSeniorMode();
const QR_INTRO_STORAGE_KEY = 'sg_qr_intro_payload';

interface SpotInfo {
  id: number;
  name: string;
  description: string;
  category: string;
  image_url: string;
  latitude: number;
  longitude: number;
  qr_intro_text: string;
}

interface IntroResponse {
  spot: SpotInfo;
  intro: string;
  cached: boolean;
  follow_up_questions: string[];
}

const loading = ref(true);
const error = ref('');
const spot = ref<SpotInfo | null>(null);
const intro = ref('');
const followUpQuestions = ref<string[]>([]);

onMounted(async () => {
  // QR code URL: /scan?id=SPOT-0001 (read from window.location.search, not hash)
  const params = new URLSearchParams(window.location.search);
  const qrCode = params.get('id') || params.get('qr');

  if (!qrCode) {
    error.value = t('qr.invalidCode');
    loading.value = false;
    return;
  }

  try {
    // 先轻量查询景点信息
    const spotData = await apiFetch<SpotInfo>(`/qr/${encodeURIComponent(qrCode)}`);
    spot.value = spotData;

    // 再触发 AI 讲解（有缓存，快速）
    const introData = await apiFetch<IntroResponse>(`/qr/${encodeURIComponent(qrCode)}/intro`, {
      method: 'POST',
      body: '{}',
    });
    intro.value = introData.intro;
    followUpQuestions.value = introData.follow_up_questions || [];
  } catch (e: unknown) {
    error.value = (e as Error)?.message || t('qr.spotNotFound');
  } finally {
    loading.value = false;
  }
});

function startTour() {
  if (!spot.value) return;
  const introText = intro.value || spot.value.qr_intro_text || spot.value.description || spot.value.name;
  sessionStorage.setItem(QR_INTRO_STORAGE_KEY, JSON.stringify({
    spot: spot.value.name,
    intro: introText,
  }));
  router.push({ name: 'digital-human', query: { qr_spot: spot.value.name, qr_direct: '1' } });
}

function chatNow() {
  router.push({ name: 'digital-human' });
}
</script>

<template>
  <main class="qr-scan-view" :class="{ 'senior-mode-page': seniorModeEnabled }">
    <button class="senior-toggle" @click="toggleSeniorMode">
      {{ seniorModeEnabled ? $t('qr.exitSeniorMode') : $t('qr.seniorMode') }}
    </button>
    <!-- Loading -->
    <div v-if="loading" class="scan-loading">
      <div class="pulse-ring"></div>
      <p>{{ $t('qr.scanning') }}</p>
    </div>

    <!-- Error -->
    <div v-else-if="error" class="scan-error">
      <div class="error-icon">⚠️</div>
      <p>{{ error }}</p>
      <button class="action-btn secondary" @click="chatNow">{{ $t('qr.backToDH') }}</button>
    </div>

    <!-- Spot Info -->
    <div v-else-if="spot" class="scan-result">
      <div class="spot-header">
        <span class="spot-badge">📍 {{ $t('qr.spotLocated') }}</span>
        <span v-if="spot.category" class="spot-category">{{ spot.category }}</span>
      </div>

      <h1 class="spot-name">{{ spot.name }}</h1>

      <div v-if="spot.image_url" class="spot-image">
        <img :src="spot.image_url" :alt="spot.name" />
      </div>

      <div v-if="intro" class="spot-intro">
        {{ intro.replace(/\[(joy|sadness|surprise|anger|fear|disgust|neutral)\]/gi, '') }}
      </div>

      <div v-if="followUpQuestions.length > 0" class="follow-up">
        <p class="follow-up-label">{{ $t('dh.scanFollowUp') }}</p>
        <div class="follow-up-list">
          <button
            v-for="(q, i) in followUpQuestions"
            :key="i"
            class="follow-up-btn"
            @click="router.push({ name: 'digital-human', query: { auto_ask: q } })"
          >
            {{ q }}
          </button>
        </div>
      </div>

      <div class="scan-actions">
        <button class="action-btn primary" @click="startTour">
          🎤 {{ $t('qr.startTour') }}
        </button>
        <button class="action-btn secondary" @click="chatNow">
          💬 {{ $t('qr.chatNow') }}
        </button>
      </div>
    </div>
  </main>
</template>

<style scoped>
.qr-scan-view {
  min-height: 100vh;
  background: linear-gradient(180deg, #0a0a0f 0%, #0f1a14 50%, #0a0a0f 100%);
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 24px;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
}

.senior-toggle {
  position: fixed;
  top: 16px;
  right: 16px;
  z-index: 2;
  padding: 9px 14px;
  border-radius: 10px;
  border: 1px solid rgba(255,255,255,0.16);
  background: rgba(255,255,255,0.08);
  color: rgba(255,255,255,0.82);
  font-size: 14px;
  cursor: pointer;
}

/* Loading */
.scan-loading {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 20px;
  color: rgba(255,255,255,0.6);
  font-size: 15px;
}
.pulse-ring {
  width: 64px;
  height: 64px;
  border-radius: 50%;
  border: 3px solid rgba(99, 226, 183, 0.2);
  border-top-color: #63e2b7;
  animation: spin 1s linear infinite;
}
@keyframes spin { to { transform: rotate(360deg); } }

/* Error */
.scan-error {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
  color: rgba(255,255,255,0.6);
  text-align: center;
}
.error-icon { font-size: 48px; }

/* Result */
.scan-result {
  max-width: 480px;
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.spot-header {
  display: flex;
  align-items: center;
  gap: 10px;
}
.spot-badge {
  font-size: 12px;
  color: #63e2b7;
  background: rgba(99,226,183,0.1);
  border: 1px solid rgba(99,226,183,0.3);
  padding: 3px 10px;
  border-radius: 20px;
}
.spot-category {
  font-size: 11px;
  color: rgba(255,255,255,0.4);
  background: rgba(255,255,255,0.06);
  padding: 2px 8px;
  border-radius: 10px;
}

.spot-name {
  font-size: 24px;
  font-weight: 700;
  color: rgba(255,255,255,0.92);
  margin: 0;
  line-height: 1.3;
}

.spot-image {
  border-radius: 12px;
  overflow: hidden;
  border: 1px solid rgba(255,255,255,0.08);
}
.spot-image img {
  width: 100%;
  height: 200px;
  object-fit: cover;
  display: block;
}

.spot-intro {
  font-size: 14px;
  line-height: 1.7;
  color: rgba(255,255,255,0.72);
  background: rgba(255,255,255,0.04);
  border: 1px solid rgba(255,255,255,0.08);
  border-radius: 12px;
  padding: 16px 20px;
}

.follow-up {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.follow-up-label {
  font-size: 12px;
  color: rgba(255,255,255,0.35);
  margin: 0;
}
.follow-up-list {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.follow-up-btn {
  font-size: 12px;
  color: #63e2b7;
  background: rgba(99,226,183,0.06);
  border: 1px solid rgba(99,226,183,0.2);
  border-radius: 16px;
  padding: 5px 12px;
  cursor: pointer;
  transition: all 0.2s;
}
.follow-up-btn:hover {
  background: rgba(99,226,183,0.15);
}

.scan-actions {
  display: flex;
  gap: 10px;
  margin-top: 8px;
}
.action-btn {
  flex: 1;
  padding: 12px 20px;
  border-radius: 10px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
  border: none;
}
.action-btn.primary {
  background: linear-gradient(135deg, #63e2b7, #18a058);
  color: #0a0a0f;
}
.action-btn.primary:hover { opacity: 0.9; transform: translateY(-1px); }
.action-btn.secondary {
  background: rgba(255,255,255,0.06);
  border: 1px solid rgba(255,255,255,0.12);
  color: rgba(255,255,255,0.7);
}
.action-btn.secondary:hover {
  background: rgba(255,255,255,0.1);
  color: rgba(255,255,255,0.9);
}

@media (max-width: 480px) {
  .qr-scan-view { padding: 16px; }
  .spot-name { font-size: 20px; }
  .spot-intro { padding: 12px 16px; font-size: 13px; }
}

/* 游客端自然景区服务风 */
.qr-scan-view {
  color: var(--visitor-ink);
  background:
    radial-gradient(ellipse at 18% 0%, rgba(232, 220, 199, 0.36), transparent 34%),
    radial-gradient(ellipse at 86% 88%, rgba(198, 107, 61, 0.2), transparent 34%),
    linear-gradient(135deg, var(--visitor-sage), var(--visitor-moss));
}

.qr-scan-view::before {
  position: fixed;
  inset: 0;
  pointer-events: none;
  content: "";
  opacity: 0.12;
  background:
    radial-gradient(circle at 20% 20%, rgba(232, 220, 199, 0.8) 0 1px, transparent 1px),
    radial-gradient(circle at 75% 72%, rgba(38, 51, 31, 0.65) 0 1px, transparent 1px);
  background-size: 18px 18px, 26px 26px;
}

.senior-toggle {
  border-color: rgba(232, 220, 199, 0.34);
  border-radius: 999px;
  color: var(--visitor-ink);
  background: var(--visitor-sand);
}

.scan-loading,
.scan-error,
.scan-result {
  position: relative;
  z-index: 1;
}

.scan-loading,
.scan-error {
  width: min(460px, 100%);
  padding: 28px;
  border: 1px solid var(--visitor-line);
  border-radius: var(--visitor-radius);
  color: var(--visitor-ink);
  background: var(--visitor-sand);
  box-shadow: var(--visitor-shadow);
}

.pulse-ring {
  border-color: rgba(96, 108, 56, 0.2);
  border-top-color: var(--visitor-moss);
}

.scan-result {
  max-width: 560px;
  padding: 24px;
  border: 1px solid var(--visitor-line);
  border-radius: 30px;
  background: var(--visitor-sand);
  box-shadow: var(--visitor-shadow);
}

.spot-badge {
  color: var(--visitor-sand);
  background: var(--visitor-moss);
  border-color: var(--visitor-moss);
}

.spot-category {
  color: var(--visitor-ink);
  background: rgba(96, 108, 56, 0.12);
}

.spot-name {
  color: var(--visitor-ink);
  font-family: Georgia, "Times New Roman", serif;
  font-size: 30px;
}

.spot-image {
  border-color: var(--visitor-line);
  border-radius: 24px;
}

.spot-intro {
  color: var(--visitor-muted);
  border-color: var(--visitor-line);
  border-radius: 22px;
  background: rgba(255, 255, 255, 0.28);
}

.follow-up-label {
  color: var(--visitor-muted);
}

.follow-up-btn {
  color: var(--visitor-ink);
  border-color: var(--visitor-line);
  background: rgba(255, 255, 255, 0.24);
}

.follow-up-btn:hover {
  background: rgba(96, 108, 56, 0.12);
}

.action-btn {
  border-radius: 999px;
}

.action-btn.primary {
  color: var(--visitor-sand);
  background: var(--visitor-moss);
}

.action-btn.secondary {
  color: var(--visitor-ink);
  border: 1px solid var(--visitor-line);
  background: rgba(255, 255, 255, 0.28);
}
</style>
