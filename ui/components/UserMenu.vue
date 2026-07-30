<template>
  <UDropdownMenu
    v-if="user"
    :items="menuItems"
    :content="{ side, align: 'start' }"
    :ui="{ content: 'w-56' }"
  >
    <UButton
      color="neutral"
      variant="ghost"
      class="gap-3"
      :class="compact ? 'px-2' : 'w-full justify-start px-2'"
    >
      <UAvatar :alt="user.name" :text="initials" size="sm" />
      <div v-if="!compact" class="flex-1 min-w-0 text-left">
        <p class="text-sm font-medium text-slate-900 dark:text-white truncate">{{ user.name }}</p>
        <p class="text-xs text-slate-500 dark:text-slate-400 truncate">{{ user.email }}</p>
      </div>
      <UIcon
        :name="side === 'top' ? 'i-lucide-chevron-up' : 'i-lucide-chevron-down'"
        class="h-4 w-4 shrink-0 text-slate-400"
      />
    </UButton>

    <template #account-header-trailing>
      <UBadge color="neutral" variant="subtle" size="xs" class="capitalize shrink-0">
        {{ user.role }}
      </UBadge>
    </template>
  </UDropdownMenu>

  <ChangePasswordModal v-model:open="showChangePassword" />
</template>

<script setup lang="ts">
import type { DropdownMenuItem } from '@nuxt/ui'

withDefaults(defineProps<{
  side?: 'top' | 'bottom'
  compact?: boolean
}>(), {
  side: 'top',
  compact: false
})

const { user, logout } = useAuth()
const colorMode = useColorMode()
const showChangePassword = ref(false)

const initials = computed(() => {
  if (!user.value?.name) {
    return '?'
  }

  return user.value.name
    .split(/\s+/)
    .map((part) => part[0])
    .slice(0, 2)
    .join('')
    .toUpperCase()
})

const menuItems = computed<DropdownMenuItem[][]>(() => {
  if (!user.value) {
    return []
  }

  const preference = colorMode.preference

  return [
    [{
      slot: 'account-header',
      label: user.value.name,
      description: user.value.email,
      avatar: { text: initials.value },
      disabled: true
    }],
    ...(user.value.has_password
      ? [[{
          label: 'Change password',
          icon: 'i-lucide-key-round',
          onSelect: () => { showChangePassword.value = true }
        }]]
      : []),
    [{
      label: 'Theme',
      icon: 'i-lucide-sun-moon',
      children: [
        {
          label: 'Light',
          icon: 'i-lucide-sun',
          type: 'checkbox',
          checked: preference === 'light',
          onSelect: () => { colorMode.preference = 'light' }
        },
        {
          label: 'Dark',
          icon: 'i-lucide-moon',
          type: 'checkbox',
          checked: preference === 'dark',
          onSelect: () => { colorMode.preference = 'dark' }
        },
        {
          label: 'System',
          icon: 'i-lucide-monitor',
          type: 'checkbox',
          checked: preference === 'system',
          onSelect: () => { colorMode.preference = 'system' }
        }
      ]
    }],
    [{
      label: 'Logout',
      icon: 'i-lucide-log-out',
      onSelect: () => { void logout() }
    }]
  ]
})
</script>
