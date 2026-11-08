// Shapes returned by the backend API (/api/v1).

export interface Progress {
  general: number
  standard: number
  stream: number
  elective: number
  total: number
  percentage: number
  stream_key: string // stream counted in `stream`: the chosen one, or the best one
  streams: Record<string, number> // credits earned in each stream
}

export interface StudentProject {
  project_id: number
  slug?: string
  final_mark: number
  marked_at: string | null
}

export interface StudentSummary {
  login: string
  display_name: string
  image: string | null
}

export interface Student extends StudentSummary {
  id: number
  staff: boolean
  alumni: boolean
  stream: string | null // chosen stream key; null until the student picks one
  registered: boolean
  synced_at: string | null
  progress: Progress
  projects: StudentProject[] // validated projects only
}
