<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { useRouter } from 'vue-router';
import { NCard, NForm, NFormItem, NInput, NButton, NButtonGroup, NSpace, useMessage } from 'naive-ui';
import { useI18n } from 'vue-i18n';
import { useAuthStore } from '../stores/auth';

const { t } = useI18n();
const router = useRouter();
const authStore = useAuthStore();
const message = useMessage();

const mode = ref<'login' | 'register'>('login');
const form = ref({ username: '', password: '' });
const registerForm = ref({ username: '', password: '', email: '' });
const loading = ref(false);
const registerLoading = ref(false);
const guestLoading = ref(false);

interface DemoAccount {
  role: 'visitor' | 'admin';
  username: string;
  password: string;
}

interface DemoInfo {
  enabled: boolean;
  accounts: DemoAccount[];
}

const demoInfo = ref<DemoInfo | null>(null);

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null;
}

function isDemoAccount(value: unknown): value is DemoAccount {
  if (!isRecord(value)) return false;
  return (value.role === 'visitor' || value.role === 'admin')
    && typeof value.username === 'string'
    && value.username.length > 0
    && typeof value.password === 'string'
    && value.password.length > 0;
}

onMounted(async () => {
  try {
    const response = await fetch('/api/v1/demo-info', {
      credentials: 'same-origin',
      cache: 'no-store',
      signal: AbortSignal.timeout(5000),
    });
    if (!response.ok) return;
    const payload: unknown = await response.json();
    if (!isRecord(payload) || payload.code !== 0 || !isRecord(payload.data)) return;
    const data = payload.data;
    if (data.enabled !== true || !Array.isArray(data.accounts) || !data.accounts.every(isDemoAccount)) return;
    demoInfo.value = { enabled: true, accounts: data.accounts };
  } catch {
    demoInfo.value = null;
  }
});

function fillDemoAccount(account: DemoAccount) {
  mode.value = 'login';
  form.value = { username: account.username, password: account.password };
}

function isPasswordPolicyValid(password: string) {
  return password.length >= 8
    && password.length <= 128
    && /[A-Z]/.test(password)
    && /[a-z]/.test(password)
    && /\d/.test(password);
}

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

async function handleRegister() {
  if (!registerForm.value.username || !registerForm.value.password) {
    message.warning(t('login.emptyFields'));
    return;
  }
  if (!isPasswordPolicyValid(registerForm.value.password)) {
    message.warning(t('login.passwordPolicyInvalid'));
    return;
  }
  registerLoading.value = true;
  try {
    const res = await fetch('/api/v1/register', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      credentials: 'include',
      signal: AbortSignal.timeout(15000),
      body: JSON.stringify(registerForm.value),
    });
    const data = await res.json();
    if (!res.ok || data.code !== 0) {
      message.error(data.message || t('login.registerFailed'));
      return;
    }
    message.success(t('login.registerSuccess'));
    form.value.username = registerForm.value.username;
    form.value.password = '';
    registerForm.value = { username: '', password: '', email: '' };
    mode.value = 'login';
  } catch {
    message.error(t('login.networkError'));
  } finally {
    registerLoading.value = false;
  }
}
</script>

<template>
  <div class="login-wrapper">
    <!-- 背景 -->
    <div class="login-bg">
      <div class="terrain terrain-1"></div>
      <div class="terrain terrain-2"></div>
      <div class="grain-layer"></div>
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

      <NButtonGroup class="mode-switch">
        <NButton :type="mode === 'login' ? 'primary' : 'default'" @click="mode = 'login'">
          {{ $t('login.modeLogin') }}
        </NButton>
        <NButton :type="mode === 'register' ? 'primary' : 'default'" @click="mode = 'register'">
          {{ $t('login.modeRegister') }}
        </NButton>
      </NButtonGroup>

      <section v-if="mode === 'login' && demoInfo?.enabled" class="demo-credentials" role="region" :aria-label="$t('login.demoTitle')">
        <div class="demo-credentials-heading">
          <strong>{{ $t('login.demoTitle') }}</strong>
          <span>{{ $t('login.demoHint') }}</span>
        </div>
        <button
          v-for="account in demoInfo.accounts"
          :key="account.role"
          type="button"
          class="demo-account-row"
          :aria-label="$t('login.demoFillAria', { role: $t(account.role === 'admin' ? 'login.demoAdmin' : 'login.demoVisitor') })"
          @click="fillDemoAccount(account)"
        >
          <span class="demo-account-role">{{ $t(account.role === 'admin' ? 'login.demoAdmin' : 'login.demoVisitor') }}</span>
          <span class="demo-account-values">
            <span>{{ $t('login.username') }}：<code>{{ account.username }}</code></span>
            <span>{{ $t('login.password') }}：<code>{{ account.password }}</code></span>
          </span>
          <span class="demo-account-action">{{ $t('login.demoFill') }}</span>
        </button>
      </section>

      <NForm v-if="mode === 'login'" @submit.prevent="handleLogin">
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
            <span class="guest-button-content">
              <svg class="guest-button-icon" viewBox="0 0 24 24" aria-hidden="true">
                <path d="M20 4c-6.8.4-11.8 3.4-14.8 8.9C3.8 15.5 3.4 18 4 20c2.1.5 4.7.1 7.3-1.3C16.7 15.7 19.6 10.8 20 4Z" />
                <path d="M7 17c2.2-3.6 5.2-6.2 9-7.8" />
              </svg>
              {{ $t('login.guestContinue') }}
            </span>
          </NButton>
        </NSpace>
      </NForm>
      <NForm v-else @submit.prevent="handleRegister">
        <NFormItem :label="$t('login.username')">
          <NInput v-model:value="registerForm.username" :placeholder="$t('login.usernamePlaceholder')" size="large" />
        </NFormItem>
        <NFormItem :label="$t('login.email')">
          <NInput v-model:value="registerForm.email" :placeholder="$t('login.emailPlaceholder')" size="large" />
        </NFormItem>
        <NFormItem :label="$t('login.password')">
          <NInput v-model:value="registerForm.password" type="password" :placeholder="$t('login.passwordPlaceholder')" size="large" show-password-on="click" />
        </NFormItem>
        <p class="password-policy">{{ $t('login.passwordPolicy') }}</p>
        <NSpace vertical :size="16" style="width: 100%; margin-top: 8px;">
          <NButton type="primary" block size="large" :loading="registerLoading" @click="handleRegister">
            {{ $t('login.registerSubmit') }}
          </NButton>
          <NButton block size="large" quaternary :loading="guestLoading" @click="handleGuestLogin">
            {{ $t('login.guestContinue') }}
          </NButton>
        </NSpace>
      </NForm>

      <p class="login-hint">{{ $t(mode === 'login' ? 'login.hint' : 'login.registerHint') }}</p>
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
  padding: 24px;
  background:
    linear-gradient(135deg, rgba(139, 157, 131, 0.94), rgba(96, 108, 56, 0.96)),
    var(--visitor-moss, #606c38);
  position: relative;
  overflow: hidden;
}

.login-bg {
  position: absolute;
  inset: 0;
  overflow: hidden;
}

.grain-layer {
  position: absolute;
  inset: 0;
  opacity: 0.16;
  background:
    radial-gradient(circle at 20% 20%, rgba(232, 220, 199, 0.45) 0 1px, transparent 1px),
    radial-gradient(circle at 70% 65%, rgba(38, 51, 31, 0.4) 0 1px, transparent 1px);
  background-size: 18px 18px, 26px 26px;
}

.terrain {
  position: absolute;
  left: 50%;
  width: 120vw;
  border-radius: 50%;
  transform: translateX(-50%);
}
.terrain-1 {
  bottom: -42vh;
  height: 72vh;
  background: rgba(232, 220, 199, 0.28);
}
.terrain-2 {
  top: -54vh;
  height: 78vh;
  background: rgba(198, 107, 61, 0.2);
}

/* 登录卡片 */
.login-card {
  width: 420px;
  background: var(--visitor-sand, #e8dcc7) !important;
  border: 1px solid rgba(96, 108, 56, 0.2) !important;
  border-radius: 28px !important;
  position: relative;
  z-index: 1;
  box-shadow: var(--visitor-shadow, 0 22px 56px rgba(38, 51, 31, 0.22));
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
  background: rgba(96, 108, 56, 0.1);
  border: 1px solid rgba(96, 108, 56, 0.18);
  display: flex;
  align-items: center;
  justify-content: center;
}
.logo-ring svg path:first-child { fill: var(--visitor-moss, #606c38); }
.logo-ring svg path:last-child { stroke: var(--visitor-moss, #606c38); }
.login-brand h1 {
  font-size: 26px;
  font-weight: 700;
  margin: 0 0 6px;
  color: var(--visitor-ink, #26331f);
  letter-spacing: 0;
}
.login-brand p {
  font-size: 12px;
  color: var(--visitor-muted, rgba(38, 51, 31, 0.68));
  letter-spacing: 0;
}

.mode-switch {
  display: grid;
  grid-template-columns: 1fr 1fr;
  width: 100%;
  margin-bottom: 20px;
}

.mode-switch :deep(.n-button) {
  width: 100%;
}

.demo-credentials {
  margin: 0 0 18px;
  padding: 12px 0;
  border-top: 1px solid rgba(96, 108, 56, 0.2);
  border-bottom: 1px solid rgba(96, 108, 56, 0.2);
}

.demo-credentials-heading {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 12px;
  margin: 0 4px 8px;
  color: var(--visitor-ink, #26331f);
}

.demo-credentials-heading strong {
  font-size: 13px;
}

.demo-credentials-heading span {
  font-size: 11px;
  color: rgba(38, 51, 31, 0.58);
  text-align: right;
}

.demo-account-row {
  display: grid;
  grid-template-columns: 64px minmax(0, 1fr) auto;
  align-items: center;
  gap: 10px;
  width: 100%;
  padding: 9px 4px;
  color: var(--visitor-ink, #26331f);
  background: transparent;
  border: 0;
  border-top: 1px solid rgba(96, 108, 56, 0.12);
  cursor: pointer;
  text-align: left;
}

.demo-account-row:hover {
  background: rgba(96, 108, 56, 0.08);
}

.demo-account-row:focus-visible {
  background: rgba(96, 108, 56, 0.08);
  outline: 2px solid var(--visitor-moss, #606c38);
  outline-offset: 2px;
}

.demo-account-role,
.demo-account-action {
  font-size: 12px;
  font-weight: 700;
}

.demo-account-values {
  display: grid;
  gap: 2px;
  min-width: 0;
  font-size: 11px;
  color: rgba(38, 51, 31, 0.72);
}

.demo-account-values code {
  color: var(--visitor-ink, #26331f);
  font-family: Consolas, monospace;
  overflow-wrap: anywhere;
}

.demo-account-action {
  color: var(--visitor-moss, #606c38);
}

.password-policy {
  margin: -8px 0 10px;
  font-size: 12px;
  line-height: 1.5;
  color: rgba(38, 51, 31, 0.58);
}

.login-hint {
  text-align: center;
  font-size: 12px;
  color: rgba(38, 51, 31, 0.58);
  margin-top: 24px;
}
.login-version {
  text-align: center;
  font-size: 11px;
  color: rgba(38, 51, 31, 0.42);
  margin-top: 8px;
  letter-spacing: 1px;
}

.login-card :deep(.n-form-item-label) {
  color: var(--visitor-ink, #26331f);
  font-weight: 700;
}

.login-card :deep(.n-input) {
  border-radius: 16px;
  background: rgba(255, 255, 255, 0.34);
}

.login-card :deep(.n-button) {
  border-radius: 18px;
}

.login-card :deep(.n-button--primary-type) {
  color: var(--visitor-sand, #e8dcc7);
  background: var(--visitor-moss, #606c38);
}

.guest-button-content {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
}

.guest-button-icon {
  width: 18px;
  height: 18px;
  fill: currentColor;
  stroke: currentColor;
  stroke-width: 1.7;
  stroke-linecap: round;
  stroke-linejoin: round;
}

@media (max-width: 480px) {
  .login-wrapper { padding: 16px; }
  .login-card { width: 100%; }
  .demo-credentials-heading { align-items: flex-start; flex-direction: column; gap: 2px; }
  .demo-credentials-heading span { text-align: left; }
  .demo-account-row { grid-template-columns: 56px minmax(0, 1fr); }
  .demo-account-action { grid-column: 2; }
}
</style>
