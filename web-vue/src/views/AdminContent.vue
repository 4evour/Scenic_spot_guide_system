<script setup lang="ts">
import { computed, h, ref, onMounted } from 'vue'
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
} = useCrudTable<GuideContent>({
  listApi: '/contents',
  idField: 'id',
  defaultForm: () => ({
    title: '',
    content_type: '讲解词',
    spot_id: undefined,
    content: '',
    audio_url: '',
  }),
  saveApi: (data, editing) => ({
    path: editing ? `/contents/${data.id}` : '/contents',
    method: editing ? 'PUT' : 'POST',
  }),
  deleteApi: (id) => ({ path: `/contents/${id}` }),
})

const contentTypeOptions = computed(() => [
  { label: t('adminContent.contentTypes.guide'), value: '讲解词' },
  { label: t('adminContent.contentTypes.faq'), value: 'FAQ' },
  { label: t('adminContent.contentTypes.history'), value: '文史资料' },
  { label: t('adminContent.contentTypes.service'), value: '服务信息' },
])

const contentTypeColorMap: Record<string, 'success' | 'info' | 'warning' | 'error' | 'default'> = {
  '讲解词': 'success',
  'FAQ': 'info',
  '文史资料': 'warning',
  '服务信息': 'default',
}

const formRef = ref<FormInst | null>(null)

const formRules = computed<FormRules>(() => ({
  title: [{ required: true, message: t('adminContent.validation.titleRequired'), trigger: 'blur' }],
  content_type: [{ required: true, message: t('adminContent.validation.typeRequired'), trigger: 'change' }],
  content: [{ required: true, message: t('adminContent.validation.contentRequired'), trigger: 'blur' }],
}))

function contentTypeLabel(value: string) {
  if (value === '讲解词') return t('adminContent.contentTypes.guide')
  if (value === 'FAQ') return t('adminContent.contentTypes.faq')
  if (value === '文史资料') return t('adminContent.contentTypes.history')
  if (value === '服务信息') return t('adminContent.contentTypes.service')
  return value
}

const columns = computed<DataTableColumns<GuideContent>>(() => [
  { title: t('adminContent.columns.title'), key: 'title', ellipsis: { tooltip: true }, width: 180 },
  {
    title: t('adminContent.columns.type'),
    key: 'content_type',
    width: 110,
    render: (row) =>
      h(
        NTag,
        { type: contentTypeColorMap[row.content_type] ?? 'default', size: 'small', bordered: false },
        { default: () => contentTypeLabel(row.content_type) },
      ),
  },
  {
    title: t('adminContent.columns.preview'),
    key: 'content',
    ellipsis: { tooltip: true },
    width: 240,
    render: (row) => (row.content?.length > 50 ? row.content.slice(0, 50) + '...' : row.content || '-'),
  },
  { title: t('adminContent.columns.spotID'), key: 'spot_id', width: 110, render: (row) => row.spot_id || '-' },
  {
    title: t('adminContent.columns.audio'),
    key: 'audio_url',
    width: 80,
    render: (row) =>
      h(
        NTag,
        { type: row.audio_url ? 'success' : 'default', size: 'small', bordered: false },
        { default: () => (row.audio_url ? t('adminContent.audio.available') : t('adminContent.audio.missing')) },
      ),
  },
  {
    title: t('adminContent.columns.actions'),
    key: 'actions',
    width: 150,
    fixed: 'right',
    render: (row) =>
      h('div', { style: 'display:flex;gap:8px' }, [
        h(
          NButton,
          { size: 'small', type: 'primary', quaternary: true, onClick: () => openEdit(row) },
          { default: () => t('adminContent.actions.edit') },
        ),
        h(
          NButton,
          { size: 'small', type: 'error', quaternary: true, onClick: () => handleDelete(row) },
          { default: () => t('adminContent.actions.delete') },
        ),
      ]),
  },
])

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
      <h2>{{ t('adminContent.title') }}</h2>
      <NButton type="primary" @click="openCreate">{{ t('adminContent.actions.create') }}</NButton>
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
      <NDrawerContent :title="isEditing ? t('adminContent.drawer.editTitle') : t('adminContent.drawer.createTitle')" closable>
        <NForm
          ref="formRef"
          :model="formData"
          :rules="formRules"
          label-placement="left"
          label-width="90"
          require-mark-placement="right-hanging"
        >
          <NFormItem :label="t('adminContent.form.title')" path="title">
            <NInput v-model:value="formData.title" :placeholder="t('adminContent.placeholders.title')" />
          </NFormItem>

          <NFormItem :label="t('adminContent.form.type')" path="content_type">
            <NSelect
              v-model:value="formData.content_type"
              :options="contentTypeOptions"
              :placeholder="t('adminContent.placeholders.type')"
            />
          </NFormItem>

          <NFormItem :label="t('adminContent.form.spotID')" path="spot_id">
            <NInputNumber
              v-model:value="formData.spot_id"
              :min="0"
              :placeholder="t('adminContent.placeholders.spotID')"
              style="width: 100%"
            />
          </NFormItem>

          <NFormItem :label="t('adminContent.form.content')" path="content">
            <NInput
              v-model:value="formData.content"
              type="textarea"
              :rows="10"
              :placeholder="t('adminContent.placeholders.content')"
            />
          </NFormItem>

          <NFormItem :label="t('adminContent.form.audioURL')" path="audio_url">
            <NInput v-model:value="formData.audio_url" :placeholder="t('adminContent.placeholders.audioURL')" />
          </NFormItem>
        </NForm>

        <template #footer>
          <div style="display: flex; justify-content: flex-end; gap: 12px">
            <NButton @click="closeDrawer">{{ t('adminContent.actions.cancel') }}</NButton>
            <NButton type="primary" :loading="saving" @click="onSave">
              {{ isEditing ? t('adminContent.actions.save') : t('adminContent.actions.submitCreate') }}
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
