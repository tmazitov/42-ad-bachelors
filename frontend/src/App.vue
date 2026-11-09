<script setup lang="ts">
import { RouterView } from 'vue-router'
import AppHeader from '@/components/AppHeader.vue'
import FullPageLoader from '@/components/FullPageLoader.vue'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
</script>

<template>
<div class="flex gap-4 justify-center p-4">
  <div class="max-w-[1160px] w-full flex flex-col gap-4">
    <AppHeader />
    <!-- Until the first /me answers we don't know if anyone is signed in -->
    <FullPageLoader v-if="!auth.checked" message="Loading…" />
    <RouterView v-else />
    <footer class="app-footer">
      <span>
        Having issues?
        <a href="https://github.com/tmazitov/42-ad-bachelors/issues" target="_blank" rel="noopener noreferrer">
          Report on GitHub
        </a>
      </span>
      <span class="footer-divider" />
      <span>
        Developed by
        <a href="https://github.com/tmazitov" target="_blank" rel="noopener noreferrer">
          tmazitov
        </a>
      </span>
    </footer>
  </div>
</div>
<FullPageLoader v-if="auth.redirecting" message="Redirecting to 42…" overlay />
</template>

<style lang="css" scoped>
.app-footer {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.75rem;
  padding: 0.75rem 1rem;
  font-size: 0.8125rem;
  color: var(--p-text-muted-color, #9ca3af);
}

.app-footer a {
  color: inherit;
  text-decoration: underline;
  text-underline-offset: 2px;
  transition: color 0.15s;
}

.app-footer a:hover {
  color: var(--p-primary-color, #3b82f6);
}

.footer-divider {
  display: inline-block;
  width: 1px;
  height: 0.875rem;
  background: currentColor;
  opacity: 0.3;
}

/* Phones: one line per item, no divider */
@media (max-width: 640px) {
  .app-footer {
    flex-direction: column;
    gap: 0.25rem;
    text-align: center;
  }

  .footer-divider {
    display: none;
  }
}
</style>