import type { SchemaField, SchemaType } from '@/api/proto'

export function shortTypeName(fullName: string): string {
  return fullName.slice(fullName.lastIndexOf('.') + 1)
}

export function fieldTypeLabel(field: SchemaField): string {
  const element = field.typeName || field.kind
  if (field.mapKey) return `map<${field.mapKey}, ${element}>`
  return field.repeated ? `repeated ${element}` : element
}

export function groupByPackage(types: SchemaType[]): Array<[string, SchemaType[]]> {
  const groups = new Map<string, SchemaType[]>()
  for (const t of types) {
    const pkg = t.packageName || '(no package)'
    groups.set(pkg, [...(groups.get(pkg) ?? []), t])
  }
  return [...groups.entries()]
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([pkg, list]) => [pkg, [...list].sort((a, b) => a.fullName.localeCompare(b.fullName))])
}
