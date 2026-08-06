<template>
  <UDashboardGroup unit="rem" storage-key="openlicensd-ui">
    <AppSidebar />
    <div class="flex-1 flex flex-col min-w-0 overflow-y-auto app-bg font-sans antialiased">
      <AppHeader />
      <main class="flex-1">
        <slot />
      </main>
    </div>
  </UDashboardGroup>
</template>
