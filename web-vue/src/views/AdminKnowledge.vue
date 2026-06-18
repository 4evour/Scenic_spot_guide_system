<script setup lang="ts">
import { computed, reactive, ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import type { FormInst, FormRules } from 'naive-ui'
import {
  NButton,
  NCard,
  NEmpty,
  NForm,
  NFormItem,
  NGrid,
  NGi,
  NInput,
  NPagination,
  NSelect,
  NSpace,
  NSpin,
  NTag,
  useDialog,
  useMessage,
} from 'naive-ui'
import KpiCard from '../components/KpiCard.vue'
import { apiFetch } from '../services/api'
import type { KnowledgeCandidate, KnowledgeItem, VisitorInsightAnalysis } from '../types/admin'

const { t, locale } = useI18n()

const categoryOptions = computed(() => [
  { label: t('adminKnowledge.categories.all'), value: '' },
  { label: t('adminKnowledge.categories.guide'), value: '讲解词' },
  { label: t('adminKnowledge.categories.history'), value: '文史资料' },
  { label: t('adminKnowledge.categories.faq'), value: '游客 FAQ' },
  { label: t('adminKnowledge.categories.route'), value: '路线推荐' },
  { label: t('adminKnowledge.categories.service'), value: '服务设施' },
  { label: t('adminKnowledge.categories.ticket'), value: '票务交通' },
  { label: t('adminKnowledge.categories.official'), value: 'official' },
  { label: t('adminKnowledge.categories.government'), value: 'government' },
  { label: t('adminKnowledge.categories.overview'), value: 'overview' },
  { label: t('adminKnowledge.categories.spot'), value: 'spot' },
  { label: t('adminKnowledge.categories.boundary'), value: 'boundary' },
])

const emptyEditor = { title: '', category: '讲解词', source: 'admin', content: '' }
const uploadCategoryOptions = computed(() => categoryOptions.value.filter(option => option.value))
const spotCategoryOptions = computed(() => [
  { label: t('adminKnowledge.spotCategories.all'), value: '' },
  { label: t('adminKnowledge.spotCategories.core'), value: '核心景点' },
  { label: t('adminKnowledge.spotCategories.performance'), value: '演艺体验' },
  { label: t('adminKnowledge.spotCategories.culture'), value: '文化建筑' },
  { label: t('adminKnowledge.spotCategories.service'), value: '服务设施' },
])

const message = useMessage()
const dialog = useDialog()

const formRef = ref<FormInst | null>(null)

const formRules = computed<FormRules>(() => ({
  content: [{ required: true, message: t('adminKnowledge.validation.contentRequired'), trigger: 'blur' }],
}))

const state = reactive({
  search: '',
  category: '',
  spotCategory: '',
  spotId: 0,
  spots: [] as Array<{ label: string; value: number; category: string }>,
  loading: false,
  saving: false,
  total: 0,
  page: 1,
  pageSize: 20,
  knowledge: [] as KnowledgeItem[],
  editingID: '',
  editor: { ...emptyEditor },
  uploadCategory: '文史资料',
  selectedFile: null as File | null,
  analysisSessionId: '',
  analyzing: false,
  analysesLoading: false,
  insightAnalyses: [] as VisitorInsightAnalysis[],
  candidatesLoading: false,
  candidates: [] as KnowledgeCandidate[],
})

const displayedCount = computed(() => state.knowledge.length)
const latestAnalysis = computed(() => state.insightAnalyses[0])
const spotOptions = computed(() => [
  { label: t('adminKnowledge.spots.all'), value: 0 },
  ...state.spots
    .filter(spot => !state.spotCategory || spot.category === state.spotCategory)
    .map(spot => ({ label: spot.label, value: spot.value })),
])

function categoryLabel(value: string) {
  if (value === '讲解词') return t('adminKnowledge.categories.guide')
  if (value === '文史资料') return t('adminKnowledge.categories.history')
  if (value === '游客 FAQ') return t('adminKnowledge.categories.faq')
  if (value === '路线推荐') return t('adminKnowledge.categories.route')
  if (value === '服务设施') return t('adminKnowledge.categories.service')
  if (value === '票务交通') return t('adminKnowledge.categories.ticket')
  if (value === 'official') return t('adminKnowledge.categories.official')
  if (value === 'government') return t('adminKnowledge.categories.government')
  if (value === 'overview') return t('adminKnowledge.categories.overview')
  if (value === 'spot') return t('adminKnowledge.categories.spot')
  if (value === 'boundary') return t('adminKnowledge.categories.boundary')
  if (value === '未分类') return t('adminKnowledge.categories.uncategorized')
  return value
}

function spotCategoryLabel(value: string) {
  if (value === '核心景点') return t('adminKnowledge.spotCategories.core')
  if (value === '演艺体验') return t('adminKnowledge.spotCategories.performance')
  if (value === '文化建筑') return t('adminKnowledge.spotCategories.culture')
  if (value === '服务设施') return t('adminKnowledge.spotCategories.service')
  return value
}

function getField(item: Record<string, unknown>, key: string) {
  return item[key] ?? item[key.charAt(0).toUpperCase() + key.slice(1)] ?? ''
}

function getCategory(metadata: string, source: string) {
  if (!metadata) return source || '未分类'
  try {
    const parsed = JSON.parse(metadata)
    return parsed.category || parsed.topic || parsed.source_type || parsed.type || parsed.domain || parsed.filename || source || '未分类'
  } catch {
    return source || '未分类'
  }
}

function normalizeKnowledge(raw: Record<string, unknown>): KnowledgeItem {
  const metadata = String(getField(raw, 'metadata') || '')
  const source = String(getField(raw, 'source') || 'admin')
  const updatedAt = String(getField(raw, 'updatedAt') || getField(raw, 'UpdatedAt') || '')
  return {
    id: String(getField(raw, 'id') || getField(raw, 'ID')),
    title: String(getField(raw, 'title')),
    source,
    content: String(getField(raw, 'content')),
    category: getCategory(metadata, source),
    knowledge_category: String(getField(raw, 'knowledge_category') || getCategory(metadata, source)),
    spot_id: Number(getField(raw, 'spot_id') || 0),
    spot_category: String(getField(raw, 'spot_category') || ''),
    metadata,
    updated: updatedAt ? new Date(updatedAt).toLocaleDateString(locale.value) : '-',
  }
}

function parseJSONList(value: string) {
  if (!value) return [] as string[]
  try {
    const parsed = JSON.parse(value)
    return Array.isArray(parsed) ? parsed.map(item => String(item)).filter(Boolean) : []
  } catch {
    return []
  }
}

function formatAnalysisTime(value: string) {
  if (!value) return '-'
  return new Date(value).toLocaleString(locale.value, {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}

async function loadSpots() {
  try {
    const spots = await apiFetch<Array<Record<string, unknown>>>('/spots')
    state.spots = spots.map(raw => ({
      label: String(getField(raw, 'name') || t('adminKnowledge.spots.fallback', { id: getField(raw, 'id') || '' })),
      value: Number(getField(raw, 'id') || 0),
      category: String(getField(raw, 'category') || ''),
    })).filter(item => item.value > 0)
  } catch {
    state.spots = []
  }
}

async function loadKnowledge() {
  state.loading = true
  try {
    const params = new URLSearchParams({
      page: String(state.page),
      page_size: String(state.pageSize),
    })
    if (state.search.trim()) params.set('keyword', state.search.trim())
    if (state.category) params.set('knowledge_category', state.category)
    if (state.spotCategory) params.set('spot_category', state.spotCategory)
    if (state.spotId) params.set('spot_id', String(state.spotId))
    const data = await apiFetch<{ list?: Array<Record<string, unknown>>; total?: number }>(`/knowledge/list?${params.toString()}`)
    state.knowledge = (data.list || []).map(normalizeKnowledge)
    state.total = Number(data.total || state.knowledge.length)
  } catch (error) {
    message.error(error instanceof Error ? error.message : t('adminKnowledge.messages.loadFailed'))
  } finally {
    state.loading = false
  }
}

function reloadFromFirstPage() {
  state.page = 1
  void loadKnowledge()
}

function resetEditor() {
  state.editingID = ''
  state.editor = { ...emptyEditor }
}

function editKnowledge(item: KnowledgeItem) {
  state.editingID = item.id
  state.editor = { title: item.title, category: item.knowledge_category || item.category, source: item.source, content: item.content }
}

function saveKnowledge() {
  formRef.value?.validate(async (errors) => {
    if (errors) return
    state.saving = true
    try {
      const body = JSON.stringify(state.editor)
      const path = state.editingID ? `/knowledge/${encodeURIComponent(state.editingID)}` : '/knowledge'
      const method = state.editingID ? 'PUT' : 'POST'
      await apiFetch(path, { method, body })
      message.success(state.editingID ? t('adminKnowledge.messages.updateSuccess') : t('adminKnowledge.messages.createSuccess'))
      resetEditor()
      await loadKnowledge()
    } catch (error) {
      message.error(error instanceof Error ? error.message : t('adminKnowledge.messages.saveFailed'))
    } finally {
      state.saving = false
    }
  })
}

function deleteKnowledge(item: KnowledgeItem) {
  dialog.warning({
    title: t('adminKnowledge.dialog.deleteTitle'),
    content: t('adminKnowledge.dialog.deleteContent', { title: item.title }),
    positiveText: t('adminKnowledge.actions.delete'),
    negativeText: t('adminKnowledge.actions.cancel'),
    onPositiveClick: async () => {
      try {
        await apiFetch(`/knowledge/${encodeURIComponent(item.id)}`, { method: 'DELETE' })
        message.success(t('adminKnowledge.messages.deleteSuccess'))
        await loadKnowledge()
      } catch (error) {
        message.error(error instanceof Error ? error.message : t('adminKnowledge.messages.deleteFailed'))
      }
    },
  })
}

function onFileChange(event: Event) {
  state.selectedFile = (event.target as HTMLInputElement).files?.[0] ?? null
}

async function uploadKnowledge() {
  if (!state.selectedFile) {
    message.warning(t('adminKnowledge.messages.chooseFile'))
    return
  }
  state.saving = true
  try {
    const form = new FormData()
    form.append('file', state.selectedFile)
    form.append('category', state.uploadCategory)
    const data = await apiFetch<{ loaded_count?: number }>('/knowledge/upload', { method: 'POST', body: form })
    message.success(t('adminKnowledge.messages.uploadSuccess', { count: data.loaded_count || 0 }))
    state.selectedFile = null
    await loadKnowledge()
  } catch (error) {
    message.error(error instanceof Error ? error.message : t('adminKnowledge.messages.uploadFailed'))
  } finally {
    state.saving = false
  }
}

async function analyzeSession() {
  if (!state.analysisSessionId.trim()) {
    message.warning(t('adminKnowledge.messages.sessionRequired'))
    return
  }
  state.analyzing = true
  try {
    await apiFetch(`/admin/insights/sessions/${encodeURIComponent(state.analysisSessionId.trim())}/analyze`, { method: 'POST', body: '{}' })
    message.success(t('adminKnowledge.messages.analyzeSuccess'))
    await Promise.all([loadInsightAnalyses(), loadCandidates()])
  } catch (error) {
    message.error(error instanceof Error ? error.message : t('adminKnowledge.messages.analyzeFailed'))
  } finally {
    state.analyzing = false
  }
}

async function loadInsightAnalyses() {
  state.analysesLoading = true
  try {
    const data = await apiFetch<{ list?: VisitorInsightAnalysis[] }>('/admin/insights/analyses?page=1&page_size=5')
    state.insightAnalyses = data.list || []
  } catch (error) {
    message.error(error instanceof Error ? error.message : t('adminKnowledge.messages.analysesLoadFailed'))
  } finally {
    state.analysesLoading = false
  }
}

async function loadCandidates() {
  state.candidatesLoading = true
  try {
    const data = await apiFetch<{ list?: KnowledgeCandidate[] }>('/admin/knowledge/candidates?status=pending&page_size=20')
    state.candidates = data.list || []
  } catch (error) {
    message.error(error instanceof Error ? error.message : t('adminKnowledge.messages.candidatesLoadFailed'))
  } finally {
    state.candidatesLoading = false
  }
}

async function approveCandidate(candidate: KnowledgeCandidate) {
  try {
    await apiFetch(`/admin/knowledge/candidates/${candidate.id}/approve`, {
      method: 'POST',
      body: JSON.stringify({
        title: candidate.title,
        content: candidate.content,
        knowledge_category: candidate.knowledge_category || '游客 FAQ',
        spot_id: candidate.spot_id,
        spot_category: candidate.spot_category,
      }),
    })
    message.success(t('adminKnowledge.messages.approveSuccess'))
    await Promise.all([loadCandidates(), loadKnowledge()])
  } catch (error) {
    message.error(error instanceof Error ? error.message : t('adminKnowledge.messages.approveFailed'))
  }
}

async function rejectCandidate(candidate: KnowledgeCandidate) {
  try {
    await apiFetch(`/admin/knowledge/candidates/${candidate.id}/reject`, { method: 'POST', body: JSON.stringify({ reason: '管理员拒绝' }) })
    message.success(t('adminKnowledge.messages.rejectSuccess'))
    await loadCandidates()
  } catch (error) {
    message.error(error instanceof Error ? error.message : t('adminKnowledge.messages.rejectFailed'))
  }
}

onMounted(async () => {
  await Promise.all([loadSpots(), loadKnowledge(), loadInsightAnalyses(), loadCandidates()])
})
</script>

<template>
  <section class="kpi-row">
    <KpiCard :label="t('adminKnowledge.kpis.items.label')" :value="String(state.total)" :note="t('adminKnowledge.kpis.items.note')" />
    <KpiCard :label="t('adminKnowledge.kpis.current.label')" :value="String(displayedCount)" :note="t('adminKnowledge.kpis.current.note')" tone="green" />
    <KpiCard :label="t('adminKnowledge.kpis.formats.label')" value="JSONL/MD" :note="t('adminKnowledge.kpis.formats.note')" tone="gold" />
    <KpiCard :label="t('adminKnowledge.kpis.cache.label')" :value="t('adminKnowledge.kpis.cache.value')" :note="t('adminKnowledge.kpis.cache.note')" />
  </section>

  <NGrid :cols="2" :x-gap="16" :y-gap="16" responsive="screen" item-responsive>
    <NGi span="2 m:1">
      <NCard :title="state.editingID ? t('adminKnowledge.editor.editTitle') : t('adminKnowledge.editor.createTitle')" :bordered="false" class="sg-card">
        <NForm ref="formRef" :model="state.editor" :rules="formRules" label-placement="left" label-width="70" require-mark-placement="right-hanging">
          <NFormItem :label="t('adminKnowledge.form.title')" path="title">
            <NInput v-model:value="state.editor.title" :placeholder="t('adminKnowledge.placeholders.title')" />
          </NFormItem>
          <NFormItem :label="t('adminKnowledge.form.category')" path="category">
            <NSelect v-model:value="state.editor.category" :options="uploadCategoryOptions" />
          </NFormItem>
          <NFormItem :label="t('adminKnowledge.form.source')" path="source">
            <NInput v-model:value="state.editor.source" :placeholder="t('adminKnowledge.placeholders.source')" />
          </NFormItem>
          <NFormItem :label="t('adminKnowledge.form.content')" path="content">
            <NInput v-model:value="state.editor.content" type="textarea" :rows="8" :placeholder="t('adminKnowledge.placeholders.content')" />
          </NFormItem>
          <NSpace>
            <NButton type="primary" :loading="state.saving" :disabled="!state.editor.content.trim()" @click="saveKnowledge">
              {{ state.editingID ? t('adminKnowledge.actions.save') : t('adminKnowledge.actions.add') }}
            </NButton>
            <NButton @click="resetEditor">{{ t('adminKnowledge.actions.clear') }}</NButton>
          </NSpace>
        </NForm>
      </NCard>
    </NGi>

    <NGi span="2 m:1">
      <NCard :title="t('adminKnowledge.upload.title')" :bordered="false" class="sg-card">
        <NSpace vertical :size="12">
          <NSpace align="center" :wrap="false">
            <input type="file" accept=".jsonl,.json,.md,.markdown,.txt" @change="onFileChange" />
            <NSelect v-model:value="state.uploadCategory" :options="uploadCategoryOptions" style="width: 160px" />
            <NButton type="primary" :loading="state.saving" :disabled="!state.selectedFile" @click="uploadKnowledge">
              {{ t('adminKnowledge.upload.submit') }}
            </NButton>
          </NSpace>
          <p class="hint-line">{{ t('adminKnowledge.upload.hint') }}</p>
        </NSpace>
      </NCard>
    </NGi>
  </NGrid>

  <NCard :bordered="false" class="sg-card" style="margin-top: 16px">
    <template #header-extra>
      <NSpace align="center" :size="12">
        <NSelect v-model:value="state.category" :options="categoryOptions" style="width: 160px" @update:value="reloadFromFirstPage" />
        <NSelect v-model:value="state.spotCategory" :options="spotCategoryOptions" style="width: 160px" @update:value="() => { state.spotId = 0; reloadFromFirstPage() }" />
        <NSelect v-model:value="state.spotId" :options="spotOptions" style="width: 160px" @update:value="reloadFromFirstPage" />
        <NInput
          v-model:value="state.search"
          :placeholder="t('adminKnowledge.filters.searchPlaceholder')"
          clearable
          style="width: 280px"
          @keydown.enter="reloadFromFirstPage"
          @clear="reloadFromFirstPage"
        />
        <NButton :loading="state.loading" @click="reloadFromFirstPage">{{ t('adminKnowledge.actions.search') }}</NButton>
        <NButton :loading="state.loading" @click="loadKnowledge">{{ t('adminKnowledge.actions.refresh') }}</NButton>
      </NSpace>
    </template>

    <template #default>
      <NSpin :show="state.loading">
        <div v-if="state.loading" class="spin-placeholder" />
        <div v-else-if="state.knowledge.length === 0">
          <NEmpty :description="t('adminKnowledge.empty.knowledge')" />
        </div>
        <div v-else class="knowledge-grid">
          <article v-for="item in state.knowledge" :key="item.id" class="knowledge-card">
            <div class="knowledge-card-header">
              <h3>{{ item.title }}</h3>
              <NSpace :size="6">
                <NTag size="small" :bordered="false" type="success">{{ categoryLabel(item.knowledge_category || item.category) }}</NTag>
                <NTag v-if="item.spot_category" size="small" :bordered="false">{{ spotCategoryLabel(item.spot_category) }}</NTag>
              </NSpace>
            </div>
            <p class="knowledge-preview">{{ item.content }}</p>
            <div class="knowledge-card-footer">
              <small>{{ t('adminKnowledge.labels.source', { source: item.source, updated: item.updated }) }}</small>
              <NSpace :size="6">
                <NButton size="small" quaternary type="primary" @click="editKnowledge(item)">{{ t('adminKnowledge.actions.edit') }}</NButton>
                <NButton size="small" quaternary type="error" @click="deleteKnowledge(item)">{{ t('adminKnowledge.actions.delete') }}</NButton>
              </NSpace>
            </div>
          </article>
        </div>
        <div v-if="state.total > state.pageSize" class="knowledge-pagination">
          <NPagination
            v-model:page="state.page"
            v-model:page-size="state.pageSize"
            :item-count="state.total"
            :page-sizes="[20, 50, 100]"
            show-size-picker
            @update:page="loadKnowledge"
            @update:page-size="reloadFromFirstPage"
          />
        </div>
      </NSpin>
    </template>
  </NCard>

  <NGrid :cols="2" :x-gap="16" :y-gap="16" responsive="screen" item-responsive style="margin-top: 16px">
    <NGi span="2 m:1">
      <NCard :title="t('adminKnowledge.analysis.title')" :bordered="false" class="sg-card">
        <template #header-extra>
          <NSpace align="center">
            <NInput v-model:value="state.analysisSessionId" :placeholder="t('adminKnowledge.analysis.sessionPlaceholder')" style="width: 220px" />
            <NButton type="primary" :loading="state.analyzing" @click="analyzeSession">{{ t('adminKnowledge.analysis.analyze') }}</NButton>
            <NButton :loading="state.analysesLoading" @click="loadInsightAnalyses">{{ t('adminKnowledge.analysis.refresh') }}</NButton>
          </NSpace>
        </template>
        <NSpin :show="state.analysesLoading">
          <NEmpty v-if="state.insightAnalyses.length === 0" :description="t('adminKnowledge.empty.analyses')" />
          <div v-else class="analysis-list">
            <article v-for="analysis in state.insightAnalyses" :key="analysis.id" class="analysis-card">
              <header>
                <strong>{{ analysis.session_id }}</strong>
                <NTag size="small" type="success" :bordered="false">{{ t('adminKnowledge.analysis.satisfaction', { score: analysis.satisfaction_score }) }}</NTag>
              </header>
              <p>{{ analysis.summary || t('adminKnowledge.analysis.emptySummary') }}</p>
              <div class="analysis-tags">
                <NTag v-for="item in parseJSONList(analysis.attention_points)" :key="`a-${analysis.id}-${item}`" size="small" :bordered="false">
                  {{ t('adminKnowledge.analysis.attentionPoint', { item }) }}
                </NTag>
                <NTag v-for="item in parseJSONList(analysis.negative_reasons)" :key="`n-${analysis.id}-${item}`" size="small" type="error" :bordered="false">
                  {{ t('adminKnowledge.analysis.negativeReason', { item }) }}
                </NTag>
              </div>
              <small>{{ formatAnalysisTime(analysis.created_at) }}</small>
            </article>
          </div>
        </NSpin>
      </NCard>
    </NGi>

    <NGi span="2 m:1">
      <NCard :title="t('adminKnowledge.candidates.title')" :bordered="false" class="sg-card">
        <template #header-extra>
          <NSpace align="center">
            <NTag v-if="latestAnalysis" size="small" :bordered="false">{{ t('adminKnowledge.candidates.latestAnalysis', { session: latestAnalysis.session_id }) }}</NTag>
            <NButton :loading="state.candidatesLoading" @click="loadCandidates">{{ t('adminKnowledge.candidates.refresh') }}</NButton>
          </NSpace>
        </template>
        <NSpin :show="state.candidatesLoading">
          <div v-if="state.candidates.length === 0">
            <NEmpty :description="t('adminKnowledge.empty.candidates')" />
          </div>
          <div v-else class="knowledge-grid compact">
            <article v-for="candidate in state.candidates" :key="candidate.id" class="knowledge-card">
              <div class="knowledge-card-header">
                <h3>{{ candidate.title }}</h3>
                <NSpace :size="6">
                  <NTag size="small" type="warning" :bordered="false">{{ categoryLabel(candidate.knowledge_category || '游客 FAQ') }}</NTag>
                  <NTag v-if="candidate.spot_category" size="small" :bordered="false">{{ spotCategoryLabel(candidate.spot_category) }}</NTag>
                </NSpace>
              </div>
              <p class="knowledge-preview">{{ candidate.content }}</p>
              <div class="knowledge-card-footer">
                <small>{{ t('adminKnowledge.labels.session', { session: candidate.session_id || '-' }) }}</small>
                <NSpace :size="6">
                  <NButton size="small" quaternary type="primary" @click="approveCandidate(candidate)">{{ t('adminKnowledge.actions.approve') }}</NButton>
                  <NButton size="small" quaternary type="error" @click="rejectCandidate(candidate)">{{ t('adminKnowledge.actions.reject') }}</NButton>
                </NSpace>
              </div>
            </article>
          </div>
        </NSpin>
      </NCard>
    </NGi>
  </NGrid>
</template>

<style scoped>
.kpi-row {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 16px;
  margin-bottom: 24px;
}

.sg-card {
  background: var(--sg-surface-card);
  border: 1px solid var(--sg-border-soft);
  border-radius: 14px;
}

.spin-placeholder {
  height: 120px;
}

.hint-line {
  font-size: 12px;
  color: var(--sg-text-hint);
  margin: 0;
}

.knowledge-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 16px;
}

.knowledge-grid.compact {
  grid-template-columns: 1fr;
}

.analysis-list {
  display: grid;
  gap: 12px;
}

.analysis-card {
  background: var(--sg-surface-soft);
  border: 1px solid var(--sg-border-soft);
  border-radius: var(--sg-radius-lg);
  padding: 16px;
}

.analysis-card header,
.analysis-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}

.analysis-card header {
  justify-content: space-between;
  margin-bottom: 8px;
}

.analysis-card strong {
  color: var(--sg-text-heading);
  font-size: 13px;
}

.analysis-card p {
  color: var(--sg-text-placeholder);
  font-size: 13px;
  line-height: 1.6;
  margin: 0 0 10px;
}

.analysis-card small {
  display: block;
  color: var(--sg-text-faint);
  font-size: 11px;
  margin-top: 10px;
}

.knowledge-card {
  background: var(--sg-surface-soft);
  border: 1px solid var(--sg-border-soft);
  border-radius: var(--sg-radius-lg);
  padding: 20px;
  display: flex;
  flex-direction: column;
  transition: all 0.2s;
}

.knowledge-card:hover {
  border-color: var(--sg-jade-border);
  background: var(--sg-surface-hover);
}

.knowledge-card-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 10px;
}

.knowledge-card-header h3 {
  font-size: 14px;
  font-weight: 600;
  color: var(--sg-text-heading);
  flex: 1;
  margin-right: 8px;
}

.knowledge-preview {
  font-size: 13px;
  color: var(--sg-text-placeholder);
  line-height: 1.6;
  flex: 1;
  display: -webkit-box;
  -webkit-line-clamp: 3;
  -webkit-box-orient: vertical;
  overflow: hidden;
  margin-bottom: 12px;
}

.knowledge-card-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-top: 12px;
  border-top: 1px solid var(--sg-border-subtle);
}

.knowledge-card-footer small {
  font-size: 11px;
  color: var(--sg-text-faint);
}

.knowledge-pagination {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}

@media (max-width: 768px) {
  .knowledge-grid {
    grid-template-columns: 1fr;
  }
}
</style>
