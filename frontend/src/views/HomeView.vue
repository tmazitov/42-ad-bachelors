<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import Profile from '@/components/profile/Profile.vue'
import GreetingsMessage from '@/components/GreetingsMessage.vue'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const route = useRoute()
const router = useRouter()

// The backend's OAuth callback sends users back with ?login_error=<reason>.
const LOGIN_ERRORS: Record<string, string> = {
  denied: 'Sign-in was cancelled on 42.',
  state: 'Sign-in expired or was tampered with. Please try again.',
  failed: 'Could not sign in with 42. Please try again.',
  busy: '42 is busy right now. Please try signing in again in a minute.',
}
const loginError = computed(() => {
  const reason = route.query.login_error
  return typeof reason === 'string' ? (LOGIN_ERRORS[reason] ?? LOGIN_ERRORS.failed) : null
})

function dismissLoginError() {
  router.replace({ query: {} })
}
</script>

<template>
  <main class="flex flex-col gap-4">
    <div v-if="loginError" class="login-error">
      <i class="pi pi-exclamation-triangle" />
      <span>{{ loginError }}</span>
      <button class="login-error__close" aria-label="Dismiss" @click="dismissLoginError">
        <i class="pi pi-times" />
      </button>
    </div>

    <Profile v-if="auth.isLoggedIn" :student="auth.user" own />
    <GreetingsMessage v-else-if="!auth.loading" />
  </main>
</template>

<style scoped>
.login-error {
  display: flex;
  align-items: center;
  gap: 0.625rem;
  padding: 0.75rem 1rem;
  border: 1px solid color-mix(in srgb, #ef4444 40%, var(--card-border));
  border-radius: 1rem;
  background: color-mix(in srgb, #ef4444 8%, var(--card-bg));
  color: #ef4444;
  font-size: 0.875rem;
}

.login-error__close {
  margin-left: auto;
  background: none;
  border: none;
  color: inherit;
  cursor: pointer;
}
</style>
