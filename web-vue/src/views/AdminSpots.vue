<script setup lang="ts">
import { ref, h, onMounted } from 'vue'
import type { DataTableColumns, FormInst, FormRules } from 'naive-ui'
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
  NPopconfirm,
} from 'naive-ui'
import { useCrudTable } from '../composables/useCrudTable'

interface SpotForm {
  Name: string
  Description: string
  Location: string
  Category: string
  Rating: number
  Price: number
  ImageURL: string
  Latitude: number
  Longitude: number
}

const categoryOptions = [
  { label: '核心景点', value: '核心景点' },
  { label: '演艺体验', value: '演艺体验' },
  { label: '文化建筑', value: '文化建筑' },
  { label: '服务设施', value: '服务设施' },
]

const categoryTagType: Record<string, 'success' | 'info' | 'warning' | 'error'> = {
  '核心景点': 'success',
  '演艺体验': 'info',
  '文化建筑': 'warning',
  '服务设施': 'error',
}

function defaultForm(): SpotForm {
  return {
    Name: '',
    Description: '',
    Location: '',
    Category: '核心景点',
    Rating: 4.0,
    Price: 0,
    ImageURL: '',
    Latitude: 0,
    Longitude: 0,
  }
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
  fetchData,
  openCreate,
  openEdit,
  closeDrawer,
  handleSave,
  handleDelete,
} = useCrudTable<Record<string, unknown>>({
  listApi: '/spots',
  saveApi: (data, edit) => ({
    path: edit ? `/spots/${(data as Record<string, unknown>).ID || (data as Record<string, unknown>).id}` : '/spots',
    method: edit ? 'PUT' : 'POST',
  }),
  deleteApi: (id) => ({ path: `/spots/${id}` }),
  defaultForm: defaultForm as unknown as () => Record<string, unknown>,
})

const formRef = ref<FormInst | null>(null)

const formRules: FormRules = {
  Name: [
    { required: true, message: '请输入景点名称', trigger: ['blur', 'input'] },
  ],
  Location: [
    { required: true, message: '请输入位置信息', trigger: ['blur', 'input'] },
  ],
  Category: [
    { required: true, message: '请选择分类', trigger: ['blur', 'change'] },
  ],
  Rating: [
    {
      type: 'number',
      min: 0,
      max: 5,
      message: '评分范围为 0-5',
      trigger: ['blur', 'input'],
    },
  ],
}

const columns: DataTableColumns<Record<string, unknown>> = [
  {
    title: '名称',
    key: 'Name',
    width: 180,
    ellipsis: { tooltip: true },
  },
  {
    title: '分类',
    key: 'Category',
    width: 120,
    render(row) {
      const cat = String(row.Category || '')
      const type = categoryTagType[cat] || 'info'
      return h(NTag, { type, size: 'small', bordered: false }, { default: () => cat || '-' })
    },
  },
  {
    title: '位置',
    key: 'Location',
    width: 200,
    ellipsis: { tooltip: true },
  },
  {
    title: '评分',
    key: 'Rating',
    width: 80,
    align: 'center',
    render(row) {
      const rating = Number(row.Rating ?? 0)
      return rating.toFixed(1)
    },
  },
  {
    title: '价格',
    key: 'Price',
    width: 100,
    align: 'right',
    render(row) {
      const price = Number(row.Price ?? 0)
      return price > 0 ? `¥${price.toFixed(0)}` : '免费'
    },
  },
  {
    title: '操作',
    key: 'actions',
    width: 160,
    align: 'center',
    fixed: 'right',
    render(row) {
      return h(NSpace, { size: 'small', justify: 'center' }, {
        default: () => [
          h(NButton, {
            size: 'small',
            tertiary: true,
            type: 'primary',
            onClick: () => openEdit(row),
          }, { default: () => '编辑' }),
          h(NPopconfirm, {
            onPositiveClick: () => handleDelete(row),
          }, {
            trigger: () =>
              h(NButton, {
                size: 'small',
                tertiary: true,
                type: 'error',
              }, { default: () => '删除' }),
            default: () => '确认删除该景点？',
          }),
        ],
      })
    },
  },
]

function getForm(): SpotForm {
  return formData.value as unknown as SpotForm
}

function rowKey(row: Record<string, unknown>): string | number {
  return (row.ID ?? row.id ?? '') as string | number
}

function onSaveClick() {
  formRef.value?.validate((errors) => {
    if (!errors) {
      handleSave()
    }
  })
}

onMounted(fetchData)
</script>

<template>
  <section class="spots-page">
    <div class="spots-header">
      <div>
        <h2 class="spots-title">景点管理</h2>
        <p class="spots-subtitle">管理景区内的所有景点信息、分类与位置</p>
      </div>
      <NButton type="primary" @click="openCreate">
        新增景点
      </NButton>
    </div>

    <div class="spots-table-wrap">
      <NDataTable
        :columns="columns"
        :data="tableData"
        :loading="loading"
        :pagination="{ ...pagination, itemCount: total }"
        :row-key="rowKey"
        :scroll-x="900"
        :bordered="false"
        size="medium"
        striped
      />
    </div>

    <NDrawer
      v-model:show="drawerVisible"
      :width="500"
      placement="right"
    >
      <NDrawerContent :title="isEditing ? '编辑景点' : '新增景点'" closable>
        <NForm
          ref="formRef"
          :model="formData as unknown as SpotForm"
          :rules="formRules"
          label-placement="left"
          label-width="80"
          require-mark-placement="right-hanging"
        >
          <NFormItem label="名称" path="Name">
            <NInput
              :value="getForm().Name"
              placeholder="请输入景点名称"
              @update:value="(v: string) => { getForm().Name = v }"
            />
          </NFormItem>

          <NFormItem label="描述" path="Description">
            <NInput
              :value="getForm().Description"
              type="textarea"
              :rows="3"
              placeholder="请输入景点描述"
              @update:value="(v: string) => { getForm().Description = v }"
            />
          </NFormItem>

          <NFormItem label="位置" path="Location">
            <NInput
              :value="getForm().Location"
              placeholder="请输入位置信息"
              @update:value="(v: string) => { getForm().Location = v }"
            />
          </NFormItem>

          <NFormItem label="分类" path="Category">
            <NSelect
              :value="getForm().Category"
              :options="categoryOptions"
              placeholder="请选择分类"
              @update:value="(v: string) => { getForm().Category = v }"
            />
          </NFormItem>

          <NFormItem label="评分" path="Rating">
            <NInputNumber
              :value="getForm().Rating"
              :min="0"
              :max="5"
              :step="0.1"
              :precision="1"
              placeholder="0-5"
              style="width: 100%"
              @update:value="(v: number | null) => { getForm().Rating = v ?? 0 }"
            />
          </NFormItem>

          <NFormItem label="价格" path="Price">
            <NInputNumber
              :value="getForm().Price"
              :min="0"
              :precision="0"
              placeholder="0 为免费"
              style="width: 100%"
              @update:value="(v: number | null) => { getForm().Price = v ?? 0 }"
            />
          </NFormItem>

          <NFormItem label="图片链接" path="ImageURL">
            <NInput
              :value="getForm().ImageURL"
              placeholder="可选，图片 URL"
              @update:value="(v: string) => { getForm().ImageURL = v }"
            />
          </NFormItem>

          <div class="coord-row">
            <NFormItem label="经度" path="Longitude" class="coord-item">
              <NInputNumber
                :value="getForm().Longitude"
                :precision="6"
                placeholder="经度"
                style="width: 100%"
                @update:value="(v: number | null) => { getForm().Longitude = v ?? 0 }"
              />
            </NFormItem>

            <NFormItem label="纬度" path="Latitude" class="coord-item">
              <NInputNumber
                :value="getForm().Latitude"
                :precision="6"
                placeholder="纬度"
                style="width: 100%"
                @update:value="(v: number | null) => { getForm().Latitude = v ?? 0 }"
              />
            </NFormItem>
          </div>
        </NForm>

        <template #footer>
          <NSpace justify="end">
            <NButton @click="closeDrawer">取消</NButton>
            <NButton type="primary" :loading="saving" @click="onSaveClick">
              {{ isEditing ? '保存修改' : '确认新增' }}
            </NButton>
          </NSpace>
        </template>
      </NDrawerContent>
    </NDrawer>
  </section>
</template>

<style scoped>
.spots-page {
  padding: 0;
}

.spots-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 24px;
}

.spots-title {
  font-size: 20px;
  font-weight: 700;
  color: rgba(255, 255, 255, 0.92);
  margin: 0 0 4px;
}

.spots-subtitle {
  font-size: 13px;
  color: rgba(255, 255, 255, 0.35);
  margin: 0;
}

.spots-table-wrap {
  background: var(--sg-surface-card, rgba(255, 255, 255, 0.03));
  border: 1px solid var(--sg-border-soft, rgba(255, 255, 255, 0.06));
  border-radius: 14px;
  padding: 20px;
  overflow: hidden;
}

.coord-row {
  display: flex;
  gap: 12px;
}

.coord-item {
  flex: 1;
}

@media (max-width: 768px) {
  .spots-header {
    flex-direction: column;
    gap: 12px;
  }
  .spots-table-wrap {
    padding: 12px;
  }
  .coord-row {
    flex-direction: column;
    gap: 0;
  }
}
</style>
