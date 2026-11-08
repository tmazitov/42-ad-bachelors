import { computed, type Ref } from 'vue'
import type { StudentProject } from '@/models/Student'
import type { Project } from '@/models/Project'
import { generalEducationModules, standardModules, electiveModules, STREAMS, type StreamDef } from '@/utils/data'

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

function credits(projects: Project[], ids: Set<number>): number {
  return projects.filter((p) => isCompleted(p, ids)).reduce((s, p) => s + p.credit, 0)
}

// The stream counted for the stream credits: the chosen one, or (until the
// student picks) the one with the most credits; ties go to the first stream.
// Mirrors curriculum.Evaluate in the backend.
export function countedStream(chosen: string | null, streamCredits: Record<string, number>): StreamDef {
  const picked = STREAMS.find((s) => s.key === chosen)
  if (picked) return picked
  return STREAMS.reduce((best, s) => ((streamCredits[s.key] ?? 0) > (streamCredits[best.key] ?? 0) ? s : best))
}

// completedProjects are the student's validated projects; streamKey is their
// chosen stream (null if not chosen yet).
export function useProgress(completedProjects: Ref<StudentProject[]>, streamKey: Ref<string | null>) {
  return computed(() => {
    const ids = new Set(completedProjects.value.map((p) => p.project_id))

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

    const streamCredits: Record<string, number> = {}
    for (const s of STREAMS) streamCredits[s.key] = credits(s.projects, ids)
    const counted = countedStream(streamKey.value, streamCredits)
    const stream = Math.min(streamCredits[counted.key] ?? 0, CREDITS_REQUIRED.stream)

    const elective = Math.min(
      electiveModules.projects.filter((p) => isCompleted(p, ids)).reduce((s, p) => s + p.credit, 0),
      CREDITS_REQUIRED.elective,
    )

    const total = general + standard + stream + elective
    const percentage = (total / CREDITS_REQUIRED.total) * 100

    return {
      general, standard, stream, elective, total, percentage,
      streamCredits,
      countedStream: counted,
      streamChosen: counted.key === streamKey.value,
    }
  })
}
