export const mappingKeys = {
  all: ['mappings'] as const,
  lists: () => [...mappingKeys.all, 'list'] as const,
  list: () => [...mappingKeys.lists()] as const,
  health: (ids: string[]) => [...mappingKeys.all, 'health', ids.slice().sort().join(',')] as const,
}
