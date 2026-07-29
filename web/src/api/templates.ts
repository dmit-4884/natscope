import type { Timestamp } from '@bufbuild/protobuf/wkt'
import { tsToMillis } from '@/utils/timestamp'
import { templatesClient } from './grpc/clients'

export interface MessageTemplate {
  id: string
  name: string
  subject: string
  messageType: string
  data: string
  headers?: Record<string, string>
  /** Resolved wildcard slot values, positional (matches the `*`/`>` count of `subject`). */
  wildcards?: string[]
  createdAt: number
  updatedAt: number
}

export interface TemplateInput {
  name: string
  subject: string
  messageType: string
  data: string
  headers?: Record<string, string>
  wildcards?: string[]
}

function fromProto(t: {
  id: string
  name: string
  subject: string
  messageType: string
  data: string
  headers: Record<string, string>
  wildcards: string[]
  createdAt?: Timestamp
  updatedAt?: Timestamp
}): MessageTemplate {
  // Proto map/repeated are never null on the wire; expose undefined for empty.
  const headers = t.headers && Object.keys(t.headers).length > 0 ? { ...t.headers } : undefined
  const wildcards = t.wildcards && t.wildcards.length > 0 ? [...t.wildcards] : undefined
  return {
    id: t.id,
    name: t.name,
    subject: t.subject,
    messageType: t.messageType,
    data: t.data,
    headers,
    wildcards,
    createdAt: tsToMillis(t.createdAt),
    updatedAt: tsToMillis(t.updatedAt),
  }
}

/** Fetches every template, walking the cursor; callers need the full set. */
export async function listTemplates(): Promise<MessageTemplate[]> {
  const items: MessageTemplate[] = []
  let pageToken = ''
  for (let page = 0; page < 100; page++) {
    const response = await templatesClient.listTemplates({
      pageSize: 500,
      pageToken,
    })
    for (const t of response.templates) items.push(fromProto(t))
    if (!response.nextPageToken) break
    pageToken = response.nextPageToken
  }
  return items
}

export async function createTemplate(input: TemplateInput): Promise<MessageTemplate> {
  const response = await templatesClient.createTemplate({
    name: input.name,
    subject: input.subject,
    messageType: input.messageType,
    data: input.data,
    headers: input.headers ?? {},
    wildcards: input.wildcards ?? [],
  })
  return fromProto(response.template!)
}

export async function updateTemplate(id: string, patch: Partial<TemplateInput>): Promise<MessageTemplate> {
  const response = await templatesClient.updateTemplate({
    id,
    name: patch.name,
    subject: patch.subject,
    messageType: patch.messageType,
    data: patch.data,
    // Wrap in Patch only when present: omit="leave alone" vs empty="clear".
    headers: patch.headers !== undefined ? { values: patch.headers } : undefined,
    wildcards: patch.wildcards !== undefined ? { values: patch.wildcards } : undefined,
  })
  return fromProto(response.template!)
}

export async function deleteTemplate(id: string): Promise<void> {
  await templatesClient.deleteTemplate({ id })
}

export async function bulkCreateTemplates(items: TemplateInput[]): Promise<number> {
  if (items.length === 0) return 0
  const response = await templatesClient.batchCreateTemplates({
    templates: items.map((t) => ({
      name: t.name,
      subject: t.subject,
      messageType: t.messageType,
      data: t.data,
      headers: t.headers ?? {},
      wildcards: t.wildcards ?? [],
    })),
  })
  return response.created
}

export async function deleteAllTemplates(): Promise<number> {
  const response = await templatesClient.deleteAllTemplates({})
  return response.deleted
}
