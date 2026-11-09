// Search results returned by the backend (/api/v1/projects, /api/v1/students).
import type { CategoryKey } from '@/utils/data'

export interface ProjectModuleRef {
  code: string
  name: string
  category: CategoryKey
}

export interface ProjectSearchResult {
  id: number
  name: string
  slug: string
  modules: ProjectModuleRef[] // empty: the project is not part of the bachelor
}
