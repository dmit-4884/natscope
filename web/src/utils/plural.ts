export function plural(count: number, singular: string, pluralForm?: string, format: (n: number) => string = String): string {
  return `${format(count)} ${count === 1 ? singular : (pluralForm ?? singular + 's')}`
}
