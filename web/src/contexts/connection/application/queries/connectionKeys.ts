export const connectionKeys = {
  all: ['connections'] as const,
  sidebarLayout: (connectionId: string) => ['connections', 'sidebarLayout', connectionId] as const,
}
