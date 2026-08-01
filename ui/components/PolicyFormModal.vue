<template>
  <UModal v-model:open="open" :title="policy ? 'Edit policy' : 'Create policy'">
    <template #header>
      <div class="flex items-center gap-2">
        <div class="flex h-8 w-8 items-center justify-center rounded-lg bg-brand-100 dark:bg-brand-900/40">
          <UIcon :name="policy ? 'i-lucide-pencil' : 'i-lucide-plus'" class="h-4 w-4 text-brand-600 dark:text-brand-400" />
        </div>
        <span class="font-semibold">{{ policy ? 'Edit policy' : 'Create policy' }}</span>
      </div>
    </template>
    <template #body>
      <UForm :state="form" class="space-y-4" @submit="onSubmit">
        <UFormField v-if="!policy" label="Product" name="product" required>
          <USelectMenu
            v-model="form.productId"
            v-model:search-term="productSearchTerm"
            :items="productItems"
            value-key="value"
            label-key="label"
            placeholder="Select a product"
            :loading="productSelect.loading"
            searchable
            class="w-full"
            @update:search-term="productSelect.onSearch"
          />
        </UFormField>

        <UFormField label="Name" name="name" required>
          <UInput v-model="form.name" placeholder="e.g. 30-day trial" />
        </UFormField>

        <UFormField label="Description" name="description">
          <UTextarea v-model="form.description" placeholder="Optional description" :rows="3" />
        </UFormField>

        <UFormField name="perpetual">
          <UCheckbox v-model="form.perpetual" label="Never expires (perpetual)" />
        </UFormField>

        <UFormField v-if="!form.perpetual" label="Duration (days)" name="durationDays" required>
          <UInput v-model.number="form.durationDays" type="number" min="1" />
        </UFormField>

        <UFormField v-if="!form.perpetual" label="Expiration basis" name="expirationBasis">
          <USelect
            v-model="form.expirationBasis"
            :items="expirationBasisOptions"
          />
        </UFormField>

        <UFormField label="Grace period (days)" name="gracePeriodDays">
          <UInput v-model.number="form.gracePeriodDays" type="number" min="0" />
        </UFormField>

        <UAlert v-if="error" color="error" variant="subtle" :title="error" class="animate-fade-in" />

        <div class="flex justify-end gap-2 pt-2">
          <UButton color="neutral" variant="outline" @click="open = false">
            Cancel
          </UButton>
          <UButton type="submit" :loading="loading">
            {{ policy ? 'Save' : 'Create' }}
          </UButton>
        </div>
      </UForm>
    </template>
  </UModal>
</template>

<script setup lang="ts">
import type { ExpirationBasis, Policy } from '~/types'

const props = defineProps<{
  policy: Policy | null
}>()

const open = defineModel<boolean>('open', { required: true })
const emit = defineEmits<{
  saved: []
}>()

const { createPolicy, updatePolicy } = useApi()
const { createProductSelect } = useServerSelect()
const productSelect = createProductSelect()

const form = reactive({
  productId: null as string | null,
  name: '',
  description: '',
  perpetual: true,
  durationDays: 30,
  expirationBasis: 'on_creation' as ExpirationBasis,
  gracePeriodDays: 0
})

const loading = ref(false)
const error = ref('')
const productSearchTerm = ref('')

const expirationBasisOptions = [
  { label: 'On creation', value: 'on_creation' },
  { label: 'On first validation', value: 'on_first_validation' }
]

const productItems = computed(() => productSelect.items)

watch(open, async (value) => {
  if (!value) {
    return
  }
  if (!props.policy) {
    await productSelect.fetchItems('')
  }
  if (props.policy) {
    form.productId = props.policy.product_id
    form.name = props.policy.name
    form.description = props.policy.description ?? ''
    form.perpetual = props.policy.duration_days == null
    form.durationDays = props.policy.duration_days ?? 30
    form.expirationBasis = props.policy.expiration_basis
    form.gracePeriodDays = props.policy.grace_period_days
  } else {
    form.productId = null
    form.name = ''
    form.description = ''
    form.perpetual = true
    form.durationDays = 30
    form.expirationBasis = 'on_creation'
    form.gracePeriodDays = 0
  }
  error.value = ''
})

const onSubmit = async () => {
  if (!props.policy && !form.productId) {
    error.value = 'Product is required'
    return
  }
  if (!form.name.trim()) {
    error.value = 'Name is required'
    return
  }
  if (!form.perpetual && (!form.durationDays || form.durationDays < 1)) {
    error.value = 'Duration must be at least 1 day'
    return
  }

  loading.value = true
  error.value = ''

  const durationDays = form.perpetual ? null : form.durationDays
  const payload = {
    name: form.name.trim(),
    description: form.description.trim() || null,
    duration_days: durationDays,
    expiration_basis: form.expirationBasis,
    grace_period_days: form.gracePeriodDays
  }

  try {
    if (props.policy) {
      await updatePolicy(props.policy.id, payload)
    } else {
      await createPolicy({
        product_id: form.productId!,
        ...payload
      })
    }
    open.value = false
    emit('saved')
  } catch {
    error.value = props.policy ? 'Failed to update policy' : 'Failed to create policy'
  } finally {
    loading.value = false
  }
}
</script>
