<template>
  <div class="space-y-4">
    <UFormField label="Product" name="product" required>
      <USelectMenu
        v-model="selectedProduct"
        :items="productItems"
        value-key="value"
        label-key="label"
        placeholder="Select a product"
        :loading="loadingProducts"
        class="w-full"
      />
    </UFormField>

    <UFormField label="Policy" name="policy" required>
      <USelectMenu
        v-model="selectedPolicy"
        :items="policyItems"
        value-key="value"
        label-key="label"
        placeholder="Select a policy"
        :loading="loadingPolicies"
        :disabled="!selectedProduct"
        class="w-full"
      />
    </UFormField>
  </div>
</template>

<script setup lang="ts">
import type { Policy, Product } from '~/types'

const productId = defineModel<string | null>('productId', { required: true })
const policyId = defineModel<string | null>('policyId', { required: true })

const { listProducts, listPolicies } = useApi()

const products = ref<Product[]>([])
const policies = ref<Policy[]>([])
const loadingProducts = ref(false)
const loadingPolicies = ref(false)

const productItems = computed(() =>
  products.value.map((product) => ({
    label: product.name,
    value: product.id
  }))
)

const policyItems = computed(() =>
  policies.value.map((policy) => ({
    label: policy.name,
    value: policy.id
  }))
)

const selectedProduct = computed({
  get: () => productId.value,
  set: (value: string | null) => {
    productId.value = value
    policyId.value = null
    policies.value = []
    if (value) {
      fetchPolicies(value)
    }
  }
})

const selectedPolicy = computed({
  get: () => policyId.value,
  set: (value: string | null) => {
    policyId.value = value
  }
})

const fetchProducts = async () => {
  loadingProducts.value = true
  try {
    products.value = await listProducts()
  } finally {
    loadingProducts.value = false
  }
}

const fetchPolicies = async (id: string) => {
  loadingPolicies.value = true
  try {
    policies.value = await listPolicies(id)
  } finally {
    loadingPolicies.value = false
  }
}

watch(productId, async (value) => {
  if (value && policies.value.length === 0) {
    await fetchPolicies(value)
  }
})

onMounted(fetchProducts)
</script>
