<template>
  <div class="min-h-screen flex">
    <AppSidebar />
    <div class="flex-1 flex flex-col min-w-0">
      <AppHeader />
      <main class="flex-1">
        <slot />
      </main>
    </div>
  </div>
</template>
