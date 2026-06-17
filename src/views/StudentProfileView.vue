<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Button } from 'primevue'
import { useAuthStore } from '@/stores/auth'
import type { FtUser } from '@/stores/auth'
import Profile from '@/components/profile/Profile.vue'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

const login = route.params.login as string
const user = ref<FtUser | null>(null)
const loading = ref(false)
const error = ref<string | null>(null)

onMounted(async () => {
  if (!auth.token) {
    router.replace('/')
    return
  }
  loading.value = true
  try {
    const res = await fetch(
      `${import.meta.env.VITE_42_API_BASE || ''}/v2/users/${login}`,
      { headers: { Authorization: `Bearer ${auth.token}` } },
    )
    if (!res.ok) throw new Error(`${res.status}`)
    user.value = await res.json()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to load profile'
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <main class="flex flex-col gap-4">
    <div class="flex items-center gap-3">
      <Button
        icon="pi pi-arrow-left"
        rounded
        outlined
        severity="secondary"
        @click="router.back()"
      />
      <span class="font-semibold text-lg">{{ login }}</span>
    </div>

    <div v-if="loading" class="flex justify-center py-12">
      <i class="pi pi-spinner pi-spin text-3xl text-gray-400" />
    </div>

    <div v-else-if="error" class="text-center py-12 text-red-400 text-sm">
      Could not load profile: {{ error }}
    </div>

    <Profile v-else-if="user" :user="user" />
  </main>
</template>
