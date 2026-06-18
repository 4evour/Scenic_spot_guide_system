<script setup lang="ts">
import { computed, reactive, ref, onMounted } from 'vue'
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
import type { KnowledgeCandidate, KnowledgeItem } from '../types/admin'

const categoryOptions = [
  { label: '全部分类', value: '' },
  { label: '讲解词', value: '讲解词' },
  { label: '文史资料', value: '文史资料' },
  { label: '游客 FAQ', value: '游客 FAQ' },
  { label: '路线推荐', value: '路线推荐' },
  { label: '服务设施', value: '服务设施' },
  { label: '票务交通', value: '票务交通' },
  { label: '官方资料', value: 'official' },
  { label: '政府资料', value: 'government' },
  { label: '景区概况', value: 'overview' },
  { label: '景点介绍', value: 'spot' },
  { label: '实时边界', value: 'boundary' },
]

const emptyEditor = { title: '', category: '讲解词', source: 'admin', content: '' }
const uploadCategoryOptions = categoryOptions.filter(option => option.value)
const spotCategoryOptions = [
  { label: '全部景点分类', value: '' },
  { label: '核心景点', value: '核心景点' },
  { label: '演艺体验', value: '演艺体验' },
  { label: '文化建筑', value: '文化建筑' },
  { label: '服务设施', value: '服务设施' },
]

const message = useMessage()
const dialog = useDialog()

const formRef = ref<FormInst | null>(null)

const formRules: FormRules = {
  content: [{ required: true, message: '请输入知识内容', trigger: 'blur' }],
}

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
  candidatesLoading: false,
  candidates: [] as KnowledgeCandidate[],
})

const displayedCount = computed(() => state.knowledge.length)
const spotOptions = computed(() => [
  { label: '全部景点', value: 0 },
  ...state.spots
    .filter(spot => !state.spotCategory || spot.category === state.spotCategory)
    .map(spot => ({ label: spot.label, value: spot.value })),
])

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
    updated: updatedAt ? new Date(updatedAt).toLocaleDateString('zh-CN') : '-',
  }
}

async function loadSpots() {
  try {
    const spots = await apiFetch<Array<Record<string, unknown>>>('/spots')
    state.spots = spots.map(raw => ({
      label: String(getField(raw, 'name') || `景点${getField(raw, 'id') || ''}`),
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
    message.error(error instanceof Error ? error.message : '知识库加载失败')
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
      message.success(state.editingID ? '知识条目已更新，数字人检索缓存已刷新。' : '知识条目已加入数字人知识库。')
      resetEditor()
      await loadKnowledge()
    } catch (error) {
      message.error(error instanceof Error ? error.message : '保存失败')
    } finally {
      state.saving = false
    }
  })
}

function deleteKnowledge(item: KnowledgeItem) {
  dialog.warning({
    title: '确认删除',
    content: `确认删除「${item.title}」吗？`,
    positiveText: '删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await apiFetch(`/knowledge/${encodeURIComponent(item.id)}`, { method: 'DELETE' })
        message.success('知识条目已删除。')
        await loadKnowledge()
      } catch (error) {
        message.error(error instanceof Error ? error.message : '删除失败')
      }
    },
  })
}

function onFileChange(event: Event) {
  state.selectedFile = (event.target as HTMLInputElement).files?.[0] ?? null
}

async function uploadKnowledge() {
  if (!state.selectedFile) {
    message.warning('请先选择知识文档。')
    return
  }
  state.saving = true
  try {
    const form = new FormData()
    form.append('file', state.selectedFile)
    form.append('category', state.uploadCategory)
    const data = await apiFetch<{ loaded_count?: number }>('/knowledge/upload', { method: 'POST', body: form })
    message.success(`上传完成，已导入 ${data.loaded_count || 0} 条知识片段。`)
    state.selectedFile = null
    await loadKnowledge()
  } catch (error) {
    message.error(error instanceof Error ? error.message : '上传失败')
  } finally {
    state.saving = false
  }
}

async function analyzeSession() {
  if (!state.analysisSessionId.trim()) {
    message.warning('请输入需要分析的会话 ID。')
    return
  }
  state.analyzing = true
  try {
    await apiFetch(`/admin/insights/sessions/${encodeURIComponent(state.analysisSessionId.trim())}/analyze`, { method: 'POST', body: '{}' })
    message.success('AI 分析完成，已生成知识候选。')
    await loadCandidates()
  } catch (error) {
    message.error(error instanceof Error ? error.message : 'AI 分析失败')
  } finally {
    state.analyzing = false
  }
}

async function loadCandidates() {
  state.candidatesLoading = true
  try {
    const data = await apiFetch<{ list?: KnowledgeCandidate[] }>('/admin/knowledge/candidates?status=pending&page_size=20')
    state.candidates = data.list || []
  } catch (error) {
    message.error(error instanceof Error ? error.message : '知识候选加载失败')
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
    message.success('候选知识已入库。')
    await Promise.all([loadCandidates(), loadKnowledge()])
  } catch (error) {
    message.error(error instanceof Error ? error.message : '入库失败')
  }
}

async function rejectCandidate(candidate: KnowledgeCandidate) {
  try {
    await apiFetch(`/admin/knowledge/candidates/${candidate.id}/reject`, { method: 'POST', body: JSON.stringify({ reason: '管理员拒绝' }) })
    message.success('候选知识已拒绝。')
    await loadCandidates()
  } catch (error) {
    message.error(error instanceof Error ? error.message : '拒绝失败')
  }
}

onMounted(async () => {
  await Promise.all([loadSpots(), loadKnowledge(), loadCandidates()])
})
</script>

<template>
  <section class="kpi-row">
    <KpiCard label="知识条目" :value="String(state.total)" note="来自 RAG 知识库" />
    <KpiCard label="当前页显示" :value="String(displayedCount)" note="服务端筛选结果" tone="green" />
    <KpiCard label="支持格式" value="JSONL/MD" note="另支持 JSON、TXT" tone="gold" />
    <KpiCard label="缓存状态" value="自动刷新" note="增删改后立即生效" />
  </section>

  <NGrid :cols="2" :x-gap="16" :y-gap="16" responsive="screen" item-responsive>
    <NGi span="2 m:1">
      <NCard :title="state.editingID ? '编辑知识条目' : '新增知识条目'" :bordered="false" class="sg-card">
        <NForm ref="formRef" :model="state.editor" :rules="formRules" label-placement="left" label-width="70" require-mark-placement="right-hanging">
          <NFormItem label="标题" path="title">
            <NInput v-model:value="state.editor.title" placeholder="例如：九龙灌浴讲解词" />
          </NFormItem>
          <NFormItem label="分类" path="category">
            <NSelect v-model:value="state.editor.category" :options="uploadCategoryOptions" />
          </NFormItem>
          <NFormItem label="来源" path="source">
            <NInput v-model:value="state.editor.source" placeholder="admin / 文件名 / 景点名称" />
          </NFormItem>
          <NFormItem label="知识内容" path="content">
            <NInput v-model:value="state.editor.content" type="textarea" :rows="8" placeholder="填写讲解词、历史背景、FAQ 问答或运营说明" />
          </NFormItem>
          <NSpace>
            <NButton type="primary" :loading="state.saving" :disabled="!state.editor.content.trim()" @click="saveKnowledge">
              {{ state.editingID ? '保存更新' : '加入知识库' }}
            </NButton>
            <NButton @click="resetEditor">清空</NButton>
          </NSpace>
        </NForm>
      </NCard>
    </NGi>

    <NGi span="2 m:1">
      <NCard title="上传知识文档" :bordered="false" class="sg-card">
        <NSpace vertical :size="12">
          <NSpace align="center" :wrap="false">
            <input type="file" accept=".jsonl,.json,.md,.markdown,.txt" @change="onFileChange" />
            <NSelect v-model:value="state.uploadCategory" :options="uploadCategoryOptions" style="width: 160px" />
            <NButton type="primary" :loading="state.saving" :disabled="!state.selectedFile" @click="uploadKnowledge">
              上传并导入
            </NButton>
          </NSpace>
          <p class="hint-line">JSONL/JSON 需包含 title、content、source、metadata 字段；Markdown/TXT 会按段落自动切片。</p>
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
          placeholder="搜索景点、讲解词、FAQ、来源..."
          clearable
          style="width: 280px"
          @keydown.enter="reloadFromFirstPage"
          @clear="reloadFromFirstPage"
        />
        <NButton :loading="state.loading" @click="reloadFromFirstPage">查询</NButton>
        <NButton :loading="state.loading" @click="loadKnowledge">刷新</NButton>
      </NSpace>
    </template>

    <template #default>
      <NSpin :show="state.loading">
        <div v-if="state.loading" class="spin-placeholder" />
        <div v-else-if="state.knowledge.length === 0">
          <NEmpty description="暂无知识条目" />
        </div>
        <div v-else class="knowledge-grid">
          <article v-for="item in state.knowledge" :key="item.id" class="knowledge-card">
            <div class="knowledge-card-header">
              <h3>{{ item.title }}</h3>
              <NSpace :size="6">
                <NTag size="small" :bordered="false" type="success">{{ item.knowledge_category || item.category }}</NTag>
                <NTag v-if="item.spot_category" size="small" :bordered="false">{{ item.spot_category }}</NTag>
              </NSpace>
            </div>
            <p class="knowledge-preview">{{ item.content }}</p>
            <div class="knowledge-card-footer">
              <small>来源：{{ item.source }} / {{ item.updated }}</small>
              <NSpace :size="6">
                <NButton size="small" quaternary type="primary" @click="editKnowledge(item)">编辑</NButton>
                <NButton size="small" quaternary type="error" @click="deleteKnowledge(item)">删除</NButton>
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

  <NCard title="AI 知识候选" :bordered="false" class="sg-card" style="margin-top: 16px">
    <template #header-extra>
      <NSpace align="center">
        <NInput v-model:value="state.analysisSessionId" placeholder="输入会话 ID 生成候选" style="width: 240px" />
        <NButton type="primary" :loading="state.analyzing" @click="analyzeSession">AI 分析会话</NButton>
        <NButton :loading="state.candidatesLoading" @click="loadCandidates">刷新候选</NButton>
      </NSpace>
    </template>
    <NSpin :show="state.candidatesLoading">
      <div v-if="state.candidates.length === 0">
        <NEmpty description="暂无待审核知识候选" />
      </div>
      <div v-else class="knowledge-grid">
        <article v-for="candidate in state.candidates" :key="candidate.id" class="knowledge-card">
          <div class="knowledge-card-header">
            <h3>{{ candidate.title }}</h3>
            <NSpace :size="6">
              <NTag size="small" type="warning" :bordered="false">{{ candidate.knowledge_category || '游客 FAQ' }}</NTag>
              <NTag v-if="candidate.spot_category" size="small" :bordered="false">{{ candidate.spot_category }}</NTag>
            </NSpace>
          </div>
          <p class="knowledge-preview">{{ candidate.content }}</p>
          <div class="knowledge-card-footer">
            <small>会话：{{ candidate.session_id || '-' }}</small>
            <NSpace :size="6">
              <NButton size="small" quaternary type="primary" @click="approveCandidate(candidate)">入库</NButton>
              <NButton size="small" quaternary type="error" @click="rejectCandidate(candidate)">拒绝</NButton>
            </NSpace>
          </div>
        </article>
      </div>
    </NSpin>
  </NCard>
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
