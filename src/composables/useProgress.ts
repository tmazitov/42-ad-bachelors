import { computed, type Ref } from 'vue'
import type { ProjectUser } from '@/composables/useProjects'
import type { Project } from '@/models/Project'
import { generalEducationModules, standardModules, streamModules, electiveModules } from '@/utils/data'

export const CREDITS_REQUIRED = {
  general:    5,
  standard:  93,
  stream:    14,
  elective:  12,
  capstone:   4,
  internship: 4,
  total:    132,
} as const

export function isCompleted(project: Project, ids: Set<number>): boolean {
  if (project.id === null) return false
  if (typeof project.id === 'number') return ids.has(project.id)
  return project.id.some((id) => ids.has(id))
}

export function useProgress(completedProjects: Ref<ProjectUser[]>) {
  return computed(() => {
    const ids = new Set(completedProjects.value.filter((p) => p['validated?']).map((p) => p.project.id))

    const genDone = generalEducationModules.projects.filter((p) => isCompleted(p, ids)).length
    const general = Math.min(genDone * (CREDITS_REQUIRED.general / 3), CREDITS_REQUIRED.general)

    const stdProjects = standardModules.projects.filter((p) => p.code !== 'BAI 498' && p.code !== 'BAI 499')
    const stdDone = stdProjects.filter((p) => isCompleted(p, ids)).length
    const standard = Math.min(stdDone * (CREDITS_REQUIRED.standard / 27), CREDITS_REQUIRED.standard)

    const stream = Math.min(
      streamModules.projects.filter((p) => isCompleted(p, ids)).reduce((s, p) => s + p.credit, 0),
      CREDITS_REQUIRED.stream,
    )

    const elective = Math.min(
      electiveModules.projects.filter((p) => isCompleted(p, ids)).reduce((s, p) => s + p.credit, 0),
      CREDITS_REQUIRED.elective,
    )

    const capstoneProject = standardModules.projects.find((p) => p.code === 'BAI 498')
    const capstone = capstoneProject && isCompleted(capstoneProject, ids) ? CREDITS_REQUIRED.capstone : 0

    const internshipProject = standardModules.projects.find((p) => p.code === 'BAI 499')
    const internship = internshipProject && isCompleted(internshipProject, ids) ? CREDITS_REQUIRED.internship : 0

    const total = general + standard + stream + elective + capstone + internship
    const percentage = (total / CREDITS_REQUIRED.total) * 100

    return { general, standard, stream, elective, capstone, internship, total, percentage }
  })
}
