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
      const data = await apiFetch<ListResult<T>>(`${options.listApi}?page=${params.page}&page_size=${params.pageSize}`)
      tableData.value = data.list || []
      total.value = data.total || 0
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
      await apiFetch(path, { method, body: JSON.stringify(formData.value) })
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
