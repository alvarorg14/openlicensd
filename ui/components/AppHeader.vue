<template>
  <header class="lg:hidden sticky top-0 z-30 border-b border-default bg-default/80 backdrop-blur-md">
    <UContainer class="flex h-14 items-center justify-between">
      <div class="flex items-center gap-2.5 min-w-0">
        <BrandMark class="h-7 w-auto text-navy-900 dark:text-white shrink-0" />
        <BrandWordmark class="h-5 w-auto text-navy-900 dark:text-white shrink-0" />
      </div>
      <UserMenu side="bottom" compact />
    </UContainer>
  </header>
</template>
