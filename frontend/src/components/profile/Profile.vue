<script setup lang="ts">
import { computed } from 'vue'
import type { Student } from '@/models/Student'
import ProgressChart from '@/components/profile/ProgressChart.vue'

const props = defineProps<{ student: Student | null; own?: boolean; loading?: boolean }>()

const projects = computed(() => props.student?.projects ?? [])
</script>

<template>
  <section class="flex flex-col gap-4">
    <ProgressChart
      :completed-projects="projects"
      :loading="loading || !student"
      :stream="student?.stream ?? null"
      :marked-modules="student?.marked_modules ?? []"
      :own="own"
      :title="own ? 'Your Progress' : `${student?.login ?? ''}'s Progress`"
    />
  </section>
</template>
