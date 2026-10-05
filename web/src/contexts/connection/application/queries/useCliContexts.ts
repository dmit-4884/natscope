import { useQuery } from '@tanstack/react-query'
import { listCliContexts, type CliContextFile } from '@/api/connections'
import { connectionKeys } from './connectionKeys'

export function useCliContexts(upload: { id: number; files: CliContextFile[] }, enabled: boolean) {
  return useQuery({
    queryKey: connectionKeys.cliContexts(upload.id, upload.files.map((f) => f.name)),
    queryFn: () => listCliContexts(upload.files),
    enabled,
    staleTime: 0,
  })
}
