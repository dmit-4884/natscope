/**
 * Fixture generator: a genuinely non-trivial proto file compiled into a FILES
 * source (nested message, enum, repeated, map). Backend shares the filesystem,
 * so we write the .proto to a temp dir and point a source at it.
 */
import { mkdtempSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import * as A from '../api-audit'

export const MESSAGE_TYPE = 'audit.codec.Outer'
export const AUDIT_PROTO = `syntax = "proto3";
package audit.codec;

enum Color {
  COLOR_UNSPECIFIED = 0;
  RED = 1;
  GREEN = 2;
  BLUE = 3;
}

message Inner {
  string label = 1;
  int32 depth = 2;
}

message Outer {
  string name = 1;
  int32 count = 2;
  repeated string tags = 3;
  Color color = 4;
  Inner inner = 5;
  map<string, int32> scores = 6;
}
`

/** Create (fresh) a compiled proto source and return its id. Deletes any prior
 *  source of the same name first so its stored file path is always valid. */
export async function ensureAuditProtoSource(name = 'AUDIT_CODEC_SRC'): Promise<string> {
  for (const s of (await A.listSources()).sources ?? []) {
    if (s.name === name) await A.deleteSource(s.id).catch(() => {})
  }
  const dir = mkdtempSync(join(tmpdir(), 'audit-proto-'))
  const path = join(dir, 'audit.proto')
  writeFileSync(path, AUDIT_PROTO)
  const created = await A.createSource({ name, sourceType: 'SOURCE_TYPE_FILES', files: [path], includeDirs: [] })
  const sourceId = created.source?.id
  if (!sourceId) throw new Error('CreateSource returned no id')
  await A.compileFiles(sourceId)
  return sourceId
}

export async function deleteAuditProtoSource(name = 'AUDIT_CODEC_SRC'): Promise<void> {
  for (const s of (await A.listSources()).sources ?? []) {
    if (s.name === name) await A.deleteSource(s.id).catch(() => {})
  }
}
