import type { Project } from './Project'

export class ProjectGroup {
  constructor(
    public name: string,
    public projects: Project[],
    public credit?: number,
  ) {}
}
