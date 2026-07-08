<script setup lang="ts">
import { computed, h, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
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
  NInputNumber,
  NSpace,
  NSwitch,
  NTag,
  useDialog,
  useMessage,
  type DataTableColumns,
} from 'naive-ui'
import { apiFetch } from '../services/api'
import type { VisitorQuery } from '../types/admin'

type QueryFilter = 'all' | 'unanswered'

const message = useMessage()
const dialog = useDialog()
const { t } = useI18n()
const loading = ref(false)
const saving = ref(false)
const deletingId = ref<number | null>(null)
const filter = ref<QueryFilter>('unanswered')
const drawerVisible = ref(false)
const queries = ref<VisitorQuery[]>([])
const editor = ref<Partial<VisitorQuery>>({
  id: 0,
  query: '',
  response: '',
  spot_id: 0,
  is_answered: false,
})

const stats = computed(() => {
  const total = queries.value.length
  const unanswered = queries.value.filter(item => !item.is_answered).length
  return { total, unanswered, answered: total - unanswered }
})

const filterOptions = computed<Array<{ label: string; value: QueryFilter }>>(() => [
  { label: t('adminQueries.filters.unanswered'), value: 'unanswered' },
  { label: t('adminQueries.filters.all'), value: 'all' },
])

function formatTime(value: string) {
  if (!value) return '-'
  return new Date(value).toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}

async function loadQueries() {
  loading.value = true
  try {
    const path = filter.value === 'unanswered' ? '/queries/unanswered' : '/queries'
    queries.value = await apiFetch<VisitorQuery[]>(path)
  } catch (error) {
    message.error(error instanceof Error ? error.message : t('adminQueries.messages.loadFailed'))
  } finally {
    loading.value = false
  }
}

function openEdit(row: VisitorQuery) {
  editor.value = { ...row }
  drawerVisible.value = true
}

function closeDrawer() {
  drawerVisible.value = false
  editor.value = {
    id: 0,
    query: '',
    response: '',
    spot_id: 0,
    is_answered: false,
  }
}

async function saveQuery() {
  if (!editor.value.id) return
  const queryText = String(editor.value.query || '').trim()
  if (!queryText) {
    message.error(t('adminQueries.messages.queryRequired'))
    return
  }
  saving.value = true
  try {
    await apiFetch(`/queries/${editor.value.id}`, {
      method: 'PUT',
      body: JSON.stringify({
        query: queryText,
        response: String(editor.value.response || '').trim(),
        spot_id: Number(editor.value.spot_id || 0),
        is_answered: Boolean(editor.value.is_answered),
      }),
    })
    message.success(t('adminQueries.messages.saveSuccess'))
    closeDrawer()
    await loadQueries()
  } catch (error) {
    message.error(error instanceof Error ? error.message : t('adminQueries.messages.saveFailed'))
  } finally {
    saving.value = false
  }
}

function confirmDelete(row: VisitorQuery) {
  dialog.warning({
    title: t('adminQueries.deleteDialog.title'),
    content: t('adminQueries.deleteDialog.content'),
    positiveText: t('adminQueries.actions.delete'),
    negativeText: t('adminQueries.actions.cancel'),
    onPositiveClick: async () => {
      deletingId.value = row.id
      try {
        await apiFetch(`/queries/${row.id}`, { method: 'DELETE' })
        message.success(t('adminQueries.messages.deleteSuccess'))
        await loadQueries()
      } catch (error) {
        message.error(error instanceof Error ? error.message : t('adminQueries.messages.deleteFailed'))
      } finally {
        deletingId.value = null
      }
    },
  })
}

async function switchFilter(value: QueryFilter) {
  if (filter.value === value) return
  filter.value = value
  await loadQueries()
}

const columns: DataTableColumns<VisitorQuery> = [
  {
    title: t('adminQueries.columns.query'),
    key: 'query',
    minWidth: 220,
    ellipsis: { tooltip: true },
  },
  {
    title: t('adminQueries.columns.status'),
    key: 'is_answered',
    width: 96,
    render(row) {
      return row.is_answered
        ? h(NTag, { type: 'success', size: 'small', bordered: false }, { default: () => t('adminQueries.status.answered') })
        : h(NTag, { type: 'warning', size: 'small', bordered: false }, { default: () => t('adminQueries.status.unanswered') })
    },
  },
  {
    title: t('adminQueries.columns.spotId'),
    key: 'spot_id',
    width: 110,
    render(row) {
      return row.spot_id || '-'
    },
  },
  {
    title: t('adminQueries.columns.responsePreview'),
    key: 'response',
    minWidth: 240,
    ellipsis: { tooltip: true },
    render(row) {
      return row.response || '-'
    },
  },
  {
    title: t('adminQueries.columns.createdAt'),
    key: 'created_at',
    width: 180,
    render(row) {
      return formatTime(row.created_at)
    },
  },
  {
    title: t('adminQueries.columns.actions'),
    key: 'actions',
    width: 160,
    fixed: 'right',
    render(row) {
      return h(NSpace, { size: 'small' }, {
        default: () => [
          h(
            NButton,
            { size: 'small', quaternary: true, type: 'primary', onClick: () => openEdit(row) },
            { default: () => t('adminQueries.actions.process') },
          ),
          h(
            NButton,
            {
              size: 'small',
              quaternary: true,
              type: 'error',
              loading: deletingId.value === row.id,
              onClick: () => confirmDelete(row),
            },
            { default: () => t('adminQueries.actions.delete') },
          ),
        ],
      })
    },
  },
]

onMounted(loadQueries)
</script>

<template>
  <section class="queries-admin">
    <div class="queries-header">
      <div>
        <h2>{{ t('adminQueries.title') }}</h2>
        <p>{{ t('adminQueries.subtitle') }}</p>
      </div>
      <NSpace>
        <div class="filter-switch">
          <button
            v-for="item in filterOptions"
            :key="item.value"
            :class="{ active: filter === item.value }"
            type="button"
            @click="switchFilter(item.value)"
          >
            {{ item.label }}
          </button>
        </div>
        <NButton :loading="loading" @click="loadQueries">{{ t('adminQueries.actions.refresh') }}</NButton>
      </NSpace>
    </div>

    <div class="query-kpis">
      <NCard :bordered="false">{{ t('adminQueries.kpis.currentList', { count: stats.total }) }}</NCard>
      <NCard :bordered="false">{{ t('adminQueries.kpis.unanswered', { count: stats.unanswered }) }}</NCard>
      <NCard :bordered="false">{{ t('adminQueries.kpis.answered', { count: stats.answered }) }}</NCard>
    </div>

    <NCard :bordered="false" class="query-card">
      <NDataTable
        v-if="queries.length > 0 || loading"
        :columns="columns"
        :data="queries"
        :loading="loading"
        :row-key="row => row.id"
        :bordered="false"
        :scroll-x="1080"
      />
      <NEmpty v-else :description="t('adminQueries.empty')" class="empty-placeholder" />
    </NCard>

    <NDrawer v-model:show="drawerVisible" :width="620" placement="right">
      <NDrawerContent :title="t('adminQueries.drawerTitle')" closable>
        <NForm label-placement="top">
          <NFormItem :label="t('adminQueries.form.query')">
            <NInput
              v-model:value="editor.query"
              type="textarea"
              :rows="4"
              maxlength="1000"
              show-count
              :placeholder="t('adminQueries.placeholders.query')"
            />
          </NFormItem>
          <NFormItem :label="t('adminQueries.form.spotId')">
            <NInputNumber
              v-model:value="editor.spot_id"
              :min="0"
              :placeholder="t('adminQueries.placeholders.spotId')"
              style="width: 100%"
            />
          </NFormItem>
          <NFormItem :label="t('adminQueries.form.response')">
            <NInput
              v-model:value="editor.response"
              type="textarea"
              :rows="8"
              maxlength="4000"
              show-count
              :placeholder="t('adminQueries.placeholders.response')"
            />
          </NFormItem>
          <NFormItem :label="t('adminQueries.form.status')">
            <NSwitch v-model:value="editor.is_answered" />
            <span class="switch-hint">
              {{ editor.is_answered ? t('adminQueries.statusHints.done') : t('adminQueries.statusHints.pending') }}
            </span>
          </NFormItem>
        </NForm>
        <template #footer>
          <NSpace justify="end">
            <NButton @click="closeDrawer">{{ t('adminQueries.actions.cancel') }}</NButton>
            <NButton type="primary" :loading="saving" @click="saveQuery">
              {{ t('adminQueries.actions.save') }}
            </NButton>
          </NSpace>
        </template>
      </NDrawerContent>
    </NDrawer>
  </section>
</template>

<style scoped>
.queries-admin {
  padding: 0;
}

.queries-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
  margin-bottom: 20px;
}

.queries-header h2 {
  margin: 0 0 4px;
  font-size: 20px;
  color: var(--sg-text-heading);
}

.queries-header p {
  margin: 0;
  font-size: 13px;
  color: var(--sg-text-hint);
}

.filter-switch {
  display: flex;
  gap: 4px;
  padding: 3px;
  background: rgba(255, 255, 255, 0.04);
  border-radius: 8px;
}

.filter-switch button {
  border: 0;
  border-radius: 6px;
  padding: 6px 12px;
  background: transparent;
  color: rgba(255, 255, 255, 0.58);
  cursor: pointer;
}

.filter-switch button.active {
  background: var(--sg-jade-bright, #63e2b7);
  color: #041213;
  font-weight: 700;
}

.query-kpis {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 12px;
  margin-bottom: 16px;
}

.query-card,
.query-kpis :deep(.n-card) {
  background: var(--sg-surface-card);
  border: 1px solid var(--sg-border-soft);
  border-radius: var(--sg-radius-xl);
}

.empty-placeholder {
  padding: 64px 0;
}

.switch-hint {
  margin-left: 10px;
  font-size: 12px;
  color: var(--sg-text-hint);
}

@media (max-width: 760px) {
  .queries-header {
    flex-direction: column;
  }
}
</style>
