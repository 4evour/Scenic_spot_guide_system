<script setup lang="ts">
import { ref } from 'vue';

const emit = defineEmits<{
  (e: 'login-success'): void;
}>();

const username = ref('');
const password = ref('');
const loading = ref(false);
const error = ref('');

async function handleLogin() {
  if (!username.value || !password.value) {
    error.value = '请输入用户名和密码';
    return;
  }
  loading.value = true;
  error.value = '';
  try {
    const res = await fetch('/api/v1/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username: username.value, password: password.value }),
    });
    const data = await res.json();
    if (data.code !== 0) {
      error.value = data.message || '登录失败';
      return;
    }
    localStorage.setItem('authToken', data.data.token);
    localStorage.setItem('user', JSON.stringify(data.data));
    emit('login-success');
  } catch {
    error.value = '网络错误，请重试';
  } finally {
    loading.value = false;
  }
}
</script>

<template>
  <div class="login-wrapper">
    <div class="login-card">
      <div class="login-brand">
        <span class="brand-icon">🏔</span>
        <h1>景区智能导览系统</h1>
        <p>Scenic Spot Guide System</p>
      </div>
      <form @submit.prevent="handleLogin">
        <label>
          <span>用户名</span>
          <input v-model="username" type="text" placeholder="请输入用户名" autocomplete="username" />
        </label>
        <label>
          <span>密码</span>
          <input v-model="password" type="password" placeholder="请输入密码" autocomplete="current-password" />
        </label>
        <p v-if="error" class="login-error">{{ error }}</p>
        <button type="submit" class="login-btn" :disabled="loading">
          {{ loading ? '登录中...' : '登 录' }}
        </button>
      </form>
      <p class="login-hint">管理员登录后可访问数据大屏和管理后台</p>
    </div>
  </div>
</template>

<style scoped>
.login-wrapper {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  background: linear-gradient(135deg, #0f1923 0%, #1a2a3a 50%, #0d1b2a 100%);
}

.login-card {
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 16px;
  padding: 40px 36px;
  width: 380px;
  backdrop-filter: blur(20px);
}

.login-brand {
  text-align: center;
  margin-bottom: 32px;
}

.brand-icon {
  font-size: 48px;
  display: block;
  margin-bottom: 12px;
}

.login-brand h1 {
  color: #e8d5a3;
  font-size: 22px;
  margin: 0 0 4px;
}

.login-brand p {
  color: rgba(255, 255, 255, 0.4);
  font-size: 12px;
  margin: 0;
  letter-spacing: 2px;
}

form label {
  display: block;
  margin-bottom: 16px;
}

form label span {
  display: block;
  color: rgba(255, 255, 255, 0.6);
  font-size: 13px;
  margin-bottom: 6px;
}

form input {
  width: 100%;
  padding: 10px 14px;
  background: rgba(255, 255, 255, 0.06);
  border: 1px solid rgba(255, 255, 255, 0.12);
  border-radius: 8px;
  color: #fff;
  font-size: 14px;
  outline: none;
  transition: border-color 0.2s;
  box-sizing: border-box;
}

form input:focus {
  border-color: #e8d5a3;
}

.login-error {
  color: #ff8b8b;
  font-size: 13px;
  margin: 8px 0;
  text-align: center;
}

.login-btn {
  width: 100%;
  padding: 12px;
  margin-top: 8px;
  background: linear-gradient(135deg, #e8d5a3, #c4a96a);
  border: none;
  border-radius: 8px;
  color: #1a1a2e;
  font-size: 15px;
  font-weight: 600;
  cursor: pointer;
  transition: opacity 0.2s;
}

.login-btn:hover {
  opacity: 0.9;
}

.login-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.login-hint {
  text-align: center;
  color: rgba(255, 255, 255, 0.3);
  font-size: 12px;
  margin-top: 20px;
}
</style>
