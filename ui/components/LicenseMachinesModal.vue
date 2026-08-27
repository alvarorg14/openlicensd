<template>
  <UModal v-model:open="open" :title="title" :ui="{ content: 'max-w-5xl' }">
    <template #header>
      <div class="flex items-center gap-2">
        <div class="flex h-8 w-8 items-center justify-center rounded-lg bg-brand-100 dark:bg-brand-900/40">
          <UIcon name="i-lucide-monitor" class="h-4 w-4 text-brand-600 dark:text-brand-400" />
        </div>
        <div>
          <span class="font-semibold">{{ title }}</span>
          <p v-if="license" class="text-xs text-muted font-normal mt-0.5">
            {{ machinesSummary }}
          </p>
        </div>
      </div>
    </template>
    <template #body>
      <UAlert v-if="error" color="error" variant="subtle" :title="error" class="animate-fade-in mb-4" />

      <div v-if="loading" class="space-y-3">
        <div v-for="i in 4" :key="i" class="h-12 rounded-lg animate-shimmer" />
      </div>

      <div
        v-else-if="items.length === 0"
        class="flex flex-col items-center justify-center py-12 px-4 text-center"
      >
        <UIcon name="i-lucide-monitor-off" class="h-8 w-8 text-dimmed mb-3" />
        <p class="text-sm text-muted">No machines have activated this license yet.</p>
      </div>

      <template v-else>
        <UTable :columns="columns" :data="items" class="mb-4">
          <template #display_name-cell="{ row }">
            <div class="space-y-1">
              <div class="font-medium text-highlighted">{{ row.original.display_name }}</div>
              <div class="font-mono text-xs text-muted break-all">{{ row.original.fingerprint }}</div>
            </div>
          </template>

          <template #status-cell="{ row }">
            <UBadge
              :color="row.original.deactivated_at ? 'neutral' : 'success'"
              variant="subtle"
            >
              {{ row.original.deactivated_at ? 'Released' : 'Active' }}
            </UBadge>
          </template>

          <template #actions-cell="{ row }">
            <div v-if="canWrite && !row.original.deactivated_at" class="flex justify-end gap-1">
              <UTooltip text="Rename">
                <UButton
                  color="neutral"
                  variant="ghost"
                  size="xs"
                  icon="i-lucide-pencil"
                  @click="openRename(row.original)"
                />
              </UTooltip>
              <UTooltip text="Release">
                <UButton
                  color="error"
                  variant="ghost"
                  size="xs"
                  icon="i-lucide-unlink"
                  :loading="actionId === row.original.id && actionType === 'release'"
                  @click="openReleaseConfirm(row.original)"
                />
              </UTooltip>
            </div>
          </template>
        </UTable>

        <div v-if="totalPages > 1" class="flex justify-center">
          <UPagination
            :page="page"
            :items-per-page="pageSize"
            :total="total"
            @update:page="setPage"
          />
        </div>
      </template>
    </template>
  </UModal>

  <UModal v-model:open="showRename" title="Rename machine">
    <template #body>
      <UForm class="space-y-4" @submit="confirmRename">
        <UFormField label="Display name" name="name">
          <UInput v-model="renameValue" placeholder="e.g. Ana's MacBook" />
        </UFormField>
        <UAlert v-if="renameError" color="error" variant="subtle" :title="renameError" />
        <div class="flex justify-end gap-2">
          <UButton color="neutral" variant="outline" @click="showRename = false">
            Cancel
          </UButton>
          <UButton type="submit" :loading="renaming">
            Save
          </UButton>
        </div>
      </UForm>
    </template>
  </UModal>

  <ConfirmModal
    v-model:open="showReleaseConfirm"
    title="Release machine"
    :description="releaseConfirmDescription"
    confirm-label="Release"
    confirm-color="error"
    :loading="actionType === 'release'"
    @confirm="confirmRelease"
  />
</template>

<script setup lang="ts">
import type { License, LicenseMachine } from '~/types'

const props = defineProps<{
  license: License | null
}>()

const open = defineModel<boolean>('open', { required: true })

const { listLicenseMachines, updateLicenseMachine, releaseLicenseMachine } = useApi()
const { canWrite } = useAuth()

const page = ref(1)
const pageSize = 25
const items = ref<LicenseMachine[]>([])
const total = ref(0)
const totalPages = ref(0)
const loading = ref(false)
const error = ref('')

const showRename = ref(false)
const renameTarget = ref<LicenseMachine | null>(null)
const renameValue = ref('')
const renameError = ref('')
const renaming = ref(false)

const showReleaseConfirm = ref(false)
const releaseTarget = ref<LicenseMachine | null>(null)
const actionId = ref<string | null>(null)
const actionType = ref<'release' | null>(null)

const columns = [
  { accessorKey: 'display_name', header: 'Machine' },
  { accessorKey: 'first_seen_at', header: 'First seen' },
  { accessorKey: 'last_seen_at', header: 'Last validated' },
  { accessorKey: 'validation_count', header: 'Validations' },
  { accessorKey: 'status', header: 'Status' },
  { id: 'actions', header: '' }
]

const title = computed(() => props.license ? `Machines — ${props.license.label}` : 'Machines')

const machinesSummary = computed(() => {
  if (!props.license) {
    return ''
  }
  if (props.license.max_activations == null) {
    return `${props.license.activation_count} active · unlimited`
  }
  return `${props.license.activation_count} / ${props.license.max_activations} active`
})

const releaseConfirmDescription = computed(() => {
  const name = releaseTarget.value?.display_name ?? 'this machine'
  return `Release "${name}"? This frees one activation slot for another machine.`
})

const formatDate = (value: string) => new Date(value).toLocaleString()

const fetchMachines = async () => {
  if (!props.license) {
    return
  }
  loading.value = true
  error.value = ''
  try {
    const response = await listLicenseMachines(props.license.id, {
      page: page.value,
      page_size: pageSize,
      sort: 'last_seen_at',
      order: 'desc'
    })
    items.value = response.items.map((machine) => ({
      ...machine,
      first_seen_at: formatDate(machine.first_seen_at),
      last_seen_at: formatDate(machine.last_seen_at)
    }))
    total.value = response.total
    totalPages.value = response.total_pages
  } catch {
    error.value = 'Failed to load machines'
  } finally {
    loading.value = false
  }
}

const setPage = (value: number) => {
  page.value = value
  fetchMachines()
}

watch(open, (value) => {
  if (value && props.license) {
    page.value = 1
    fetchMachines()
  }
})

const openRename = (machine: LicenseMachine) => {
  renameTarget.value = machine
  renameValue.value = machine.name ?? machine.display_name
  renameError.value = ''
  showRename.value = true
}

const confirmRename = async () => {
  if (!props.license || !renameTarget.value) {
    return
  }
  renaming.value = true
  renameError.value = ''
  try {
    await updateLicenseMachine(
      props.license.id,
      renameTarget.value.id,
      renameValue.value.trim() || null
    )
    showRename.value = false
    await fetchMachines()
  } catch {
    renameError.value = 'Failed to rename machine'
  } finally {
    renaming.value = false
  }
}

const openReleaseConfirm = (machine: LicenseMachine) => {
  releaseTarget.value = machine
  showReleaseConfirm.value = true
}

const confirmRelease = async () => {
  if (!props.license || !releaseTarget.value) {
    return
  }
  actionId.value = releaseTarget.value.id
  actionType.value = 'release'
  try {
    await releaseLicenseMachine(props.license.id, releaseTarget.value.id)
    showReleaseConfirm.value = false
    await fetchMachines()
  } catch {
    error.value = 'Failed to release machine'
  } finally {
    actionId.value = null
    actionType.value = null
    releaseTarget.value = null
  }
}
</script>
