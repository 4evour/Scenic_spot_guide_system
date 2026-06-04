import { ref, reactive, type Ref } from 'vue'
import { useMessage, useDialog } from 'naive-ui'
import { apiFetch } from '../services/api'

interface ListParams {
  page: number
  pageSize: number
  keyword?: string
  [key: string]: unknown
}

interface ListResult<T> {
  list: T[]
  total: number
}

interface CrudOptions<T extends Record<string, unknown>> {
  listApi: string
  saveApi?: (data: Partial<T>, isEdit: boolean) => { path: string; method: string }
  deleteApi?: (id: string) => { path: string }
  idField?: string
  defaultForm: () => Partial<T>
}

/** 将 Go 后端的 PascalCase 字段名转为 camelCase（处理 ID→id, Name→name 等） */
function toCamelCase(obj: Record<string, unknown>): Record<string, unknown> {
  const result: Record<string, unknown> = {}
  for (const [key, value] of Object.entries(obj)) {
    let camel = key
    if (key === key.toUpperCase() && key.length <= 3) {
      camel = key.toLowerCase()
    } else {
      camel = key.charAt(0).toLowerCase() + key.slice(1)
    }
    result[camel] = value
  }
  return result
}

/** 将前端 camelCase 字段名转为 PascalCase 供 Go 后端解析（id→ID, name→Name） */
function toPascalCase(obj: Record<string, unknown>): Record<string, unknown> {
  const special: Record<string, string> = { id: 'ID', url: 'URL', uid: 'UID' }
  const result: Record<string, unknown> = {}
  for (const [key, value] of Object.entries(obj)) {
    const pascal = special[key] || key.charAt(0).toUpperCase() + key.slice(1)
    result[pascal] = value
  }
  return result
}

export function useCrudTable<T extends Record<string, unknown>>(options: CrudOptions<T>) {
  const message = useMessage()
  const dialog = useDialog()

  const loading = ref(false)
  const saving = ref(false)
  const tableData = ref<T[]>([]) as Ref<T[]>
  const total = ref(0)
  const drawerVisible = ref(false)
  const formData = ref<Partial<T>>(options.defaultForm()) as Ref<Partial<T>>
  const isEditing = ref(false)
  const editingId = ref('')

  const pagination = reactive({
    page: 1,
    pageSize: 10,
    showSizePicker: true,
    pageSizes: [10, 20, 50],
    onChange: (page: number) => { pagination.page = page; fetchData() },
    onUpdatePageSize: (size: number) => { pagination.pageSize = size; pagination.page = 1; fetchData() },
  })

  const idField = options.idField || 'id'

  async function fetchData() {
    loading.value = true
    try {
      const params: ListParams = { page: pagination.page, pageSize: pagination.pageSize }
      const data = await apiFetch<ListResult<T> | T[]>(`${options.listApi}?page=${params.page}&page_size=${params.pageSize}`)
      // 兼容两种后端返回格式：分页对象 {list, total} 或普通数组 [...]
      // 同时将 Go 的 PascalCase 字段名转为 camelCase
      let items: T[]
      if (Array.isArray(data)) {
        items = data.map(item => toCamelCase(item as Record<string, unknown>) as T)
        total.value = items.length
      } else {
        items = (data.list || []).map(item => toCamelCase(item as Record<string, unknown>) as T)
        total.value = data.total || 0
      }
      tableData.value = items
      pagination.page = params.page
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载失败')
    } finally {
      loading.value = false
    }
  }

  function openCreate() {
    isEditing.value = false
    editingId.value = ''
    formData.value = options.defaultForm()
    drawerVisible.value = true
  }

  function openEdit(row: T) {
    isEditing.value = true
    editingId.value = String(row[idField] || '')
    formData.value = { ...row }
    drawerVisible.value = true
  }

  function closeDrawer() {
    drawerVisible.value = false
    formData.value = options.defaultForm()
  }

  async function handleSave() {
    if (!options.saveApi) return
    saving.value = true
    try {
      const { path, method } = options.saveApi(formData.value, isEditing.value)
      // 将 camelCase 表单数据转为 PascalCase 供 Go 后端解析
      const body = toPascalCase(formData.value as Record<string, unknown>)
      await apiFetch(path, { method, body: JSON.stringify(body) })
      message.success(isEditing.value ? '更新成功' : '创建成功')
      closeDrawer()
      await fetchData()
    } catch (error) {
      message.error(error instanceof Error ? error.message : '保存失败')
    } finally {
      saving.value = false
    }
  }

  function handleDelete(row: T) {
    if (!options.deleteApi) return
    const id = String(row[idField] || '')
    dialog.warning({
      title: '确认删除',
      content: '删除后不可恢复，确定要删除吗？',
      positiveText: '删除',
      negativeText: '取消',
      onPositiveClick: async () => {
        try {
          const { path } = options.deleteApi!(id)
          await apiFetch(path, { method: 'DELETE' })
          message.success('删除成功')
          await fetchData()
        } catch (error) {
          message.error(error instanceof Error ? error.message : '删除失败')
        }
      },
    })
  }

  return {
    loading,
    saving,
    tableData,
    total,
    pagination,
    drawerVisible,
    formData,
    isEditing,
    editingId,
    fetchData,
    openCreate,
    openEdit,
    closeDrawer,
    handleSave,
    handleDelete,
  }
}
