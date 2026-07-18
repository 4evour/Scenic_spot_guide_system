<script setup lang="ts">
import { computed, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'

const props = withDefaults(
  defineProps<{
    loading?: boolean
    error?: string
  }>(),
  {
    loading: false,
    error: '',
  },
)

const emit = defineEmits<{
  submit: [payload: { file: File; message: string }]
  close: []
}>()

const { t } = useI18n()
const fileInput = ref<HTMLInputElement | null>(null)
const selectedFile = ref<File | null>(null)
const message = ref('')
const localError = ref('')
const previewUrl = ref('')

const acceptedTypes = 'image/jpeg,image/png,image/webp,audio/wav,audio/mpeg,audio/ogg,audio/webm,video/mp4,video/webm'

const fileKind = computed(() => {
  const type = selectedFile.value?.type || ''
  if (type.startsWith('image/')) return 'image'
  if (type.startsWith('audio/')) return 'audio'
  if (type.startsWith('video/')) return 'video'
  return 'file'
})

const fileKindLabel = computed(() => t(`dh.multimodal.types.${fileKind.value}`))

function maxSizeFor(type: string) {
  if (type.startsWith('image/')) return 10 * 1024 * 1024
  if (type.startsWith('audio/')) return 20 * 1024 * 1024
  if (type.startsWith('video/')) return 50 * 1024 * 1024
  return 0
}

function formatFileSize(size: number) {
  if (size < 1024 * 1024) return `${Math.max(1, Math.round(size / 1024))} KB`
  return `${(size / 1024 / 1024).toFixed(1)} MB`
}

function clearPreviewUrl() {
  if (previewUrl.value) {
    URL.revokeObjectURL(previewUrl.value)
    previewUrl.value = ''
  }
}

function selectFile(file: File | undefined) {
  clearPreviewUrl()
  selectedFile.value = null
  localError.value = ''
  if (!file) return

  const maxSize = maxSizeFor(file.type)
  if (!maxSize || file.size <= 0 || file.size > maxSize) {
    localError.value = t('dh.multimodal.fileTooLarge', { size: formatFileSize(maxSize) })
    return
  }

  selectedFile.value = file
  if (fileKind.value !== 'file') {
    previewUrl.value = URL.createObjectURL(file)
  }
}

function onFileChange(event: Event) {
  const input = event.target as HTMLInputElement
  selectFile(input.files?.[0])
  input.value = ''
}

function openFilePicker() {
  if (!props.loading) fileInput.value?.click()
}

function removeFile() {
  clearPreviewUrl()
  selectedFile.value = null
  localError.value = ''
}

function submit() {
  if (props.loading || !selectedFile.value) return
  if (localError.value) return
  emit('submit', { file: selectedFile.value, message: message.value.trim() })
}

watch(
  () => props.error,
  (value) => {
    if (value) localError.value = value
  },
)

onUnmounted(clearPreviewUrl)
</script>

<template>
  <section class="multimodal-input" aria-live="polite">
    <div class="multimodal-header">
      <div>
        <strong>{{ t('dh.multimodal.title') }}</strong>
        <span>{{ t('dh.multimodal.subtitle') }}</span>
      </div>
      <button
        type="button"
        class="multimodal-close"
        :aria-label="t('dh.multimodal.close')"
        :title="t('dh.multimodal.close')"
        @click="emit('close')"
      >
        ×
      </button>
    </div>

    <input
      ref="fileInput"
      class="visually-hidden"
      type="file"
      :accept="acceptedTypes"
      @change="onFileChange"
    />

    <button type="button" class="multimodal-picker" :disabled="loading" @click="openFilePicker">
      <span class="multimodal-picker-icon" aria-hidden="true">＋</span>
      <span>{{ selectedFile ? t('dh.multimodal.changeFile') : t('dh.multimodal.chooseFile') }}</span>
    </button>

    <div v-if="selectedFile" class="multimodal-file">
      <img v-if="fileKind === 'image' && previewUrl" class="multimodal-preview" :src="previewUrl" :alt="selectedFile.name" />
      <audio v-else-if="fileKind === 'audio' && previewUrl" class="multimodal-audio-preview" :src="previewUrl" controls />
      <video v-else-if="fileKind === 'video' && previewUrl" class="multimodal-video-preview" :src="previewUrl" controls preload="metadata" />
      <div class="multimodal-file-info">
        <strong>{{ selectedFile.name }}</strong>
        <span>{{ fileKindLabel }} · {{ formatFileSize(selectedFile.size) }}</span>
      </div>
      <button
        type="button"
        class="multimodal-remove"
        :aria-label="t('dh.multimodal.removeFile')"
        :title="t('dh.multimodal.removeFile')"
        :disabled="loading"
        @click="removeFile"
      >
        ×
      </button>
    </div>

    <textarea
      v-model="message"
      class="multimodal-message"
      rows="2"
      :placeholder="t('dh.multimodal.promptPlaceholder')"
      :disabled="loading"
    />

    <p v-if="localError" class="multimodal-error" role="alert">{{ localError }}</p>

    <div class="multimodal-actions">
      <span>{{ t('dh.multimodal.formatHint') }}</span>
      <button type="button" class="multimodal-submit" :disabled="!selectedFile || loading" @click="submit">
        {{ loading ? t('dh.multimodal.analyzing') : t('dh.multimodal.analyze') }}
      </button>
    </div>
  </section>
</template>

<style scoped>
.multimodal-input {
  margin: 0 16px 10px;
  padding: 12px;
  border: 1px solid rgba(99, 226, 183, 0.24);
  border-radius: 14px;
  background: rgba(255, 255, 255, 0.05);
}

.multimodal-header,
.multimodal-actions,
.multimodal-file {
  display: flex;
  align-items: center;
  gap: 8px;
}

.multimodal-header {
  justify-content: space-between;
  margin-bottom: 10px;
}

.multimodal-header div {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.multimodal-header strong {
  color: var(--sg-text-body, rgba(255, 255, 255, 0.9));
  font-size: 13px;
}

.multimodal-header span,
.multimodal-actions > span {
  color: var(--sg-text-faint, rgba(255, 255, 255, 0.48));
  font-size: 11px;
}

.multimodal-close,
.multimodal-remove {
  width: 28px;
  height: 28px;
  border: 1px solid rgba(255, 255, 255, 0.12);
  border-radius: 50%;
  background: transparent;
  color: inherit;
  cursor: pointer;
  font-size: 18px;
  line-height: 1;
}

.multimodal-picker {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 9px 10px;
  border: 1px dashed rgba(99, 226, 183, 0.38);
  border-radius: 9px;
  background: rgba(99, 226, 183, 0.06);
  color: var(--sg-jade-bright, #63e2b7);
  cursor: pointer;
  font-size: 12px;
  text-align: left;
}

.multimodal-picker:disabled,
.multimodal-submit:disabled,
.multimodal-remove:disabled {
  cursor: not-allowed;
  opacity: 0.55;
}

.multimodal-picker-icon {
  font-size: 18px;
}

.multimodal-file {
  min-width: 0;
  margin-top: 8px;
  padding: 7px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 9px;
}

.multimodal-preview {
  width: 42px;
  height: 42px;
  flex: 0 0 42px;
  border-radius: 6px;
  object-fit: cover;
}

.multimodal-audio-preview {
  width: min(230px, 48%);
  height: 32px;
}

.multimodal-video-preview {
  width: 96px;
  height: 54px;
  flex: 0 0 96px;
  border-radius: 6px;
  background: #000;
  object-fit: cover;
}

.multimodal-file-info {
  min-width: 0;
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 2px;
}

.multimodal-file-info strong {
  overflow: hidden;
  color: var(--sg-text-body, rgba(255, 255, 255, 0.88));
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.multimodal-file-info span {
  color: var(--sg-text-faint, rgba(255, 255, 255, 0.48));
  font-size: 11px;
}

.multimodal-message {
  box-sizing: border-box;
  width: 100%;
  min-height: 54px;
  margin-top: 8px;
  padding: 8px 10px;
  resize: vertical;
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.05);
  color: var(--sg-text-body, rgba(255, 255, 255, 0.88));
  font: inherit;
  font-size: 12px;
  outline: none;
}

.multimodal-message:focus {
  border-color: var(--sg-jade-bright, #63e2b7);
}

.multimodal-error {
  margin: 7px 0 0;
  color: #e88989;
  font-size: 11px;
}

.multimodal-actions {
  justify-content: space-between;
  margin-top: 8px;
}

.multimodal-submit {
  padding: 7px 12px;
  border: 0;
  border-radius: 8px;
  background: var(--sg-jade-bright, #63e2b7);
  color: var(--sg-bg-ink, #0a0a0f);
  cursor: pointer;
  font-size: 12px;
  font-weight: 600;
}

.visually-hidden {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
  border: 0;
}

:global(.visitor-shell) .multimodal-input {
  border-color: var(--visitor-line);
  background: rgba(255, 255, 255, 0.22);
}

:global(.visitor-shell) .multimodal-header strong,
:global(.visitor-shell) .multimodal-file-info strong,
:global(.visitor-shell) .multimodal-message {
  color: var(--visitor-ink);
}

:global(.visitor-shell) .multimodal-header span,
:global(.visitor-shell) .multimodal-actions > span,
:global(.visitor-shell) .multimodal-file-info span {
  color: var(--visitor-muted);
}

@media (max-width: 600px) {
  .multimodal-input {
    margin-right: 10px;
    margin-left: 10px;
  }

  .multimodal-actions {
    align-items: flex-end;
  }

  .multimodal-actions > span {
    max-width: 58%;
    line-height: 1.4;
  }
}
</style>
