<script setup lang="ts">
definePageMeta({
  middleware: 'auth'
})

await navigateTo('/licenses', { replace: true })
</script>
