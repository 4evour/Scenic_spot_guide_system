<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  NButton,
  NCard,
  NColorPicker,
  NDivider,
  NForm,
  NFormItem,
  NInput,
  NSelect,
  NSlider,
  NStatistic,
  NSwitch,
  useMessage,
  type FormInst,
  type FormRules,
} from 'naive-ui'
import { apiFetch } from '../services/api'
import type { AvatarConfig } from '../types/admin'
import { defaultAvatarConfig } from '../types/admin'
import type { DigitalHumanAvatarOption } from '../types/digitalHuman'

const message = useMessage()
const { t } = useI18n()

const appearanceOptions = [
  { label: t('adminAvatar.options.appearance.friendly'), value: '亲和型国风讲解员' },
  { label: t('adminAvatar.options.appearance.formal'), value: '端庄礼仪讲解员' },
  { label: t('adminAvatar.options.appearance.youthful'), value: '青春活力文旅推荐官' },
  { label: t('adminAvatar.options.appearance.zen'), value: '禅意文化讲述者' },
]
const costumeOptions = [
  { label: t('adminAvatar.options.costume.hanfu'), value: '古典汉服' },
  { label: t('adminAvatar.options.costume.uniform'), value: '景区文旅制服' },
  { label: t('adminAvatar.options.costume.landscape'), value: '山水讲解员制服' },
  { label: t('adminAvatar.options.costume.zenRobe'), value: '禅意素雅长衫' },
  { label: t('adminAvatar.options.costume.festival'), value: '节庆主题服饰' },
]
const voiceOptions = [
  { label: t('adminAvatar.options.voice.gentle'), value: '温柔自然女声' },
  { label: t('adminAvatar.options.voice.professional'), value: '沉稳专业女声' },
  { label: t('adminAvatar.options.voice.energetic'), value: '活力亲切女声' },
  { label: t('adminAvatar.options.voice.ceremonial'), value: '端庄礼仪女声' },
]
const voiceIdOptions = [
  { label: t('adminAvatar.options.voiceId.xiaoxiao'), value: 'female_xiaoxiao' },
  { label: t('adminAvatar.options.voiceId.xiaoyi'), value: 'female_xiaoyi' },
  { label: t('adminAvatar.options.voiceId.yunxi'), value: 'female_yunxi' },
  { label: t('adminAvatar.options.voiceId.yunyang'), value: 'male_yunyang' },
]
const toneOptions = [
  { label: t('adminAvatar.options.tone.warm'), value: '温暖、端庄、亲切' },
  { label: t('adminAvatar.options.tone.clear'), value: '专业、清晰、克制' },
  { label: t('adminAvatar.options.tone.lively'), value: '活泼、轻快、有陪伴感' },
  { label: t('adminAvatar.options.tone.zen'), value: '舒缓、禅意、适合文化讲解' },
]
const emotionOptions = [
  { label: t('adminAvatar.options.emotion.joy'), value: 'joy' },
  { label: t('adminAvatar.options.emotion.neutral'), value: 'neutral' },
  { label: t('adminAvatar.options.emotion.surprise'), value: 'surprise' },
  { label: t('adminAvatar.options.emotion.sadness'), value: 'sadness' },
]

const state = reactive({
  loading: false,
  saving: false,
  avatar: { ...defaultAvatarConfig } as AvatarConfig,
  avatarOptions: [] as DigitalHumanAvatarOption[],
})

const formRef = ref<FormInst | null>(null)

const avatarUpdatedNote = computed(() => `${state.avatar.costume} / ${state.avatar.voice_type}`)
const digitalHumanOptions = computed(() =>
  state.avatarOptions.map(item => ({
    label: `${item.name}（${item.id}）`,
    value: item.id,
  })),
)
const selectedDigitalHuman = computed(() =>
  state.avatarOptions.find(item => item.id === state.avatar.default_avatar_id),
)

function normalizeAvatarConfig(raw: Partial<AvatarConfig>): AvatarConfig {
  return {
    ...defaultAvatarConfig, ...raw,
    default_avatar_id: raw.default_avatar_id || defaultAvatarConfig.default_avatar_id,
    allow_avatar_switch: raw.allow_avatar_switch ?? defaultAvatarConfig.allow_avatar_switch,
    speed: Number(raw.speed ?? defaultAvatarConfig.speed),
    volume: Number(raw.volume ?? defaultAvatarConfig.volume),
    emotion_level: Number(raw.emotion_level ?? defaultAvatarConfig.emotion_level),
    voice_id: raw.voice_id || defaultAvatarConfig.voice_id,
    tts_rate: raw.tts_rate || defaultAvatarConfig.tts_rate,
  }
}

async function loadAvatarConfig() {
  state.loading = true
  try {
    const [data, options] = await Promise.all([
      apiFetch<Partial<AvatarConfig>>('/admin/digital-human/config'),
      apiFetch<DigitalHumanAvatarOption[]>('/digital-human/avatar-options'),
    ])
    state.avatar = normalizeAvatarConfig(data)
    state.avatarOptions = options
  } catch (error) {
    message.error(error instanceof Error ? error.message : t('adminAvatar.messages.loadFailed'))
  } finally {
    state.loading = false
  }
}

async function saveAvatarConfig() {
  formRef.value?.validate(async (errors) => {
    if (errors) return
    state.saving = true
    try {
      await apiFetch('/admin/digital-human/config', {
        method: 'PUT',
        body: JSON.stringify(state.avatar),
      })
      message.success(t('adminAvatar.messages.saveSuccess'))
      await loadAvatarConfig()
    } catch (error) {
      message.error(error instanceof Error ? error.message : t('adminAvatar.messages.saveFailed'))
    } finally {
      state.saving = false
    }
  })
}

const rules: FormRules = {
  name: [{ required: true, message: t('adminAvatar.messages.nameRequired'), trigger: 'blur' }],
}

onMounted(loadAvatarConfig)
</script>

<template>
  <div class="two-col">
    <NCard :title="t('adminAvatar.previewTitle')" class="avatar-preview">
      <div
        class="avatar-holo"
        :style="{ background: `radial-gradient(circle, ${state.avatar.color}, #52f0ee)` }"
      >
        {{ state.avatar.name }}
      </div>
      <h3>{{ state.avatar.appearance }}</h3>
      <p class="avatar-theme">{{ state.avatar.culture_theme }}</p>
      <ul class="avatar-summary">
        <li><span>{{ t('adminAvatar.summary.defaultAvatar') }}</span><strong>{{ selectedDigitalHuman?.name || state.avatar.default_avatar_id }}</strong></li>
        <li><span>{{ t('adminAvatar.summary.costume') }}</span><strong>{{ state.avatar.costume }}</strong></li>
        <li><span>{{ t('adminAvatar.summary.voice') }}</span><strong>{{ state.avatar.voice_type }}</strong></li>
        <li><span>{{ t('adminAvatar.summary.tone') }}</span><strong>{{ state.avatar.voice_tone }}</strong></li>
      </ul>

      <NDivider />

      <div class="avatar-stats">
        <NStatistic :label="t('adminAvatar.form.speed')" :value="state.avatar.speed" :precision="1" />
        <NStatistic :label="t('adminAvatar.form.volume')" :value="state.avatar.volume" />
        <NStatistic :label="t('adminAvatar.form.emotionLevel')" :value="state.avatar.emotion_level" />
      </div>

      <small class="hint-line">{{ t('adminAvatar.currentPlan', { plan: avatarUpdatedNote }) }}</small>
    </NCard>

    <NCard :title="t('adminAvatar.formTitle')" class="form-panel">
      <NForm
        ref="formRef"
        :model="state.avatar"
        :rules="rules"
        label-placement="left"
        label-width="100"
        require-mark-placement="right-hanging"
      >
        <NDivider title-placement="left">{{ t('adminAvatar.sections.appearance') }}</NDivider>

        <NFormItem :label="t('adminAvatar.form.defaultAvatar')" path="default_avatar_id">
          <NSelect
            v-model:value="state.avatar.default_avatar_id"
            :options="digitalHumanOptions"
            :loading="state.loading && state.avatarOptions.length === 0"
          />
        </NFormItem>

        <NFormItem :label="t('adminAvatar.form.allowSwitch')" path="allow_avatar_switch">
          <NSwitch v-model:value="state.avatar.allow_avatar_switch">
            <template #checked>{{ t('adminAvatar.switch.allow') }}</template>
            <template #unchecked>{{ t('adminAvatar.switch.locked') }}</template>
          </NSwitch>
        </NFormItem>

        <NFormItem :label="t('adminAvatar.form.name')" path="name">
          <NInput v-model:value="state.avatar.name" :placeholder="t('adminAvatar.placeholders.name')" />
        </NFormItem>

        <NFormItem :label="t('adminAvatar.form.appearance')" path="appearance">
          <NSelect v-model:value="state.avatar.appearance" :options="appearanceOptions" />
        </NFormItem>

        <NFormItem :label="t('adminAvatar.form.costume')" path="costume">
          <NSelect v-model:value="state.avatar.costume" :options="costumeOptions" />
        </NFormItem>

        <NFormItem :label="t('adminAvatar.form.color')" path="color">
          <NColorPicker v-model:value="state.avatar.color" :show-alpha="false" />
        </NFormItem>

        <NFormItem :label="t('adminAvatar.form.cultureTheme')" path="culture_theme">
          <NInput v-model:value="state.avatar.culture_theme" type="textarea" :rows="3" />
        </NFormItem>

        <NFormItem :label="t('adminAvatar.form.greeting')" path="greeting">
          <NInput v-model:value="state.avatar.greeting" type="textarea" :rows="3" />
        </NFormItem>

        <NDivider title-placement="left">{{ t('adminAvatar.sections.voice') }}</NDivider>

        <NFormItem :label="t('adminAvatar.form.voice')" path="voice_type">
          <NSelect v-model:value="state.avatar.voice_type" :options="voiceOptions" />
        </NFormItem>

        <NFormItem :label="t('adminAvatar.form.voiceId')" path="voice_id">
          <NSelect v-model:value="state.avatar.voice_id" :options="voiceIdOptions" />
        </NFormItem>

        <NFormItem :label="t('adminAvatar.form.tone')" path="voice_tone">
          <NSelect v-model:value="state.avatar.voice_tone" :options="toneOptions" />
        </NFormItem>

        <NFormItem :label="t('adminAvatar.form.speed')" path="speed">
          <NSlider
            v-model:value="state.avatar.speed"
            :min="0.6"
            :max="1.4"
            :step="0.1"
            :format-tooltip="(v: number) => `${v.toFixed(1)}x`"
          />
        </NFormItem>

        <NFormItem :label="t('adminAvatar.form.volume')" path="volume">
          <NSlider v-model:value="state.avatar.volume" :min="0" :max="100" :step="1" />
        </NFormItem>

        <NFormItem :label="t('adminAvatar.form.defaultEmotion')" path="default_emotion">
          <NSelect v-model:value="state.avatar.default_emotion" :options="emotionOptions" />
        </NFormItem>

        <NFormItem :label="t('adminAvatar.form.emotionLevel')" path="emotion_level">
          <NSlider v-model:value="state.avatar.emotion_level" :min="1" :max="5" :step="1" />
        </NFormItem>
      </NForm>

      <div class="button-row">
        <NButton
          type="primary"
          :loading="state.saving"
          :disabled="state.loading"
          @click="saveAvatarConfig"
        >
          {{ t('adminAvatar.actions.save') }}
        </NButton>
        <NButton :loading="state.loading" @click="loadAvatarConfig">
          {{ t('adminAvatar.actions.reload') }}
        </NButton>
      </div>
    </NCard>
  </div>
</template>

<style scoped>
.two-col {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
  margin-bottom: 24px;
}

.avatar-preview {
  text-align: center;
}

.avatar-holo {
  width: 120px;
  height: 120px;
  border-radius: 50%;
  margin: 0 auto 16px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 28px;
  font-weight: 700;
  color: rgba(255, 255, 255, 0.9);
  box-shadow: 0 0 40px rgba(99, 226, 183, 0.15);
}

.avatar-preview h3 {
  font-size: 15px;
  font-weight: 600;
  color: rgba(255, 255, 255, 0.88);
  margin-bottom: 8px;
}

.avatar-theme {
  font-size: 13px;
  color: rgba(255, 255, 255, 0.45);
  margin-bottom: 16px;
}

.avatar-summary {
  list-style: none;
  padding: 0;
  text-align: left;
  margin: 0;
}

.avatar-summary li {
  display: flex;
  justify-content: space-between;
  padding: 8px 0;
  border-bottom: 1px solid rgba(255, 255, 255, 0.04);
  font-size: 13px;
}

.avatar-summary span {
  color: rgba(255, 255, 255, 0.4);
}

.avatar-summary strong {
  color: rgba(255, 255, 255, 0.8);
}

.avatar-stats {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 12px;
  text-align: center;
  margin-bottom: 12px;
}

.hint-line {
  font-size: 12px;
  color: rgba(255, 255, 255, 0.3);
  margin-top: 8px;
}

.button-row {
  display: flex;
  gap: 12px;
  margin-top: 24px;
}

@media (max-width: 1200px) {
  .two-col {
    grid-template-columns: 1fr;
  }
}
</style>
