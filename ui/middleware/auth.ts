export default defineNuxtRouteMiddleware(async () => {
  const { isAuthenticated, authReady, fetchMe } = useAuth()

  if (!authReady.value) {
    await fetchMe()
  }

  if (!isAuthenticated.value) {
    return navigateTo('/login')
  }
})
