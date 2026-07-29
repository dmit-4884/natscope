import { createClient } from "@connectrpc/connect"

// Service definitions
import { ConnectionsService } from "../../gen/services/grpc/nats/v1/connections/nats_connections_service_pb"
import { StreamsService } from "../../gen/services/grpc/nats/v1/streams/nats_streams_service_pb"
import { MessagesService } from "../../gen/services/grpc/nats/v1/messages/nats_messages_service_pb"
import { PublishService } from "../../gen/services/grpc/nats/v1/publish/nats_publish_service_pb"
import { ManagementService } from "../../gen/services/grpc/nats/v1/management/nats_management_service_pb"
import { StatsService } from "../../gen/services/grpc/nats/v1/stats/nats_stats_service_pb"
import { LiveService } from "../../gen/services/grpc/nats/v1/live/nats_live_service_pb"
import { RegistryService } from "../../gen/services/grpc/proto/v1/registry/proto_registry_service_pb"
import { CodecService } from "../../gen/services/grpc/proto/v1/codec/proto_codec_service_pb"
import { SourcesService } from "../../gen/services/grpc/proto/v1/sources/proto_sources_service_pb"
import { SelectionsService } from "../../gen/services/grpc/proto/v1/selections/proto_selections_service_pb"
import { MappingsService } from "../../gen/services/grpc/mappings/v1/mappings/mappings_service_pb"
import { HistoryService } from "../../gen/services/grpc/history/v1/history/history_service_pb"
import { SettingsService } from "../../gen/services/grpc/settings/v1/settings/settings_service_pb"
import { TemplatesService } from "../../gen/services/grpc/templates/v1/templates/templates_service_pb"
import { WorkspaceService } from "../../gen/services/grpc/workspace/v1/workspace/workspace_service_pb"
import { transport } from "./transport"

// NATS services
export const connectionsClient = createClient(ConnectionsService, transport)
export const streamsClient = createClient(StreamsService, transport)
export const messagesClient = createClient(MessagesService, transport)
export const publishClient = createClient(PublishService, transport)
export const managementClient = createClient(ManagementService, transport)
export const statsClient = createClient(StatsService, transport)
export const liveClient = createClient(LiveService, transport)

// Proto services
export const registryClient = createClient(RegistryService, transport)
export const codecClient = createClient(CodecService, transport)
export const sourcesClient = createClient(SourcesService, transport)
export const selectionsClient = createClient(SelectionsService, transport)

// Other services
export const mappingsClient = createClient(MappingsService, transport)
export const historyClient = createClient(HistoryService, transport)
export const settingsClient = createClient(SettingsService, transport)
export const templatesClient = createClient(TemplatesService, transport)
export const workspaceClient = createClient(WorkspaceService, transport)
