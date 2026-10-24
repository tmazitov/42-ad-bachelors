import { computed, type Ref } from 'vue'
import type { ProjectUser } from '@/composables/useProjects'
import type { Project } from '@/models/Project'
import { generalEducationModules, standardModules, streamModules, electiveModules } from '@/utils/data'

export const CREDITS_REQUIRED = {
  general:   5,
  standard: 101,
  stream:   14,
  elective: 12,
  total:   132,
} as const

export function isCompleted(project: Project, ids: Set<number>): boolean {
  if (project.id === null) return false
  if (typeof project.id === 'number') return ids.has(project.id)
  return project.id.some((id) => ids.has(id))
}

export function useProgress(completedProjects: Ref<ProjectUser[]>) {
  return computed(() => {
    const ids = new Set(completedProjects.value.filter((p) => p['validated?']).map((p) => p.project.id))

    const general = Math.min(
      generalEducationModules.projects
        .filter((p) => isCompleted(p, ids))
        .reduce((s, p) => s + p.credit, 0),
      CREDITS_REQUIRED.general,
    )

    const standard = Math.min(
      standardModules.projects
        .filter((p) => isCompleted(p, ids))
        .reduce((s, p) => s + p.credit, 0),
      CREDITS_REQUIRED.standard,
    )

    const stream = Math.min(
      streamModules.projects.filter((p) => isCompleted(p, ids)).reduce((s, p) => s + p.credit, 0),
      CREDITS_REQUIRED.stream,
    )

    const elective = Math.min(
      electiveModules.projects.filter((p) => isCompleted(p, ids)).reduce((s, p) => s + p.credit, 0),
      CREDITS_REQUIRED.elective,
    )

    const total = general + standard + stream + elective
    const percentage = (total / CREDITS_REQUIRED.total) * 100

    return { general, standard, stream, elective, total, percentage }
  })
}
