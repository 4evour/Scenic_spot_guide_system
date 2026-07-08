<script setup lang="ts">
import { onMounted, computed, ref, h } from 'vue'
import type { Ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  NButton,
  NDataTable,
  NModal,
  NForm,
  NFormItem,
  NInput,
  NSelect,
  NTag,
  NSpace,
  type DataTableColumns,
  type FormInst,
  type FormRules,
} from 'naive-ui'
import { useCrudTable } from '../composables/useCrudTable'

type User = {
  [key: string]: unknown
  id: string
  username: string
  email: string
  role: string
  created_at: string
  updated_at: string
  password?: string
}

const { t, locale } = useI18n()
const formRef = ref<FormInst | null>(null)

const crud = useCrudTable<User>({
  listApi: '/admin/users',
  saveApi: (data, edit) => ({
    path: edit ? `/admin/users/${data.id}` : '/admin/users',
    method: edit ? 'PUT' : 'POST',
  }),
  deleteApi: (id) => ({ path: `/admin/users/${id}` }),
  idField: 'id',
  defaultForm: () => ({
    username: '',
    email: '',
    role: 'visitor',
    password: '',
  }),
})

const { loading, saving, total, pagination, drawerVisible, isEditing, fetchData, openCreate, openEdit, closeDrawer, handleSave, handleDelete } = crud

const tableData = crud.tableData as Ref<User[]>
const formData = crud.formData as Ref<Partial<User>>

const roleOptions = computed(() => [
  { label: t('adminUsers.roles.admin'), value: 'admin' },
  { label: t('adminUsers.roles.visitor'), value: 'visitor' },
])

const passwordRule = computed(() => ({
  required: !isEditing.value,
  message: t('adminUsers.validation.passwordRequired'),
  trigger: 'blur' as const,
}))

function validatePassword(_rule: unknown, value: string | undefined): boolean {
  if (isEditing.value && !value) return true
  return /^(?=.*[a-z])(?=.*[A-Z])(?=.*\d).{8,128}$/.test(value || '')
}

const formRules = computed<FormRules>(() => ({
  username: [
    { required: true, message: t('adminUsers.validation.usernameRequired'), trigger: 'blur' },
    { min: 3, max: 32, message: t('adminUsers.validation.usernameLength'), trigger: 'blur' },
    { pattern: /^[a-zA-Z0-9_]+$/, message: t('adminUsers.validation.usernameLength'), trigger: 'blur' },
  ],
  email: [
    { type: 'email', message: t('adminUsers.validation.emailInvalid'), trigger: 'blur' },
  ],
  password: [
    passwordRule.value,
    { validator: validatePassword, message: t('adminUsers.validation.passwordPolicy'), trigger: 'blur' },
  ],
  role: [
    { required: true, message: t('adminUsers.validation.roleRequired'), trigger: 'change' },
  ],
}))

const modalTitle = computed(() => isEditing.value ? t('adminUsers.drawer.editTitle') : t('adminUsers.drawer.createTitle'))

const columns = computed<DataTableColumns<User>>(() => [
  { title: t('adminUsers.columns.username'), key: 'username', ellipsis: { tooltip: true } },
  { title: t('adminUsers.columns.email'), key: 'email', ellipsis: { tooltip: true } },
  {
    title: t('adminUsers.columns.role'),
    key: 'role',
    width: 100,
    render(row) {
      const isAdmin = row.role === 'admin'
      return h(
        NTag,
        {
          type: isAdmin ? 'info' : 'success',
          bordered: false,
          size: 'small',
        },
        { default: () => (isAdmin ? t('adminUsers.roles.admin') : t('adminUsers.roles.visitor')) },
      )
    },
  },
  {
    title: t('adminUsers.columns.createdAt'),
    key: 'created_at',
    width: 180,
    render(row) {
      if (!row.created_at) return '-'
      return new Date(row.created_at).toLocaleString(locale.value, {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
      })
    },
  },
  {
    title: t('adminUsers.columns.actions'),
    key: 'actions',
    width: 160,
    render(row) {
      return h(NSpace, { size: 'small' }, {
        default: () => [
          h(
            NButton,
            {
              size: 'small',
              quaternary: true,
              type: 'info',
              onClick: () => openEdit(row),
            },
            { default: () => t('adminUsers.actions.edit') },
          ),
          h(
            NButton,
            {
              size: 'small',
              quaternary: true,
              type: 'error',
              onClick: () => handleDelete(row),
            },
            { default: () => t('adminUsers.actions.delete') },
          ),
        ],
      })
    },
  },
])

async function handleFormSubmit() {
  try {
    await formRef.value?.validate()
  } catch {
    return
  }
  await handleSave()
}

onMounted(fetchData)
</script>

<template>
  <div class="admin-users">
    <div class="page-header">
      <div>
        <h2>{{ t('adminUsers.title') }}</h2>
        <p class="page-subtitle">{{ t('adminUsers.subtitle') }}</p>
      </div>
      <NButton type="primary" @click="openCreate">
        {{ t('adminUsers.actions.create') }}
      </NButton>
    </div>

    <div class="table-card">
      <NDataTable
        :columns="columns"
        :data="tableData"
        :loading="loading"
        :pagination="{ ...pagination, itemCount: total }"
        :bordered="false"
        :single-line="false"
        striped
        remote
      />
    </div>

    <NModal
      v-model:show="drawerVisible"
      :title="modalTitle"
      preset="card"
      :style="{ width: '520px' }"
      :mask-closable="false"
      :close-on-esc="true"
    >
      <NForm
        ref="formRef"
        :model="formData"
        :rules="formRules"
        label-placement="left"
        label-width="80"
        require-mark-placement="right-hanging"
      >
        <NFormItem :label="t('adminUsers.form.username')" path="username">
          <NInput
            v-model:value="formData.username"
            :placeholder="t('adminUsers.placeholders.username')"
            :maxlength="32"
          />
        </NFormItem>
        <NFormItem :label="t('adminUsers.form.email')" path="email">
          <NInput
            v-model:value="formData.email"
            :placeholder="t('adminUsers.placeholders.email')"
          />
        </NFormItem>
        <NFormItem :label="t('adminUsers.form.password')" path="password">
          <NInput
            v-model:value="formData.password"
            type="password"
            show-password-on="click"
            :placeholder="isEditing ? t('adminUsers.placeholders.passwordEdit') : t('adminUsers.placeholders.password')"
          />
        </NFormItem>
        <NFormItem :label="t('adminUsers.form.role')" path="role">
          <NSelect
            v-model:value="formData.role"
            :options="roleOptions"
            :placeholder="t('adminUsers.placeholders.role')"
          />
        </NFormItem>
      </NForm>
      <template #footer>
        <NSpace justify="end">
          <NButton @click="closeDrawer">{{ t('adminUsers.actions.cancel') }}</NButton>
          <NButton type="primary" :loading="saving" @click="handleFormSubmit">
            {{ isEditing ? t('adminUsers.actions.save') : t('adminUsers.actions.create') }}
          </NButton>
        </NSpace>
      </template>
    </NModal>
  </div>
</template>

<style scoped>
.admin-users {
  padding: 0;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 24px;
}

.page-header h2 {
  font-size: 18px;
  font-weight: 600;
  color: rgba(255, 255, 255, 0.88);
  margin: 0 0 4px;
}

.page-subtitle {
  font-size: 13px;
  color: rgba(255, 255, 255, 0.45);
  margin: 0;
}

.table-card {
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid rgba(255, 255, 255, 0.06);
  border-radius: 14px;
  padding: 20px;
}
</style>
