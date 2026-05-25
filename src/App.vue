<script setup lang="ts">
import { RouterView } from 'vue-router'
import AppHeader from '@/components/AppHeader.vue'
</script>

<template>
<div class="flex gap-4 justify-center p-4">
  <div class="max-w-[900px] w-full flex flex-col gap-4">
    <AppHeader />
    <RouterView />
  </div>
</div>
</template>

<style lang="css" scoped>

</style>