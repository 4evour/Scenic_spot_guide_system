<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import {
  NCard,
  NForm,
  NFormItem,
  NInput,
  NSwitch,
  NSelect,
  NButton,
  NDivider,
  NSkeleton,
  type FormInst,
  type FormRules,
} from 'naive-ui'
import { useMessage } from 'naive-ui'
import { apiFetch } from '../services/api'
import type { SystemSettings } from '../types/admin'
import { defaultSettings } from '../types/admin'

const message = useMessage()

const loading = ref(false)
const saving = ref(false)

const settings = reactive<SystemSettings>({ ...defaultSettings })

const formRef = ref<FormInst | null>(null)

const backupFrequencyOptions = [
  { label: '每日', value: '每日' },
  { label: '每周', value: '每周' },
  { label: '每月', value: '每月' },
  { label: '手动', value: '手动' },
]

const rules: FormRules = {
  scenic_name: [{ required: true, message: '请输入景区名称', trigger: 'blur' }],
  data_retention: [
    { required: true, message: '请输入数据保留天数', trigger: 'blur' },
    {
      validator: (_rule, value: string) => {
        const num = Number(value)
        if (Number.isNaN(num) || num < 1 || num > 365) {
          return new Error('1-365天')
        }
        return true
      },
      trigger: 'blur',
    },
  ],
}

async function loadSettings() {
  loading.value = true
  try {
    const data = await apiFetch<Partial<SystemSettings>>('/admin/settings')
    Object.assign(settings, { ...defaultSettings, ...data })
  } catch (error) {
    message.error(error instanceof Error ? error.message : '系统设置加载失败')
  } finally {
    loading.value = false
  }
}

async function saveSettings() {
  try {
    await formRef.value?.validate()
  } catch {
    return
  }
  saving.value = true
  try {
    await apiFetch('/admin/settings', { method: 'PUT', body: JSON.stringify(settings) })
    message.success('系统设置已保存')
  } catch (error) {
    message.error(error instanceof Error ? error.message : '保存失败')
  } finally {
    saving.value = false
  }
}

onMounted(loadSettings)
</script>

<template>
  <NCard title="系统设置" :bordered="false" class="settings-card">
    <template v-if="loading">
      <NSkeleton v-for="n in 6" :key="n" :height="40" :width="n <= 4 ? '100%' : '60%'" style="margin-bottom: 20px" />
    </template>

    <template v-else>
      <NForm
        ref="formRef"
        :model="settings"
        :rules="rules"
        label-placement="left"
        label-width="120"
        require-mark-placement="right-hanging"
      >
        <NDivider>基本信息</NDivider>

        <NFormItem label="景区名称" path="scenic_name">
          <NInput v-model:value="settings.scenic_name" placeholder="请输入景区名称" />
        </NFormItem>

        <NFormItem label="景区简介" path="scenic_desc">
          <NInput
            v-model:value="settings.scenic_desc"
            type="textarea"
            :rows="3"
            placeholder="请输入景区简介"
          />
        </NFormItem>

        <NFormItem label="服务热线" path="service_hotline">
          <NInput v-model:value="settings.service_hotline" placeholder="请输入服务热线" />
        </NFormItem>

        <NDivider>系统功能</NDivider>

        <NFormItem label="启用用户登录">
          <NSwitch v-model:value="settings.enable_login" />
        </NFormItem>

        <NFormItem label="启用语音服务">
          <NSwitch v-model:value="settings.enable_voice" />
        </NFormItem>

        <NFormItem label="启用游客感受度分析">
          <NSwitch v-model:value="settings.enable_filter" />
        </NFormItem>

        <NDivider>数据管理</NDivider>

        <NFormItem label="数据保留天数" path="data_retention">
          <NInput v-model:value="settings.data_retention" placeholder="1-365" />
        </NFormItem>

        <NFormItem label="备份频率" path="backup_frequency">
          <NSelect
            v-model:value="settings.backup_frequency"
            :options="backupFrequencyOptions"
            placeholder="请选择备份频率"
          />
        </NFormItem>

        <div class="button-row">
          <NButton type="primary" :loading="saving" @click="saveSettings">
            保存设置
          </NButton>
          <NButton @click="loadSettings">重置</NButton>
        </div>
      </NForm>
    </template>
  </NCard>
</template>

<style scoped>
.settings-card {
  max-width: 640px;
  background: var(--sg-surface-card);
  border: 1px solid var(--sg-border-soft);
  border-radius: var(--sg-radius-xl);
}

.button-row {
  display: flex;
  gap: var(--sg-space-3);
  margin-top: var(--sg-space-6);
}
</style>
