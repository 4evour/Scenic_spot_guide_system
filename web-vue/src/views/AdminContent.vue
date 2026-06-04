<script setup lang="ts">
import { h, ref, onMounted } from 'vue'
import {
  NButton,
  NDataTable,
  NDrawer,
  NDrawerContent,
  NForm,
  NFormItem,
  NInput,
  NInputNumber,
  NSelect,
  NTag,
  type DataTableColumns,
  type FormInst,
  type FormRules,
} from 'naive-ui'
import { useCrudTable } from '../composables/useCrudTable'

interface GuideContent {
  [key: string]: unknown
  id: string
  spot_id: number
  title: string
  content: string
  content_type: string
  audio_url: string
  duration: number
}

const {
  loading,
  saving,
  tableData,
  total,
  pagination,
  drawerVisible,
  formData,
  isEditing,
  openCreate,
  openEdit,
  closeDrawer,
  handleSave,
  handleDelete,
  fetchData,
} = useCrudTable<GuideContent>({
  listApi: '/guide-content',
  idField: 'id',
  defaultForm: () => ({
    title: '',
    content_type: '讲解词',
    spot_id: undefined,
    content: '',
    audio_url: '',
  }),
  saveApi: (data, editing) => ({
    path: editing ? `/guide-content/${data.id}` : '/guide-content',
    method: editing ? 'PUT' : 'POST',
  }),
  deleteApi: (id) => ({ path: `/guide-content/${id}` }),
})

const contentTypeOptions = [
  { label: '讲解词', value: '讲解词' },
  { label: 'FAQ', value: 'FAQ' },
  { label: '文史资料', value: '文史资料' },
  { label: '服务信息', value: '服务信息' },
]

const contentTypeColorMap: Record<string, 'success' | 'info' | 'warning' | 'error' | 'default'> = {
  '讲解词': 'success',
  'FAQ': 'info',
  '文史资料': 'warning',
  '服务信息': 'default',
}

const formRef = ref<FormInst | null>(null)

const formRules: FormRules = {
  title: [{ required: true, message: '请输入标题', trigger: 'blur' }],
  content_type: [{ required: true, message: '请选择类型', trigger: 'change' }],
  content: [{ required: true, message: '请输入内容', trigger: 'blur' }],
}

const columns: DataTableColumns<GuideContent> = [
  { title: '标题', key: 'title', ellipsis: { tooltip: true }, width: 180 },
  {
    title: '类型',
    key: 'content_type',
    width: 110,
    render: (row) =>
      h(
        NTag,
        { type: contentTypeColorMap[row.content_type] ?? 'default', size: 'small', bordered: false },
        { default: () => row.content_type },
      ),
  },
  {
    title: '内容预览',
    key: 'content',
    ellipsis: { tooltip: true },
    width: 240,
    render: (row) => (row.content?.length > 50 ? row.content.slice(0, 50) + '...' : row.content || '-'),
  },
  { title: '关联景点ID', key: 'spot_id', width: 110, render: (row) => row.spot_id || '-' },
  {
    title: '音频',
    key: 'audio_url',
    width: 80,
    render: (row) =>
      h(
        NTag,
        { type: row.audio_url ? 'success' : 'default', size: 'small', bordered: false },
        { default: () => (row.audio_url ? '有' : '无') },
      ),
  },
  {
    title: '操作',
    key: 'actions',
    width: 150,
    fixed: 'right',
    render: (row) =>
      h('div', { style: 'display:flex;gap:8px' }, [
        h(
          NButton,
          { size: 'small', type: 'primary', quaternary: true, onClick: () => openEdit(row) },
          { default: () => '编辑' },
        ),
        h(
          NButton,
          { size: 'small', type: 'error', quaternary: true, onClick: () => handleDelete(row) },
          { default: () => '删除' },
        ),
      ]),
  },
]

function onSave() {
  formRef.value?.validate((errors) => {
    if (!errors) {
      handleSave()
    }
  })
}

onMounted(fetchData)
</script>

<template>
  <div class="admin-content">
    <div class="page-header">
      <h2>讲解内容管理</h2>
      <NButton type="primary" @click="openCreate">+ 新增内容</NButton>
    </div>

    <NDataTable
      :columns="columns"
      :data="tableData"
      :loading="loading"
      :pagination="{ ...pagination, itemCount: total }"
      :row-key="(row: GuideContent) => row.id"
      :scroll-x="900"
      remote
      striped
    />

    <NDrawer v-model:show="drawerVisible" :width="700" placement="right">
      <NDrawerContent :title="isEditing ? '编辑内容' : '新增内容'" closable>
        <NForm
          ref="formRef"
          :model="formData"
          :rules="formRules"
          label-placement="left"
          label-width="90"
          require-mark-placement="right-hanging"
        >
          <NFormItem label="标题" path="title">
            <NInput v-model:value="formData.title" placeholder="请输入标题" />
          </NFormItem>

          <NFormItem label="类型" path="content_type">
            <NSelect
              v-model:value="formData.content_type"
              :options="contentTypeOptions"
              placeholder="请选择类型"
            />
          </NFormItem>

          <NFormItem label="关联景点ID" path="spot_id">
            <NInputNumber
              v-model:value="formData.spot_id"
              :min="0"
              placeholder="请输入景点ID"
              style="width: 100%"
            />
          </NFormItem>

          <NFormItem label="内容" path="content">
            <NInput
              v-model:value="formData.content"
              type="textarea"
              :rows="10"
              placeholder="请输入讲解内容"
            />
          </NFormItem>

          <NFormItem label="音频URL" path="audio_url">
            <NInput v-model:value="formData.audio_url" placeholder="可选，音频文件链接" />
          </NFormItem>
        </NForm>

        <template #footer>
          <div style="display: flex; justify-content: flex-end; gap: 12px">
            <NButton @click="closeDrawer">取消</NButton>
            <NButton type="primary" :loading="saving" @click="onSave">
              {{ isEditing ? '保存' : '创建' }}
            </NButton>
          </div>
        </template>
      </NDrawerContent>
    </NDrawer>
  </div>
</template>

<style scoped>
.admin-content {
  padding: 24px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.page-header h2 {
  font-size: 18px;
  font-weight: 600;
  color: var(--sg-text-body, rgba(255, 255, 255, 0.9));
  margin: 0;
}
</style>
