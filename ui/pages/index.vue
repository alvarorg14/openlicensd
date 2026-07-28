<template>
  <div class="min-h-screen">
    <AppHeader />

    <UContainer class="py-6 space-y-6">
      <!-- Page header -->
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

      <!-- Stat cards -->
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

      <!-- Licenses table card -->
      <UCard
        class="shadow-app border-0 ring-1 ring-slate-200/80 dark:ring-slate-800/80 animate-fade-in-up stagger-2 overflow-hidden"
      >
        <!-- Toolbar -->
        <div class="flex flex-col gap-3 sm:flex-row sm:items-center border-b border-slate-200/80 dark:border-slate-800/80 pb-4 mb-4">
          <UInput
            v-model="searchQuery"
            icon="i-lucide-search"
            placeholder="Search by label or key prefix..."
            class="sm:flex-1 transition-app"
          />
          <USelect
            v-model="statusFilter"
            :items="statusFilterOptions"
            class="sm:w-48"
          />
        </div>

        <!-- Loading skeleton -->
        <div v-if="loading" class="space-y-3">
          <div v-for="i in 5" :key="i" class="h-12 rounded-lg animate-shimmer" />
        </div>

        <!-- Empty state -->
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

        <!-- Table -->
        <UTable
          v-else
          :columns="columns"
          :data="filteredLicenses"
          :loading="false"
          class="[&_tbody_tr]:transition-app [&_tbody_tr:hover]:bg-slate-50 dark:[&_tbody_tr:hover]:bg-slate-800/30"
        >
          <template #label-cell="{ row }">
            <span class="font-medium text-slate-900 dark:text-white">{{ row.original.label }}</span>
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

          <template #last_validated_at-cell="{ row }">
            <span v-if="!row.original.last_validated_at" class="text-slate-500">Never</span>
            <span v-else class="text-slate-700 dark:text-slate-300">{{ formatDate(row.original.last_validated_at) }}</span>
          </template>

          <template #validation_count-cell="{ row }">
            <span class="tabular-nums text-slate-700 dark:text-slate-300">{{ row.original.validation_count }}</span>
          </template>

          <template #created_at-cell="{ row }">
            <span class="text-slate-700 dark:text-slate-300">{{ formatDate(row.original.created_at) }}</span>
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
    </UContainer>

    <CreateLicenseModal v-model:open="showCreate" @created="onCreated" />
    <EditLicenseModal
      v-model:open="showEdit"
      :license="editingLicense"
      @updated="onUpdated"
    />
    <LicenseKeyModal v-model:open="showKeyModal" :license-key="createdKey" :label="createdLabel" />

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
  </div>
</template>

<script setup lang="ts">
import type { DropdownMenuItem } from '@nuxt/ui'
import type { License, LicenseStatus } from '~/types'

definePageMeta({
  middleware: 'auth'
})

const {
  listLicenses,
  revokeLicense,
  activateLicense,
  deleteLicense
} = useAuth()

const licenses = ref<License[]>([])
const loading = ref(true)
const loadError = ref('')
const showCreate = ref(false)
const showEdit = ref(false)
const showKeyModal = ref(false)
const showRevokeConfirm = ref(false)
const showDeleteConfirm = ref(false)
const createdKey = ref('')
const createdLabel = ref('')
const editingLicense = ref<License | null>(null)
const confirmTarget = ref<License | null>(null)
const actionId = ref<string | null>(null)
const actionType = ref<'revoke' | 'activate' | 'delete' | null>(null)
const searchQuery = ref('')
const statusFilter = ref<'all' | LicenseStatus>('all')

const statusFilterOptions = [
  { label: 'All statuses', value: 'all' },
  { label: 'Active', value: 'active' },
  { label: 'Expired', value: 'expired' },
  { label: 'Revoked', value: 'revoked' }
]

const columns = [
  { accessorKey: 'label', header: 'Label' },
  { accessorKey: 'key_prefix', header: 'Key prefix' },
  { accessorKey: 'expires_at', header: 'Expires' },
  { accessorKey: 'revoked', header: 'Status' },
  { accessorKey: 'last_validated_at', header: 'Last validated' },
  { accessorKey: 'validation_count', header: 'Validations' },
  { accessorKey: 'created_at', header: 'Created' },
  { id: 'actions', header: '' }
]

const formatDate = (value: string) => new Date(value).toLocaleString()

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

    if (!query) {
      return true
    }

    return (
      license.label.toLowerCase().includes(query)
      || license.key_prefix.toLowerCase().includes(query)
    )
  })
})

const revokeConfirmDescription = computed(() => {
  const label = confirmTarget.value?.label ?? 'this license'
  return `Are you sure you want to revoke "${label}"? The key will stop working immediately.`
})

const deleteConfirmDescription = computed(() => {
  const label = confirmTarget.value?.label ?? 'this license'
  return `Are you sure you want to permanently delete "${label}"? This action cannot be undone.`
})

const getActionItems = (license: License): DropdownMenuItem[][] => {
  const items: DropdownMenuItem[] = [
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
