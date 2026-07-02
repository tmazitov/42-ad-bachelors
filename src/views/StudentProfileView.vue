<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Button, Tag } from 'primevue'
import { useAuthStore } from '@/stores/auth'
import type { FtUser } from '@/stores/auth'
import { apiFetch } from '@/services/apiFetch'
import Profile from '@/components/profile/Profile.vue'

interface ProjectInProgress {
  id: number
  'validated?': boolean
  project: { id: number; name: string; slug: string }
}

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

const user = ref<FtUser | null>(null)
const inProgress = ref<ProjectInProgress[]>([])
const loading = ref(false)
const error = ref<string | null>(null)

const avatarUrl = computed(() =>
  user.value?.image?.versions?.medium ?? user.value?.image?.link ?? null
)

watch(
  () => route.params.login as string,
  async (login) => {
    if (!auth.token) {
      router.replace('/')
      return
    }
    user.value = null
    inProgress.value = []
    error.value = null
    loading.value = true
    try {
      const base = import.meta.env.VITE_42_API_BASE || ''
      const [userRes, progressRes] = await Promise.all([
        apiFetch(`${base}/v2/users/${login}`),
        apiFetch(`${base}/v2/users/${login}/projects_users?filter[status]=in_progress&per_page=20`),
      ])

      if (!userRes.ok) throw new Error(`${userRes.status}`)
      user.value = await userRes.json()

      if (progressRes.ok) {
        const data: ProjectInProgress[] = await progressRes.json()
        inProgress.value = data.filter(pu => !pu['validated?'])
      }
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load profile'
    } finally {
      loading.value = false
    }
  },
  { immediate: true },
)
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
      <span class="font-semibold text-lg">{{ route.params.login }}</span>
    </div>

    <div v-if="loading" class="flex justify-center py-12">
      <i class="pi pi-spinner pi-spin text-3xl text-gray-400" />
    </div>

    <div v-else-if="error" class="text-center py-12 text-red-400 text-sm">
      Could not load profile: {{ error }}
    </div>

    <template v-else-if="user">
      <div class="profile-row">
      <div class="profile-card">
        <div class="profile-avatar">
          <span class="profile-avatar__initial">{{ user.login[0]?.toUpperCase() }}</span>
          <img
            v-if="avatarUrl"
            :src="avatarUrl"
            class="profile-avatar__img"
            alt=""
            @load="(e) => (e.target as HTMLElement).classList.add('is-loaded')"
          />
        </div>

        <div class="profile-info">
          <span class="profile-displayname">{{ user.displayname }}</span>
          <a
            :href="`https://profile.intra.42.fr/users/${user.login}`"
            target="_blank"
            rel="noopener noreferrer"
            class="profile-login"
          >{{ user.login }}</a>

          <div v-if="user['alumni?'] || user['staff?'] || user.location" class="profile-chips">
            <Tag v-if="user['alumni?']" value="Alumni" severity="secondary" icon="pi pi-graduation-cap" />
            <Tag v-if="user['staff?']" value="Staff" severity="warn" icon="pi pi-shield" />
            <Tag v-if="user.location" :value="user.location" severity="success" icon="pi pi-map-marker" />
          </div>
        </div>
      </div>

      <div v-if="inProgress.length" class="in-progress-card">
        <span class="in-progress-card__title">
          <i class="pi pi-box" />
          Current projects
        </span>
        <div class="in-progress-card__list">
          <a
            v-for="pu in inProgress"
            :key="pu.id"
            :href="`https://projects.intra.42.fr/projects/${pu.project.slug}`"
            target="_blank"
            rel="noopener noreferrer"
            class="in-progress-chip"
          >{{ pu.project.name }}</a>
        </div>
      </div>
      </div>

      <Profile :user="user" />
    </template>
  </main>
</template>

<style scoped>
.profile-row {
  display: flex;
  gap: 1rem;
}

.profile-row > .profile-card,
.profile-row > .in-progress-card {
  flex: 1 1 0;
  min-width: 0;
}

@media (max-width: 640px) {
  .profile-row {
    flex-direction: column;
  }
}

.profile-card {
  display: flex;
  align-items: center;
  gap: 1.25rem;
  padding: 1rem 1.25rem;
  border: 1px solid var(--card-border);
  border-radius: 1.5rem;
  background: var(--card-bg);
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
}

.profile-avatar {
  position: relative;
  width: 64px;
  height: 64px;
  border-radius: 50%;
  overflow: hidden;
  background: var(--p-primary-color, #3b82f6);
  flex-shrink: 0;
}

.profile-avatar__initial {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 700;
  font-size: 1.5rem;
  color: #fff;
  user-select: none;
}

.profile-avatar__img {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
  opacity: 0;
  transition: opacity 0.2s ease;
}

.profile-avatar__img.is-loaded {
  opacity: 1;
}

.profile-info {
  display: flex;
  flex-direction: column;
  gap: 0.2rem;
  min-width: 0;
}

.profile-displayname {
  font-weight: 600;
  font-size: 1.1rem;
  line-height: 1.3;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.profile-login {
  font-size: 0.875rem;
  color: var(--p-text-muted-color, #9ca3af);
  line-height: 1.3;
  text-decoration: none;
  width: fit-content;
}

.profile-login:hover {
  text-decoration: underline;
}

.profile-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 0.375rem;
  margin-top: 0.375rem;
}

/* ── In-progress card ────────────────── */
.in-progress-card {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  padding: 1rem 1.25rem;
  border: 1px solid var(--card-border);
  border-radius: 1.5rem;
  background: var(--card-bg);
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
}

.in-progress-card__title {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--p-text-muted-color, #9ca3af);
}

.in-progress-card__list {
  display: flex;
  flex-wrap: wrap;
  gap: 0.375rem;
}

.in-progress-chip {
  font-size: 0.75rem;
  font-weight: 500;
  padding: 0.25rem 0.625rem;
  border-radius: 999px;
  border: 1px solid var(--card-border);
  background: var(--color-background-mute);
  color: inherit;
  text-decoration: none;
  transition: border-color 0.15s, color 0.15s;
  white-space: nowrap;
}

.in-progress-chip:hover {
  border-color: var(--p-primary-color, #3b82f6);
  color: var(--p-primary-color, #3b82f6);
}
</style>
