import type { Policy, Product } from '~/types'

interface SelectItem {
  label: string
  value: string
}

export const useServerSelect = () => {
  const { listProducts, listPolicies } = useApi()

  const createProductSelect = () => {
    const itemsState = ref<SelectItem[]>([])
    const loading = ref(false)
    let requestId = 0
    let debounceTimer: ReturnType<typeof setTimeout> | undefined

    const clearItems = () => {
      itemsState.value = []
    }

    const fetchItems = async (search = '') => {
      const currentRequest = ++requestId
      loading.value = true
      try {
        const result = await listProducts({ search: search || undefined, page_size: 25 })
        if (currentRequest !== requestId) {
          return
        }
        itemsState.value = result.items.map((product: Product) => ({
          label: product.name,
          value: product.id
        }))
      } finally {
        if (currentRequest === requestId) {
          loading.value = false
        }
      }
    }

    const onSearch = (value: string) => {
      if (debounceTimer) {
        clearTimeout(debounceTimer)
      }
      debounceTimer = setTimeout(() => fetchItems(value), 300)
    }

    onMounted(() => fetchItems(''))
    onUnmounted(() => {
      if (debounceTimer) {
        clearTimeout(debounceTimer)
      }
    })

    return reactive({
      items: computed(() => itemsState.value),
      loading,
      fetchItems,
      onSearch,
      clearItems
    })
  }

  const createPolicySelect = (productId: Ref<string | null | undefined>) => {
    const itemsState = ref<SelectItem[]>([])
    const loading = ref(false)
    let requestId = 0
    let debounceTimer: ReturnType<typeof setTimeout> | undefined

    const clearItems = () => {
      itemsState.value = []
    }

    const fetchItems = async (search = '') => {
      const currentRequest = ++requestId
      loading.value = true
      try {
        const result = await listPolicies({
          search: search || undefined,
          product_id: productId.value || undefined,
          page_size: 25
        })
        if (currentRequest !== requestId) {
          return
        }
        itemsState.value = result.items.map((policy: Policy) => ({
          label: policy.name,
          value: policy.id
        }))
      } finally {
        if (currentRequest === requestId) {
          loading.value = false
        }
      }
    }

    const onSearch = (value: string) => {
      if (debounceTimer) {
        clearTimeout(debounceTimer)
      }
      debounceTimer = setTimeout(() => fetchItems(value), 300)
    }

    watch(productId, () => {
      clearItems()
      if (productId.value) {
        fetchItems('')
      }
    })

    onMounted(() => {
      if (productId.value) {
        fetchItems('')
      }
    })
    onUnmounted(() => {
      if (debounceTimer) {
        clearTimeout(debounceTimer)
      }
    })

    return reactive({
      items: computed(() => itemsState.value),
      loading,
      fetchItems,
      onSearch,
      clearItems
    })
  }

  return { createProductSelect, createPolicySelect }
}
