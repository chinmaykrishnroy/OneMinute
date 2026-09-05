export type EventType = "room.join" | "connection.ready" | "match.found" | "webrtc.offer" | "webrtc.answer" | "webrtc.ice" | "presence.heartbeat" | "match.leave" | "match.ended" | "error";
export type Envelope = { version: 1; type: EventType; requestId?: string; matchId?: string; payload: unknown };
export function message(type: EventType, payload: unknown, matchId?: string): Envelope {
  return { version: 1, type, requestId: crypto.randomUUID(), matchId, payload };
}
export function parse(data: string): Envelope {
  const value: unknown = JSON.parse(data);
  if (!value || typeof value !== "object" || !("version" in value) || value.version !== 1 || !("type" in value) || typeof value.type !== "string" || !("payload" in value)) throw new Error("Invalid signaling event.");
  return value as Envelope;
}
