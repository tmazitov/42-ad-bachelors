<script setup lang="ts">
defineProps<{
  login: string
  image: string | null
  size?: number
  ring?: string
}>()
</script>

<template>
  <div
    class="avatar"
    :style="{
      width: `${size ?? 36}px`,
      height: `${size ?? 36}px`,
      fontSize: `${(size ?? 36) * 0.4}px`,
      boxShadow: ring ? `0 0 0 3px var(--card-bg), 0 0 0 5px ${ring}` : undefined,
    }"
  >
    <span class="avatar__initial">{{ login[0]?.toUpperCase() }}</span>
    <img
      v-if="image"
      :src="image"
      class="avatar__img"
      alt=""
      loading="lazy"
      @load="(e) => (e.target as HTMLElement).classList.add('is-loaded')"
    />
  </div>
</template>

<style scoped>
.avatar {
  position: relative;
  border-radius: 50%;
  background: var(--p-primary-color, #3b82f6);
  flex-shrink: 0;
}

.avatar__initial {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 700;
  color: #fff;
  user-select: none;
}

.avatar__img {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  border-radius: 50%;
  object-fit: cover;
  opacity: 0;
  transition: opacity 0.2s;
}

.avatar__img.is-loaded {
  opacity: 1;
}
</style>
