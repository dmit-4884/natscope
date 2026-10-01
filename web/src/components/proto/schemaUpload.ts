import type { SchemaUploadContent } from '@/api/protoSources'
import { formatBytes } from '@/utils/formatters'
import { plural } from '@/utils/plural'

export interface PickedFile {
  file: File
  path: string
}

export interface PreparedUpload {
  content: SchemaUploadContent
  summary: string
}

const DESCRIPTOR_SET_EXTENSIONS = ['.binpb', '.pb', '.desc', '.protoset', '.bin']
const CONFIG_NAMES = ['buf.yaml', 'buf.work.yaml', 'buf.lock']

const baseName = (path: string) => path.slice(path.lastIndexOf('/') + 1)
const hasExtension = (path: string, extensions: string[]) => extensions.some((ext) => path.toLowerCase().endsWith(ext))
const isHidden = (path: string) => path.split('/').some((part) => part.startsWith('.') || part === 'node_modules')

function isDescriptorSet(path: string): boolean {
  return hasExtension(path, DESCRIPTOR_SET_EXTENSIONS)
}

export function pickedFromList(files: FileList | File[]): PickedFile[] {
  return [...files].map((file) => ({ file, path: file.webkitRelativePath || file.name }))
}

function readFileEntry(entry: FileSystemFileEntry): Promise<File> {
  return new Promise((resolve, reject) => entry.file(resolve, reject))
}

function readDirectoryBatch(reader: FileSystemDirectoryReader): Promise<FileSystemEntry[]> {
  return new Promise((resolve, reject) => reader.readEntries(resolve, reject))
}

async function collectEntry(entry: FileSystemEntry, out: PickedFile[]): Promise<void> {
  if (entry.isFile) {
    out.push({ file: await readFileEntry(entry as FileSystemFileEntry), path: entry.fullPath.replace(/^\//, '') })
    return
  }
  if (!entry.isDirectory) return
  const reader = (entry as FileSystemDirectoryEntry).createReader()
  for (let batch = await readDirectoryBatch(reader); batch.length > 0; batch = await readDirectoryBatch(reader)) {
    for (const child of batch) await collectEntry(child, out)
  }
}

export async function pickedFromDrop(transfer: DataTransfer): Promise<PickedFile[]> {
  const entries = [...transfer.items]
    .map((item) => item.webkitGetAsEntry?.() ?? null)
    .filter((entry): entry is FileSystemEntry => entry !== null)
  if (entries.length === 0) return pickedFromList(transfer.files)
  const out: PickedFile[] = []
  for (const entry of entries) await collectEntry(entry, out)
  return out
}

export async function prepareUpload(picked: PickedFile[]): Promise<PreparedUpload | null> {
  const sets = picked.filter((p) => isDescriptorSet(p.path))
  if (picked.length === 1 && sets.length === 1) {
    const { file, path } = sets[0]
    return {
      content: { kind: 'descriptorSet', data: new Uint8Array(await file.arrayBuffer()) },
      summary: `Descriptor set ${baseName(path)} · ${formatBytes(file.size)}`,
    }
  }
  const usable = picked.filter(
    (p) => !isHidden(p.path) && (p.path.toLowerCase().endsWith('.proto') || CONFIG_NAMES.includes(baseName(p.path))),
  )
  const protoCount = usable.filter((p) => p.path.toLowerCase().endsWith('.proto')).length
  if (protoCount === 0) return null
  const files = await Promise.all(usable.map(async ({ file, path }) => ({ path, content: await file.text() })))
  const configs = usable.length - protoCount
  const parts = [plural(protoCount, '.proto file')]
  if (configs > 0) parts.push(plural(configs, 'buf config'))
  const skipped = picked.length - usable.length
  if (skipped > 0) parts.push(`${skipped} other skipped`)
  return { content: { kind: 'files', files }, summary: parts.join(' · ') }
}
