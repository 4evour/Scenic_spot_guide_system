<script setup lang="ts">
import { computed, ref, h, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
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

const { t } = useI18n()

const categoryOptions = computed(() => [
  { label: t('adminSpots.categories.core'), value: '核心景点' },
  { label: t('adminSpots.categories.performance'), value: '演艺体验' },
  { label: t('adminSpots.categories.culture'), value: '文化建筑' },
  { label: t('adminSpots.categories.service'), value: '服务设施' },
])

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

const formRules = computed<FormRules>(() => ({
  name: [
    { required: true, message: t('adminSpots.validation.nameRequired'), trigger: ['blur', 'input'] },
  ],
  location: [
    { required: true, message: t('adminSpots.validation.locationRequired'), trigger: ['blur', 'input'] },
  ],
  category: [
    { required: true, message: t('adminSpots.validation.categoryRequired'), trigger: ['blur', 'change'] },
  ],
  rating: [
    {
      type: 'number',
      min: 0,
      max: 5,
      message: t('adminSpots.validation.ratingRange'),
      trigger: ['blur', 'input'],
    },
  ],
}))

function categoryLabel(value: string) {
  if (value === '核心景点') return t('adminSpots.categories.core')
  if (value === '演艺体验') return t('adminSpots.categories.performance')
  if (value === '文化建筑') return t('adminSpots.categories.culture')
  if (value === '服务设施') return t('adminSpots.categories.service')
  return value
}

const columns = computed<DataTableColumns<Record<string, unknown>>>(() => [
  {
    title: t('adminSpots.columns.name'),
    key: 'name',
    width: 180,
    ellipsis: { tooltip: true },
  },
  {
    title: t('adminSpots.columns.category'),
    key: 'category',
    width: 120,
    render(row) {
      const cat = String(row.category || '')
      const type = categoryTagType[cat] || 'info'
      return h(NTag, { type, size: 'small', bordered: false }, { default: () => (cat ? categoryLabel(cat) : '-') })
    },
  },
  {
    title: t('adminSpots.columns.location'),
    key: 'location',
    width: 200,
    ellipsis: { tooltip: true },
  },
  {
    title: t('adminSpots.columns.rating'),
    key: 'rating',
    width: 80,
    align: 'center',
    render(row) {
      const rating = Number(row.rating ?? 0)
      return rating.toFixed(1)
    },
  },
  {
    title: t('adminSpots.columns.price'),
    key: 'price',
    width: 100,
    align: 'right',
    render(row) {
      const price = Number(row.price ?? 0)
      return price > 0 ? `¥${price.toFixed(0)}` : t('adminSpots.price.free')
    },
  },
  {
    title: t('adminSpots.columns.qrCode'),
    key: 'qr_code',
    width: 140,
    render(row) {
      const code = String(row.qr_code || '')
      const enabled = Boolean(row.qr_enabled)
      if (!code) return h('span', { style: 'color:rgba(255,255,255,.25);font-size:12px' }, t('adminSpots.status.notConfigured'))
      return h(NTag, { type: enabled ? 'success' : 'default', size: 'small', bordered: false }, { default: () => code })
    },
  },
  {
    title: t('adminSpots.columns.geofence'),
    key: 'geofence_enabled',
    width: 120,
    render(row) {
      return h(NTag, { type: row.geofence_enabled ? 'success' : 'default', size: 'small', bordered: false }, {
        default: () => row.geofence_enabled ? `${Number(row.geofence_radius_m || 100)}m` : t('adminSpots.status.disabled'),
      })
    },
  },
  {
    title: t('adminSpots.columns.actions'),
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
          }, { default: () => t('adminSpots.actions.edit') }),
          h(NButton, {
            size: 'small',
            tertiary: true,
            type: 'error',
            onClick: () => handleDelete(row),
          }, { default: () => t('adminSpots.actions.delete') }),
        ],
      })
    },
  },
])

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
        <h2 class="spots-title">{{ t('adminSpots.title') }}</h2>
        <p class="spots-subtitle">{{ t('adminSpots.subtitle') }}</p>
      </div>
      <NButton type="primary" @click="openCreate">
        {{ t('adminSpots.actions.create') }}
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
      <NDrawerContent :title="isEditing ? t('adminSpots.drawer.editTitle') : t('adminSpots.drawer.createTitle')" closable>
        <NForm
          ref="formRef"
          :model="formData as unknown as SpotForm"
          :rules="formRules"
          label-placement="left"
          label-width="80"
          require-mark-placement="right-hanging"
        >
          <NFormItem :label="t('adminSpots.form.name')" path="name">
            <NInput
              :value="getForm().name"
              :placeholder="t('adminSpots.placeholders.name')"
              @update:value="(v: string) => { getForm().name = v }"
            />
          </NFormItem>

          <NFormItem :label="t('adminSpots.form.description')" path="description">
            <NInput
              :value="getForm().description"
              type="textarea"
              :rows="3"
              :placeholder="t('adminSpots.placeholders.description')"
              @update:value="(v: string) => { getForm().description = v }"
            />
          </NFormItem>

          <NFormItem :label="t('adminSpots.form.location')" path="location">
            <NInput
              :value="getForm().location"
              :placeholder="t('adminSpots.placeholders.location')"
              @update:value="(v: string) => { getForm().location = v }"
            />
          </NFormItem>

          <NFormItem :label="t('adminSpots.form.category')" path="category">
            <NSelect
              :value="getForm().category"
              :options="categoryOptions"
              :placeholder="t('adminSpots.placeholders.category')"
              @update:value="(v: string) => { getForm().category = v }"
            />
          </NFormItem>

          <NFormItem :label="t('adminSpots.form.rating')" path="rating">
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

          <NFormItem :label="t('adminSpots.form.price')" path="price">
            <NInputNumber
              :value="getForm().price"
              :min="0"
              :precision="0"
              :placeholder="t('adminSpots.placeholders.price')"
              style="width: 100%"
              @update:value="(v: number | null) => { getForm().price = v ?? 0 }"
            />
          </NFormItem>

          <NFormItem :label="t('adminSpots.form.imageURL')" path="image_url">
            <NInput
              :value="getForm().image_url"
              :placeholder="t('adminSpots.placeholders.imageURL')"
              @update:value="(v: string) => { getForm().image_url = v }"
            />
          </NFormItem>

          <div class="coord-row">
            <NFormItem :label="t('adminSpots.form.longitude')" path="longitude" class="coord-item">
              <NInputNumber
                :value="getForm().longitude"
                :precision="6"
                :placeholder="t('adminSpots.placeholders.longitude')"
                style="width: 100%"
                @update:value="(v: number | null) => { getForm().longitude = v ?? 0 }"
              />
            </NFormItem>

            <NFormItem :label="t('adminSpots.form.latitude')" path="latitude" class="coord-item">
              <NInputNumber
                :value="getForm().latitude"
                :precision="6"
                :placeholder="t('adminSpots.placeholders.latitude')"
                style="width: 100%"
                @update:value="(v: number | null) => { getForm().latitude = v ?? 0 }"
              />
            </NFormItem>
          </div>

          <NFormItem :label="t('adminSpots.form.sortOrder')" path="sort_order">
            <NInputNumber
              :value="getForm().sort_order"
              :precision="0"
              :placeholder="t('adminSpots.placeholders.sortOrder')"
              style="width: 100%"
              @update:value="(v: number | null) => { getForm().sort_order = v ?? 0 }"
            />
          </NFormItem>
          <NFormItem :label="t('adminSpots.form.qrCode')" path="qr_code">
            <NInput
              :value="getForm().qr_code"
              :placeholder="t('adminSpots.placeholders.qrCode')"
              @update:value="(v: string) => { getForm().qr_code = v }"
            />
          </NFormItem>

          <NFormItem :label="t('adminSpots.form.qrIntro')" path="qr_intro_text">
            <NInput
              :value="getForm().qr_intro_text"
              type="textarea"
              :rows="2"
              :placeholder="t('adminSpots.placeholders.qrIntro')"
              @update:value="(v: string) => { getForm().qr_intro_text = v }"
            />
          </NFormItem>

          <NFormItem :label="t('adminSpots.form.qrEnabled')" path="qr_enabled">
            <NSwitch
              :value="getForm().qr_enabled"
              @update:value="(v: boolean) => { getForm().qr_enabled = v }"
            />
            <span style="margin-left:8px;font-size:12px;color:rgba(255,255,255,.4)">
              {{ getForm().qr_enabled ? t('adminSpots.switches.qrEnabled') : t('adminSpots.switches.qrDisabled') }}
            </span>
          </NFormItem>

          <NFormItem :label="t('adminSpots.form.geofenceEnabled')" path="geofence_enabled">
            <NSwitch
              :value="getForm().geofence_enabled"
              @update:value="(v: boolean) => { getForm().geofence_enabled = v }"
            />
            <span style="margin-left:8px;font-size:12px;color:rgba(255,255,255,.4)">
              {{ getForm().geofence_enabled ? t('adminSpots.switches.geofenceEnabled') : t('adminSpots.switches.geofenceDisabled') }}
            </span>
          </NFormItem>

          <div class="coord-row">
            <NFormItem :label="t('adminSpots.form.geofenceRadius')" path="geofence_radius_m" class="coord-item">
              <NInputNumber
                :value="getForm().geofence_radius_m"
                :min="20"
                :max="1000"
                :precision="0"
                style="width: 100%"
                @update:value="(v: number | null) => { getForm().geofence_radius_m = v ?? 100 }"
              />
            </NFormItem>
            <NFormItem :label="t('adminSpots.form.geofenceCooldown')" path="geofence_cooldown_minutes" class="coord-item">
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

          <NFormItem :label="t('adminSpots.form.geofenceIntro')" path="geofence_intro_text">
            <NInput
              :value="getForm().geofence_intro_text"
              type="textarea"
              :rows="2"
              :placeholder="t('adminSpots.placeholders.geofenceIntro')"
              @update:value="(v: string) => { getForm().geofence_intro_text = v }"
            />
          </NFormItem>
        </NForm>

        <template #footer>
          <NSpace justify="end">
            <NButton @click="closeDrawer">{{ t('adminSpots.actions.cancel') }}</NButton>
            <NButton type="primary" :loading="saving" @click="onSaveClick">
              {{ isEditing ? t('adminSpots.actions.save') : t('adminSpots.actions.submitCreate') }}
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
