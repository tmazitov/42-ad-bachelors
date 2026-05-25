<script setup lang="ts">
import { onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { exchangeCode } from '@/services/oauth'

const router = useRouter()
const route = useRoute()
const auth = useAuthStore()

onMounted(async () => {
  const code = route.query.code as string | undefined

  if (!code) {
    router.replace('/')
    return
  }

  try {
    const token = await exchangeCode(code)
    auth.setToken(token)
    await auth.fetchUser()
  } catch (e) {
    console.error('OAuth callback failed:', e)
  } finally {
    router.replace('/')
  }
})
</script>

<template>
  <div class="flex items-center justify-center h-screen gap-3">
    <i class="pi pi-spin pi-spinner text-2xl" />
    <span class="text-lg">Authenticating…</span>
  </div>
</template>
