import type { SectionLayout } from '@/contexts/connection'

export interface ArrangedSection {
  pinned: string[]
  rest: string[]
}

export function arrangeSection(names: string[], layout: SectionLayout, query = ''): ArrangedSection {
  const available = new Set(names)
  const placed = new Set<string>()
  const take = (list: string[]) =>
    list.filter((name) => {
      if (!available.has(name) || placed.has(name)) return false
      placed.add(name)
      return true
    })

  const pinned = take(layout.pinned)
  const ordered = take(layout.order)
  const unordered = names.filter((name) => !placed.has(name)).sort((a, b) => a.localeCompare(b))

  const needle = query.trim().toLowerCase()
  const matches = (name: string) => !needle || name.toLowerCase().includes(needle)
  return { pinned: pinned.filter(matches), rest: [...ordered, ...unordered].filter(matches) }
}

export function togglePin(layout: SectionLayout, name: string): SectionLayout {
  if (layout.pinned.includes(name)) {
    return { pinned: layout.pinned.filter((n) => n !== name), order: layout.order }
  }
  return { pinned: [...layout.pinned, name], order: layout.order.filter((n) => n !== name) }
}

export function moveItem<T>(items: T[], from: number, to: number): T[] {
  const next = [...items]
  const [item] = next.splice(from, 1)
  next.splice(to, 0, item)
  return next
}

export function dropIndex(from: number, over: number, after: boolean): number {
  const target = after ? over + 1 : over
  return from < target ? target - 1 : target
}
