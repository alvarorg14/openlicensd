<template>
  <UModal v-model:open="open" :title="product ? 'Edit product' : 'Create product'">
    <template #header>
      <div class="flex items-center gap-2">
        <div class="flex h-8 w-8 items-center justify-center rounded-lg bg-indigo-100 dark:bg-indigo-900/40">
          <UIcon :name="product ? 'i-lucide-pencil' : 'i-lucide-plus'" class="h-4 w-4 text-indigo-600 dark:text-indigo-400" />
        </div>
        <span class="font-semibold">{{ product ? 'Edit product' : 'Create product' }}</span>
      </div>
    </template>
    <template #body>
      <UForm :state="form" class="space-y-4" @submit="onSubmit">
        <UFormField label="Name" name="name" required>
          <UInput v-model="form.name" placeholder="e.g. Acme Widget Pro" />
        </UFormField>

        <UFormField label="Code" name="code" required>
          <UInput v-model="form.code" placeholder="e.g. acme-widget" />
        </UFormField>

        <UFormField label="Description" name="description">
          <UTextarea v-model="form.description" placeholder="Optional description" :rows="3" />
        </UFormField>

        <UAlert v-if="error" color="error" variant="subtle" :title="error" class="animate-fade-in" />

        <div class="flex justify-end gap-2 pt-2">
          <UButton color="neutral" variant="outline" @click="open = false">
            Cancel
          </UButton>
          <UButton type="submit" :loading="loading">
            {{ product ? 'Save' : 'Create' }}
          </UButton>
        </div>
      </UForm>
    </template>
  </UModal>
</template>

<script setup lang="ts">
import type { Product } from '~/types'

const props = defineProps<{
  product: Product | null
}>()

const open = defineModel<boolean>('open', { required: true })
const emit = defineEmits<{
  saved: []
}>()

const { createProduct, updateProduct } = useApi()

const form = reactive({
  name: '',
  code: '',
  description: ''
})

const loading = ref(false)
const error = ref('')

watch(open, (value) => {
  if (!value) {
    return
  }
  if (props.product) {
    form.name = props.product.name
    form.code = props.product.code
    form.description = props.product.description ?? ''
  } else {
    form.name = ''
    form.code = ''
    form.description = ''
  }
  error.value = ''
})

const onSubmit = async () => {
  if (!form.name.trim()) {
    error.value = 'Name is required'
    return
  }
  if (!form.code.trim()) {
    error.value = 'Code is required'
    return
  }

  loading.value = true
  error.value = ''

  const payload = {
    name: form.name.trim(),
    code: form.code.trim(),
    description: form.description.trim() || null
  }

  try {
    if (props.product) {
      await updateProduct(props.product.id, payload)
    } else {
      await createProduct(payload)
    }
    open.value = false
    emit('saved')
  } catch {
    error.value = props.product ? 'Failed to update product' : 'Failed to create product'
  } finally {
    loading.value = false
  }
}
</script>
