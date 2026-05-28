<script setup lang="ts">
import { ref } from 'vue';
import { NCard, NForm, NFormItem, NInput, NButton, NSpace, useMessage } from 'naive-ui';

const emit = defineEmits<{ (e: 'login-success'): void }>();
const message = useMessage();

const form = ref({ username: '', password: '' });
const loading = ref(false);

async function handleLogin() {
  if (!form.value.username || !form.value.password) {
    message.warning('请输入用户名和密码');
    return;
  }
  loading.value = true;
  try {
    const res = await fetch('/api/v1/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(form.value),
    });
    const data = await res.json();
    if (data.code !== 0) {
      message.error(data.message || '登录失败');
      return;
    }
    localStorage.setItem('authToken', data.data.token);
    localStorage.setItem('user', JSON.stringify(data.data));
    message.success('登录成功');
    emit('login-success');
  } catch {
    message.error('网络错误，请重试');
  } finally {
    loading.value = false;
  }
}
</script>

<template>
  <div class="login-wrapper">
    <div class="login-bg"></div>
    <NCard class="login-card" :bordered="false">
      <div class="login-brand">
        <svg width="48" height="48" viewBox="0 0 24 24" fill="none">
          <path d="M12 2L2 7l10 5 10-5-10-5z" fill="#63e2b7" opacity="0.8"/>
          <path d="M2 17l10 5 10-5M2 12l10 5 10-5" stroke="#63e2b7" stroke-width="1.5" fill="none" opacity="0.5"/>
        </svg>
        <h1>景区智能导览系统</h1>
        <p>Scenic Spot Guide System</p>
      </div>

      <NForm @submit.prevent="handleLogin">
        <NFormItem label="用户名">
          <NInput v-model:value="form.username" placeholder="请输入用户名" size="large" />
        </NFormItem>
        <NFormItem label="密码">
          <NInput v-model:value="form.password" type="password" placeholder="请输入密码" size="large" show-password-on="click" />
        </NFormItem>
        <NSpace vertical :size="16" style="width: 100%; margin-top: 8px;">
          <NButton type="primary" block size="large" :loading="loading" @click="handleLogin">
            登 录
          </NButton>
        </NSpace>
      </NForm>

      <p class="login-hint">管理员登录后可访问数据大屏和管理后台</p>
    </NCard>
  </div>
</template>

<style scoped>
.login-wrapper {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  background: #0a0a0f;
  position: relative;
  overflow: hidden;
}
.login-bg {
  position: absolute;
  inset: 0;
  background:
    radial-gradient(ellipse at 20% 50%, rgba(99, 226, 183, 0.08) 0%, transparent 60%),
    radial-gradient(ellipse at 80% 20%, rgba(99, 226, 183, 0.05) 0%, transparent 50%);
}
.login-card {
  width: 400px;
  background: rgba(255, 255, 255, 0.04) !important;
  border: 1px solid rgba(255, 255, 255, 0.08) !important;
  border-radius: 16px !important;
  backdrop-filter: blur(20px);
  position: relative;
  z-index: 1;
}
.login-brand {
  text-align: center;
  margin-bottom: 32px;
}
.login-brand h1 {
  font-size: 22px;
  margin: 12px 0 4px;
  color: rgba(255, 255, 255, 0.88);
}
.login-brand p {
  font-size: 12px;
  color: rgba(255, 255, 255, 0.35);
  letter-spacing: 2px;
}
.login-hint {
  text-align: center;
  font-size: 12px;
  color: rgba(255, 255, 255, 0.25);
  margin-top: 20px;
}
</style>
