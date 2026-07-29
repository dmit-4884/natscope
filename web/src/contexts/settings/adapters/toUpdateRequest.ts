import type { MessageInitShape } from '@bufbuild/protobuf'
import type { UpdateSettingsRequestSchema } from '@/gen/services/grpc/settings/v1/settings/settings_service_pb'
import type { UserSettingsUpdate } from '../domain/entities/UserSettings'

/** Domain update DTO -> API mutation request. */
export function toUpdateRequest(
  update: UserSettingsUpdate,
): MessageInitShape<typeof UpdateSettingsRequestSchema> {
  const req: MessageInitShape<typeof UpdateSettingsRequestSchema> = {}
  if (update.messages && Object.keys(update.messages).length > 0) {
    req.messages = { ...update.messages }
  }
  if (update.live && Object.keys(update.live).length > 0) {
    req.live = { ...update.live }
  }
  if (update.display && Object.keys(update.display).length > 0) {
    req.display = { ...update.display }
  }
  if (update.publish && Object.keys(update.publish).length > 0) {
    req.publish = { ...update.publish }
  }
  if (update.behavior && Object.keys(update.behavior).length > 0) {
    req.behavior = { ...update.behavior }
  }
  return req
}
