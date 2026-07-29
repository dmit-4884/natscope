import { mkdirSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { test as base } from '@playwright/test'
import { findConnectionByName, ensureStream, ensureProtoMapping } from './api'

// Worker setup: local connection, E2E stream (max_msg_size small on purpose
// for oversize tests), proto source from a temp file (backend shares the FS).
export const CONNECTION_NAME = 'local'
export const STREAM = 'E2E_PUBLISH'
export const PATTERN = 'e2e.publish.*.*'
export const PLAIN_SUBJECT = 'e2e.plain'
export const PROTO_PATTERN = 'e2e.proto.*'
export const PROTO_MESSAGE_TYPE = 'e2e.E2EMessage'
export const MAX_MSG_SIZE = 1024

const PROTO_SOURCE_NAME = 'e2e-proto-files'
const PROTO_FILE_CONTENT = `syntax = "proto3";

package e2e;

message E2EMessage {
  string name = 1;
  int32 count = 2;
  repeated string tags = 3;
}
`

function writeProtoFile(): string {
  const dir = join(tmpdir(), 'natscope-e2e-proto')
  mkdirSync(dir, { recursive: true })
  const path = join(dir, 'e2e.proto')
  writeFileSync(path, PROTO_FILE_CONTENT)
  return path
}

interface Env {
  connectionId: string
  connectionUrl: string
  protoSourceId: string
}

// WorkerEnv is resolved once per worker so expensive backend setup
// (CompileFiles etc.) doesn't repeat for every test.
interface WorkerEnv {
  connectionId: string
  connectionUrl: string
  protoSourceId: string
  connectionName: string
  connectionUrls: string[]
}

export const test = base.extend<{ env: Env }, { _workerEnv: WorkerEnv }>({
  // Worker-scoped: runs once per Playwright worker process.
  _workerEnv: [
    // eslint-disable-next-line no-empty-pattern
    async ({}, use) => {
      const conn = await findConnectionByName(CONNECTION_NAME)
      await ensureStream(conn.id, STREAM, [PATTERN, PLAIN_SUBJECT, PROTO_PATTERN], MAX_MSG_SIZE)
      const protoSourceId = await ensureProtoMapping(
        PROTO_SOURCE_NAME,
        writeProtoFile(),
        PROTO_PATTERN,
        PROTO_MESSAGE_TYPE,
      )
      await use({
        connectionId: conn.id,
        connectionUrl: conn.urls[0],
        protoSourceId,
        connectionName: conn.name,
        connectionUrls: conn.urls,
      })
    },
    { scope: 'worker' },
  ],

  // Test-scoped: seeds localStorage (requires a browser context) and
  // forwards the stable worker data to tests as the `env` fixture.
  env: async ({ context, _workerEnv }, use) => {
    // Boot the SPA as already-connected: the app reads the active
    // connection id + minimal info from localStorage on load.
    await context.addInitScript(
      ([id, name, urls]) => {
        localStorage.setItem('nats_active_connection_id', id as string)
        localStorage.setItem(
          'nats_active_connection_info',
          JSON.stringify({ id, name, urls }),
        )
      },
      [_workerEnv.connectionId, _workerEnv.connectionName, _workerEnv.connectionUrls] as const,
    )

    await use({
      connectionId: _workerEnv.connectionId,
      connectionUrl: _workerEnv.connectionUrl,
      protoSourceId: _workerEnv.protoSourceId,
    })
  },
})

export { expect } from '@playwright/test'

/** Publish-tab URL for the e2e stream. */
export const publishUrl = `/streams/${STREAM}/publish`
