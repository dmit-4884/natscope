export function isJetStreamControlReply(reply: string): boolean {
  return reply.startsWith('$JS.ACK.') || reply.startsWith('$JS.FC.')
}
