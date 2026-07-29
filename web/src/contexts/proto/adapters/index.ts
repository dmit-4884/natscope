import { ProtoMessage } from '../domain/entities/ProtoMessage'

/** Groups messages by package for tree-like display. */
export function groupMessagesByPackage(messages: ProtoMessage[]): Record<string, string[]> {
  const grouped: Record<string, string[]> = {}

  for (const msg of messages) {
    const pkg = msg.packageName || 'default'
    if (!grouped[pkg]) {
      grouped[pkg] = []
    }
    grouped[pkg].push(msg.fullName)
  }

  // Sort messages within each package
  for (const pkg of Object.keys(grouped)) {
    grouped[pkg].sort()
  }

  return grouped
}
