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
        <UTable :columns="columns" :data="licenses" :loading="loading">
          <template #expires_at-cell="{ row }">
            <span v-if="!row.original.expires_at" class="text-gray-500">Never</span>
            <span v-else>{{ formatDate(row.original.expires_at) }}</span>
          </template>

          <template #revoked-cell="{ row }">
            <UBadge :color="row.original.revoked ? 'error' : 'success'" variant="subtle">
              {{ row.original.revoked ? 'Revoked' : 'Active' }}
            </UBadge>
          </template>

          <template #created_at-cell="{ row }">
            {{ formatDate(row.original.created_at) }}
          </template>

          <template #actions-cell="{ row }">
            <UButton
              v-if="!row.original.revoked"
              size="xs"
              color="error"
              variant="soft"
              :loading="revokingId === row.original.id"
              @click="revoke(row.original.id)"
            >
              Revoke
            </UButton>
          </template>
        </UTable>
      </UCard>
    </UContainer>

    <CreateLicenseModal v-model:open="showCreate" @created="onCreated" />
    <LicenseKeyModal v-model:open="showKeyModal" :license-key="createdKey" :label="createdLabel" />
  </div>
</template>

<script setup lang="ts">
import type { License } from '~/types'

definePageMeta({
  middleware: 'auth'
})

const { logout, listLicenses, revokeLicense } = useAuth()

const licenses = ref<License[]>([])
const loading = ref(true)
const loadError = ref('')
const showCreate = ref(false)
const showKeyModal = ref(false)
const createdKey = ref('')
const createdLabel = ref('')
const revokingId = ref<string | null>(null)

const columns = [
  { accessorKey: 'label', header: 'Label' },
  { accessorKey: 'key_prefix', header: 'Key prefix' },
  { accessorKey: 'expires_at', header: 'Expires' },
  { accessorKey: 'revoked', header: 'Status' },
  { accessorKey: 'created_at', header: 'Created' },
  { id: 'actions', header: '' }
]

const formatDate = (value: string) => new Date(value).toLocaleString()

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

const revoke = async (id: string) => {
  revokingId.value = id
  try {
    await revokeLicense(id)
    await fetchLicenses()
  } catch {
    loadError.value = 'Failed to revoke license'
  } finally {
    revokingId.value = null
  }
}

onMounted(fetchLicenses)
</script>
