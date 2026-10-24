// How a project is done on intra; 'mixed' when the module's alternatives (or
// the project's sessions) differ. null when the module is not tracked.
export type Team = 'solo' | 'group' | 'mixed'

export class Project {
  constructor(
    public code: string,
    public name: string,
    public credit: number,
    public id: number | number[] | null = null,
    public team: Team | null = null,
  ) {}
}
