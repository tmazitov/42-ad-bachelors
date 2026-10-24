<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Avatar, Button, Menu } from 'primevue'
import type { MenuItem } from 'primevue/menuitem'
import { useThemeStore } from '@/stores/theme'
import { useAuthStore } from '@/stores/auth'
import { CREDITS_REQUIRED } from '@/composables/useProgress'
import SearchModal from '@/components/search/SearchModal.vue'
import { proxiedImage } from '@/utils/avatar'

const router = useRouter()
const route = useRoute()
const theme = useThemeStore()
const auth = useAuthStore()

const avatarUrl = computed(() => proxiedImage(auth.user?.image))

const searchVisible = ref(false)

const PROGRESS_SEGMENTS = [
  { key: 'general' as const, color: '#22c55e' },
  { key: 'standard' as const, color: '#3b82f6' },
  { key: 'stream' as const, color: '#8b5cf6' },
  { key: 'elective' as const, color: '#f97316' },
]

const profileMenu = ref<InstanceType<typeof Menu> | null>(null)
const profileMenuOpen = ref(false)

const profileMenuItems = computed<MenuItem[]>(() => [
  {
    label: theme.isDark ? 'Light mode' : 'Dark mode',
    icon: theme.isDark ? 'pi pi-sun' : 'pi pi-moon',
    command: () => theme.toggle(),
  },
  {
    label: 'Log out',
    icon: 'pi pi-sign-out',
    command: async () => {
      await auth.logout()
      router.push('/')
    },
  },
])

function toggleProfileMenu(event: Event) {
  profileMenu.value?.toggle(event)
}

function openMyProgress() {
  profileMenu.value?.hide()
  router.push('/')
}
</script>

<template>
  <header class="app-header">
    <button class="app-header__home flex items-center gap-2" @click="router.push('/')">
      <img src="/42-bachelors.png" alt="42" class="app-header__logo rounded-full" loading="lazy" />
      <span class="app-header__title">42 AD Bachelor's Tracker</span>
    </button>

    <div class="app-header__buttons flex items-center gap-2">
      <template v-if="auth.isLoggedIn && auth.user">
        <Button
          label="Leaderboard"
          icon="pi pi-trophy"
          rounded
          outlined
          severity="secondary"
          class="app-header__leaderboard"
          @click="router.push('/leaderboard')"
        />
        <Button
          icon="pi pi-search"
          rounded
          outlined
          severity="secondary"
          aria-label="Search students"
          @click="searchVisible = true"
        />

        <button
          type="button"
          class="app-header__profile"
          aria-haspopup="true"
          :aria-expanded="profileMenuOpen"
          aria-controls="profile-menu"
          @click="toggleProfileMenu"
        >
          <Avatar
            :image="avatarUrl ?? undefined"
            :label="avatarUrl ? undefined : auth.user.login[0]?.toUpperCase()"
            shape="circle"
            :pt="{ image: { style: 'object-fit: cover; width: 100%; height: 100%;', loading: 'lazy' } }"
          />
          <span class="font-medium text-md">{{ auth.user.login }}</span>
          <i class="pi pi-chevron-down app-header__chevron" />
        </button>
        <Menu
          id="profile-menu"
          ref="profileMenu"
          :model="profileMenuItems"
          popup
          @show="profileMenuOpen = true"
          @hide="profileMenuOpen = false"
        >
          <template #start>
            <button type="button" class="menu-progress" title="Open my progress" @click="openMyProgress">
              <span class="menu-progress__name">{{ auth.user.display_name }}</span>
              <span class="menu-progress__login">{{ auth.user.login }}</span>

              <div class="menu-progress__row">
                <span class="menu-progress__label">Bachelor progress</span>
                <span class="menu-progress__pct">
                  {{ auth.user.progress.percentage.toFixed(1) }}%
                </span>
              </div>

              <div class="menu-progress__bar">
                <div
                  v-for="seg in PROGRESS_SEGMENTS"
                  :key="seg.key"
                  :style="{
                    width: `${(auth.user.progress[seg.key] / CREDITS_REQUIRED.total) * 100}%`,
                    background: seg.color,
                  }"
                />
              </div>

              <span class="menu-progress__credits">
                {{ auth.user.progress.total }} / {{ CREDITS_REQUIRED.total }} credits
              </span>
            </button>
          </template>
        </Menu>
      </template>

      <template v-else>
        <Button
          label="Log In"
          icon="pi pi-sign-in"
          class="h-10"
          rounded
          :loading="auth.loading || auth.redirecting || route.name === 'auth-callback'"
          @click="auth.login()"
        />
        <Button
          :icon="theme.isDark ? 'pi pi-sun' : 'pi pi-moon'"
          :aria-label="theme.isDark ? 'Switch to light mode' : 'Switch to dark mode'"
          rounded
          outlined
          @click="theme.toggle()"
        />
      </template>
    </div>
  </header>

  <SearchModal v-model:visible="searchVisible" />
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

.app-header__home {
  background: none;
  border: none;
  cursor: pointer;
  color: inherit;
  padding: 0;
}

.app-header__logo {
  width: 40px;
  height: 40px;
  object-fit: contain;
}

/* Leaderboard, search and profile share one height */
.app-header__leaderboard {
  height: 2.5rem;
}

.app-header__profile {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  height: 2.5rem;
  padding: 0 0.75rem 0 0.1875rem;
  border: 1px solid var(--card-border);
  border-radius: 999px;
  background: none;
  color: inherit;
  cursor: pointer;
  transition: background 0.15s, border-color 0.15s;
}

.app-header__profile:hover,
.app-header__profile[aria-expanded='true'] {
  background: var(--color-background-mute);
}

.menu-progress {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  width: 15rem;
  padding: 0.75rem 1rem 0.875rem;
  border: none;
  border-bottom: 1px solid var(--card-border);
  margin-bottom: 0.25rem;
  background: none;
  color: inherit;
  font: inherit;
  text-align: left;
  cursor: pointer;
  transition: background 0.15s;
}

.menu-progress:hover,
.menu-progress:focus-visible {
  background: var(--color-background-mute);
  outline: none;
}

.menu-progress__name {
  font-weight: 600;
  line-height: 1.2;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.menu-progress__login,
.menu-progress__label,
.menu-progress__credits {
  font-size: 0.75rem;
  color: var(--p-text-muted-color, #9ca3af);
}

.menu-progress__row {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  margin-top: 0.5rem;
}

.menu-progress__pct {
  font-size: 1.25rem;
  font-weight: 700;
  line-height: 1;
}

.menu-progress__bar {
  display: flex;
  height: 6px;
  border-radius: 999px;
  overflow: hidden;
  background: var(--card-border);
}

.menu-progress__bar > div {
  height: 100%;
}

.app-header__chevron {
  font-size: 0.7rem;
  color: var(--p-text-muted-color, #9ca3af);
}

@media (max-width: 768px) {
  .app-header__title {
    display: none;
  }
}

@media (max-width: 480px) {
  /* Keep everything on one row on phones: icon-only leaderboard, avatar-only profile */
  .app-header__leaderboard :deep(.p-button-label),
  .app-header__profile > span {
    display: none;
  }

  .app-header__leaderboard {
    width: 2.5rem;
    padding: 0;
  }

  .app-header__profile {
    padding-right: 0.5rem;
  }
}
</style>
