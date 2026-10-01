import { syntaxTree } from '@codemirror/language'
import type { Completion, CompletionContext, CompletionResult, CompletionSource } from '@codemirror/autocomplete'
import type { EditorState } from '@codemirror/state'
import type { SchemaEnum, SchemaField, SchemaMessage, TypeDescription } from '@/api/proto'
import { fieldTypeLabel } from '@/contexts/proto'

const WELL_KNOWN = 'google.protobuf.'

type SyntaxNode = ReturnType<ReturnType<typeof syntaxTree>['resolveInner']>

interface SchemaIndex {
  root: string
  messages: Map<string, SchemaMessage>
  enums: Map<string, SchemaEnum>
}

type Slot = { kind: 'object'; message: SchemaMessage } | { kind: 'value'; field: SchemaField }

type Cursor = { kind: 'key'; path: string[]; present: string[] } | { kind: 'value'; path: string[] }

function fieldNamed(message: SchemaMessage, key: string): SchemaField | undefined {
  return message.fields.find((f) => f.name === key || f.jsonName === key)
}

function slotAt(index: SchemaIndex, path: string[]): Slot | undefined {
  let message = index.messages.get(index.root)
  let i = 0
  while (message) {
    if (i === path.length) return { kind: 'object', message }
    const field = fieldNamed(message, path[i++])
    if (!field) return undefined
    if (field.mapKey) {
      if (i === path.length) return undefined
      i++
    }
    if (field.kind !== 'message' || field.typeName.startsWith(WELL_KNOWN)) {
      return i === path.length ? { kind: 'value', field } : undefined
    }
    message = index.messages.get(field.typeName)
  }
  return undefined
}

function unquote(state: EditorState, node: SyntaxNode): string {
  const text = state.sliceDoc(node.from, node.to)
  try {
    return JSON.parse(text) as string
  } catch {
    return text.replace(/^"|"$/g, '')
  }
}

function enclosing(node: SyntaxNode | null, names: string[]): SyntaxNode | null {
  for (let n = node; n; n = n.parent) if (names.includes(n.name)) return n
  return null
}

function keysAbove(state: EditorState, node: SyntaxNode | null): string[] {
  const path: string[] = []
  for (let n = node; n; n = n.parent) {
    if (n.name !== 'Property') continue
    const name = n.getChild('PropertyName')
    if (name) path.unshift(unquote(state, name))
  }
  return path
}

function cursorAt(state: EditorState, pos: number, wordFrom: number): Cursor | null {
  const tree = syntaxTree(state)
  const before = state.sliceDoc(0, wordFrom).replace(/\s+$/, '')
  const prev = before.slice(-1)
  const at = tree.resolveInner(before.length, -1)

  if (prev === ':') {
    const property = enclosing(at, ['Property'])
    return property ? { kind: 'value', path: keysAbove(state, property) } : null
  }
  if (prev !== '{' && prev !== ',' && prev !== '[') return null

  const container = enclosing(at, ['Object', 'Array'])
  if (!container) return prev === '{' && before.length === 1 ? { kind: 'key', path: [], present: [] } : null
  if (container.name === 'Array') return { kind: 'value', path: keysAbove(state, container) }
  if (prev === '[') return null

  const present = container
    .getChildren('Property')
    .map((p) => p.getChild('PropertyName'))
    .filter((n): n is SyntaxNode => !!n && (n.to < wordFrom || n.from > pos))
    .map((n) => unquote(state, n))
  return { kind: 'key', path: keysAbove(state, container), present }
}

function insertQuoted(text: string, opened: boolean, suffix: string): Completion['apply'] {
  return (view, _completion, from, to) => {
    const end = opened && view.state.sliceDoc(to, to + 1) === '"' ? to + 1 : to
    const insert = `${opened ? '' : '"'}${text}"${suffix}`
    view.dispatch({ changes: { from, to: end, insert }, selection: { anchor: from + insert.length } })
  }
}

function declared(i: number, deprecated: boolean): number {
  return deprecated ? -99 : Math.max(-98, -i)
}

function keyOptions(message: SchemaMessage, present: string[], opened: boolean): Completion[] {
  return message.fields
    .filter((f) => !present.includes(f.name) && !present.includes(f.jsonName))
    .map((f, i) => ({
      label: f.name,
      type: f.kind === 'message' ? 'class' : f.kind === 'enum' ? 'enum' : 'property',
      detail: f.deprecated ? `${fieldTypeLabel(f)} · deprecated` : fieldTypeLabel(f),
      info: f.comment || undefined,
      boost: declared(i, f.deprecated),
      apply: insertQuoted(f.name, opened, ': '),
    }))
}

function valueOptions(index: SchemaIndex, field: SchemaField, opened: boolean): Completion[] {
  if (field.kind === 'enum') {
    return (index.enums.get(field.typeName)?.values ?? []).map((v, i) => ({
      label: v.name,
      type: 'enum',
      detail: String(v.number),
      info: v.comment || undefined,
      boost: declared(i, v.deprecated),
      apply: insertQuoted(v.name, opened, ''),
    }))
  }
  if (field.kind === 'bool' && !opened) {
    return [
      { label: 'true', type: 'keyword' },
      { label: 'false', type: 'keyword' },
    ]
  }
  return []
}

/** Completes field names and enum values anywhere in a JSON document of the described message type. */
export function protoCompletionSource(description: TypeDescription, messageType: string): CompletionSource {
  const index: SchemaIndex = {
    root: messageType,
    messages: new Map(description.messages.map((m) => [m.fullName, m])),
    enums: new Map(description.enums.map((e) => [e.fullName, e])),
  }
  return (context: CompletionContext): CompletionResult | null => {
    const word = context.matchBefore(/"[\w]*$|[A-Za-z_][\w]*$/)
    if (!word && !context.explicit) return null
    const opened = word?.text.startsWith('"') ?? false
    const wordFrom = word ? word.from : context.pos
    const cursor = cursorAt(context.state, context.pos, wordFrom)
    if (!cursor) return null

    const slot = slotAt(index, cursor.path)
    const options =
      cursor.kind === 'key'
        ? slot?.kind === 'object'
          ? keyOptions(slot.message, cursor.present, opened)
          : []
        : slot?.kind === 'value'
          ? valueOptions(index, slot.field, opened)
          : []
    if (options.length === 0) return null
    return { from: wordFrom + (opened ? 1 : 0), options, validFor: /^[\w]*$/ }
  }
}
