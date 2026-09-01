<template>
  <UContainer class="py-6 pb-20 lg:pb-6 space-y-6">
    <div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between animate-fade-in-up">
      <div>
        <h2 class="text-2xl font-medium tracking-brand text-highlighted">API Tokens</h2>
        <p class="text-sm text-muted mt-0.5">Manage machine-to-machine credentials for automation</p>
      </div>
      <UButton
        color="primary"
        icon="i-lucide-plus"
        size="md"
        class="transition-app shrink-0"
        @click="showForm = true"
      >
        Create token
      </UButton>
    </div>

    <UAlert v-if="error" color="error" variant="subtle" :title="error" class="animate-fade-in" />

    <UCard class="shadow-app border-0 ring-1 ring-default overflow-hidden">
      <div class="flex flex-col gap-3 sm:flex-row sm:items-center border-b border-default pb-4 mb-4">
        <UInput
          :model-value="search"
          icon="i-lucide-search"
          placeholder="Search by name or prefix..."
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
          {{ total === 0 && !search ? 'No API tokens yet' : 'No matching tokens' }}
        </h3>
        <p class="text-sm text-muted mb-6">
          Create tokens for CI, Terraform, and other automation workflows.
        </p>
        <UButton v-if="total === 0 && !search" color="primary" icon="i-lucide-plus" @click="showForm = true">
          Create token
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

          <template #token_prefix-cell="{ row }">
            <code class="text-xs font-mono text-toned">{{ row.original.token_prefix }}…</code>
          </template>

          <template #role-cell="{ row }">
            <UBadge color="neutral" variant="subtle" size="sm" class="capitalize">{{ row.original.role }}</UBadge>
          </template>

          <template #status-cell="{ row }">
            <UBadge
              :color="row.original.revoked_at ? 'error' : 'success'"
              variant="subtle"
              size="sm"
            >
              {{ row.original.revoked_at ? 'Revoked' : 'Active' }}
            </UBadge>
          </template>

          <template #last_used_at-cell="{ row }">
            <span>{{ row.original.last_used_at ? formatDate(row.original.last_used_at) : '—' }}</span>
          </template>

          <template #expires_at-cell="{ row }">
            <span>{{ row.original.expires_at ? formatDate(row.original.expires_at) : 'Never' }}</span>
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

    <ApiTokenFormModal v-model:open="showForm" @created="onCreated" />

    <ApiTokenSecretModal
      v-model:open="showSecretModal"
      :token="createdToken"
      :name="createdName"
    />

    <DetailsModal
      v-model:open="showDetails"
      :title="detailsToken?.name ?? 'Token details'"
      icon="i-lucide-key-round"
      icon-bg-class="bg-brand-100 dark:bg-brand-900/40"
      icon-class="text-brand-600 dark:text-brand-400"
      :items="detailsItems"
    />

    <ConfirmModal
      v-model:open="showRevokeConfirm"
      title="Revoke API token"
      :description="revokeConfirmDescription"
      confirm-label="Revoke"
      confirm-color="warning"
      :loading="revoking"
      @confirm="confirmRevoke"
    />

    <ConfirmModal
      v-model:open="showDeleteConfirm"
      title="Delete API token"
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
import type { ApiToken, DetailItem } from '~/types'

definePageMeta({
  middleware: ['auth', 'admin']
})

const { listApiTokens, revokeApiToken, deleteApiToken } = useApi()

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
} = usePaginatedList<ApiToken>({
  fetcher: (params) => listApiTokens(params)
})

const showForm = ref(false)
const showSecretModal = ref(false)
const showDetails = ref(false)
const showRevokeConfirm = ref(false)
const showDeleteConfirm = ref(false)
const createdToken = ref('')
const createdName = ref('')
const detailsToken = ref<ApiToken | null>(null)
const revokeTarget = ref<ApiToken | null>(null)
const deleteTarget = ref<ApiToken | null>(null)
const actionId = ref<string | null>(null)
const revoking = ref(false)
const deleting = ref(false)

const columns = [
  { accessorKey: 'name', header: 'Name', enableSorting: true },
  { accessorKey: 'token_prefix', header: 'Prefix' },
  { accessorKey: 'role', header: 'Role', enableSorting: true },
  { accessorKey: 'status', header: 'Status' },
  { accessorKey: 'last_used_at', header: 'Last used', enableSorting: true },
  { accessorKey: 'expires_at', header: 'Expires', enableSorting: true },
  { id: 'actions', header: '' }
]

const formatDate = (value: string) => new Date(value).toLocaleString()

const detailsItems = computed((): DetailItem[] => {
  const token = detailsToken.value
  if (!token) {
    return []
  }
  return [
    { label: 'Name', value: token.name },
    { label: 'Prefix', value: `${token.token_prefix}…`, mono: true },
    { label: 'Role', value: token.role },
    { label: 'Status', value: token.revoked_at ? 'Revoked' : 'Active' },
    { label: 'Last used', value: token.last_used_at ? formatDate(token.last_used_at) : '—' },
    { label: 'Expires', value: token.expires_at ? formatDate(token.expires_at) : 'Never' },
    { label: 'Created', value: formatDate(token.created_at) }
  ]
})

const revokeConfirmDescription = computed(() => {
  const name = revokeTarget.value?.name ?? 'this token'
  return `Revoke "${name}"? Requests using this token will be rejected immediately.`
})

const deleteConfirmDescription = computed(() => {
  const name = deleteTarget.value?.name ?? 'this token'
  return `Delete "${name}"? This cannot be undone.`
})

const onCreated = (token: ApiToken) => {
  createdToken.value = token.token || ''
  createdName.value = token.name
  showSecretModal.value = true
  refresh()
}

const openDetails = (token: ApiToken) => {
  detailsToken.value = token
  showDetails.value = true
}

const openRevoke = (token: ApiToken) => {
  revokeTarget.value = token
  showRevokeConfirm.value = true
}

const openDelete = (token: ApiToken) => {
  deleteTarget.value = token
  showDeleteConfirm.value = true
}

const getActionItems = (token: ApiToken): DropdownMenuItem[][] => {
  const menuItems: DropdownMenuItem[] = [
    { label: 'View details', icon: 'i-lucide-info', onSelect: () => openDetails(token) }
  ]

  const destructive: DropdownMenuItem[] = []
  if (!token.revoked_at) {
    destructive.push({
      label: 'Revoke',
      icon: 'i-lucide-ban',
      color: 'warning',
      onSelect: () => openRevoke(token)
    })
  }
  destructive.push({
    label: 'Delete',
    icon: 'i-lucide-trash-2',
    color: 'error',
    onSelect: () => openDelete(token)
  })

  return [menuItems, destructive]
}

const confirmRevoke = async () => {
  if (!revokeTarget.value) {
    return
  }
  actionId.value = revokeTarget.value.id
  revoking.value = true
  try {
    await revokeApiToken(revokeTarget.value.id)
    showRevokeConfirm.value = false
    await refresh()
  } catch {
    error.value = 'Failed to revoke API token'
  } finally {
    actionId.value = null
    revoking.value = false
    revokeTarget.value = null
  }
}

const confirmDelete = async () => {
  if (!deleteTarget.value) {
    return
  }
  actionId.value = deleteTarget.value.id
  deleting.value = true
  try {
    await deleteApiToken(deleteTarget.value.id)
    showDeleteConfirm.value = false
    await refresh()
  } catch {
    error.value = 'Failed to delete API token'
  } finally {
    actionId.value = null
    deleting.value = false
    deleteTarget.value = null
  }
}
</script>
