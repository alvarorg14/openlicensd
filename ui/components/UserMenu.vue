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
      :class="[
        compact ? 'px-2' : 'w-full justify-start px-2',
        onDark ? 'hover:bg-white/10' : ''
      ]"
    >
      <UAvatar :alt="user.name" :text="initials" size="sm" />
      <div v-if="!compact" class="flex-1 min-w-0 text-left">
        <p
          class="text-sm font-medium truncate"
          :class="onDark ? 'text-white' : 'text-highlighted'"
        >
          {{ user.name }}
        </p>
        <p
          class="text-xs truncate"
          :class="onDark ? 'text-navy-300' : 'text-muted dark:text-dimmed'"
        >
          {{ user.email }}
        </p>
      </div>
      <UIcon
        :name="side === 'top' ? 'i-lucide-chevron-up' : 'i-lucide-chevron-down'"
        class="h-4 w-4 shrink-0"
        :class="onDark ? 'text-navy-300' : 'text-dimmed'"
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
import { GITHUB_URL } from '~/constants/app'

const props = withDefaults(defineProps<{
  side?: 'top' | 'bottom'
  compact?: boolean
  onDark?: boolean
}>(), {
  side: 'top',
  compact: false,
  onDark: false
})

const auth = useAuth()
const user = auth.user
const logout = auth.logout
const serverVersion = auth.serverVersion
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
      label: 'Star us on GitHub',
      icon: 'i-lucide-github',
      to: GITHUB_URL,
      target: '_blank'
    }],
    [{
      label: 'Logout',
      icon: 'i-lucide-log-out',
      onSelect: () => { void logout() }
    }],
    ...(props.compact && serverVersion.value
      ? [[{
          label: serverVersion.value,
          icon: 'i-lucide-info',
          disabled: true
        }]]
      : [])
  ]
})
</script>
