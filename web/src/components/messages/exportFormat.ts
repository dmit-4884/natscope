import type { Message } from '@/types/nats'
import { decodeBase64ToUtf8 } from '@/utils/base64'

export type ExportFormat = 'json' | 'ndjson' | 'csv'

export interface ExportSerializeOptions {
  format: ExportFormat
  includeMetadata: boolean
  includeHeaders: boolean
  includeDecoded: boolean
}

export interface ExportSerializer {
  mime: string
  extension: string
  begin: () => string
  chunk: (messages: Message[]) => string
  end: () => string
}

const MIME: Record<ExportFormat, string> = {
  json: 'application/json',
  ndjson: 'application/x-ndjson',
  csv: 'text/csv',
}

function parsePayload(msg: Message): unknown {
  try {
    return JSON.parse(decodeBase64ToUtf8(msg.data_base64))
  } catch {
    return msg.data_base64
  }
}

function formatMessage(msg: Message, options: ExportSerializeOptions): unknown {
  if (!options.includeMetadata) {
    if (options.includeDecoded && msg.decoded) return msg.decoded
    return parsePayload(msg)
  }

  const formatted: Record<string, unknown> = {
    sequence: msg.sequence,
    subject: msg.subject,
    timestamp: msg.timestamp,
  }
  if (options.includeHeaders && msg.headers) formatted.headers = msg.headers
  if (options.includeDecoded && msg.decoded) {
    formatted.data = msg.decoded
    formatted.decoded_type = msg.decoded_type
  } else {
    formatted.data = parsePayload(msg)
  }
  return formatted
}

function escapeCsvValue(value: string): string {
  // Guard against CSV formula injection: a leading =, +, - or @ makes
  // spreadsheet apps evaluate the cell as a formula. Prefix with a single
  // quote to force text.
  const safe = /^[=+\-@]/.test(value) ? `'${value}` : value
  if (safe.includes(',') || safe.includes('"') || safe.includes('\n')) {
    return `"${safe.replace(/"/g, '""')}"`
  }
  return safe
}

function csvRow(formatted: unknown, options: ExportSerializeOptions): string {
  if (!options.includeMetadata) return escapeCsvValue(JSON.stringify(formatted))
  const typed = formatted as Record<string, unknown>
  const row = [String(typed.sequence || ''), escapeCsvValue(String(typed.subject || '')), String(typed.timestamp || '')]
  if (options.includeHeaders) row.push(escapeCsvValue(JSON.stringify(typed.headers || {})))
  row.push(escapeCsvValue(JSON.stringify(typed.data)))
  return row.join(',')
}

function csvHeader(options: ExportSerializeOptions): string {
  const headers: string[] = []
  if (options.includeMetadata) {
    headers.push('sequence', 'subject', 'timestamp')
    if (options.includeHeaders) headers.push('headers')
  }
  headers.push('data')
  return headers.join(',')
}

function indentJson(item: unknown): string {
  return '  ' + JSON.stringify(item, null, 2).replace(/\n/g, '\n  ')
}

export function createExportSerializer(options: ExportSerializeOptions): ExportSerializer {
  const base = { mime: MIME[options.format], extension: options.format }
  let wroteAny = false

  switch (options.format) {
    case 'json':
      return {
        ...base,
        begin: () => '[',
        chunk: (messages) => {
          if (messages.length === 0) return ''
          const body = messages.map((msg) => indentJson(formatMessage(msg, options))).join(',\n')
          const lead = wroteAny ? ',\n' : '\n'
          wroteAny = true
          return `${lead}${body}`
        },
        end: () => (wroteAny ? '\n]' : ']'),
      }
    case 'ndjson':
      return {
        ...base,
        begin: () => '',
        chunk: (messages) => messages.map((msg) => `${JSON.stringify(formatMessage(msg, options))}\n`).join(''),
        end: () => '',
      }
    case 'csv':
      return {
        ...base,
        begin: () => `${csvHeader(options)}\n`,
        chunk: (messages) => messages.map((msg) => `${csvRow(formatMessage(msg, options), options)}\n`).join(''),
        end: () => '',
      }
  }
}
