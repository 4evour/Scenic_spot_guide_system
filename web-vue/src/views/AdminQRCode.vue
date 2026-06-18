<script setup lang="ts">
import { computed, h, onMounted, reactive, ref } from 'vue'
import {
  NButton,
  NCard,
  NDataTable,
  NDrawer,
  NDrawerContent,
  NEmpty,
  NForm,
  NFormItem,
  NInput,
  NSpace,
  NSpin,
  NSwitch,
  NTag,
  useMessage,
} from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { apiFetch } from '../services/api'

type QRSpot = {
  id: number
  name: string
  category: string
  qr_intro_text: string
  qr_enabled: boolean
  qr_code?: string
}

const message = useMessage()
const drawerVisible = ref(false)
const state = reactive({
  loading: false,
  generating: false,
  saving: false,
  spots: [] as QRSpot[],
  stats: { spots_with_qr: 0, cache_entries: 0, cache_ttl_min: 0 },
  editor: {
    id: 0,
    name: '',
    qr_code: '',
    qr_intro_text: '',
    qr_enabled: false,
  },
})

const enabledCount = computed(() => state.spots.filter(item => item.qr_enabled).length)

function scanURL(row: QRSpot) {
  const code = row.qr_code || `SPOT-${String(row.id).padStart(4, '0')}`
  return `${window.location.origin}/scan?id=${encodeURIComponent(code)}`
}

async function loadData() {
  state.loading = true
  try {
    const [spots, stats] = await Promise.all([
      apiFetch<QRSpot[]>('/admin/qr/spots'),
      apiFetch<typeof state.stats>('/admin/qr/stats').catch(() => state.stats),
    ])
    state.spots = spots
    state.stats = stats
  } catch (error) {
    message.error(error instanceof Error ? error.message : '二维码数据加载失败')
  } finally {
    state.loading = false
  }
}

async function generateAll() {
  state.generating = true
  try {
    const data = await apiFetch<{ generated: number }>('/admin/qr/batch-generate', { method: 'POST', body: '{}' })
    message.success(`已生成 ${data.generated || 0} 个二维码 ID。`)
    await loadData()
  } catch (error) {
    message.error(error instanceof Error ? error.message : '批量生成失败')
  } finally {
    state.generating = false
  }
}

async function copyLink(row: QRSpot) {
  try {
    await navigator.clipboard.writeText(scanURL(row))
    message.success('扫码链接已复制。')
  } catch {
    message.error('复制失败，请手动复制链接。')
  }
}

function downloadQR(row: QRSpot, format: 'png' | 'svg') {
  window.open(`/api/v1/admin/qr/spots/${row.id}/image?format=${format}`, '_blank')
}

function openEdit(row: QRSpot) {
  state.editor = {
    id: row.id,
    name: row.name,
    qr_code: row.qr_code || '',
    qr_intro_text: row.qr_intro_text || '',
    qr_enabled: row.qr_enabled,
  }
  drawerVisible.value = true
}

async function saveQRCode() {
  if (state.editor.qr_code.length > 100) {
    message.error('二维码 ID 最多 100 个字符。')
    return
  }
  state.saving = true
  try {
    await apiFetch(`/admin/qr/spots/${state.editor.id}`, {
      method: 'PUT',
      body: JSON.stringify({
        qr_code: state.editor.qr_code.trim(),
        qr_intro_text: state.editor.qr_intro_text.trim(),
        qr_enabled: state.editor.qr_enabled,
      }),
    })
    message.success('二维码配置已保存。')
    drawerVisible.value = false
    await loadData()
  } catch (error) {
    message.error(error instanceof Error ? error.message : '二维码配置保存失败')
  } finally {
    state.saving = false
  }
}

const columns: DataTableColumns<QRSpot> = [
  { title: '景点', key: 'name', width: 180, ellipsis: { tooltip: true } },
  { title: '分类', key: 'category', width: 120 },
  {
    title: '状态',
    key: 'qr_enabled',
    width: 100,
    render(row) {
      return row.qr_enabled
        ? h(NTag, { type: 'success', size: 'small', bordered: false }, { default: () => '已启用' })
        : h(NTag, { size: 'small', bordered: false }, { default: () => '未启用' })
    },
  },
  { title: '二维码 ID', key: 'qr_code', width: 140 },
  {
    title: '操作',
    key: 'actions',
    width: 320,
    render(row) {
      return h(NSpace, { size: 6 }, {
        default: () => [
          h(NButton, { size: 'small', tertiary: true, type: 'primary', onClick: () => openEdit(row) }, { default: () => '编辑' }),
          h(NButton, { size: 'small', tertiary: true, onClick: () => copyLink(row) }, { default: () => '复制链接' }),
          h(NButton, { size: 'small', tertiary: true, type: 'primary', onClick: () => downloadQR(row, 'png') }, { default: () => 'PNG' }),
          h(NButton, { size: 'small', tertiary: true, type: 'primary', onClick: () => downloadQR(row, 'svg') }, { default: () => 'SVG' }),
        ],
      })
    },
  },
]

onMounted(loadData)
</script>

<template>
  <section class="qr-admin">
    <div class="qr-header">
      <div>
        <h2>二维码管理</h2>
        <p>生成、复制和下载景点扫码导览二维码。</p>
      </div>
      <NSpace>
        <NButton :loading="state.loading" @click="loadData">刷新</NButton>
        <NButton type="primary" :loading="state.generating" @click="generateAll">批量生成</NButton>
      </NSpace>
    </div>

    <div class="qr-kpis">
      <NCard :bordered="false">已启用：{{ enabledCount }}</NCard>
      <NCard :bordered="false">有二维码：{{ state.stats.spots_with_qr }}</NCard>
      <NCard :bordered="false">缓存：{{ state.stats.cache_entries }}</NCard>
    </div>

    <NCard :bordered="false" class="qr-card">
      <NSpin :show="state.loading">
        <NDataTable
          v-if="state.spots.length > 0"
          :columns="columns"
          :data="state.spots"
          :row-key="row => row.id"
          :bordered="false"
          :scroll-x="900"
        />
        <NEmpty v-else description="暂无景点二维码配置" />
      </NSpin>
    </NCard>

    <NDrawer v-model:show="drawerVisible" :width="520" placement="right">
      <NDrawerContent :title="`编辑二维码：${state.editor.name}`" closable>
        <NForm label-placement="left" label-width="90">
          <NFormItem label="二维码 ID">
            <NInput v-model:value="state.editor.qr_code" maxlength="100" show-count placeholder="例如 SPOT-0001" />
          </NFormItem>
          <NFormItem label="启用扫码">
            <NSwitch v-model:value="state.editor.qr_enabled" />
            <span class="switch-hint">{{ state.editor.qr_enabled ? '游客可扫码进入讲解' : '扫码入口停用' }}</span>
          </NFormItem>
          <NFormItem label="讲解词">
            <NInput
              v-model:value="state.editor.qr_intro_text"
              type="textarea"
              :rows="6"
              placeholder="留空时由后端根据景点信息生成讲解"
            />
          </NFormItem>
        </NForm>
        <template #footer>
          <NSpace justify="end">
            <NButton @click="drawerVisible = false">取消</NButton>
            <NButton type="primary" :loading="state.saving" @click="saveQRCode">保存</NButton>
          </NSpace>
        </template>
      </NDrawerContent>
    </NDrawer>
  </section>
</template>

<style scoped>
.qr-admin {
  padding: 0;
}

.qr-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 20px;
}

.qr-header h2 {
  margin: 0 0 4px;
  font-size: 20px;
  color: var(--sg-text-heading);
}

.qr-header p {
  margin: 0;
  font-size: 13px;
  color: var(--sg-text-hint);
}

.qr-kpis {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 12px;
  margin-bottom: 16px;
}

.qr-card,
.qr-kpis :deep(.n-card) {
  background: var(--sg-surface-card);
  border: 1px solid var(--sg-border-soft);
  border-radius: var(--sg-radius-xl);
}

.switch-hint {
  margin-left: 10px;
  font-size: 12px;
  color: var(--sg-text-hint);
}
</style>
