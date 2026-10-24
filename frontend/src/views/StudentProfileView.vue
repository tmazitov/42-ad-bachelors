<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Button, Tag } from 'primevue'
import { api, ApiError } from '@/services/api'
import type { Student } from '@/models/Student'
import { useAuthStore } from '@/stores/auth'
import Profile from '@/components/profile/Profile.vue'
import { proxiedImage } from '@/utils/avatar'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

const student = ref<Student | null>(null)
const loading = ref(false)
const error = ref<string | null>(null)

const avatarUrl = computed(() => proxiedImage(student.value?.image))
const isOwn = computed(() => !!student.value && student.value.id === auth.user?.id)

watch(
  () => route.params.login as string,
  async (login) => {
    student.value = null
    error.value = null
    loading.value = true
    try {
      // The backend serves stored data and refreshes it from 42 when stale.
      student.value = await api<Student>(`/students/${encodeURIComponent(login)}`)
    } catch (e) {
      error.value = e instanceof ApiError && e.status === 404
        ? 'No such student on 42.'
        : e instanceof Error ? e.message : 'Failed to load profile'
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

    <template v-else-if="student">
      <div class="profile-card">
        <div class="profile-avatar">
          <span class="profile-avatar__initial">{{ student.login[0]?.toUpperCase() }}</span>
          <img
            v-if="avatarUrl"
            :src="avatarUrl"
            class="profile-avatar__img"
            alt=""
            @load="(e) => (e.target as HTMLElement).classList.add('is-loaded')"
          />
        </div>

        <div class="profile-info">
          <span class="profile-displayname">{{ student.display_name }}</span>
          <a
            :href="`https://profile.intra.42.fr/users/${student.login}`"
            target="_blank"
            rel="noopener noreferrer"
            class="profile-login"
          >{{ student.login }}</a>

          <div v-if="student.alumni || student.staff" class="profile-chips">
            <Tag v-if="student.alumni" value="Alumni" severity="secondary" icon="pi pi-graduation-cap" />
            <Tag v-if="student.staff" value="Staff" severity="warn" icon="pi pi-shield" />
          </div>
        </div>
      </div>

      <Profile :student="student" :own="isOwn" />
    </template>
  </main>
</template>

<style scoped>
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
</style>
