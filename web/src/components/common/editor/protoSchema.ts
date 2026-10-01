import type { TypeDescription } from '@/api/proto'

/** Message type the editor completes, with every type it reaches. */
export interface ProtoSchema {
  messageType: string
  description: TypeDescription
}
