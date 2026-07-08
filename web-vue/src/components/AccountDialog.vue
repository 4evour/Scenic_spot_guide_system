<script setup lang="ts">
import { reactive, ref } from 'vue';
import { NButton, NForm, NFormItem, NInput, NModal, NSpace, useMessage } from 'naive-ui';
import { useI18n } from 'vue-i18n';
import { useAuthStore } from '../stores/auth';

const show = defineModel<boolean>('show', { default: false });

const { t } = useI18n();
const message = useMessage();
const authStore = useAuthStore();

const upgradeForm = reactive({ username: '', password: '', email: '' });
const passwordForm = reactive({ currentPassword: '', newPassword: '', confirmPassword: '' });
const loading = ref(false);

function isPasswordPolicyValid(password: string) {
  return password.length >= 8
    && password.length <= 128
    && /[A-Z]/.test(password)
    && /[a-z]/.test(password)
    && /\d/.test(password);
}

function resetForms() {
  upgradeForm.username = '';
  upgradeForm.password = '';
  upgradeForm.email = '';
  passwordForm.currentPassword = '';
  passwordForm.newPassword = '';
  passwordForm.confirmPassword = '';
}

async function handleUpgrade() {
  if (!upgradeForm.username || !upgradeForm.password) {
    message.warning(t('account.usernamePasswordRequired'));
    return;
  }
  if (!isPasswordPolicyValid(upgradeForm.password)) {
    message.warning(t('account.passwordPolicy'));
    return;
  }
  loading.value = true;
  try {
    const ok = await authStore.upgradeAccount(upgradeForm.username, upgradeForm.password, upgradeForm.email || undefined);
    if (!ok) {
      message.error(t('account.upgradeFailed'));
      return;
    }
    message.success(t('account.upgradeSuccess'));
    resetForms();
    show.value = false;
  } finally {
    loading.value = false;
  }
}

async function handleChangePassword() {
  if (!passwordForm.currentPassword || !passwordForm.newPassword) {
    message.warning(t('account.passwordFieldsRequired'));
    return;
  }
  if (passwordForm.newPassword !== passwordForm.confirmPassword) {
    message.warning(t('account.passwordMismatch'));
    return;
  }
  if (!isPasswordPolicyValid(passwordForm.newPassword)) {
    message.warning(t('account.passwordPolicy'));
    return;
  }
  loading.value = true;
  try {
    const ok = await authStore.changePassword(passwordForm.currentPassword, passwordForm.newPassword);
    if (!ok) {
      message.error(t('account.passwordChangeFailed'));
      return;
    }
    message.success(t('account.passwordChangeSuccess'));
    resetForms();
    show.value = false;
  } finally {
    loading.value = false;
  }
}
</script>

<template>
  <NModal v-model:show="show" preset="card" class="account-dialog" :bordered="false">
    <template #header>
      <div class="account-title">{{ $t('account.title') }}</div>
    </template>

    <div class="account-summary">
      <span>{{ $t('account.current') }}</span>
      <strong>{{ authStore.displayName || $t('account.guestName') }}</strong>
      <small>{{ $t(`account.roles.${authStore.user?.role || 'guest'}`) }}</small>
    </div>

    <NForm v-if="authStore.isGuest" @submit.prevent="handleUpgrade">
      <p class="account-hint">{{ $t('account.guestHint') }}</p>
      <NFormItem :label="$t('account.username')">
        <NInput v-model:value="upgradeForm.username" :placeholder="$t('account.usernamePlaceholder')" />
      </NFormItem>
      <NFormItem :label="$t('account.email')">
        <NInput v-model:value="upgradeForm.email" :placeholder="$t('account.emailPlaceholder')" />
      </NFormItem>
      <NFormItem :label="$t('account.password')">
        <NInput v-model:value="upgradeForm.password" type="password" show-password-on="click" :placeholder="$t('account.passwordPlaceholder')" />
      </NFormItem>
      <p class="account-policy">{{ $t('account.passwordPolicy') }}</p>
      <NSpace justify="end">
        <NButton @click="show = false">{{ $t('common.cancel') }}</NButton>
        <NButton type="primary" :loading="loading" @click="handleUpgrade">{{ $t('account.upgrade') }}</NButton>
      </NSpace>
    </NForm>

    <NForm v-else @submit.prevent="handleChangePassword">
      <p class="account-hint">{{ $t('account.passwordHint') }}</p>
      <NFormItem :label="$t('account.currentPassword')">
        <NInput v-model:value="passwordForm.currentPassword" type="password" show-password-on="click" :placeholder="$t('account.currentPasswordPlaceholder')" />
      </NFormItem>
      <NFormItem :label="$t('account.newPassword')">
        <NInput v-model:value="passwordForm.newPassword" type="password" show-password-on="click" :placeholder="$t('account.passwordPlaceholder')" />
      </NFormItem>
      <NFormItem :label="$t('account.confirmPassword')">
        <NInput v-model:value="passwordForm.confirmPassword" type="password" show-password-on="click" :placeholder="$t('account.confirmPasswordPlaceholder')" />
      </NFormItem>
      <p class="account-policy">{{ $t('account.passwordPolicy') }}</p>
      <NSpace justify="end">
        <NButton @click="show = false">{{ $t('common.cancel') }}</NButton>
        <NButton type="primary" :loading="loading" @click="handleChangePassword">{{ $t('account.changePassword') }}</NButton>
      </NSpace>
    </NForm>
  </NModal>
</template>

<style scoped>
.account-dialog {
  width: min(420px, calc(100vw - 32px));
  background: var(--visitor-sand, #e8dcc7);
  border-radius: 24px;
}

.account-title {
  color: var(--visitor-ink, #26331f);
  font-weight: 800;
}

.account-summary {
  display: grid;
  gap: 4px;
  margin-bottom: 18px;
  padding: 12px;
  border: 1px solid rgba(96, 108, 56, 0.16);
  border-radius: 16px;
  background: rgba(255, 255, 255, 0.24);
  color: var(--visitor-ink, #26331f);
}

.account-summary span,
.account-summary small,
.account-hint,
.account-policy {
  color: rgba(38, 51, 31, 0.62);
  font-size: 12px;
}

.account-summary strong {
  font-size: 17px;
}

.account-hint {
  margin: 0 0 14px;
  line-height: 1.6;
}

.account-policy {
  margin: -8px 0 16px;
  line-height: 1.5;
}
</style>
