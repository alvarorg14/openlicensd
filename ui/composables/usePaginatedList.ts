import type { Paginated } from '~/types'

type SortOrder = 'asc' | 'desc'

interface UsePaginatedListOptions<T, P extends Record<string, unknown>> {
  fetcher: (params: P & { page: number, page_size: number, search?: string, sort?: string, order?: SortOrder }) => Promise<Paginated<T>>
  defaultSort?: string
  defaultOrder?: SortOrder
  pageSize?: number
  debounceMs?: number
  initialFilters?: Partial<P>
}

export const usePaginatedList = <T, P extends Record<string, unknown> = Record<string, never>>(
  options: UsePaginatedListOptions<T, P>
) => {
  const page = ref(1)
  const pageSize = ref(options.pageSize ?? 25)
  const search = ref('')
  const debouncedSearch = ref('')
  const sort = ref(options.defaultSort ?? 'created_at')
  const order = ref<SortOrder>(options.defaultOrder ?? 'desc')
  const filters = ref({ ...(options.initialFilters ?? {}) }) as Ref<Partial<P>>

  const items = ref<T[]>([]) as Ref<T[]>
  const total = ref(0)
  const totalPages = ref(0)
  const loading = ref(true)
  const error = ref('')

  let requestId = 0
  let debounceTimer: ReturnType<typeof setTimeout> | undefined

  const buildParams = () => ({
    ...filters.value,
    page: page.value,
    page_size: pageSize.value,
    search: debouncedSearch.value || undefined,
    sort: sort.value,
    order: order.value
  } as P & { page: number, page_size: number, search?: string, sort?: string, order?: SortOrder })

  const refresh = async () => {
    const currentRequest = ++requestId
    loading.value = true
    error.value = ''
    try {
      const result = await options.fetcher(buildParams())
      if (currentRequest !== requestId) {
        return
      }
      items.value = result.items
      total.value = result.total
      totalPages.value = result.total_pages
      page.value = result.page
      if (pageSize.value !== result.page_size) {
        pageSize.value = result.page_size
      }
    } catch {
      if (currentRequest !== requestId) {
        return
      }
      error.value = 'Failed to load data'
      items.value = []
      total.value = 0
      totalPages.value = 0
    } finally {
      if (currentRequest === requestId) {
        loading.value = false
      }
    }
  }

  const setSearch = (value: string) => {
    search.value = value
    if (debounceTimer) {
      clearTimeout(debounceTimer)
    }
    debounceTimer = setTimeout(() => {
      debouncedSearch.value = value.trim()
      page.value = 1
      refresh()
    }, options.debounceMs ?? 300)
  }

  const setFilter = <K extends keyof P>(key: K, value: P[K] | undefined) => {
    const next = { ...filters.value }
    if (value === undefined || value === null || value === '') {
      const { [key]: _removed, ...rest } = next
      filters.value = rest as Partial<P>
    } else {
      filters.value = { ...next, [key]: value }
    }
    page.value = 1
    refresh()
  }

  const setPage = (value: number) => {
    page.value = value
    refresh()
  }

  const setSort = (column: string, nextOrder?: SortOrder) => {
    if (sort.value === column && !nextOrder) {
      order.value = order.value === 'asc' ? 'desc' : 'asc'
    } else {
      sort.value = column
      if (nextOrder) {
        order.value = nextOrder
      }
    }
    page.value = 1
    refresh()
  }

  const sorting = computed({
    get: () => [{ id: sort.value, desc: order.value === 'desc' }],
    set: (value: { id: string, desc: boolean }[]) => {
      const next = value[0]
      if (!next) {
        return
      }
      sort.value = next.id
      order.value = next.desc ? 'desc' : 'asc'
      page.value = 1
      refresh()
    }
  })

  watch(pageSize, () => {
    page.value = 1
    refresh()
  })

  onMounted(refresh)
  onUnmounted(() => {
    if (debounceTimer) {
      clearTimeout(debounceTimer)
    }
  })

  return {
    page,
    pageSize,
    search,
    sort,
    order,
    filters,
    items,
    total,
    totalPages,
    loading,
    error,
    sorting,
    refresh,
    setSearch,
    setFilter,
    setPage,
    setSort
  }
}
