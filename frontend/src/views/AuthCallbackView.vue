<script setup lang="ts">
import { onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api, ApiError } from '@/services/api'
import { useAuthStore } from '@/stores/auth'
import FullPageLoader from '@/components/FullPageLoader.vue'

// 42 redirects here after the user approves the sign-in. The backend
// exchanges the code (and syncs the student's projects), which takes a few
// seconds, so this page shows a loader meanwhile.
const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

function param(name: string): string {
  const v = route.query[name]
  return typeof v === 'string' ? v : ''
}

onMounted(async () => {
  try {
    await api('/auth/callback', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ code: param('code'), state: param('state'), error: param('error') }),
    })
    await auth.fetchMe(true)
    router.replace('/')
  } catch (e) {
    const reason = e instanceof ApiError ? e.message : 'failed'
    router.replace({ path: '/', query: { login_error: reason } })
  }
})
</script>

<template>
  <FullPageLoader message="Signing you in…" />
</template>
