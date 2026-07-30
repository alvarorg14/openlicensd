<template>
  <div class="space-y-4">
    <UFormField label="Product" name="product" required>
      <USelectMenu
        v-model="selectedProduct"
        v-model:search-term="productSearchTerm"
        :items="productSelect.items"
        value-key="value"
        label-key="label"
        placeholder="Select a product"
        :loading="productSelect.loading"
        searchable
        class="w-full"
        @update:search-term="productSelect.onSearch"
      />
    </UFormField>

    <UFormField label="Policy" name="policy" required>
      <USelectMenu
        v-model="selectedPolicy"
        v-model:search-term="policySearchTerm"
        :items="policySelect.items"
        value-key="value"
        label-key="label"
        placeholder="Select a policy"
        :loading="policySelect.loading"
        :disabled="!selectedProduct"
        searchable
        class="w-full"
        @update:search-term="policySelect.onSearch"
      />
    </UFormField>
  </div>
</template>

<script setup lang="ts">
const productId = defineModel<string | null>('productId', { required: true })
const policyId = defineModel<string | null>('policyId', { required: true })

const { createProductSelect, createPolicySelect } = useServerSelect()
const productSelect = createProductSelect()
const policySelect = createPolicySelect(productId)

const productSearchTerm = ref('')
const policySearchTerm = ref('')

const selectedProduct = computed({
  get: () => productId.value,
  set: (value: string | null) => {
    productId.value = value
    policyId.value = null
    policySelect.clearItems()
    if (value) {
      policySelect.fetchItems('')
    }
  }
})

const selectedPolicy = computed({
  get: () => policyId.value,
  set: (value: string | null) => {
    policyId.value = value
  }
})

watch(productId, async (value) => {
  if (value && policySelect.items.length === 0) {
    await policySelect.fetchItems('')
  }
})
</script>
