<template>
  <UContainer class="py-6 pb-20 lg:pb-6 space-y-6">
    <div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between animate-fade-in-up">
      <div>
        <h2 class="text-2xl font-bold text-slate-900 dark:text-white">Policies</h2>
        <p class="text-sm text-slate-500 dark:text-slate-400 mt-0.5">Define expiration rules per product</p>
      </div>
      <UButton
        color="primary"
        icon="i-lucide-plus"
        size="md"
        class="transition-app shrink-0"
        @click="openCreate"
      >
        Create policy
      </UButton>
    </div>

    <UAlert v-if="loadError" color="error" variant="subtle" :title="loadError" class="animate-fade-in" />

    <UCard class="shadow-app border-0 ring-1 ring-slate-200/80 dark:ring-slate-800/80 overflow-hidden">
      <div class="flex flex-col gap-3 sm:flex-row sm:items-center border-b border-slate-200/80 dark:border-slate-800/80 pb-4 mb-4">
        <UInput
          v-model="searchQuery"
          icon="i-lucide-search"
          placeholder="Search by name or product..."
          class="sm:flex-1"
        />
        <USelectMenu
          v-model="productFilter"
          :items="productFilterItems"
          value-key="value"
          label-key="label"
          placeholder="All products"
          class="sm:w-56"
        />
      </div>

      <div v-if="loading" class="space-y-3">
        <div v-for="i in 4" :key="i" class="h-12 rounded-lg animate-shimmer" />
      </div>

      <div
        v-else-if="filteredPolicies.length === 0"
        class="flex flex-col items-center justify-center py-16 px-4 text-center"
      >
        <h3 class="text-lg font-semibold text-slate-900 dark:text-white mb-1">
          {{ policies.length === 0 ? 'No policies yet' : 'No matching policies' }}
        </h3>
        <p class="text-sm text-slate-500 dark:text-slate-400 mb-6">
          Create a policy to define how licenses expire.
        </p>
        <UButton v-if="policies.length === 0" color="primary" icon="i-lucide-plus" @click="openCreate">
          Create policy
        </UButton>
      </div>

      <UTable
        v-else
        :columns="columns"
        :data="filteredPolicies"
        class="[&_tbody_tr]:transition-app [&_tbody_tr:hover]:bg-slate-50 dark:[&_tbody_tr:hover]:bg-slate-800/30 [&_tbody_tr]:cursor-pointer"
        @select="(_e, row) => openDetails(row.original)"
      >
        <template #name-cell="{ row }">
          <span class="font-medium text-slate-900 dark:text-white">{{ row.original.name }}</span>
        </template>

        <template #product_name-cell="{ row }">
          <span>{{ row.original.product_name }}</span>
        </template>

        <template #description-cell="{ row }">
          <UTooltip
            v-if="row.original.description"
            :text="row.original.description"
          >
            <span class="block max-w-md truncate text-slate-600 dark:text-slate-400">
              {{ row.original.description }}
            </span>
          </UTooltip>
          <span v-else class="text-slate-500">—</span>
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
    </UCard>

    <PolicyFormModal v-model:open="showForm" :policy="editingPolicy" @saved="fetchPolicies" />

    <DetailsModal
      v-model:open="showDetails"
      :title="detailsPolicy?.name ?? 'Policy details'"
      icon="i-lucide-shield"
      icon-bg-class="bg-indigo-100 dark:bg-indigo-900/40"
      icon-class="text-indigo-600 dark:text-indigo-400"
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
import type { DetailItem, ExpirationBasis, Policy, Product } from '~/types'

type PolicyRow = Policy & { product_name: string }

definePageMeta({
  middleware: 'auth'
})

const { listPolicies, listProducts, deletePolicy } = useApi()

const policies = ref<PolicyRow[]>([])
const products = ref<Product[]>([])
const loading = ref(true)
const loadError = ref('')
const searchQuery = ref('')
const productFilter = ref<string | null>(null)
const showForm = ref(false)
const showDetails = ref(false)
const editingPolicy = ref<Policy | null>(null)
const detailsPolicy = ref<PolicyRow | null>(null)
const showDeleteConfirm = ref(false)
const deleteTarget = ref<Policy | null>(null)
const actionId = ref<string | null>(null)
const deleting = ref(false)

const columns = [
  { accessorKey: 'name', header: 'Name' },
  { accessorKey: 'product_name', header: 'Product' },
  { accessorKey: 'description', header: 'Description' },
  { accessorKey: 'duration', header: 'Duration' },
  { accessorKey: 'expiration_basis', header: 'Basis' },
  { accessorKey: 'grace_period_days', header: 'Grace (days)' },
  { id: 'actions', header: '' }
]

const productFilterItems = computed(() => [
  { label: 'All products', value: null },
  ...products.value.map((product) => ({ label: product.name, value: product.id }))
])

const formatBasis = (basis: ExpirationBasis) =>
  basis === 'on_first_validation' ? 'First validation' : 'Creation'

const formatDuration = (policy: Policy) => {
  if (policy.duration_days == null) {
    return 'Perpetual'
  }
  return `${policy.duration_days} days`
}

const filteredPolicies = computed(() => {
  const query = searchQuery.value.trim().toLowerCase()
  return policies.value.filter((policy) => {
    if (productFilter.value && policy.product_id !== productFilter.value) {
      return false
    }
    if (!query) {
      return true
    }
    return (
      policy.name.toLowerCase().includes(query)
      || policy.product_name.toLowerCase().includes(query)
    )
  })
})

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

const openDetails = (policy: PolicyRow) => {
  detailsPolicy.value = policy
  showDetails.value = true
}

const openDelete = (policy: Policy) => {
  deleteTarget.value = policy
  showDeleteConfirm.value = true
}

const getActionItems = (policy: Policy): DropdownMenuItem[][] => [
  [
    { label: 'View details', icon: 'i-lucide-info', onSelect: () => openDetails(policy as PolicyRow) },
    { label: 'Edit', icon: 'i-lucide-pencil', onSelect: () => openEdit(policy) }
  ],
  [
    { label: 'Delete', icon: 'i-lucide-trash-2', color: 'error', onSelect: () => openDelete(policy) }
  ]
]

const fetchPolicies = async () => {
  loading.value = true
  loadError.value = ''
  try {
    const [policyList, productList] = await Promise.all([
      listPolicies(),
      listProducts()
    ])
    products.value = productList
    const productMap = new Map(productList.map((product) => [product.id, product.name]))
    policies.value = policyList.map((policy) => ({
      ...policy,
      product_name: productMap.get(policy.product_id) ?? 'Unknown'
    }))
  } catch {
    loadError.value = 'Failed to load policies'
  } finally {
    loading.value = false
  }
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
    await fetchPolicies()
  } catch {
    loadError.value = 'Failed to delete policy. It may still be assigned to licenses.'
  } finally {
    actionId.value = null
    deleting.value = false
    deleteTarget.value = null
  }
}

onMounted(fetchPolicies)
</script>
