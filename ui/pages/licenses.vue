<template>
  <UContainer class="py-6 pb-20 lg:pb-6 space-y-6">
    <div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between animate-fade-in-up">
      <div>
        <h2 class="text-2xl font-medium tracking-brand text-highlighted">Licenses</h2>
        <p class="text-sm text-muted mt-0.5">Manage and monitor your license keys</p>
      </div>
      <UButton
        v-if="canWrite"
        color="primary"
        icon="i-lucide-plus"
        size="md"
        class="transition-app shrink-0"
        @click="showCreate = true"
      >
        Create license
      </UButton>
    </div>

    <div class="grid grid-cols-2 lg:grid-cols-4 gap-4">
      <StatCard
        label="Total"
        :count="stats.total"
        icon="i-lucide-key"
        icon-bg-class="bg-brand-100 dark:bg-brand-900/40"
        icon-class="text-brand-600 dark:text-brand-400"
        stagger-class="stagger-1"
      />
      <StatCard
        label="Active"
        :count="stats.active"
        icon="i-lucide-check-circle"
        icon-bg-class="bg-emerald-100 dark:bg-emerald-900/40"
        icon-class="text-emerald-600 dark:text-emerald-400"
        stagger-class="stagger-2"
      />
      <StatCard
        label="Expired"
        :count="stats.expired"
        icon="i-lucide-clock"
        icon-bg-class="bg-amber-100 dark:bg-amber-900/40"
        icon-class="text-amber-600 dark:text-amber-400"
        stagger-class="stagger-3"
      />
      <StatCard
        label="Revoked"
        :count="stats.revoked"
        icon="i-lucide-ban"
        icon-bg-class="bg-red-100 dark:bg-red-900/40"
        icon-class="text-red-600 dark:text-red-400"
        stagger-class="stagger-4"
      />
    </div>

    <UAlert v-if="error" color="error" variant="subtle" :title="error" class="animate-fade-in" />

    <UCard
      class="shadow-app border-0 ring-1 ring-default animate-fade-in-up stagger-2 overflow-hidden"
    >
      <div class="flex flex-col gap-3 border-b border-default pb-4 mb-4">
        <UInput
          :model-value="search"
          icon="i-lucide-search"
          placeholder="Search by label or key prefix..."
          class="transition-app"
          @update:model-value="setSearch"
        />
        <div class="grid grid-cols-1 sm:grid-cols-3 gap-3">
          <USelect
            v-model="statusFilter"
            :items="statusFilterOptions"
          />
          <USelectMenu
            v-model="productFilter"
            v-model:search-term="productSearchTerm"
            :items="productFilterItems"
            value-key="value"
            label-key="label"
            placeholder="All products"
            searchable
            :loading="productSelect.loading"
            @update:search-term="productSelect.onSearch"
          />
          <USelectMenu
            v-model="policyFilter"
            v-model:search-term="policySearchTerm"
            :items="policyFilterItems"
            value-key="value"
            label-key="label"
            placeholder="All policies"
            searchable
            :loading="policySelect.loading"
            @update:search-term="policySelect.onSearch"
          />
        </div>
      </div>

      <div v-if="loading" class="space-y-3">
        <div v-for="i in 5" :key="i" class="h-12 rounded-lg animate-shimmer" />
      </div>

      <div
        v-else-if="items.length === 0"
        class="flex flex-col items-center justify-center py-16 px-4 text-center animate-fade-in"
      >
        <div class="flex h-14 w-14 items-center justify-center rounded-full bg-elevated mb-4">
          <UIcon name="i-lucide-key" class="h-7 w-7 text-dimmed" />
        </div>
        <h3 class="text-lg font-medium tracking-brand text-highlighted mb-1">
          {{ total === 0 && !hasFilters ? 'No licenses yet' : 'No matching licenses' }}
        </h3>
        <p class="text-sm text-muted mb-6 max-w-sm">
          {{ total === 0 && !hasFilters
            ? 'Create your first license key to get started.'
            : 'Try adjusting your search or filter criteria.' }}
        </p>
        <UButton
          v-if="total === 0 && !hasFilters && canWrite"
          color="primary"
          icon="i-lucide-plus"
          @click="showCreate = true"
        >
          Create license
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
          <template #label-cell="{ row }">
            <UTooltip :text="row.original.label">
              <span class="block max-w-[16rem] truncate font-medium text-highlighted">
                {{ row.original.label }}
              </span>
            </UTooltip>
          </template>

          <template #product_name-cell="{ row }">
            <UTooltip :text="row.original.product_name">
              <span class="block max-w-[10rem] truncate text-toned">
                {{ row.original.product_name }}
              </span>
            </UTooltip>
          </template>

          <template #policy_name-cell="{ row }">
            <UTooltip :text="row.original.policy_name">
              <span class="block max-w-[10rem] truncate text-toned">
                {{ row.original.policy_name }}
              </span>
            </UTooltip>
          </template>

          <template #key_prefix-cell="{ row }">
            <code class="text-xs font-mono bg-elevated px-2 py-0.5 rounded text-toned">
              {{ row.original.key_prefix }}
            </code>
          </template>

          <template #expires_at-cell="{ row }">
            <span v-if="!row.original.expires_at" class="text-muted">Never</span>
            <span v-else class="text-toned">{{ formatDate(row.original.expires_at) }}</span>
          </template>

          <template #revoked-cell="{ row }">
            <UBadge :color="statusColor(row.original)" variant="subtle" class="gap-1.5">
              <span
                class="h-1.5 w-1.5 rounded-full shrink-0"
                :class="statusDotClass(row.original)"
              />
              {{ statusLabel(row.original) }}
            </UBadge>
          </template>

          <template #actions-cell="{ row }">
            <UDropdownMenu :items="getActionItems(row.original)">
              <UButton
                color="neutral"
                variant="ghost"
                icon="i-lucide-ellipsis-vertical"
                size="sm"
                :loading="actionId === row.original.id"
                class="transition-app"
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

    <CreateLicenseModal v-model:open="showCreate" @created="onCreated" />
    <EditLicenseModal
      v-model:open="showEdit"
      :license="editingLicense"
      @updated="onUpdated"
    />
    <LicenseKeyModal v-model:open="showKeyModal" :license-key="createdKey" :label="createdLabel" />

    <DetailsModal
      v-model:open="showDetails"
      :title="detailsTitle"
      icon="i-lucide-key"
      icon-bg-class="bg-brand-100 dark:bg-brand-900/40"
      icon-class="text-brand-600 dark:text-brand-400"
      :items="detailsItems"
    >
      <template #top>
        <div v-if="detailsLicense" class="flex items-center gap-2">
          <span class="font-medium text-highlighted">{{ detailsLicense.label }}</span>
          <UBadge :color="statusColor(detailsLicense)" variant="subtle" class="gap-1.5">
            <span
              class="h-1.5 w-1.5 rounded-full shrink-0"
              :class="statusDotClass(detailsLicense)"
            />
            {{ statusLabel(detailsLicense) }}
          </UBadge>
        </div>
      </template>
    </DetailsModal>

    <ConfirmModal
      v-model:open="showRevokeConfirm"
      title="Revoke license"
      :description="revokeConfirmDescription"
      confirm-label="Revoke"
      confirm-color="error"
      :loading="actionType === 'revoke'"
      @confirm="confirmRevoke"
    />

    <ConfirmModal
      v-model:open="showDeleteConfirm"
      title="Delete license"
      :description="deleteConfirmDescription"
      confirm-label="Delete"
      confirm-color="error"
      :loading="actionType === 'delete'"
      @confirm="confirmDelete"
    />
  </UContainer>
</template>

<script setup lang="ts">
import type { DropdownMenuItem } from '@nuxt/ui'
import type { DetailItem, License, LicenseStats, LicenseStatus } from '~/types'

definePageMeta({
  middleware: 'auth'
})

const {
  listLicenses,
  getLicenseStats,
  revokeLicense,
  activateLicense,
  deleteLicense
} = useApi()
const { canWrite } = useAuth()

const statusFilter = ref<'all' | LicenseStatus>('all')
const productFilter = ref<string | null>(null)
const policyFilter = ref<string | null>(null)
const productSearchTerm = ref('')
const policySearchTerm = ref('')

const { createProductSelect, createPolicySelect } = useServerSelect()
const productSelect = createProductSelect()
const policySelect = createPolicySelect(productFilter)

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
} = usePaginatedList<License, { status?: LicenseStatus, product_id?: string, policy_id?: string }>({
  fetcher: (params) => listLicenses(params)
})

const stats = ref<LicenseStats>({ total: 0, active: 0, expired: 0, revoked: 0 })
const showCreate = ref(false)
const showEdit = ref(false)
const showKeyModal = ref(false)
const showDetails = ref(false)
const showRevokeConfirm = ref(false)
const showDeleteConfirm = ref(false)
const createdKey = ref('')
const createdLabel = ref('')
const editingLicense = ref<License | null>(null)
const detailsLicense = ref<License | null>(null)
const confirmTarget = ref<License | null>(null)
const actionId = ref<string | null>(null)
const actionType = ref<'revoke' | 'activate' | 'delete' | null>(null)

const statusFilterOptions = [
  { label: 'All statuses', value: 'all' },
  { label: 'Active', value: 'active' },
  { label: 'Expired', value: 'expired' },
  { label: 'Revoked', value: 'revoked' }
]

const columns = [
  { accessorKey: 'label', header: 'Label', enableSorting: true },
  { accessorKey: 'product_name', header: 'Product', enableSorting: true },
  { accessorKey: 'policy_name', header: 'Policy', enableSorting: true },
  { accessorKey: 'key_prefix', header: 'Key prefix' },
  { accessorKey: 'expires_at', header: 'Expires', enableSorting: true },
  { accessorKey: 'revoked', header: 'Status' },
  { id: 'actions', header: '' }
]

const productFilterItems = computed(() => [
  { label: 'All products', value: null },
  ...productSelect.items
])

const policyFilterItems = computed(() => [
  { label: 'All policies', value: null },
  ...policySelect.items
])

const hasFilters = computed(() =>
  Boolean(search.value || statusFilter.value !== 'all' || productFilter.value || policyFilter.value)
)

watch(statusFilter, (value) => {
  setFilter('status', value === 'all' ? undefined : value)
})

watch(productFilter, (value) => {
  setFilter('product_id', value ?? undefined)
  policyFilter.value = null
  setFilter('policy_id', undefined)
  policySelect.clearItems()
  if (value) {
    policySelect.fetchItems('')
  }
})

watch(policyFilter, (value) => {
  setFilter('policy_id', value ?? undefined)
})

const formatDate = (value: string) => new Date(value).toLocaleString()

const formatDateOrNever = (value: string | null) => (value ? formatDate(value) : 'Never')

const formatCreatedBy = (license: License) => {
  if (!license.created_by_name && !license.created_by_email) {
    return '—'
  }
  if (!license.created_by_email) {
    return license.created_by_name as string
  }
  if (!license.created_by_name) {
    return license.created_by_email
  }
  return `${license.created_by_name} (${license.created_by_email})`
}

const getLicenseStatus = (license: License): LicenseStatus => {
  if (license.revoked) {
    return 'revoked'
  }
  if (license.expires_at && new Date(license.expires_at) < new Date()) {
    return 'expired'
  }
  return 'active'
}

const statusLabel = (license: License) => {
  const status = getLicenseStatus(license)
  if (status === 'revoked') return 'Revoked'
  if (status === 'expired') return 'Expired'
  return 'Active'
}

const statusColor = (license: License) => {
  const status = getLicenseStatus(license)
  if (status === 'revoked') return 'error'
  if (status === 'expired') return 'warning'
  return 'success'
}

const statusDotClass = (license: License) => {
  const status = getLicenseStatus(license)
  if (status === 'revoked') return 'bg-red-500'
  if (status === 'expired') return 'bg-amber-500'
  return 'bg-emerald-500'
}

const detailsTitle = computed(() => detailsLicense.value?.label ?? 'License details')

const detailsItems = computed((): DetailItem[] => {
  const license = detailsLicense.value
  if (!license) {
    return []
  }
  return [
    { label: 'Product', value: license.product_name },
    { label: 'Policy', value: license.policy_name },
    { label: 'Key prefix', value: license.key_prefix, mono: true },
    { label: 'Expires', value: formatDateOrNever(license.expires_at) },
    { label: 'Activated', value: formatDateOrNever(license.activated_at) },
    { label: 'Last validated', value: formatDateOrNever(license.last_validated_at) },
    { label: 'Validations', value: String(license.validation_count) },
    { label: 'Created', value: formatDate(license.created_at) },
    { label: 'Created by', value: formatCreatedBy(license) }
  ]
})

const revokeConfirmDescription = computed(() => {
  const label = confirmTarget.value?.label ?? 'this license'
  return `Are you sure you want to revoke "${label}"? The key will stop working immediately.`
})

const deleteConfirmDescription = computed(() => {
  const label = confirmTarget.value?.label ?? 'this license'
  return `Are you sure you want to permanently delete "${label}"? This action cannot be undone.`
})

const fetchStats = async () => {
  try {
    stats.value = await getLicenseStats()
  } catch {
    // Stats are supplementary; list error handling covers primary failure.
  }
}

const openDetails = (license: License) => {
  detailsLicense.value = license
  showDetails.value = true
}

const getActionItems = (license: License): DropdownMenuItem[][] => {
  const menuItems: DropdownMenuItem[] = [
    {
      label: 'View details',
      icon: 'i-lucide-info',
      onSelect: () => openDetails(license)
    }
  ]

  if (!canWrite.value) {
    return [menuItems]
  }

  menuItems.push({
    label: 'Edit',
    icon: 'i-lucide-pencil',
    onSelect: () => openEdit(license)
  })

  if (!license.revoked) {
    menuItems.push({
      label: 'Revoke',
      icon: 'i-lucide-ban',
      color: 'error',
      onSelect: () => openRevokeConfirm(license)
    })
  } else {
    menuItems.push({
      label: 'Activate',
      icon: 'i-lucide-check',
      color: 'success',
      onSelect: () => activate(license.id)
    })
  }

  return [
    menuItems,
    [
      {
        label: 'Delete',
        icon: 'i-lucide-trash-2',
        color: 'error',
        onSelect: () => openDeleteConfirm(license)
      }
    ]
  ]
}

const reload = async () => {
  await Promise.all([refresh(), fetchStats()])
}

const onCreated = (license: License) => {
  createdKey.value = license.key || ''
  createdLabel.value = license.label
  showKeyModal.value = true
  reload()
}

const onUpdated = () => {
  reload()
}

const openEdit = (license: License) => {
  editingLicense.value = license
  showEdit.value = true
}

const openRevokeConfirm = (license: License) => {
  confirmTarget.value = license
  showRevokeConfirm.value = true
}

const openDeleteConfirm = (license: License) => {
  confirmTarget.value = license
  showDeleteConfirm.value = true
}

const confirmRevoke = async () => {
  if (!confirmTarget.value) {
    return
  }

  actionId.value = confirmTarget.value.id
  actionType.value = 'revoke'
  try {
    await revokeLicense(confirmTarget.value.id)
    showRevokeConfirm.value = false
    await reload()
  } catch {
    error.value = 'Failed to revoke license'
  } finally {
    actionId.value = null
    actionType.value = null
    confirmTarget.value = null
  }
}

const confirmDelete = async () => {
  if (!confirmTarget.value) {
    return
  }

  actionId.value = confirmTarget.value.id
  actionType.value = 'delete'
  try {
    await deleteLicense(confirmTarget.value.id)
    showDeleteConfirm.value = false
    await reload()
  } catch {
    error.value = 'Failed to delete license'
  } finally {
    actionId.value = null
    actionType.value = null
    confirmTarget.value = null
  }
}

const activate = async (id: string) => {
  actionId.value = id
  actionType.value = 'activate'
  try {
    await activateLicense(id)
    await reload()
  } catch {
    error.value = 'Failed to activate license'
  } finally {
    actionId.value = null
    actionType.value = null
  }
}

onMounted(fetchStats)
</script>
