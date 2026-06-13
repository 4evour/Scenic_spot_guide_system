<script setup lang="ts">
import { ref, onErrorCaptured } from 'vue'
import { useI18n } from 'vue-i18n'
import { NResult, NButton } from 'naive-ui'

const { t } = useI18n()

const error = ref<Error | null>(null)

onErrorCaptured((err) => {
  error.value = err
  return false
})

function handleRetry() {
  error.value = null
}
</script>

<template>
  <template v-if="error">
    <div class="error-boundary">
      <NResult
        status="error"
        :title="t('error.title')"
        :description="error.message"
      >
        <template #footer>
          <NButton type="primary" @click="handleRetry">{{ t('error.retry') }}</NButton>
        </template>
      </NResult>
    </div>
  </template>
  <template v-else>
    <slot />
  </template>
</template>

<style scoped>
.error-boundary {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 400px;
  padding: 40px;
}
</style>
