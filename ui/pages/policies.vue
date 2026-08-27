<template>
  <UContainer class="py-6 pb-20 lg:pb-6 space-y-6">
    <div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between animate-fade-in-up">
      <div>
        <h2 class="text-2xl font-medium tracking-brand text-highlighted">Policies</h2>
        <p class="text-sm text-muted mt-0.5">Define expiration rules per product</p>
      </div>
      <UButton
        v-if="canWrite"
        color="primary"
        icon="i-lucide-plus"
        size="md"
        class="transition-app shrink-0"
        @click="openCreate"
      >
        Create policy
      </UButton>
    </div>

    <UAlert v-if="error" color="error" variant="subtle" :title="error" class="animate-fade-in" />

    <UCard class="shadow-app border-0 ring-1 ring-default overflow-hidden">
      <div class="flex flex-col gap-3 sm:flex-row sm:items-center border-b border-default pb-4 mb-4">
        <UInput
          :model-value="search"
          icon="i-lucide-search"
          placeholder="Search by name or product..."
          class="sm:flex-1"
          @update:model-value="setSearch"
        />
        <USelectMenu
          v-model="productFilter"
          v-model:search-term="productSearchTerm"
          :items="productFilterItems"
          value-key="value"
          label-key="label"
          placeholder="All products"
          searchable
          class="sm:w-56"
          :loading="productSelect.loading"
          @update:search-term="productSelect.onSearch"
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
          {{ total === 0 && !search && !productFilter ? 'No policies yet' : 'No matching policies' }}
        </h3>
        <p class="text-sm text-muted mb-6">
          Create a policy to define how licenses expire.
        </p>
        <UButton v-if="total === 0 && !search && !productFilter && canWrite" color="primary" icon="i-lucide-plus" @click="openCreate">
          Create policy
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

          <template #product_name-cell="{ row }">
            <span>{{ row.original.product_name }}</span>
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

          <template #duration-cell="{ row }">
            <span>{{ formatDuration(row.original) }}</span>
          </template>

          <template #expiration_basis-cell="{ row }">
            <span>{{ formatBasis(row.original.expiration_basis) }}</span>
          </template>

          <template #grace_period_days-cell="{ row }">
            <span>{{ row.original.grace_period_days }}</span>
          </template>

          <template #max_activations-cell="{ row }">
            <span>{{ formatMaxActivations(row.original) }}</span>
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

    <PolicyFormModal v-model:open="showForm" :policy="editingPolicy" @saved="refresh" />

    <DetailsModal
      v-model:open="showDetails"
      :title="detailsPolicy?.name ?? 'Policy details'"
      icon="i-lucide-shield"
      icon-bg-class="bg-brand-100 dark:bg-brand-900/40"
      icon-class="text-brand-600 dark:text-brand-400"
      :items="detailsItems"
    />

    <ConfirmModal
      v-model:open="showDeleteConfirm"
      title="Delete policy"
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
import type { DetailItem, ExpirationBasis, Policy } from '~/types'

definePageMeta({
  middleware: 'auth'
})

const { listPolicies, deletePolicy } = useApi()
const { canWrite } = useAuth()
const { createProductSelect } = useServerSelect()
const productSelect = createProductSelect()

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
  setPage,
  setFilter
} = usePaginatedList<Policy, { product_id?: string }>({
  fetcher: (params) => listPolicies(params)
})

const productFilter = ref<string | null>(null)
const productSearchTerm = ref('')
const showForm = ref(false)
const showDetails = ref(false)
const editingPolicy = ref<Policy | null>(null)
const detailsPolicy = ref<Policy | null>(null)
const showDeleteConfirm = ref(false)
const deleteTarget = ref<Policy | null>(null)
const actionId = ref<string | null>(null)
const deleting = ref(false)

const productFilterItems = computed(() => [
  { label: 'All products', value: null },
  ...productSelect.items
])

const columns = [
  { accessorKey: 'name', header: 'Name', enableSorting: true },
  { accessorKey: 'product_name', header: 'Product', enableSorting: true },
  { accessorKey: 'description', header: 'Description' },
  { accessorKey: 'duration', header: 'Duration' },
  { accessorKey: 'expiration_basis', header: 'Basis' },
  { accessorKey: 'grace_period_days', header: 'Grace (days)', enableSorting: true },
  { accessorKey: 'max_activations', header: 'Max activations', enableSorting: true },
  { id: 'actions', header: '' }
]

watch(productFilter, (value) => {
  setFilter('product_id', value ?? undefined)
})

const formatBasis = (basis: ExpirationBasis) =>
  basis === 'on_first_validation' ? 'First validation' : 'Creation'

const formatDuration = (policy: Policy) => {
  if (policy.duration_days == null) {
    return 'Perpetual'
  }
  return `${policy.duration_days} days`
}

const formatMaxActivations = (policy: Policy) => {
  if (policy.max_activations == null) {
    return 'Unlimited'
  }
  return String(policy.max_activations)
}

const detailsItems = computed((): DetailItem[] => {
  const policy = detailsPolicy.value
  if (!policy) {
    return []
  }
  return [
    { label: 'Name', value: policy.name },
    { label: 'Product', value: policy.product_name },
    { label: 'Description', value: policy.description || '—', multiline: true },
    { label: 'Duration', value: formatDuration(policy) },
    { label: 'Basis', value: formatBasis(policy.expiration_basis) },
    { label: 'Grace period', value: `${policy.grace_period_days} days` },
    { label: 'Max activations', value: formatMaxActivations(policy) },
    { label: 'Created', value: formatDate(policy.created_at) }
  ]
})

const formatDate = (value: string) => new Date(value).toLocaleString()

const deleteConfirmDescription = computed(() => {
  const name = deleteTarget.value?.name ?? 'this policy'
  return `Are you sure you want to delete "${name}"? Policies assigned to licenses cannot be deleted.`
})

const openCreate = () => {
  editingPolicy.value = null
  showForm.value = true
}

const openEdit = (policy: Policy) => {
  editingPolicy.value = policy
  showForm.value = true
}

const openDetails = (policy: Policy) => {
  detailsPolicy.value = policy
  showDetails.value = true
}

const openDelete = (policy: Policy) => {
  deleteTarget.value = policy
  showDeleteConfirm.value = true
}

const getActionItems = (policy: Policy): DropdownMenuItem[][] => {
  const menuItems: DropdownMenuItem[] = [
    { label: 'View details', icon: 'i-lucide-info', onSelect: () => openDetails(policy) }
  ]
  if (canWrite.value) {
    menuItems.push({ label: 'Edit', icon: 'i-lucide-pencil', onSelect: () => openEdit(policy) })
    return [
      menuItems,
      [{ label: 'Delete', icon: 'i-lucide-trash-2', color: 'error', onSelect: () => openDelete(policy) }]
    ]
  }
  return [menuItems]
}

const confirmDelete = async () => {
  if (!deleteTarget.value) {
    return
  }
  actionId.value = deleteTarget.value.id
  deleting.value = true
  try {
    await deletePolicy(deleteTarget.value.id)
    showDeleteConfirm.value = false
    await refresh()
  } catch {
    error.value = 'Failed to delete policy. It may still be assigned to licenses.'
  } finally {
    actionId.value = null
    deleting.value = false
    deleteTarget.value = null
  }
}
</script>
