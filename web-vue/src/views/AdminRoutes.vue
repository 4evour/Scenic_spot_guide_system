<script setup lang="ts">
import { computed, ref, h } from 'vue'
import { useI18n } from 'vue-i18n'
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

const { t } = useI18n()

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
  listApi: '/routes',
  saveApi: (data, edit) => ({
    path: edit ? `/routes/${data.id}` : '/routes',
    method: edit ? 'PUT' : 'POST',
  }),
  deleteApi: (id) => ({ path: `/routes/${id}` }),
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

const difficultyOptions = computed(() => [
  { label: t('adminRoutes.difficulty.easy'), value: 'easy' },
  { label: t('adminRoutes.difficulty.medium'), value: 'medium' },
  { label: t('adminRoutes.difficulty.hard'), value: 'hard' },
])

const difficultyTagType: Record<string, 'success' | 'warning' | 'error'> = {
  easy: 'success',
  medium: 'warning',
  hard: 'error',
}

function difficultyLabel(value: string) {
  if (value === 'easy') return t('adminRoutes.difficulty.easy')
  if (value === 'medium') return t('adminRoutes.difficulty.medium')
  if (value === 'hard') return t('adminRoutes.difficulty.hard')
  return value
}

function formatDuration(minutes: number): string {
  const hours = Math.floor(minutes / 60)
  const mins = minutes % 60
  if (hours > 0 && mins > 0) return `${hours}${t('adminRoutes.units.hour')}${mins}${t('adminRoutes.units.minute')}`
  if (hours > 0) return `${hours}${t('adminRoutes.units.hour')}`
  return `${mins}${t('adminRoutes.units.minute')}`
}

function countSpots(spots: string): number {
  if (!spots || !spots.trim()) return 0
  return spots.split(',').filter((s) => s.trim()).length
}

const rules = computed<FormRules>(() => ({
  name: [{ required: true, message: t('adminRoutes.validation.nameRequired'), trigger: 'blur' }],
  duration: [
    { type: 'number', required: true, message: t('adminRoutes.validation.durationRequired'), trigger: 'blur' },
    { type: 'number', min: 1, message: t('adminRoutes.validation.durationMin'), trigger: 'blur' },
  ],
  difficulty: [{ required: true, message: t('adminRoutes.validation.difficultyRequired'), trigger: 'change' }],
  rating: [
    { type: 'number', min: 0, max: 5, message: t('adminRoutes.validation.ratingRange'), trigger: 'blur' },
  ],
}))

const columns = computed<DataTableColumns<TourRoute>>(() => [
  { title: t('adminRoutes.columns.name'), key: 'name', ellipsis: { tooltip: true } },
  {
    title: t('adminRoutes.columns.description'),
    key: 'description',
    ellipsis: { tooltip: true },
    render(row) {
      const text = row.description || '-'
      return text.length > 40 ? text.slice(0, 40) + '...' : text
    },
  },
  {
    title: t('adminRoutes.columns.spotCount'),
    key: 'spots',
    width: 80,
    align: 'center',
    render(row) {
      return countSpots(row.spots)
    },
  },
  {
    title: t('adminRoutes.columns.duration'),
    key: 'duration',
    width: 120,
    render(row) {
      return formatDuration(row.duration)
    },
  },
  {
    title: t('adminRoutes.columns.difficulty'),
    key: 'difficulty',
    width: 80,
    align: 'center',
    render(row) {
      const type = difficultyTagType[row.difficulty]
      if (!type) return row.difficulty
      return h(NTag, { type, size: 'small', round: true }, { default: () => difficultyLabel(row.difficulty) })
    },
  },
  {
    title: t('adminRoutes.columns.rating'),
    key: 'rating',
    width: 70,
    align: 'center',
    render(row) {
      return Number(row.rating).toFixed(1)
    },
  },
  {
    title: t('adminRoutes.columns.actions'),
    key: 'actions',
    width: 150,
    align: 'center',
    render(row) {
      return h(NSpace, { justify: 'center', size: 'small' }, {
        default: () => [
          h(NButton, { size: 'small', type: 'info', onClick: () => openEdit(row) }, { default: () => t('adminRoutes.actions.edit') }),
          h(NButton, { size: 'small', type: 'error', onClick: () => handleDelete(row) }, { default: () => t('adminRoutes.actions.delete') }),
        ],
      })
    },
  },
])

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
      <h2 class="admin-routes__title">{{ t('adminRoutes.title') }}</h2>
      <NButton type="primary" @click="openCreate">{{ t('adminRoutes.actions.create') }}</NButton>
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
      <NDrawerContent :title="isEditing ? t('adminRoutes.drawer.editTitle') : t('adminRoutes.drawer.createTitle')">
        <NForm
          ref="formRef"
          :model="formData"
          :rules="rules"
          label-placement="left"
          label-width="80"
          require-mark-placement="right-hanging"
        >
          <NFormItem :label="t('adminRoutes.form.name')" path="name">
            <NInput v-model:value="formData.name" :placeholder="t('adminRoutes.placeholders.name')" />
          </NFormItem>

          <NFormItem :label="t('adminRoutes.form.description')" path="description">
            <NInput
              v-model:value="formData.description"
              type="textarea"
              :placeholder="t('adminRoutes.placeholders.description')"
              :rows="3"
            />
          </NFormItem>

          <NFormItem :label="t('adminRoutes.form.spots')" path="spots">
            <NInput
              v-model:value="formData.spots"
              type="textarea"
              :placeholder="t('adminRoutes.placeholders.spots')"
              :rows="3"
            />
          </NFormItem>

          <NFormItem :label="t('adminRoutes.form.duration')" path="duration">
            <NInputNumber
              v-model:value="formData.duration"
              :min="1"
              :placeholder="t('adminRoutes.placeholders.duration')"
              style="width: 100%"
            />
          </NFormItem>

          <NFormItem :label="t('adminRoutes.form.difficulty')" path="difficulty">
            <NSelect
              v-model:value="formData.difficulty"
              :options="difficultyOptions"
              :placeholder="t('adminRoutes.placeholders.difficulty')"
            />
          </NFormItem>

          <NFormItem :label="t('adminRoutes.form.rating')" path="rating">
            <NInputNumber
              v-model:value="formData.rating"
              :min="0"
              :max="5"
              :step="0.1"
              :precision="1"
              :placeholder="t('adminRoutes.placeholders.rating')"
              style="width: 100%"
            />
          </NFormItem>
        </NForm>

        <template #footer>
          <NSpace>
            <NButton @click="closeDrawer">{{ t('adminRoutes.actions.cancel') }}</NButton>
            <NButton type="primary" :loading="saving" @click="onSubmit">
              {{ isEditing ? t('adminRoutes.actions.update') : t('adminRoutes.actions.create') }}
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
