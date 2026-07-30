export default defineNuxtPlugin(async () => {
  const { fetchMe, authReady } = useAuth()
  await fetchMe()
  authReady.value = true
})
