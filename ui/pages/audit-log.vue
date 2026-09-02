<template>
  <UContainer class="py-6 pb-20 lg:pb-6 space-y-6">
    <div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between animate-fade-in-up">
      <div>
        <h2 class="text-2xl font-medium tracking-brand text-highlighted">Audit Log</h2>
        <p class="text-sm text-muted mt-0.5">Append-only record of admin actions</p>
      </div>
    </div>

    <UAlert v-if="error" color="error" variant="subtle" :title="error" class="animate-fade-in" />

    <UCard class="shadow-app border-0 ring-1 ring-default overflow-hidden">
      <div class="flex flex-col gap-3 sm:flex-row sm:items-center border-b border-default pb-4 mb-4">
        <UInput
          :model-value="search"
          icon="i-lucide-search"
          placeholder="Search actions, actors, resources..."
          class="sm:flex-1"
          @update:model-value="setSearch"
        />
        <USelect
          v-model="resourceTypeFilter"
          :items="resourceTypeOptions"
          placeholder="All resources"
          class="sm:w-48"
        />
        <USelect
          v-model="actionFilter"
          :items="actionOptions"
          placeholder="All actions"
          :disabled="!resourceTypeFilter"
          class="sm:w-48"
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
          {{ total === 0 && !search && !actionFilter && !resourceTypeFilter ? 'No audit events yet' : 'No matching events' }}
        </h3>
        <p class="text-sm text-muted">
          Admin mutations are recorded here automatically.
        </p>
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
          <template #occurred_at-cell="{ row }">
            <span>{{ formatDate(row.original.occurred_at) }}</span>
          </template>

          <template #resource_type-cell="{ row }">
            <UBadge
              :color="resourceTypeBadgeColor(row.original.resource_type)"
              variant="subtle"
              size="sm"
            >
              {{ formatResourceTypeLabel(row.original.resource_type) }}
            </UBadge>
          </template>

          <template #action-cell="{ row }">
            <span class="text-sm text-highlighted">{{ formatActionLabel(row.original.action) }}</span>
          </template>

          <template #resource-cell="{ row }">
            <div class="flex flex-col">
              <span class="text-sm text-highlighted">{{ row.original.resource_label || row.original.resource_type }}</span>
              <span v-if="row.original.resource_id" class="text-xs text-muted font-mono">{{ row.original.resource_id }}</span>
            </div>
          </template>

          <template #actor-cell="{ row }">
            <div class="flex flex-col">
              <span class="text-sm text-highlighted">{{ row.original.actor_name }}</span>
              <span
                v-if="row.original.auth_method === 'session'"
                class="text-xs text-muted"
              >
                {{ row.original.actor_email || '—' }}
              </span>
              <code
                v-else-if="row.original.actor_token_prefix"
                class="text-xs font-mono text-muted"
              >
                {{ row.original.actor_token_prefix }}…
              </code>
            </div>
          </template>
        </UTable>

        <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between pt-4 border-t border-default mt-4">
          <USelect
            v-model="pageSize"
            :items="pageSizeOptions"
            class="sm:w-32"
          />
          <UPagination
            v-if="totalPages > 1"
            :page="page"
            :items-per-page="pageSize"
            :total="total"
            @update:page="setPage"
          />
        </div>
      </template>
    </UCard>

    <DetailsModal
      v-model:open="showDetails"
      :title="detailsTitle"
      icon="i-lucide-scroll-text"
      icon-bg-class="bg-brand-100 dark:bg-brand-900/40"
      icon-class="text-brand-600 dark:text-brand-400"
      :items="detailsItems"
    />
  </UContainer>
</template>

<script setup lang="ts">
import type { AuditEvent, DetailItem } from '~/types'
import {
  actionsByResourceType,
  formatActionLabel,
  formatResourceTypeLabel,
  resourceTypeBadgeColors
} from '~/utils/auditLog'

definePageMeta({
  middleware: ['auth', 'admin']
})

const { listAuditEvents } = useApi()

const actionFilter = ref<string | undefined>(undefined)
const resourceTypeFilter = ref<string | undefined>(undefined)

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
  setSearch,
  setPage,
  setFilter
} = usePaginatedList<AuditEvent, { action?: string, resource_type?: string }>({
  fetcher: (params) => listAuditEvents(params),
  defaultSort: 'occurred_at'
})

const showDetails = ref(false)
const detailsEvent = ref<AuditEvent | null>(null)

const columns = [
  { accessorKey: 'occurred_at', header: 'When', enableSorting: true },
  { accessorKey: 'resource_type', header: 'Type', enableSorting: true },
  { accessorKey: 'action', header: 'Action', enableSorting: true },
  { accessorKey: 'resource', header: 'Resource' },
  { accessorKey: 'actor', header: 'Actor', enableSorting: true }
]

const resourceTypeOptions = [
  { label: 'All resources', value: undefined },
  { label: 'License', value: 'license' },
  { label: 'Product', value: 'product' },
  { label: 'Policy', value: 'policy' },
  { label: 'Machine', value: 'machine' },
  { label: 'User', value: 'user' },
  { label: 'API token', value: 'api_token' }
]

const pageSizeOptions = [
  { label: '25 / page', value: 25 },
  { label: '50 / page', value: 50 },
  { label: '100 / page', value: 100 }
]

const actionOptions = computed(() => {
  const options: Array<{ label: string, value: string | undefined }> = [
    { label: 'All actions', value: undefined }
  ]
  if (!resourceTypeFilter.value) {
    return options
  }
  const actions = actionsByResourceType[resourceTypeFilter.value] ?? []
  for (const action of actions) {
    options.push({
      label: formatActionLabel(action),
      value: action
    })
  }
  return options
})

const resourceTypeBadgeColor = (resourceType: string) => {
  return resourceTypeBadgeColors[resourceType] ?? 'neutral'
}

const formatDate = (value: string) => new Date(value).toLocaleString()

const detailsTitle = computed(() => {
  if (!detailsEvent.value) {
    return 'Event details'
  }
  return formatActionLabel(detailsEvent.value.action)
})

const detailsItems = computed((): DetailItem[] => {
  const event = detailsEvent.value
  if (!event) {
    return []
  }
  const items: DetailItem[] = [
    { label: 'When', value: formatDate(event.occurred_at) },
    { label: 'Action', value: event.action, mono: true },
    { label: 'Resource type', value: formatResourceTypeLabel(event.resource_type) },
    { label: 'Resource', value: event.resource_label || '—' },
    { label: 'Resource ID', value: event.resource_id || '—', mono: true },
    { label: 'Actor', value: event.actor_name },
    { label: 'Actor email', value: event.actor_email || '—' },
    { label: 'Actor role', value: event.actor_role },
    { label: 'Auth method', value: event.auth_method },
    { label: 'Client IP', value: event.client_ip || '—', mono: true },
    { label: 'User agent', value: event.user_agent || '—' },
    { label: 'Request ID', value: event.request_id || '—', mono: true },
    { label: 'Request', value: `${event.request_method} ${event.request_path}`, mono: true },
    { label: 'Status', value: String(event.response_status) }
  ]
  if (event.metadata && Object.keys(event.metadata).length > 0) {
    items.push({ label: 'Metadata', value: JSON.stringify(event.metadata, null, 2), mono: true })
  }
  return items
})

watch(actionFilter, (value) => {
  setFilter('action', value)
})

watch(resourceTypeFilter, (value) => {
  setFilter('resource_type', value)
  if (!value) {
    actionFilter.value = undefined
    return
  }
  const validActions = actionsByResourceType[value] ?? []
  if (actionFilter.value && !validActions.includes(actionFilter.value)) {
    actionFilter.value = undefined
  }
})

const openDetails = (event: AuditEvent) => {
  detailsEvent.value = event
  showDetails.value = true
}
</script>
