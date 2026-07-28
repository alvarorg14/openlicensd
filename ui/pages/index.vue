<template>
  <div class="min-h-screen bg-gray-50 dark:bg-gray-900">
    <UContainer class="py-8 space-y-6">
      <div class="flex items-center justify-between">
        <div>
          <h1 class="text-2xl font-bold">Licenses</h1>
          <p class="text-sm text-gray-500">Manage license keys</p>
        </div>
        <div class="flex gap-2">
          <UButton color="primary" icon="i-lucide-plus" @click="showCreate = true">
            Create license
          </UButton>
          <UButton color="neutral" variant="outline" icon="i-lucide-log-out" @click="logout">
            Logout
          </UButton>
        </div>
      </div>

      <UAlert v-if="loadError" color="error" variant="subtle" :title="loadError" />

      <UCard>
        <div class="mb-4 flex flex-col gap-3 sm:flex-row sm:items-center">
          <UInput
            v-model="searchQuery"
            icon="i-lucide-search"
            placeholder="Search by label or key prefix..."
            class="sm:flex-1"
          />
          <USelect
            v-model="statusFilter"
            :items="statusFilterOptions"
            class="sm:w-48"
          />
        </div>

        <UTable :columns="columns" :data="filteredLicenses" :loading="loading">
          <template #expires_at-cell="{ row }">
            <span v-if="!row.original.expires_at" class="text-gray-500">Never</span>
            <span v-else>{{ formatDate(row.original.expires_at) }}</span>
          </template>

          <template #revoked-cell="{ row }">
            <UBadge :color="statusColor(row.original)" variant="subtle">
              {{ statusLabel(row.original) }}
            </UBadge>
          </template>

          <template #last_validated_at-cell="{ row }">
            <span v-if="!row.original.last_validated_at" class="text-gray-500">Never</span>
            <span v-else>{{ formatDate(row.original.last_validated_at) }}</span>
          </template>

          <template #validation_count-cell="{ row }">
            {{ row.original.validation_count }}
          </template>

          <template #created_at-cell="{ row }">
            {{ formatDate(row.original.created_at) }}
          </template>

          <template #actions-cell="{ row }">
            <div class="flex flex-wrap gap-1">
              <UButton
                size="xs"
                color="neutral"
                variant="soft"
                icon="i-lucide-pencil"
                @click="openEdit(row.original)"
              >
                Edit
              </UButton>

              <UButton
                v-if="!row.original.revoked"
                size="xs"
                color="error"
                variant="soft"
                icon="i-lucide-ban"
                :loading="actionId === row.original.id && actionType === 'revoke'"
                @click="openRevokeConfirm(row.original)"
              >
                Revoke
              </UButton>

              <UButton
                v-if="row.original.revoked"
                size="xs"
                color="success"
                variant="soft"
                icon="i-lucide-check"
                :loading="actionId === row.original.id && actionType === 'activate'"
                @click="activate(row.original.id)"
              >
                Activate
              </UButton>

              <UButton
                size="xs"
                color="error"
                variant="outline"
                icon="i-lucide-trash-2"
                :loading="actionId === row.original.id && actionType === 'delete'"
                @click="openDeleteConfirm(row.original)"
              >
                Delete
              </UButton>
            </div>
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
import type { License, LicenseStatus } from '~/types'

definePageMeta({
  middleware: 'auth'
})

const {
  logout,
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
