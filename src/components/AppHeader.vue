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
    <span class="flex items-center gap-2">
      <img src="/42-bachelors.png" alt="42" class="app-header__logo rounded-full" />
      
      <span class="app-header__title">
        42 AD Bachelor's Tracker
      </span>
    </span>


    <div class="app-header__buttons flex items-center gap-4">

      <div class="flex items-center gap-3" v-if="auth.isLoggedIn && auth.user">
        <Avatar
          :image="avatarUrl ?? undefined"
          :label="avatarUrl ? undefined : auth.user.login[0]?.toUpperCase()"
          shape="circle"
          class="cursor-pointer"
          :pt="{ image: { style: 'object-fit: cover; width: 100%; height: 100%;' } }"
        />
        <span class="font-medium text-md">{{ auth.user.login }}</span>
      </div>
      <span class="flex gap-2">
        <Button v-if="auth.isLoggedIn && auth.user"
          icon="pi pi-sign-out"
          rounded
          outlined
          severity="secondary"
          @click="auth.logout()"
        />
        <Button
          v-if="!(auth.isLoggedIn && auth.user)"
          label="Log In"
          icon="pi pi-sign-in"
          class="h-10"
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
      </span>
    </div>
  </header>
</template>

<style scoped>
.app-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.5rem;
  border: 1px solid var(--card-border);
  border-radius: 1.5rem;
  background: var(--card-bg);
}

.app-header__logo {
  width: 40px;
  height: 40px;
  object-fit: contain;
}

@media (max-width: 768px) {
  .app-header__buttons {
    flex-wrap: wrap;
  }
  .app-header__title {
    display: none;
  }
}

</style>
