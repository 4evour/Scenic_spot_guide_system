<script setup lang="ts">
import { computed, ref, reactive, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
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

const { t } = useI18n()
const message = useMessage()

const loading = ref(false)
const saving = ref(false)

const settings = reactive<SystemSettings>({ ...defaultSettings })

const formRef = ref<FormInst | null>(null)

const backupFrequencyOptions = computed(() => [
  { label: t('adminSettings.backupFrequency.daily'), value: '每日' },
  { label: t('adminSettings.backupFrequency.weekly'), value: '每周' },
  { label: t('adminSettings.backupFrequency.monthly'), value: '每月' },
  { label: t('adminSettings.backupFrequency.manual'), value: '手动' },
])

const rules = computed<FormRules>(() => ({
  scenic_name: [{ required: true, message: t('adminSettings.validation.scenicNameRequired'), trigger: 'blur' }],
  data_retention: [
    { required: true, message: t('adminSettings.validation.dataRetentionRequired'), trigger: 'blur' },
    {
      validator: (_rule, value: string) => {
        const num = Number(value)
        if (Number.isNaN(num) || num < 1 || num > 365) {
          return new Error(t('adminSettings.validation.dataRetentionRange'))
        }
        return true
      },
      trigger: 'blur',
    },
  ],
}))

async function loadSettings() {
  loading.value = true
  try {
    const data = await apiFetch<Partial<SystemSettings>>('/admin/settings')
    Object.assign(settings, { ...defaultSettings, ...data })
  } catch (error) {
    message.error(error instanceof Error ? error.message : t('adminSettings.messages.loadFailed'))
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
    message.success(t('adminSettings.messages.saveSuccess'))
  } catch (error) {
    message.error(error instanceof Error ? error.message : t('adminSettings.messages.saveFailed'))
  } finally {
    saving.value = false
  }
}

onMounted(loadSettings)
</script>

<template>
  <NCard :title="t('adminSettings.title')" :bordered="false" class="settings-card">
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
        <NDivider>{{ t('adminSettings.sections.basic') }}</NDivider>

        <NFormItem :label="t('adminSettings.form.scenicName')" path="scenic_name">
          <NInput v-model:value="settings.scenic_name" :placeholder="t('adminSettings.placeholders.scenicName')" />
        </NFormItem>

        <NFormItem :label="t('adminSettings.form.scenicDesc')" path="scenic_desc">
          <NInput
            v-model:value="settings.scenic_desc"
            type="textarea"
            :rows="3"
            :placeholder="t('adminSettings.placeholders.scenicDesc')"
          />
        </NFormItem>

        <NFormItem :label="t('adminSettings.form.serviceHotline')" path="service_hotline">
          <NInput v-model:value="settings.service_hotline" :placeholder="t('adminSettings.placeholders.serviceHotline')" />
        </NFormItem>

        <NDivider>{{ t('adminSettings.sections.features') }}</NDivider>

        <NFormItem :label="t('adminSettings.form.enableLogin')">
          <NSwitch v-model:value="settings.enable_login" />
        </NFormItem>

        <NFormItem :label="t('adminSettings.form.enableVoice')">
          <NSwitch v-model:value="settings.enable_voice" />
        </NFormItem>

        <NFormItem :label="t('adminSettings.form.enableFilter')">
          <NSwitch v-model:value="settings.enable_filter" />
        </NFormItem>

        <NDivider>{{ t('adminSettings.sections.data') }}</NDivider>

        <NFormItem :label="t('adminSettings.form.dataRetention')" path="data_retention">
          <NInput v-model:value="settings.data_retention" placeholder="1-365" />
        </NFormItem>

        <NFormItem :label="t('adminSettings.form.backupFrequency')" path="backup_frequency">
          <NSelect
            v-model:value="settings.backup_frequency"
            :options="backupFrequencyOptions"
            :placeholder="t('adminSettings.placeholders.backupFrequency')"
          />
        </NFormItem>

        <div class="button-row">
          <NButton type="primary" :loading="saving" @click="saveSettings">
            {{ t('adminSettings.actions.save') }}
          </NButton>
          <NButton @click="loadSettings">{{ t('adminSettings.actions.reset') }}</NButton>
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
