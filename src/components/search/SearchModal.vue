<script setup lang="ts">
import { watch, nextTick, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useStudentSearch } from '@/composables/useStudentSearch'

const props = defineProps<{ visible: boolean }>()
const emit = defineEmits<{ 'update:visible': [value: boolean] }>()

const router = useRouter()
const { query, results, loading, page, totalPages, goToPage, reset } = useStudentSearch()
const inputRef = ref<HTMLInputElement | null>(null)

function close() {
  emit('update:visible', false)
}

watch(() => props.visible, (val) => {
  if (!val) {
    reset()
  } else {
    nextTick(() => inputRef.value?.focus())
  }
})

function selectUser(login: string) {
  close()
  router.push(`/profile/${login}`)
}
</script>

<template>
  <Teleport to="body">
    <Transition name="search-modal">
      <div
        v-if="visible"
        class="search-overlay"
        @click.self="close"
        @keydown.esc="close"
      >
        <div class="search-box" role="dialog" aria-modal="true" aria-label="Search students">
          <div class="drag-handle" />

          <div class="search-field-wrap">
            <i class="pi pi-search search-icon" />
            <input
              ref="inputRef"
              v-model="query"
              placeholder="Search students by login…"
              class="search-input"
              autocomplete="off"
              spellcheck="false"
              @keydown.esc="close"
            />
            <i v-if="loading" class="pi pi-spinner pi-spin input-spinner" />
            <button class="close-btn" aria-label="Close" @click="close">
              <i class="pi pi-times" />
            </button>
          </div>

          <div class="results-wrap">
            <template v-if="results.length">
              <button
                v-for="user in results"
                :key="user.id"
                class="result-item"
                @click="selectUser(user.login)"
              >
                <div class="result-avatar">
                  <span class="result-avatar__initial">{{ user.login[0]?.toUpperCase() }}</span>
                  <img
                    v-if="user.image?.versions?.small ?? user.image?.link"
                    :src="(user.image?.versions?.small ?? user.image?.link)!"
                    class="result-avatar__img"
                    alt=""
                    @load="(e) => (e.target as HTMLElement).classList.add('is-loaded')"
                  />
                </div>
                <div class="result-info">
                  <span class="result-login">{{ user.login }}</span>
                  <span class="result-name">{{ user.displayname }}</span>
                </div>
              </button>
            </template>

            <p v-else-if="query.trim() && !loading" class="empty-hint">No students found</p>
            <p v-else-if="!query.trim()" class="empty-hint">Type a login to search</p>
          </div>

          <div v-if="totalPages > 1" class="pagination">
            <button
              class="page-btn"
              :disabled="page === 1 || loading"
              aria-label="Previous page"
              @click="goToPage(page - 1)"
            >
              <i class="pi pi-chevron-left" />
            </button>
            <span class="page-indicator">{{ page }} / {{ totalPages }}</span>
            <button
              class="page-btn"
              :disabled="page === totalPages || loading"
              aria-label="Next page"
              @click="goToPage(page + 1)"
            >
              <i class="pi pi-chevron-right" />
            </button>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
/* ── Overlay ─────────────────────────── */
.search-overlay {
  position: fixed;
  inset: 0;
  z-index: 9999;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.45);
}

/* ── Box (desktop default) ───────────── */
.search-box {
  width: 440px;
  max-width: 92vw;
  padding: 1rem;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  border-radius: 1.5rem;
  background: var(--card-bg);
  border: 1px solid var(--card-border);
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.2);
}

.drag-handle {
  display: none;
}

/* ── Search field ────────────────────── */
.search-field-wrap {
  position: relative;
  display: flex;
  align-items: center;
}

.search-icon {
  position: absolute;
  left: 0.75rem;
  color: var(--p-text-muted-color, #9ca3af);
  font-size: 0.85rem;
  pointer-events: none;
}

.search-input {
  width: 100%;
  padding: 0.6rem 3.25rem 0.6rem 2.25rem;
  border: 1px solid var(--p-inputtext-border-color, var(--card-border));
  border-radius: 0.75rem;
  background: var(--p-inputtext-background, transparent);
  color: inherit;
  font-size: 0.925rem;
  outline: none;
  transition: border-color 0.15s;
}

.search-input:focus {
  border-color: var(--p-primary-color, #3b82f6);
}

.input-spinner {
  position: absolute;
  right: 2.75rem;
  color: var(--p-text-muted-color, #9ca3af);
  font-size: 0.85rem;
  pointer-events: none;
}

.close-btn {
  position: absolute;
  right: 0.4rem;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 1.75rem;
  height: 1.75rem;
  border-radius: 50%;
  border: none;
  background: transparent;
  color: var(--p-text-muted-color, #9ca3af);
  cursor: pointer;
  font-size: 0.75rem;
  transition: background 0.15s;
}

.close-btn:hover {
  background: var(--p-content-hover-background, rgba(0, 0, 0, 0.07));
}

/* ── Avatar ──────────────────────────── */
.result-avatar {
  position: relative;
  width: 2rem;
  height: 2rem;
  border-radius: 50%;
  overflow: hidden;
  background: var(--p-primary-color, #3b82f6);
  flex-shrink: 0;
}

.result-avatar__initial {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 600;
  font-size: 0.8125rem;
  color: #fff;
  user-select: none;
}

.result-avatar__img {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
  opacity: 0;
  transition: opacity 0.2s ease;
}

.result-avatar__img.is-loaded {
  opacity: 1;
}

/* ── Results ─────────────────────────── */
.results-wrap {
  display: flex;
  flex-direction: column;
  gap: 0.125rem;
  min-height: 2rem;
}

.result-item {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.5rem 0.625rem;
  border-radius: 0.75rem;
  width: 100%;
  text-align: left;
  cursor: pointer;
  border: none;
  background: transparent;
  color: inherit;
  transition: background 0.15s;
}

.result-item:hover {
  background: var(--p-content-hover-background, rgba(0, 0, 0, 0.05));
}

.result-info {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.result-login {
  font-weight: 500;
  font-size: 0.875rem;
  line-height: 1.3;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.result-name {
  font-size: 0.75rem;
  color: var(--p-text-muted-color, #9ca3af);
  line-height: 1.3;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.empty-hint {
  text-align: center;
  font-size: 0.875rem;
  color: var(--p-text-muted-color, #9ca3af);
  padding: 0.75rem 0;
  margin: 0;
}

/* ── Pagination ──────────────────────── */
.pagination {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  padding-top: 0.5rem;
  border-top: 1px solid var(--card-border);
}

.page-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 2rem;
  height: 2rem;
  border-radius: 0.5rem;
  border: none;
  background: transparent;
  color: inherit;
  cursor: pointer;
  font-size: 0.75rem;
  transition: background 0.15s;
}

.page-btn:hover:not(:disabled) {
  background: var(--p-content-hover-background, rgba(0, 0, 0, 0.05));
}

.page-btn:disabled {
  opacity: 0.3;
  cursor: default;
}

.page-indicator {
  font-size: 0.875rem;
  color: var(--p-text-muted-color, #9ca3af);
  min-width: 3.5rem;
  text-align: center;
}

/* ── Desktop transitions ─────────────── */
/* Backdrop fades via background-color so the box opacity stays independent */
.search-modal-enter-active,
.search-modal-leave-active {
  transition: background-color 0.2s ease;
}

.search-modal-enter-from,
.search-modal-leave-to {
  background-color: transparent;
}

.search-modal-enter-active .search-box,
.search-modal-leave-active .search-box {
  transition: transform 0.2s ease, opacity 0.2s ease;
}

.search-modal-enter-from .search-box,
.search-modal-leave-to .search-box {
  transform: scale(0.96) translateY(-6px);
  opacity: 0;
}

/* ── Mobile: bottom sheet ────────────── */
@media (max-width: 640px) {
  .search-overlay {
    align-items: flex-end;
  }

  .search-box {
    width: 100%;
    max-width: 100%;
    border-radius: 1.5rem 1.5rem 0 0;
    border-bottom: none;
    border-left: none;
    border-right: none;
    padding: 0.625rem 1rem max(1rem, env(safe-area-inset-bottom));
    box-shadow: 0 -4px 32px rgba(0, 0, 0, 0.15);
    gap: 0.625rem;
  }

  .drag-handle {
    display: block;
    width: 2.5rem;
    height: 4px;
    border-radius: 2px;
    background: var(--p-text-muted-color, #9ca3af);
    opacity: 0.35;
    margin: 0 auto;
    flex-shrink: 0;
  }

  /* Override box animation: slide up, no scale/fade */
  .search-modal-enter-from .search-box,
  .search-modal-leave-to .search-box {
    transform: translateY(100%);
    opacity: 1;
  }

  .search-modal-enter-active .search-box {
    transition: transform 0.35s cubic-bezier(0.32, 0.72, 0, 1);
  }

  .search-modal-leave-active .search-box {
    transition: transform 0.28s cubic-bezier(0.32, 0.72, 0, 1);
  }
}
</style>
