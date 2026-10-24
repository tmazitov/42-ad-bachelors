<script setup lang="ts">
import Profile from '@/components/profile/Profile.vue'
import ProjectLookup from '@/components/ProjectLookup.vue'
import GreetingsMessage from '@/components/GreetingsMessage.vue'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
</script>

<template>
  <main class="flex flex-col gap-4">
    <template v-if="auth.isLoggedIn">
      <Profile />
      <!-- <ProjectLookup /> -->
    </template>
    <GreetingsMessage v-else />
  </main>
</template>
