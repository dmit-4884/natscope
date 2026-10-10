import { BASE_URL } from './grpc/transport'

const OBJECTS_URL = `${BASE_URL}/api/objects`

export function objectDownloadUrl(connectionId: string, bucket: string, name: string): string {
  return `${OBJECTS_URL}?${new URLSearchParams({ connection: connectionId, bucket, name })}`
}

export async function uploadObject(connectionId: string, bucket: string, file: File, description?: string): Promise<void> {
  const query = new URLSearchParams({ connection: connectionId, bucket, name: file.name })
  if (description) query.set('description', description)
  const response = await fetch(`${OBJECTS_URL}?${query}`, { method: 'PUT', body: file })
  if (response.ok) return
  const body = (await response.json().catch(() => null)) as { message?: string } | null
  throw new Error(body?.message ?? `Upload failed: HTTP ${response.status}`)
}
