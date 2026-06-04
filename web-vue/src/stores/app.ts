import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useAppStore = defineStore('app', () => {
  const siderCollapsed = ref(
    localStorage.getItem('sg-sider-collapsed') === 'true',
  )

  function toggleSider() {
    siderCollapsed.value = !siderCollapsed.value
    localStorage.setItem('sg-sider-collapsed', String(siderCollapsed.value))
  }

  return { siderCollapsed, toggleSider }
})
