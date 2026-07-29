// Dynamic value templates for JSON editor
export interface Helper {
  name: string
  template: string
  description: string
  generate: () => string
}

export const helpers: Helper[] = [
  {
    name: 'UUID',
    template: '{{uuid}}',
    description: 'Random UUID v4',
    generate: () => crypto.randomUUID(),
  },
  {
    name: 'Timestamp',
    template: '{{timestamp}}',
    description: 'ISO timestamp',
    generate: () => new Date().toISOString(),
  },
  {
    name: 'Unix',
    template: '{{unix}}',
    description: 'Unix timestamp (seconds)',
    generate: () => String(Math.floor(Date.now() / 1000)),
  },
  {
    name: 'Unix (ms)',
    template: '{{unix_ms}}',
    description: 'Unix timestamp (milliseconds)',
    generate: () => String(Date.now()),
  },
  {
    name: 'Date',
    template: '{{date}}',
    description: 'Current date YYYY-MM-DD',
    generate: () => new Date().toISOString().split('T')[0],
  },
  {
    name: 'Time',
    template: '{{time}}',
    description: 'Current time HH:MM:SS',
    generate: () => new Date().toTimeString().split(' ')[0],
  },
  {
    name: 'Random Int',
    template: '{{random_int}}',
    description: 'Random integer 1-1000',
    generate: () => String(Math.floor(Math.random() * 1000) + 1),
  },
  {
    name: 'Random String',
    template: '{{random_string}}',
    description: 'Random 8-char string',
    generate: () => Math.random().toString(36).substring(2, 10),
  },
]

/**
 * Replace {{helper}} in text with generated values; quotes in JSON context.
 * $ prefix shares: {{$uuid}} generates once. Pass the same sharedValues object
 * across calls (e.g. subject + body) so {{$uuid}} matches in both.
 */
export function processHelpers(text: string, sharedValues?: Record<string, string>): string {
  let result = text
  const shared = sharedValues ?? {}

  // Generate shared ($-prefix) values, skipping any already present
  for (const helper of helpers) {
    const sharedTemplate = helper.template.replace('{{', '{{$')
    if (text.includes(sharedTemplate) && !shared[sharedTemplate]) {
      shared[sharedTemplate] = helper.generate()
    }
  }

  // Process shared ($-prefix) helpers first
  for (const helper of helpers) {
    const sharedTemplate = helper.template.replace('{{', '{{$')
    if (shared[sharedTemplate]) {
      const escapedTemplate = sharedTemplate.replace(/[{}$]/g, '\\$&')
      const value = shared[sharedTemplate]

      // In JSON context (after :), add quotes if not already quoted
      const jsonContextRegex = new RegExp(`(:\\s*)${escapedTemplate}(?!")`, 'g')
      result = result.replace(jsonContextRegex, `$1"${value}"`)
      // Remaining (quoted or non-JSON) get the bare value
      const remainingRegex = new RegExp(escapedTemplate, 'g')
      result = result.replace(remainingRegex, value)
    }
  }

  // Process regular helpers (each generates a new value)
  for (const helper of helpers) {
    const escapedTemplate = helper.template.replace(/[{}]/g, '\\$&')

    // In JSON context (after :), add quotes if not already quoted
    const jsonContextRegex = new RegExp(`(:\\s*)${escapedTemplate}(?!")`, 'g')
    result = result.replace(jsonContextRegex, (_, prefix) => `${prefix}"${helper.generate()}"`)
    // Remaining (quoted or non-JSON) get the bare value
    const remainingRegex = new RegExp(escapedTemplate, 'g')
    result = result.replace(remainingRegex, helper.generate)
  }

  return result
}
