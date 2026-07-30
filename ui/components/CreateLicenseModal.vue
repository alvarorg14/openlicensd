<template>
  <UModal v-model:open="open" title="Create license">
    <template #header>
      <div class="flex items-center gap-2">
        <div class="flex h-8 w-8 items-center justify-center rounded-lg bg-indigo-100 dark:bg-indigo-900/40">
          <UIcon name="i-lucide-plus" class="h-4 w-4 text-indigo-600 dark:text-indigo-400" />
        </div>
        <span class="font-semibold">Create license</span>
      </div>
    </template>
    <template #body>
      <UForm :state="form" class="space-y-4" @submit="onSubmit">
        <UFormField label="Label" name="label" required>
          <UInput v-model="form.label" placeholder="e.g. Acme Corp production" />
        </UFormField>

        <ProductPolicySelect
          v-model:product-id="form.productId"
          v-model:policy-id="form.policyId"
        />

        <UFormField name="usePolicyExpiration">
          <UCheckbox v-model="form.usePolicyExpiration" label="Use policy expiration" />
        </UFormField>

        <p v-if="form.usePolicyExpiration && policyExpirationHint" class="text-sm text-slate-500 dark:text-slate-400">
          {{ policyExpirationHint }}
        </p>

        <UFormField v-if="!form.usePolicyExpiration" label="Expiration date" name="expiresAt">
          <UInput v-model="form.expiresAt" type="datetime-local" />
        </UFormField>

        <UAlert v-if="error" color="error" variant="subtle" :title="error" class="animate-fade-in" />

        <div class="flex justify-end gap-2 pt-2">
          <UButton color="neutral" variant="outline" @click="open = false">
            Cancel
          </UButton>
          <UButton type="submit" :loading="loading">
            Create
          </UButton>
        </div>
      </UForm>
    </template>
  </UModal>
</template>

<script setup lang="ts">
import type { License, Policy } from '~/types'

const open = defineModel<boolean>('open', { required: true })
const emit = defineEmits<{
  created: [license: License]
}>()

const { createLicense, listPolicies } = useApi()

const form = reactive({
  label: '',
  productId: null as string | null,
  policyId: null as string | null,
  usePolicyExpiration: true,
  expiresAt: ''
})

const loading = ref(false)
const error = ref('')
const selectedPolicy = ref<Policy | null>(null)

const policyExpirationHint = computed(() => {
  if (!selectedPolicy.value) {
    return ''
  }
  if (selectedPolicy.value.duration_days == null) {
    return 'This policy never expires.'
  }
  if (selectedPolicy.value.expiration_basis === 'on_first_validation') {
    return `Expires ${selectedPolicy.value.duration_days} days after first validation.`
  }
  return `Expires ${selectedPolicy.value.duration_days} days from creation.`
})

watch(() => form.policyId, async (policyId) => {
  selectedPolicy.value = null
  if (!policyId || !form.productId) {
    return
  }
  const policies = await listPolicies(form.productId)
  selectedPolicy.value = policies.find((policy) => policy.id === policyId) ?? null
})

watch(open, (value) => {
  if (value) {
    form.label = ''
    form.productId = null
    form.policyId = null
    form.usePolicyExpiration = true
    form.expiresAt = ''
    selectedPolicy.value = null
    error.value = ''
  }
})

const onSubmit = async () => {
  if (!form.label.trim()) {
    error.value = 'Label is required'
    return
  }
  if (!form.productId) {
    error.value = 'Product is required'
    return
  }
  if (!form.policyId) {
    error.value = 'Policy is required'
    return
  }
  if (!form.usePolicyExpiration && !form.expiresAt) {
    error.value = 'Expiration date is required when not using policy expiration'
    return
  }

  loading.value = true
  error.value = ''

  try {
    const body: {
      label: string
      product_id: string
      policy_id: string
      expires_at?: string | null
    } = {
      label: form.label.trim(),
      product_id: form.productId,
      policy_id: form.policyId
    }

    if (!form.usePolicyExpiration) {
      body.expires_at = new Date(form.expiresAt).toISOString()
    }

    const license = await createLicense(body)
    open.value = false
    emit('created', license)
  } catch {
    error.value = 'Failed to create license'
  } finally {
    loading.value = false
  }
}
</script>
