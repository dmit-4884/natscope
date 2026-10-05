export const connectionKeys = {
  all: ['connections'] as const,
  sidebarLayout: (connectionId: string) => ['connections', 'sidebarLayout', connectionId] as const,
  cliContexts: (upload: number, files: string[]) => ['connections', 'cliContexts', upload, files] as const,
}
