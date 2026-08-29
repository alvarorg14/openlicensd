<template>
  <UContainer class="py-6 pb-20 lg:pb-6 space-y-6">
    <div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between animate-fade-in-up">
      <div>
        <h2 class="text-2xl font-medium tracking-brand text-highlighted">Users</h2>
        <p class="text-sm text-muted mt-0.5">Manage admin accounts and roles</p>
      </div>
      <UButton
        color="primary"
        icon="i-lucide-plus"
        size="md"
        class="transition-app shrink-0"
        @click="openCreate"
      >
        Create user
      </UButton>
    </div>

    <UAlert v-if="error" color="error" variant="subtle" :title="error" class="animate-fade-in" />

    <UCard class="shadow-app border-0 ring-1 ring-default overflow-hidden">
      <div class="flex flex-col gap-3 sm:flex-row sm:items-center border-b border-default pb-4 mb-4">
        <UInput
          :model-value="search"
          icon="i-lucide-search"
          placeholder="Search by name or email..."
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
          {{ total === 0 && !search ? 'No users yet' : 'No matching users' }}
        </h3>
        <p class="text-sm text-muted mb-6">
          Create users to grant access to the admin console.
        </p>
        <UButton v-if="total === 0 && !search" color="primary" icon="i-lucide-plus" @click="openCreate">
          Create user
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

          <template #email-cell="{ row }">
            <span class="text-toned">{{ row.original.email }}</span>
          </template>

          <template #role-cell="{ row }">
            <UBadge color="neutral" variant="subtle" size="sm" class="capitalize">{{ row.original.role }}</UBadge>
          </template>

          <template #auth_provider-cell="{ row }">
            <UBadge color="neutral" variant="outline" size="sm" class="capitalize">
              {{ row.original.auth_provider }}
            </UBadge>
          </template>

          <template #status-cell="{ row }">
            <UBadge
              :color="row.original.disabled_at ? 'error' : 'success'"
              variant="subtle"
              size="sm"
            >
              {{ row.original.disabled_at ? 'Disabled' : 'Active' }}
            </UBadge>
          </template>

          <template #last_login_at-cell="{ row }">
            <span>{{ row.original.last_login_at ? formatDate(row.original.last_login_at) : '—' }}</span>
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

    <UserFormModal v-model:open="showForm" :user="editingUser" @saved="refresh" />

    <DetailsModal
      v-model:open="showDetails"
      :title="detailsUser?.name ?? 'User details'"
      icon="i-lucide-user"
      icon-bg-class="bg-brand-100 dark:bg-brand-900/40"
      icon-class="text-brand-600 dark:text-brand-400"
      :items="detailsItems"
    />

    <ConfirmModal
      v-model:open="showDeleteConfirm"
      title="Delete user"
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
import type { DetailItem, User } from '~/types'

definePageMeta({
  middleware: ['auth', 'admin']
})

const { listUsers, deleteUser, disableUser, enableUser } = useApi()
const { user: currentUser } = useAuth()

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
} = usePaginatedList<User>({
  fetcher: (params) => listUsers(params)
})

const showForm = ref(false)
const showDetails = ref(false)
const editingUser = ref<User | null>(null)
const detailsUser = ref<User | null>(null)
const showDeleteConfirm = ref(false)
const deleteTarget = ref<User | null>(null)
const actionId = ref<string | null>(null)
const deleting = ref(false)

const columns = [
  { accessorKey: 'name', header: 'Name', enableSorting: true },
  { accessorKey: 'email', header: 'Email', enableSorting: true },
  { accessorKey: 'role', header: 'Role', enableSorting: true },
  { accessorKey: 'auth_provider', header: 'Provider' },
  { accessorKey: 'status', header: 'Status' },
  { accessorKey: 'last_login_at', header: 'Last login', enableSorting: true },
  { id: 'actions', header: '' }
]

const formatDate = (value: string) => new Date(value).toLocaleString()

const detailsItems = computed((): DetailItem[] => {
  const user = detailsUser.value
  if (!user) {
    return []
  }
  return [
    { label: 'Name', value: user.name },
    { label: 'Email', value: user.email },
    { label: 'Role', value: user.role },
    { label: 'Status', value: user.disabled_at ? 'Disabled' : 'Active' },
    { label: 'Auth provider', value: user.auth_provider },
    { label: 'Last login', value: user.last_login_at ? formatDate(user.last_login_at) : '—' },
    { label: 'Created', value: formatDate(user.created_at) }
  ]
})

const deleteConfirmDescription = computed(() => {
  const name = deleteTarget.value?.name ?? 'this user'
  return `Are you sure you want to delete "${name}"? This cannot be undone.`
})

const openCreate = () => {
  editingUser.value = null
  showForm.value = true
}

const openEdit = (user: User) => {
  editingUser.value = user
  showForm.value = true
}

const openDetails = (user: User) => {
  detailsUser.value = user
  showDetails.value = true
}

const openDelete = (user: User) => {
  deleteTarget.value = user
  showDeleteConfirm.value = true
}

const isSelf = (user: User) => currentUser.value?.id === user.id

const getActionItems = (user: User): DropdownMenuItem[][] => {
  const menuItems: DropdownMenuItem[] = [
    { label: 'View details', icon: 'i-lucide-info', onSelect: () => openDetails(user) },
    { label: 'Edit', icon: 'i-lucide-pencil', onSelect: () => openEdit(user) }
  ]

  if (!isSelf(user)) {
    if (user.disabled_at) {
      menuItems.push({
        label: 'Enable',
        icon: 'i-lucide-user-check',
        onSelect: () => toggleDisabled(user, false)
      })
    } else {
      menuItems.push({
        label: 'Disable',
        icon: 'i-lucide-user-x',
        color: 'warning',
        onSelect: () => toggleDisabled(user, true)
      })
    }
  }

  const destructive: DropdownMenuItem[] = []
  if (!isSelf(user)) {
    destructive.push({
      label: 'Delete',
      icon: 'i-lucide-trash-2',
      color: 'error',
      onSelect: () => openDelete(user)
    })
  }

  return destructive.length > 0 ? [menuItems, destructive] : [menuItems]
}

const toggleDisabled = async (user: User, disable: boolean) => {
  actionId.value = user.id
  try {
    if (disable) {
      await disableUser(user.id)
    } else {
      await enableUser(user.id)
    }
    await refresh()
  } catch {
    error.value = disable ? 'Failed to disable user' : 'Failed to enable user'
  } finally {
    actionId.value = null
  }
}

const confirmDelete = async () => {
  if (!deleteTarget.value) {
    return
  }
  actionId.value = deleteTarget.value.id
  deleting.value = true
  try {
    await deleteUser(deleteTarget.value.id)
    showDeleteConfirm.value = false
    await refresh()
  } catch {
    error.value = 'Failed to delete user'
  } finally {
    actionId.value = null
    deleting.value = false
    deleteTarget.value = null
  }
}
</script>
