<script setup lang="ts">
import { onMounted, computed, ref, h } from 'vue'
import type { Ref } from 'vue'
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
  NEmpty,
  NAlert,
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

const formRef = ref<FormInst | null>(null)
const apiReady = ref(true)

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

const roleOptions = [
  { label: '管理员', value: 'admin' },
  { label: '游客', value: 'visitor' },
]

const passwordRule = computed(() => ({
  required: !isEditing.value,
  message: '请输入密码',
  trigger: 'blur' as const,
}))

function validatePassword(_rule: unknown, value: string | undefined): boolean {
  if (isEditing.value && !value) return true
  return /^(?=.*[a-z])(?=.*[A-Z])(?=.*\d).{8,128}$/.test(value || '')
}

const formRules = computed<FormRules>(() => ({
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' },
    { min: 2, max: 32, message: '用户名长度为 2-32 个字符', trigger: 'blur' },
  ],
  email: [
    { type: 'email', message: '请输入有效的邮箱地址', trigger: 'blur' },
  ],
  password: [
    passwordRule.value,
    { validator: validatePassword, message: '密码需 8-128 位且包含大小写字母和数字', trigger: 'blur' },
  ],
  role: [
    { required: true, message: '请选择角色', trigger: 'change' },
  ],
}))

const modalTitle = computed(() => isEditing.value ? '编辑用户' : '新增用户')

const columns: DataTableColumns<User> = [
  { title: '用户名', key: 'username', ellipsis: { tooltip: true } },
  { title: '邮箱', key: 'email', ellipsis: { tooltip: true } },
  {
    title: '角色',
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
        { default: () => (isAdmin ? '管理员' : '游客') },
      )
    },
  },
  {
    title: '创建时间',
    key: 'created_at',
    width: 180,
    render(row) {
      if (!row.created_at) return '-'
      return new Date(row.created_at).toLocaleString('zh-CN', {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
      })
    },
  },
  {
    title: '操作',
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
            { default: () => '编辑' },
          ),
          h(
            NButton,
            {
              size: 'small',
              quaternary: true,
              type: 'error',
              onClick: () => handleDelete(row),
            },
            { default: () => '删除' },
          ),
        ],
      })
    },
  },
]

async function handleFormSubmit() {
  try {
    await formRef.value?.validate()
  } catch {
    return
  }
  await handleSave()
}

async function loadUsers() {
  try {
    await fetchData()
    apiReady.value = true
  } catch (error) {
    const msg = error instanceof Error ? error.message : ''
    if (msg.includes('404') || msg.includes('Not Found') || msg.includes('请求失败 (404)')) {
      apiReady.value = false
    }
  }
}

onMounted(loadUsers)
</script>

<template>
  <div class="admin-users">
    <div class="page-header">
      <div>
        <h2>用户管理</h2>
        <p class="page-subtitle">管理系统的用户账号和权限</p>
      </div>
      <NButton type="primary" @click="openCreate">
        新增用户
      </NButton>
    </div>

    <NAlert
      v-if="!apiReady"
      type="warning"
      :show-icon="true"
      class="api-alert"
    >
      用户管理 API 尚未就绪，请先通过命令行创建管理员账号，或联系后端开发人员启用用户管理接口。
    </NAlert>

    <div class="table-card">
      <NDataTable
        v-if="apiReady"
        :columns="columns"
        :data="tableData"
        :loading="loading"
        :pagination="{ ...pagination, itemCount: total }"
        :bordered="false"
        :single-line="false"
        striped
        remote
      />
      <NEmpty v-else description="API 未就绪，无法加载用户列表" class="empty-placeholder" />
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
        <NFormItem label="用户名" path="username">
          <NInput
            v-model:value="formData.username"
            placeholder="请输入用户名"
            :maxlength="32"
          />
        </NFormItem>
        <NFormItem label="邮箱" path="email">
          <NInput
            v-model:value="formData.email"
            placeholder="请输入邮箱地址"
          />
        </NFormItem>
        <NFormItem label="密码" path="password">
          <NInput
            v-model:value="formData.password"
            type="password"
            show-password-on="click"
            :placeholder="isEditing ? '留空则不修改密码' : '请输入密码'"
          />
        </NFormItem>
        <NFormItem label="角色" path="role">
          <NSelect
            v-model:value="formData.role"
            :options="roleOptions"
            placeholder="请选择角色"
          />
        </NFormItem>
      </NForm>
      <template #footer>
        <NSpace justify="end">
          <NButton @click="closeDrawer">取消</NButton>
          <NButton type="primary" :loading="saving" @click="handleFormSubmit">
            {{ isEditing ? '保存' : '创建' }}
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

.api-alert {
  margin-bottom: 16px;
}

.table-card {
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid rgba(255, 255, 255, 0.06);
  border-radius: 14px;
  padding: 20px;
}

.empty-placeholder {
  padding: 64px 0;
}
</style>
