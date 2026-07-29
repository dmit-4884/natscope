import type { UserSettings as ProtoUserSettings } from '@/gen/types/settings/user_settings_pb'
import { tsToMillis } from '@/utils/timestamp'
import { UserSettings } from '../domain/entities/UserSettings'

/** Proto -> domain UserSettings; missing fields fall back to VO defaults. */
export function toDomainSettings(proto: ProtoUserSettings): UserSettings {
  return UserSettings.fromApi({
    id: proto.id,
    messages: proto.messages
      ? {
          fetchMethod: proto.messages.fetchMethod,
          defaultPageSize: proto.messages.defaultPageSize,
          defaultDirection: proto.messages.defaultDirection,
          maxPayloadBytesInList: proto.messages.maxPayloadBytesInList,
          defaultExportFormat: proto.messages.defaultExportFormat,
          exportRangeLimit: proto.messages.exportRangeLimit,
        }
      : null,
    live: proto.live
      ? {
          subscriptionMode: proto.live.subscriptionMode,
          maxDisplayRate: proto.live.maxDisplayRate,
        }
      : null,
    display: proto.display
      ? {
          density: proto.display.density,
          defaultViewMode: proto.display.defaultViewMode,
          timestampFormat: proto.display.timestampFormat,
          jsonIndentSize: proto.display.jsonIndentSize,
          autoScrollLive: proto.display.autoScrollLive,
        }
      : null,
    publish: proto.publish
      ? { publishTimeoutSec: proto.publish.publishTimeoutSec }
      : null,
    behavior: proto.behavior
      ? {
          confirmDeleteConsumer: proto.behavior.confirmDeleteConsumer,
          confirmDeleteMessage: proto.behavior.confirmDeleteMessage,
          confirmDeleteKvKey: proto.behavior.confirmDeleteKvKey,
          confirmDeleteObject: proto.behavior.confirmDeleteObject,
          confirmPurgeKvHistory: proto.behavior.confirmPurgeKvHistory,
          secureDeleteDefault: proto.behavior.secureDeleteDefault,
        }
      : null,
    createdAt: tsToMillis(proto.createdAt),
    updatedAt: tsToMillis(proto.updatedAt),
  })
}
