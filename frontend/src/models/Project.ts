export class Project {
  constructor(
    public code: string,
    public name: string,
    public credit: number,
    public id: number | number[] | null = null,
  ) {}
}
