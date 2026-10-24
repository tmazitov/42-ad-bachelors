<script setup lang="ts">
defineProps<{ message: string; overlay?: boolean }>()
</script>

<template>
  <div class="page-loader" :class="{ 'page-loader--overlay': overlay }" role="status" aria-live="polite">
    <div class="page-loader__spinner" />
    <span class="page-loader__message">{{ message }}</span>
  </div>
</template>

<style scoped>
.page-loader {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 1rem;
  padding: 4rem 1rem;
}

.page-loader--overlay {
  position: fixed;
  inset: 0;
  z-index: 10000;
  padding: 0;
  background: color-mix(in srgb, var(--color-background) 85%, transparent);
  backdrop-filter: blur(2px);
}

.page-loader__spinner {
  width: 3rem;
  height: 3rem;
  border-radius: 50%;
  border: 4px solid var(--card-border);
  border-top-color: var(--p-primary-color, #10b981);
  animation: page-loader-spin 0.8s linear infinite;
}

@keyframes page-loader-spin {
  to { transform: rotate(360deg); }
}

.page-loader__message {
  font-size: 0.9rem;
  color: var(--p-text-muted-color, #9ca3af);
}
</style>
