<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
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
  useMessage,
  type FormInst,
  type FormRules,
} from 'naive-ui'
import { apiFetch } from '../services/api'
import type { AvatarConfig } from '../types/admin'
import { defaultAvatarConfig } from '../types/admin'

const message = useMessage()

const appearanceOptions = [
  { label: '亲和型国风讲解员', value: '亲和型国风讲解员' },
  { label: '端庄礼仪讲解员', value: '端庄礼仪讲解员' },
  { label: '青春活力文旅推荐官', value: '青春活力文旅推荐官' },
  { label: '禅意文化讲述者', value: '禅意文化讲述者' },
]
const costumeOptions = [
  { label: '古典汉服', value: '古典汉服' },
  { label: '景区文旅制服', value: '景区文旅制服' },
  { label: '山水讲解员制服', value: '山水讲解员制服' },
  { label: '禅意素雅长衫', value: '禅意素雅长衫' },
  { label: '节庆主题服饰', value: '节庆主题服饰' },
]
const voiceOptions = [
  { label: '温柔自然女声', value: '温柔自然女声' },
  { label: '沉稳专业女声', value: '沉稳专业女声' },
  { label: '活力亲切女声', value: '活力亲切女声' },
  { label: '端庄礼仪女声', value: '端庄礼仪女声' },
]
const toneOptions = [
  { label: '温暖、端庄、亲切', value: '温暖、端庄、亲切' },
  { label: '专业、清晰、克制', value: '专业、清晰、克制' },
  { label: '活泼、轻快、有陪伴感', value: '活泼、轻快、有陪伴感' },
  { label: '舒缓、禅意、适合文化讲解', value: '舒缓、禅意、适合文化讲解' },
]
const emotionOptions = [
  { label: '亲切微笑', value: 'joy' },
  { label: '自然平和', value: 'neutral' },
  { label: '热情提示', value: 'surprise' },
  { label: '温和致歉', value: 'sadness' },
]

const state = reactive({
  loading: false,
  saving: false,
  avatar: { ...defaultAvatarConfig } as AvatarConfig,
})

const formRef = ref<FormInst | null>(null)

const avatarUpdatedNote = computed(() => `${state.avatar.costume} / ${state.avatar.voice_type}`)

function normalizeAvatarConfig(raw: Partial<AvatarConfig>): AvatarConfig {
  return {
    ...defaultAvatarConfig, ...raw,
    speed: Number(raw.speed ?? defaultAvatarConfig.speed),
    volume: Number(raw.volume ?? defaultAvatarConfig.volume),
    emotion_level: Number(raw.emotion_level ?? defaultAvatarConfig.emotion_level),
  }
}

async function loadAvatarConfig() {
  state.loading = true
  try {
    const data = await apiFetch<Partial<AvatarConfig>>('/admin/digital-human/config')
    state.avatar = normalizeAvatarConfig(data)
  } catch (error) {
    message.error(error instanceof Error ? error.message : '数字人配置加载失败')
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
      message.success('数字人形象配置已保存。')
      await loadAvatarConfig()
    } catch (error) {
      message.error(error instanceof Error ? error.message : '保存失败')
    } finally {
      state.saving = false
    }
  })
}

const rules: FormRules = {
  name: [{ required: true, message: '请输入数字人名称', trigger: 'blur' }],
}

onMounted(loadAvatarConfig)
</script>

<template>
  <div class="two-col">
    <NCard title="数字人预览" class="avatar-preview">
      <div
        class="avatar-holo"
        :style="{ background: `radial-gradient(circle, ${state.avatar.color}, #52f0ee)` }"
      >
        {{ state.avatar.name }}
      </div>
      <h3>{{ state.avatar.appearance }}</h3>
      <p class="avatar-theme">{{ state.avatar.culture_theme }}</p>
      <ul class="avatar-summary">
        <li><span>服装</span><strong>{{ state.avatar.costume }}</strong></li>
        <li><span>声音</span><strong>{{ state.avatar.voice_type }}</strong></li>
        <li><span>语气</span><strong>{{ state.avatar.voice_tone }}</strong></li>
      </ul>

      <NDivider />

      <div class="avatar-stats">
        <NStatistic label="语速" :value="state.avatar.speed" :precision="1" />
        <NStatistic label="音量" :value="state.avatar.volume" />
        <NStatistic label="表情强度" :value="state.avatar.emotion_level" />
      </div>

      <small class="hint-line">当前方案：{{ avatarUpdatedNote }}</small>
    </NCard>

    <NCard title="形象与声音设定" class="form-panel">
      <NForm
        ref="formRef"
        :model="state.avatar"
        :rules="rules"
        label-placement="left"
        label-width="100"
        require-mark-placement="right-hanging"
      >
        <NDivider title-placement="left">形象设定</NDivider>

        <NFormItem label="数字人名称" path="name">
          <NInput v-model:value="state.avatar.name" placeholder="请输入数字人名称" />
        </NFormItem>

        <NFormItem label="外观定位" path="appearance">
          <NSelect v-model:value="state.avatar.appearance" :options="appearanceOptions" />
        </NFormItem>

        <NFormItem label="服装风格" path="costume">
          <NSelect v-model:value="state.avatar.costume" :options="costumeOptions" />
        </NFormItem>

        <NFormItem label="主视觉颜色" path="color">
          <NColorPicker v-model:value="state.avatar.color" :show-alpha="false" />
        </NFormItem>

        <NFormItem label="景区文化主题" path="culture_theme">
          <NInput v-model:value="state.avatar.culture_theme" type="textarea" :rows="3" />
        </NFormItem>

        <NFormItem label="欢迎语" path="greeting">
          <NInput v-model:value="state.avatar.greeting" type="textarea" :rows="3" />
        </NFormItem>

        <NDivider title-placement="left">声音与表达</NDivider>

        <NFormItem label="讲解声音" path="voice_type">
          <NSelect v-model:value="state.avatar.voice_type" :options="voiceOptions" />
        </NFormItem>

        <NFormItem label="讲解语气" path="voice_tone">
          <NSelect v-model:value="state.avatar.voice_tone" :options="toneOptions" />
        </NFormItem>

        <NFormItem label="语速" path="speed">
          <NSlider
            v-model:value="state.avatar.speed"
            :min="0.6"
            :max="1.4"
            :step="0.1"
            :format-tooltip="(v: number) => `${v.toFixed(1)}x`"
          />
        </NFormItem>

        <NFormItem label="音量" path="volume">
          <NSlider v-model:value="state.avatar.volume" :min="0" :max="100" :step="1" />
        </NFormItem>

        <NFormItem label="默认表情" path="default_emotion">
          <NSelect v-model:value="state.avatar.default_emotion" :options="emotionOptions" />
        </NFormItem>

        <NFormItem label="表情强度" path="emotion_level">
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
          保存配置
        </NButton>
        <NButton :loading="state.loading" @click="loadAvatarConfig">
          重新加载
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
