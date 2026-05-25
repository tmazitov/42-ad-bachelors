<script setup lang="ts">
import { computed } from 'vue'
import { Avatar, Button } from 'primevue'
import { useThemeStore } from '@/stores/theme'
import { useAuthStore } from '@/stores/auth'
import { redirectToLogin } from '@/services/oauth'

const theme = useThemeStore()
const auth = useAuthStore()

if (auth.token && !auth.user) {
  auth.fetchUser()
}

const avatarUrl = computed(() =>
  auth.user?.image?.versions?.medium ?? auth.user?.image?.link ?? null
)
</script>

<template>
  <header class="app-header">
    <span class="app-header__title">42 AD Bachelors Tracker</span>
    <div class="flex items-center gap-2">

      <template v-if="auth.isLoggedIn && auth.user">
        <div class="flex items-center gap-3">
          <Avatar
            :image="avatarUrl ?? undefined"
            :label="avatarUrl ? undefined : auth.user.login[0]?.toUpperCase()"
            shape="circle"
            class="cursor-pointer"
            :pt="{ image: { style: 'object-fit: cover; width: 100%; height: 100%;' } }"
          />
          <span class="font-medium text-md">{{ auth.user.login }}</span>
        </div>
        <Button
          icon="pi pi-sign-out"
          rounded
          outlined
          severity="secondary"
          @click="auth.logout()"
        />
      </template>
      <Button
        v-else
        label="Log In"
        icon="pi pi-sign-in"
        rounded
        :loading="auth.loading"
        @click="redirectToLogin()"
      />
      <Button
        :icon="theme.isDark ? 'pi pi-sun' : 'pi pi-moon'"
        :aria-label="theme.isDark ? 'Switch to light mode' : 'Switch to dark mode'"
        rounded
        outlined
        @click="theme.toggle()"
      />
    </div>
  </header>
</template>

<style scoped>
.app-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.5rem 0.5rem 0.5rem 1rem;
  border: 1px solid var(--p-gray-200);
  border-radius: 1.5rem;
  background: var(--p-gray-50);
}

.app-header__title {
  font-size: 1.1rem;
  font-weight: 600;
}
</style>
