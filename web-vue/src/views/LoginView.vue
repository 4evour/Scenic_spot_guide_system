<script setup lang="ts">
import { ref } from 'vue';
import { useRouter } from 'vue-router';
import { NCard, NForm, NFormItem, NInput, NButton, NSpace, useMessage } from 'naive-ui';
import { useI18n } from 'vue-i18n';
import { useAuthStore } from '../stores/auth';

const { t } = useI18n();
const router = useRouter();
const authStore = useAuthStore();
const message = useMessage();

const form = ref({ username: '', password: '' });
const loading = ref(false);
const guestLoading = ref(false);

async function handleLogin() {
  if (!form.value.username || !form.value.password) {
    message.warning(t('login.emptyFields'));
    return;
  }
  loading.value = true;
  try {
    const res = await fetch('/api/v1/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      credentials: 'include',
      signal: AbortSignal.timeout(15000),
      body: JSON.stringify(form.value),
    });
    const data = await res.json();
    if (!res.ok || data.code !== 0) {
      message.error(data.message || t('login.failed'));
      return;
    }
    message.success(t('login.success'));
    authStore.invalidateAuth();
    await authStore.fetchUser();
    router.push({ name: authStore.isAdmin ? 'dashboard' : 'map' });
  } catch {
    message.error(t('login.networkError'));
  } finally {
    loading.value = false;
  }
}

async function handleGuestLogin() {
  guestLoading.value = true;
  try {
    const ok = await authStore.ensureGuestSession();
    if (ok) {
      message.success(t('login.guestSuccess'));
      router.push({ name: 'map' });
    } else {
      message.error(t('login.guestFailed'));
    }
  } catch {
    message.error(t('login.networkError'));
  } finally {
    guestLoading.value = false;
  }
}
</script>

<template>
  <div class="login-wrapper">
    <!-- 动态背景 -->
    <div class="login-bg">
      <div class="orb orb-1"></div>
      <div class="orb orb-2"></div>
      <div class="orb orb-3"></div>
      <div class="grid-lines"></div>
    </div>

    <NCard class="login-card" :bordered="false">
      <div class="login-brand">
        <div class="logo-ring">
          <svg width="56" height="56" viewBox="0 0 24 24" fill="none">
            <path d="M12 2L2 7l10 5 10-5-10-5z" fill="#63e2b7" opacity="0.9"/>
            <path d="M2 17l10 5 10-5M2 12l10 5 10-5" stroke="#63e2b7" stroke-width="1.5" fill="none" opacity="0.5"/>
          </svg>
        </div>
        <h1>{{ $t('login.title') }}</h1>
        <p>{{ $t('login.subtitle') }}</p>
      </div>

      <NForm @submit.prevent="handleLogin">
        <NFormItem :label="$t('login.username')">
          <NInput v-model:value="form.username" :placeholder="$t('login.usernamePlaceholder')" size="large" />
        </NFormItem>
        <NFormItem :label="$t('login.password')">
          <NInput v-model:value="form.password" type="password" :placeholder="$t('login.passwordPlaceholder')" size="large" show-password-on="click" />
        </NFormItem>
        <NSpace vertical :size="16" style="width: 100%; margin-top: 8px;">
          <NButton type="primary" block size="large" :loading="loading" @click="handleLogin">
            {{ $t('login.submit') }}
          </NButton>
          <NButton block size="large" quaternary :loading="guestLoading" @click="handleGuestLogin">
            {{ $t('login.guestContinue') }}
          </NButton>
        </NSpace>
      </NForm>

      <p class="login-hint">{{ $t('login.hint') }}</p>
      <p class="login-version">{{ $t('login.version') }}</p>
    </NCard>
  </div>
</template>

<style scoped>
.login-wrapper {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  background: #061012;
  position: relative;
  overflow: hidden;
}

/* 动态背景 */
.login-bg {
  position: absolute;
  inset: 0;
  overflow: hidden;
}
.grid-lines {
  position: absolute;
  inset: 0;
  background-image:
    linear-gradient(rgba(82, 240, 238, 0.03) 1px, transparent 1px),
    linear-gradient(90deg, rgba(82, 240, 238, 0.03) 1px, transparent 1px);
  background-size: 60px 60px;
}
.orb {
  position: absolute;
  border-radius: 50%;
  filter: blur(80px);
  animation: float 20s ease-in-out infinite;
}
.orb-1 {
  width: 400px; height: 400px;
  background: rgba(99, 226, 183, 0.08);
  top: -10%; left: -5%;
  animation-delay: 0s;
}
.orb-2 {
  width: 300px; height: 300px;
  background: rgba(82, 240, 238, 0.06);
  bottom: -10%; right: -5%;
  animation-delay: -7s;
}
.orb-3 {
  width: 200px; height: 200px;
  background: rgba(244, 199, 101, 0.04);
  top: 50%; left: 60%;
  animation-delay: -14s;
}
@keyframes float {
  0%, 100% { transform: translate(0, 0) scale(1); }
  33% { transform: translate(30px, -20px) scale(1.05); }
  66% { transform: translate(-20px, 15px) scale(0.95); }
}

/* 登录卡片 */
.login-card {
  width: 420px;
  background: rgba(8, 24, 28, 0.6) !important;
  border: 1px solid rgba(99, 226, 183, 0.12) !important;
  border-radius: 20px !important;
  backdrop-filter: blur(24px);
  -webkit-backdrop-filter: blur(24px);
  position: relative;
  z-index: 1;
  box-shadow:
    0 0 60px rgba(99, 226, 183, 0.06),
    0 24px 48px rgba(0, 0, 0, 0.4);
  padding: 8px;
}

.login-brand {
  text-align: center;
  margin-bottom: 36px;
}
.logo-ring {
  width: 80px;
  height: 80px;
  margin: 0 auto 16px;
  border-radius: 50%;
  background: rgba(99, 226, 183, 0.06);
  border: 1px solid rgba(99, 226, 183, 0.15);
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 0 30px rgba(99, 226, 183, 0.08);
}
.login-brand h1 {
  font-size: 24px;
  font-weight: 700;
  margin: 0 0 6px;
  color: rgba(255, 255, 255, 0.92);
  letter-spacing: 1px;
}
.login-brand p {
  font-size: 12px;
  color: rgba(255, 255, 255, 0.3);
  letter-spacing: 3px;
  text-transform: uppercase;
}

.login-hint {
  text-align: center;
  font-size: 12px;
  color: rgba(255, 255, 255, 0.2);
  margin-top: 24px;
}
.login-version {
  text-align: center;
  font-size: 11px;
  color: rgba(255, 255, 255, 0.1);
  margin-top: 8px;
  letter-spacing: 1px;
}

@media (max-width: 480px) {
  .login-card { width: calc(100% - 32px); }
}
</style>
