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
  NSwitch,
} from 'naive-ui'
import { useCrudTable } from '../composables/useCrudTable'

interface SpotForm {
  id?: number
  name: string
  description: string
  location: string
  category: string
  rating: number
  price: number
  image_url: string
  latitude: number
  longitude: number
  sort_order: number
  qr_code: string
  qr_intro_text: string
  qr_enabled: boolean
  geofence_enabled: boolean
  geofence_radius_m: number
  geofence_intro_text: string
  geofence_cooldown_minutes: number
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
    name: '',
    description: '',
    location: '',
    category: '核心景点',
    rating: 4.0,
    price: 0,
    image_url: '',
    latitude: 0,
    longitude: 0,
    sort_order: 0,
    qr_code: '',
    qr_intro_text: '',
    qr_enabled: false,
    geofence_enabled: false,
    geofence_radius_m: 100,
    geofence_intro_text: '',
    geofence_cooldown_minutes: 1440,
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
    path: edit ? `/spots/${data.id}` : '/spots',
    method: edit ? 'PUT' : 'POST',
  }),
  deleteApi: (id) => ({ path: `/spots/${id}` }),
  idField: 'id',
  defaultForm: defaultForm as unknown as () => Record<string, unknown>,
})

const formRef = ref<FormInst | null>(null)

const formRules: FormRules = {
  name: [
    { required: true, message: '请输入景点名称', trigger: ['blur', 'input'] },
  ],
  location: [
    { required: true, message: '请输入位置信息', trigger: ['blur', 'input'] },
  ],
  category: [
    { required: true, message: '请选择分类', trigger: ['blur', 'change'] },
  ],
  rating: [
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
    key: 'name',
    width: 180,
    ellipsis: { tooltip: true },
  },
  {
    title: '分类',
    key: 'category',
    width: 120,
    render(row) {
      const cat = String(row.category || '')
      const type = categoryTagType[cat] || 'info'
      return h(NTag, { type, size: 'small', bordered: false }, { default: () => cat || '-' })
    },
  },
  {
    title: '位置',
    key: 'location',
    width: 200,
    ellipsis: { tooltip: true },
  },
  {
    title: '评分',
    key: 'rating',
    width: 80,
    align: 'center',
    render(row) {
      const rating = Number(row.rating ?? 0)
      return rating.toFixed(1)
    },
  },
  {
    title: '价格',
    key: 'price',
    width: 100,
    align: 'right',
    render(row) {
      const price = Number(row.price ?? 0)
      return price > 0 ? `¥${price.toFixed(0)}` : '免费'
    },
  },
  {
    title: '二维码',
    key: 'qr_code',
    width: 140,
    render(row) {
      const code = String(row.qr_code || '')
      const enabled = Boolean(row.qr_enabled)
      if (!code) return h('span', { style: 'color:rgba(255,255,255,.25);font-size:12px' }, '未配置')
      return h(NTag, { type: enabled ? 'success' : 'default', size: 'small', bordered: false }, { default: () => code })
    },
  },
  {
    title: '电子围栏',
    key: 'geofence_enabled',
    width: 120,
    render(row) {
      return h(NTag, { type: row.geofence_enabled ? 'success' : 'default', size: 'small', bordered: false }, {
        default: () => row.geofence_enabled ? `${Number(row.geofence_radius_m || 100)}m` : '未启用',
      })
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
          h(NButton, {
            size: 'small',
            tertiary: true,
            type: 'error',
            onClick: () => handleDelete(row),
          }, { default: () => '删除' }),
        ],
      })
    },
  },
]

function getForm(): SpotForm {
  return formData.value as unknown as SpotForm
}

function rowKey(row: Record<string, unknown>): string | number {
  return (row.id ?? '') as string | number
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
          <NFormItem label="名称" path="name">
            <NInput
              :value="getForm().name"
              placeholder="请输入景点名称"
              @update:value="(v: string) => { getForm().name = v }"
            />
          </NFormItem>

          <NFormItem label="描述" path="description">
            <NInput
              :value="getForm().description"
              type="textarea"
              :rows="3"
              placeholder="请输入景点描述"
              @update:value="(v: string) => { getForm().description = v }"
            />
          </NFormItem>

          <NFormItem label="位置" path="location">
            <NInput
              :value="getForm().location"
              placeholder="请输入位置信息"
              @update:value="(v: string) => { getForm().location = v }"
            />
          </NFormItem>

          <NFormItem label="分类" path="category">
            <NSelect
              :value="getForm().category"
              :options="categoryOptions"
              placeholder="请选择分类"
              @update:value="(v: string) => { getForm().category = v }"
            />
          </NFormItem>

          <NFormItem label="评分" path="rating">
            <NInputNumber
              :value="getForm().rating"
              :min="0"
              :max="5"
              :step="0.1"
              :precision="1"
              placeholder="0-5"
              style="width: 100%"
              @update:value="(v: number | null) => { getForm().rating = v ?? 0 }"
            />
          </NFormItem>

          <NFormItem label="价格" path="price">
            <NInputNumber
              :value="getForm().price"
              :min="0"
              :precision="0"
              placeholder="0 为免费"
              style="width: 100%"
              @update:value="(v: number | null) => { getForm().price = v ?? 0 }"
            />
          </NFormItem>

          <NFormItem label="图片链接" path="image_url">
            <NInput
              :value="getForm().image_url"
              placeholder="可选，图片 URL"
              @update:value="(v: string) => { getForm().image_url = v }"
            />
          </NFormItem>

          <div class="coord-row">
            <NFormItem label="经度" path="longitude" class="coord-item">
              <NInputNumber
                :value="getForm().longitude"
                :precision="6"
                placeholder="经度"
                style="width: 100%"
                @update:value="(v: number | null) => { getForm().longitude = v ?? 0 }"
              />
            </NFormItem>

            <NFormItem label="纬度" path="latitude" class="coord-item">
              <NInputNumber
                :value="getForm().latitude"
                :precision="6"
                placeholder="纬度"
                style="width: 100%"
                @update:value="(v: number | null) => { getForm().latitude = v ?? 0 }"
              />
            </NFormItem>
          </div>

          <NFormItem label="排序" path="sort_order">
            <NInputNumber
              :value="getForm().sort_order"
              :precision="0"
              placeholder="数值越小越靠前"
              style="width: 100%"
              @update:value="(v: number | null) => { getForm().sort_order = v ?? 0 }"
            />
          </NFormItem>
          <NFormItem label="二维码 ID" path="qr_code">
            <NInput
              :value="getForm().qr_code"
              placeholder="留空则自动生成，如 SPOT-0001"
              @update:value="(v: string) => { getForm().qr_code = v }"
            />
          </NFormItem>

          <NFormItem label="开场白" path="qr_intro_text">
            <NInput
              :value="getForm().qr_intro_text"
              type="textarea"
              :rows="2"
              placeholder="扫码后数字人自动说的开场白（留空则自动AI生成）"
              @update:value="(v: string) => { getForm().qr_intro_text = v }"
            />
          </NFormItem>

          <NFormItem label="启用扫码" path="qr_enabled">
            <NSwitch
              :value="getForm().qr_enabled"
              @update:value="(v: boolean) => { getForm().qr_enabled = v }"
            />
            <span style="margin-left:8px;font-size:12px;color:rgba(255,255,255,.4)">
              {{ getForm().qr_enabled ? '游客可扫码触发讲解' : '扫码功能已关闭' }}
            </span>
          </NFormItem>

          <NFormItem label="到点讲解" path="geofence_enabled">
            <NSwitch
              :value="getForm().geofence_enabled"
              @update:value="(v: boolean) => { getForm().geofence_enabled = v }"
            />
            <span style="margin-left:8px;font-size:12px;color:rgba(255,255,255,.4)">
              {{ getForm().geofence_enabled ? '游客到达附近自动触发' : '电子围栏已关闭' }}
            </span>
          </NFormItem>

          <div class="coord-row">
            <NFormItem label="半径(m)" path="geofence_radius_m" class="coord-item">
              <NInputNumber
                :value="getForm().geofence_radius_m"
                :min="20"
                :max="1000"
                :precision="0"
                style="width: 100%"
                @update:value="(v: number | null) => { getForm().geofence_radius_m = v ?? 100 }"
              />
            </NFormItem>
            <NFormItem label="冷却(分)" path="geofence_cooldown_minutes" class="coord-item">
              <NInputNumber
                :value="getForm().geofence_cooldown_minutes"
                :min="1"
                :max="10080"
                :precision="0"
                style="width: 100%"
                @update:value="(v: number | null) => { getForm().geofence_cooldown_minutes = v ?? 1440 }"
              />
            </NFormItem>
          </div>

          <NFormItem label="触发文案" path="geofence_intro_text">
            <NInput
              :value="getForm().geofence_intro_text"
              type="textarea"
              :rows="2"
              placeholder="到达该景点时优先播报的数字人语音文案，留空则使用讲解内容"
              @update:value="(v: string) => { getForm().geofence_intro_text = v }"
            />
          </NFormItem>
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
