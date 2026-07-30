<template>
  <UContainer class="py-6 pb-20 lg:pb-6 space-y-6">
    <div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between animate-fade-in-up">
      <div>
        <h2 class="text-2xl font-bold text-slate-900 dark:text-white">Licenses</h2>
        <p class="text-sm text-slate-500 dark:text-slate-400 mt-0.5">Manage and monitor your license keys</p>
      </div>
      <UButton
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
        icon-bg-class="bg-indigo-100 dark:bg-indigo-900/40"
        icon-class="text-indigo-600 dark:text-indigo-400"
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

    <UAlert v-if="loadError" color="error" variant="subtle" :title="loadError" class="animate-fade-in" />

    <UCard
      class="shadow-app border-0 ring-1 ring-slate-200/80 dark:ring-slate-800/80 animate-fade-in-up stagger-2 overflow-hidden"
    >
      <div class="flex flex-col gap-3 border-b border-slate-200/80 dark:border-slate-800/80 pb-4 mb-4">
        <UInput
          v-model="searchQuery"
          icon="i-lucide-search"
          placeholder="Search by label or key prefix..."
          class="transition-app"
        />
        <div class="grid grid-cols-1 sm:grid-cols-3 gap-3">
          <USelect
            v-model="statusFilter"
            :items="statusFilterOptions"
          />
          <USelectMenu
            v-model="productFilter"
            :items="productFilterItems"
            value-key="value"
            label-key="label"
            placeholder="All products"
          />
          <USelectMenu
            v-model="policyFilter"
            :items="policyFilterItems"
            value-key="value"
            label-key="label"
            placeholder="All policies"
          />
        </div>
      </div>

      <div v-if="loading" class="space-y-3">
        <div v-for="i in 5" :key="i" class="h-12 rounded-lg animate-shimmer" />
      </div>

      <div
        v-else-if="filteredLicenses.length === 0"
        class="flex flex-col items-center justify-center py-16 px-4 text-center animate-fade-in"
      >
        <div class="flex h-14 w-14 items-center justify-center rounded-full bg-slate-100 dark:bg-slate-800 mb-4">
          <UIcon name="i-lucide-key" class="h-7 w-7 text-slate-400" />
        </div>
        <h3 class="text-lg font-semibold text-slate-900 dark:text-white mb-1">
          {{ licenses.length === 0 ? 'No licenses yet' : 'No matching licenses' }}
        </h3>
        <p class="text-sm text-slate-500 dark:text-slate-400 mb-6 max-w-sm">
          {{ licenses.length === 0
            ? 'Create your first license key to get started.'
            : 'Try adjusting your search or filter criteria.' }}
        </p>
        <UButton
          v-if="licenses.length === 0"
          color="primary"
          icon="i-lucide-plus"
          @click="showCreate = true"
        >
          Create license
        </UButton>
      </div>

      <UTable
        v-else
        :columns="columns"
        :data="filteredLicenses"
        :loading="false"
        class="[&_tbody_tr]:transition-app [&_tbody_tr:hover]:bg-slate-50 dark:[&_tbody_tr:hover]:bg-slate-800/30 [&_tbody_tr]:cursor-pointer"
        @select="(_e, row) => openDetails(row.original)"
      >
        <template #label-cell="{ row }">
          <UTooltip :text="row.original.label">
            <span class="block max-w-[16rem] truncate font-medium text-slate-900 dark:text-white">
              {{ row.original.label }}
            </span>
          </UTooltip>
        </template>

        <template #product_name-cell="{ row }">
          <UTooltip :text="row.original.product_name">
            <span class="block max-w-[10rem] truncate text-slate-700 dark:text-slate-300">
              {{ row.original.product_name }}
            </span>
          </UTooltip>
        </template>

        <template #policy_name-cell="{ row }">
          <UTooltip :text="row.original.policy_name">
            <span class="block max-w-[10rem] truncate text-slate-700 dark:text-slate-300">
              {{ row.original.policy_name }}
            </span>
          </UTooltip>
        </template>

        <template #key_prefix-cell="{ row }">
          <code class="text-xs font-mono bg-slate-100 dark:bg-slate-800 px-2 py-0.5 rounded text-slate-700 dark:text-slate-300">
            {{ row.original.key_prefix }}
          </code>
        </template>

        <template #expires_at-cell="{ row }">
          <span v-if="!row.original.expires_at" class="text-slate-500">Never</span>
          <span v-else class="text-slate-700 dark:text-slate-300">{{ formatDate(row.original.expires_at) }}</span>
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
      icon-bg-class="bg-indigo-100 dark:bg-indigo-900/40"
      icon-class="text-indigo-600 dark:text-indigo-400"
      :items="detailsItems"
    >
      <template #top>
        <div v-if="detailsLicense" class="flex items-center gap-2">
          <span class="font-medium text-slate-900 dark:text-white">{{ detailsLicense.label }}</span>
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
import type { DetailItem, License, LicenseStatus } from '~/types'

definePageMeta({
  middleware: 'auth'
})

const {
  listLicenses,
  revokeLicense,
  activateLicense,
  deleteLicense
} = useApi()

const licenses = ref<License[]>([])
const loading = ref(true)
const loadError = ref('')
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
const searchQuery = ref('')
const statusFilter = ref<'all' | LicenseStatus>('all')
const productFilter = ref<string | null>(null)
const policyFilter = ref<string | null>(null)

const statusFilterOptions = [
  { label: 'All statuses', value: 'all' },
  { label: 'Active', value: 'active' },
  { label: 'Expired', value: 'expired' },
  { label: 'Revoked', value: 'revoked' }
]

const columns = [
  { accessorKey: 'label', header: 'Label' },
  { accessorKey: 'product_name', header: 'Product' },
  { accessorKey: 'policy_name', header: 'Policy' },
  { accessorKey: 'key_prefix', header: 'Key prefix' },
  { accessorKey: 'expires_at', header: 'Expires' },
  { accessorKey: 'revoked', header: 'Status' },
  { id: 'actions', header: '' }
]

const formatDate = (value: string) => new Date(value).toLocaleString()

const formatDateOrNever = (value: string | null) => (value ? formatDate(value) : 'Never')

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

const productFilterItems = computed(() => {
  const seen = new Map<string, string>()
  for (const license of licenses.value) {
    if (!seen.has(license.product_id)) {
      seen.set(license.product_id, license.product_name)
    }
  }
  return [
    { label: 'All products', value: null },
    ...[...seen.entries()]
      .sort((a, b) => a[1].localeCompare(b[1]))
      .map(([id, name]) => ({ label: name, value: id }))
  ]
})

const policyFilterItems = computed(() => {
  const seen = new Map<string, string>()
  for (const license of licenses.value) {
    if (productFilter.value && license.product_id !== productFilter.value) {
      continue
    }
    if (!seen.has(license.policy_id)) {
      seen.set(license.policy_id, license.policy_name)
    }
  }
  return [
    { label: 'All policies', value: null },
    ...[...seen.entries()]
      .sort((a, b) => a[1].localeCompare(b[1]))
      .map(([id, name]) => ({ label: name, value: id }))
  ]
})

watch(productFilter, () => {
  if (!policyFilter.value) {
    return
  }
  const stillValid = policyFilterItems.value.some((item) => item.value === policyFilter.value)
  if (!stillValid) {
    policyFilter.value = null
  }
})

const stats = computed(() => {
  const counts = { total: licenses.value.length, active: 0, expired: 0, revoked: 0 }
  for (const license of licenses.value) {
    const status = getLicenseStatus(license)
    counts[status]++
  }
  return counts
})

const filteredLicenses = computed(() => {
  const query = searchQuery.value.trim().toLowerCase()

  return licenses.value.filter((license) => {
    const status = getLicenseStatus(license)
    if (statusFilter.value !== 'all' && status !== statusFilter.value) {
      return false
    }

    if (productFilter.value && license.product_id !== productFilter.value) {
      return false
    }

    if (policyFilter.value && license.policy_id !== policyFilter.value) {
      return false
    }

    if (!query) {
      return true
    }

    return (
      license.label.toLowerCase().includes(query)
      || license.key_prefix.toLowerCase().includes(query)
    )
  })
})

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
    { label: 'Created', value: formatDate(license.created_at) }
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

const openDetails = (license: License) => {
  detailsLicense.value = license
  showDetails.value = true
}

const getActionItems = (license: License): DropdownMenuItem[][] => {
  const items: DropdownMenuItem[] = [
    {
      label: 'View details',
      icon: 'i-lucide-info',
      onSelect: () => openDetails(license)
    },
    {
      label: 'Edit',
      icon: 'i-lucide-pencil',
      onSelect: () => openEdit(license)
    }
  ]

  if (!license.revoked) {
    items.push({
      label: 'Revoke',
      icon: 'i-lucide-ban',
      color: 'error',
      onSelect: () => openRevokeConfirm(license)
    })
  } else {
    items.push({
      label: 'Activate',
      icon: 'i-lucide-check',
      color: 'success',
      onSelect: () => activate(license.id)
    })
  }

  return [
    items,
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

const fetchLicenses = async () => {
  loading.value = true
  loadError.value = ''
  try {
    licenses.value = await listLicenses()
  } catch {
    loadError.value = 'Failed to load licenses'
  } finally {
    loading.value = false
  }
}

const onCreated = (license: License) => {
  createdKey.value = license.key || ''
  createdLabel.value = license.label
  showKeyModal.value = true
  fetchLicenses()
}

const onUpdated = () => {
  fetchLicenses()
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
    await fetchLicenses()
  } catch {
    loadError.value = 'Failed to revoke license'
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
    await fetchLicenses()
  } catch {
    loadError.value = 'Failed to delete license'
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
    await fetchLicenses()
  } catch {
    loadError.value = 'Failed to activate license'
  } finally {
    actionId.value = null
    actionType.value = null
  }
}

onMounted(fetchLicenses)
</script>
