<template>
  <UContainer class="py-6 pb-20 lg:pb-6 space-y-6">
    <div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between animate-fade-in-up">
      <div>
        <h2 class="text-2xl font-medium tracking-brand text-highlighted">Products</h2>
        <p class="text-sm text-muted mt-0.5">Manage the products your licenses belong to</p>
      </div>
      <UButton
        v-if="canWrite"
        color="primary"
        icon="i-lucide-plus"
        size="md"
        class="transition-app shrink-0"
        @click="openCreate"
      >
        Create product
      </UButton>
    </div>

    <UAlert v-if="error" color="error" variant="subtle" :title="error" class="animate-fade-in" />

    <UCard class="shadow-app border-0 ring-1 ring-default overflow-hidden">
      <div class="flex flex-col gap-3 sm:flex-row sm:items-center border-b border-default pb-4 mb-4">
        <UInput
          :model-value="search"
          icon="i-lucide-search"
          placeholder="Search by name or code..."
          class="sm:flex-1"
          @update:model-value="setSearch"
        />
      </div>

      <div v-if="loading" class="space-y-3">
        <div v-for="i in 4" :key="i" class="h-12 rounded-lg animate-shimmer" />
      </div>

      <div
        v-else-if="items.length === 0"
        class="flex flex-col items-center justify-center py-16 px-4 text-center"
      >
        <h3 class="text-lg font-medium tracking-brand text-highlighted mb-1">
          {{ total === 0 && !search ? 'No products yet' : 'No matching products' }}
        </h3>
        <p class="text-sm text-muted mb-6">
          Create a product before issuing licenses.
        </p>
        <UButton v-if="total === 0 && !search && canWrite" color="primary" icon="i-lucide-plus" @click="openCreate">
          Create product
        </UButton>
      </div>

      <template v-else>
        <UTable
          v-model:sorting="sorting"
          :columns="columns"
          :data="items"
          :sorting-options="{ manualSorting: true }"
          class="[&_tbody_tr]:transition-app [&_tbody_tr:hover]:bg-muted [&_tbody_tr]:cursor-pointer"
          @select="(_e, row) => openDetails(row.original)"
        >
          <template #name-cell="{ row }">
            <span class="font-medium text-highlighted">{{ row.original.name }}</span>
          </template>

          <template #code-cell="{ row }">
            <code class="text-xs font-mono bg-elevated px-2 py-0.5 rounded">
              {{ row.original.code }}
            </code>
          </template>

          <template #description-cell="{ row }">
            <UTooltip
              v-if="row.original.description"
              :text="row.original.description"
            >
              <span class="block max-w-md truncate text-toned">
                {{ row.original.description }}
              </span>
            </UTooltip>
            <span v-else class="text-muted">—</span>
          </template>

          <template #created_at-cell="{ row }">
            <span>{{ formatDate(row.original.created_at) }}</span>
          </template>

          <template #actions-cell="{ row }">
            <UDropdownMenu :items="getActionItems(row.original)">
              <UButton
                color="neutral"
                variant="ghost"
                icon="i-lucide-ellipsis-vertical"
                size="sm"
                :loading="actionId === row.original.id"
              />
            </UDropdownMenu>
          </template>
        </UTable>

        <div v-if="totalPages > 1" class="flex justify-end pt-4 border-t border-default mt-4">
          <UPagination
            :page="page"
            :items-per-page="pageSize"
            :total="total"
            @update:page="setPage"
          />
        </div>
      </template>
    </UCard>

    <ProductFormModal v-model:open="showForm" :product="editingProduct" @saved="refresh" />

    <DetailsModal
      v-model:open="showDetails"
      :title="detailsProduct?.name ?? 'Product details'"
      icon="i-lucide-package"
      icon-bg-class="bg-brand-100 dark:bg-brand-900/40"
      icon-class="text-brand-600 dark:text-brand-400"
      :items="detailsItems"
    />

    <ConfirmModal
      v-model:open="showDeleteConfirm"
      title="Delete product"
      :description="deleteConfirmDescription"
      confirm-label="Delete"
      confirm-color="error"
      :loading="deleting"
      @confirm="confirmDelete"
    />
  </UContainer>
</template>

<script setup lang="ts">
import type { DropdownMenuItem } from '@nuxt/ui'
import type { DetailItem, Product } from '~/types'

definePageMeta({
  middleware: 'auth'
})

const { listProducts, deleteProduct } = useApi()
const { canWrite } = useAuth()

const {
  page,
  pageSize,
  search,
  items,
  total,
  totalPages,
  loading,
  error,
  sorting,
  refresh,
  setSearch,
  setPage
} = usePaginatedList<Product>({
  fetcher: (params) => listProducts(params)
})

const showForm = ref(false)
const showDetails = ref(false)
const editingProduct = ref<Product | null>(null)
const detailsProduct = ref<Product | null>(null)
const showDeleteConfirm = ref(false)
const deleteTarget = ref<Product | null>(null)
const actionId = ref<string | null>(null)
const deleting = ref(false)

const columns = [
  { accessorKey: 'name', header: 'Name', enableSorting: true },
  { accessorKey: 'code', header: 'Code', enableSorting: true },
  { accessorKey: 'description', header: 'Description' },
  { accessorKey: 'created_at', header: 'Created', enableSorting: true },
  { id: 'actions', header: '' }
]

const formatDate = (value: string) => new Date(value).toLocaleString()

const detailsItems = computed((): DetailItem[] => {
  const product = detailsProduct.value
  if (!product) {
    return []
  }
  return [
    { label: 'Name', value: product.name },
    { label: 'Code', value: product.code, mono: true },
    { label: 'Description', value: product.description || '—', multiline: true },
    { label: 'Created', value: formatDate(product.created_at) },
    { label: 'Updated', value: formatDate(product.updated_at) }
  ]
})

const deleteConfirmDescription = computed(() => {
  const name = deleteTarget.value?.name ?? 'this product'
  return `Are you sure you want to delete "${name}"? Products with policies or licenses cannot be deleted.`
})

const openCreate = () => {
  editingProduct.value = null
  showForm.value = true
}

const openEdit = (product: Product) => {
  editingProduct.value = product
  showForm.value = true
}

const openDetails = (product: Product) => {
  detailsProduct.value = product
  showDetails.value = true
}

const openDelete = (product: Product) => {
  deleteTarget.value = product
  showDeleteConfirm.value = true
}

const getActionItems = (product: Product): DropdownMenuItem[][] => {
  const items: DropdownMenuItem[] = [
    { label: 'View details', icon: 'i-lucide-info', onSelect: () => openDetails(product) }
  ]
  if (canWrite.value) {
    items.push({ label: 'Edit', icon: 'i-lucide-pencil', onSelect: () => openEdit(product) })
    return [
      items,
      [{ label: 'Delete', icon: 'i-lucide-trash-2', color: 'error', onSelect: () => openDelete(product) }]
    ]
  }
  return [items]
}

const confirmDelete = async () => {
  if (!deleteTarget.value) {
    return
  }
  actionId.value = deleteTarget.value.id
  deleting.value = true
  try {
    await deleteProduct(deleteTarget.value.id)
    showDeleteConfirm.value = false
    await refresh()
  } catch {
    error.value = 'Failed to delete product. It may still have policies or licenses.'
  } finally {
    actionId.value = null
    deleting.value = false
    deleteTarget.value = null
  }
}
</script>
