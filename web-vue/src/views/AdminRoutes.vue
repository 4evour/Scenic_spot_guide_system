<script setup lang="ts">
import { ref, h } from 'vue'
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
  NSpace,
  type DataTableColumns,
  type FormInst,
  type FormRules,
} from 'naive-ui'
import { useCrudTable } from '../composables/useCrudTable'

type TourRoute = {
  id: number
  name: string
  description: string
  spots: string
  duration: number
  difficulty: string
  rating: number
  created_at: string
  updated_at: string
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
} = useCrudTable<TourRoute>({
  listApi: '/tour-routes',
  saveApi: (data, edit) => ({
    path: edit ? `/tour-routes/${data.id}` : '/tour-routes',
    method: edit ? 'PUT' : 'POST',
  }),
  deleteApi: (id) => ({ path: `/tour-routes/${id}` }),
  idField: 'id',
  defaultForm: () => ({
    name: '',
    description: '',
    spots: '',
    duration: 60,
    difficulty: 'easy',
    rating: 4.0,
  }),
})

const formRef = ref<FormInst | null>(null)

const difficultyOptions = [
  { label: '轻松', value: 'easy' },
  { label: '中等', value: 'medium' },
  { label: '挑战', value: 'hard' },
]

const difficultyTagMap: Record<string, { label: string; type: 'success' | 'warning' | 'error' }> = {
  easy: { label: '轻松', type: 'success' },
  medium: { label: '中等', type: 'warning' },
  hard: { label: '挑战', type: 'error' },
}

function formatDuration(minutes: number): string {
  const hours = Math.floor(minutes / 60)
  const mins = minutes % 60
  if (hours > 0 && mins > 0) return `${hours}小时${mins}分钟`
  if (hours > 0) return `${hours}小时`
  return `${mins}分钟`
}

function countSpots(spots: string): number {
  if (!spots || !spots.trim()) return 0
  return spots.split(',').filter((s) => s.trim()).length
}

const rules: FormRules = {
  name: [{ required: true, message: '请输入路线名称', trigger: 'blur' }],
  duration: [
    { type: 'number', required: true, message: '请输入时长', trigger: 'blur' },
    { type: 'number', min: 1, message: '时长至少为 1 分钟', trigger: 'blur' },
  ],
  difficulty: [{ required: true, message: '请选择难度', trigger: 'change' }],
  rating: [
    { type: 'number', min: 0, max: 5, message: '评分范围 0-5', trigger: 'blur' },
  ],
}

const columns: DataTableColumns<TourRoute> = [
  { title: '名称', key: 'name', ellipsis: { tooltip: true } },
  {
    title: '描述',
    key: 'description',
    ellipsis: { tooltip: true },
    render(row) {
      const text = row.description || '-'
      return text.length > 40 ? text.slice(0, 40) + '...' : text
    },
  },
  {
    title: '景点数',
    key: 'spots',
    width: 80,
    align: 'center',
    render(row) {
      return countSpots(row.spots)
    },
  },
  {
    title: '时长',
    key: 'duration',
    width: 120,
    render(row) {
      return formatDuration(row.duration)
    },
  },
  {
    title: '难度',
    key: 'difficulty',
    width: 80,
    align: 'center',
    render(row) {
      const tag = difficultyTagMap[row.difficulty]
      if (!tag) return row.difficulty
      return h(NTag, { type: tag.type, size: 'small', round: true }, { default: () => tag.label })
    },
  },
  {
    title: '评分',
    key: 'rating',
    width: 70,
    align: 'center',
    render(row) {
      return Number(row.rating).toFixed(1)
    },
  },
  {
    title: '操作',
    key: 'actions',
    width: 150,
    align: 'center',
    render(row) {
      return h(NSpace, { justify: 'center', size: 'small' }, {
        default: () => [
          h(NButton, { size: 'small', type: 'info', onClick: () => openEdit(row) }, { default: () => '编辑' }),
          h(NButton, { size: 'small', type: 'error', onClick: () => handleDelete(row) }, { default: () => '删除' }),
        ],
      })
    },
  },
]

async function onSubmit() {
  try {
    await formRef.value?.validate()
    await handleSave()
  } catch {
    // form validation failed; error messages already shown by naive-ui
  }
}

fetchData()
</script>

<template>
  <div class="admin-routes">
    <div class="admin-routes__header">
      <h2 class="admin-routes__title">路线管理</h2>
      <NButton type="primary" @click="openCreate">新增路线</NButton>
    </div>

    <NDataTable
      :columns="columns"
      :data="tableData"
      :loading="loading"
      :pagination="{ ...pagination, itemCount: total }"
      :bordered="false"
      :row-key="(row: TourRoute) => row.id"
      size="small"
      striped
    />

    <NDrawer v-model:show="drawerVisible" :width="600" placement="right">
      <NDrawerContent :title="isEditing ? '编辑路线' : '新增路线'">
        <NForm
          ref="formRef"
          :model="formData"
          :rules="rules"
          label-placement="left"
          label-width="80"
          require-mark-placement="right-hanging"
        >
          <NFormItem label="名称" path="name">
            <NInput v-model:value="formData.name" placeholder="请输入路线名称" />
          </NFormItem>

          <NFormItem label="描述" path="description">
            <NInput
              v-model:value="formData.description"
              type="textarea"
              placeholder="请输入路线描述"
              :rows="3"
            />
          </NFormItem>

          <NFormItem label="景点列表" path="spots">
            <NInput
              v-model:value="formData.spots"
              type="textarea"
              placeholder="景点名称用逗号分隔，如：西湖,断桥,雷峰塔"
              :rows="3"
            />
          </NFormItem>

          <NFormItem label="时长(分钟)" path="duration">
            <NInputNumber
              v-model:value="formData.duration"
              :min="1"
              placeholder="请输入时长"
              style="width: 100%"
            />
          </NFormItem>

          <NFormItem label="难度" path="difficulty">
            <NSelect
              v-model:value="formData.difficulty"
              :options="difficultyOptions"
              placeholder="请选择难度"
            />
          </NFormItem>

          <NFormItem label="评分" path="rating">
            <NInputNumber
              v-model:value="formData.rating"
              :min="0"
              :max="5"
              :step="0.1"
              :precision="1"
              placeholder="请输入评分"
              style="width: 100%"
            />
          </NFormItem>
        </NForm>

        <template #footer>
          <NSpace>
            <NButton @click="closeDrawer">取消</NButton>
            <NButton type="primary" :loading="saving" @click="onSubmit">
              {{ isEditing ? '更新' : '创建' }}
            </NButton>
          </NSpace>
        </template>
      </NDrawerContent>
    </NDrawer>
  </div>
</template>

<style scoped>
.admin-routes {
  padding: var(--space-section, 24px);
}

.admin-routes__header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.admin-routes__title {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
}
</style>
