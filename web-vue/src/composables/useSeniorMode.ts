import { computed, ref, watch } from 'vue'

const SENIOR_MODE_KEY = 'sg_senior_mode'
const seniorModeEnabled = ref(localStorage.getItem(SENIOR_MODE_KEY) === 'true')

watch(seniorModeEnabled, (enabled) => {
  localStorage.setItem(SENIOR_MODE_KEY, String(enabled))
})

export function useSeniorMode() {
  const ttsRate = computed(() => (seniorModeEnabled.value ? '-20%' : '+0%'))

  function toggleSeniorMode() {
    seniorModeEnabled.value = !seniorModeEnabled.value
  }

  function setSeniorMode(enabled: boolean) {
    seniorModeEnabled.value = enabled
  }

  return {
    seniorModeEnabled,
    ttsRate,
    toggleSeniorMode,
    setSeniorMode,
  }
}

export function getBrowserSpeechRate() {
  return localStorage.getItem(SENIOR_MODE_KEY) === 'true' ? 0.75 : 0.95
}
