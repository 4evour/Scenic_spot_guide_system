import { ref, reactive, type Ref } from 'vue'
import { useI18n } from 'vue-i18n'
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
  const { t } = useI18n()
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
      let items: T[]
      if (Array.isArray(data)) {
        items = data as T[]
        total.value = items.length
      } else {
        items = (data.list || []) as T[]
        total.value = data.total || 0
      }
      tableData.value = items
      pagination.page = params.page
    } catch (error) {
      message.error(error instanceof Error ? error.message : t('common.loadFailed'))
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
      const body = formData.value as Record<string, unknown>
      await apiFetch(path, { method, body: JSON.stringify(body) })
      message.success(isEditing.value ? t('common.updateSuccess') : t('common.createSuccess'))
      closeDrawer()
      await fetchData()
    } catch (error) {
      message.error(error instanceof Error ? error.message : t('common.saveFailed'))
    } finally {
      saving.value = false
    }
  }

  function handleDelete(row: T) {
    if (!options.deleteApi) return
    const id = String(row[idField] || '')
    dialog.warning({
      title: t('common.confirmDelete'),
      content: t('common.deleteWarning'),
      positiveText: t('common.delete'),
      negativeText: t('common.cancel'),
      onPositiveClick: async () => {
        try {
          const { path } = options.deleteApi!(id)
          await apiFetch(path, { method: 'DELETE' })
          message.success(t('common.deleteSuccess'))
          await fetchData()
        } catch (error) {
          message.error(error instanceof Error ? error.message : t('common.deleteFailed'))
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
