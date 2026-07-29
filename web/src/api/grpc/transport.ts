import { createGrpcWebTransport } from "@connectrpc/connect-web"

const BASE_URL = import.meta.env.VITE_GRPC_URL || window.location.origin

export const transport = createGrpcWebTransport({
  baseUrl: BASE_URL,
})
