import { message, parse, type Envelope, type EventType } from "@/lib/signaling/protocol";
import { acquireMedia, type Capture, type MediaSource } from "./media";

export type Chat = { from: "You" | "Peer"; text: string };
export type LabView = {
  phase: string; error: string; local: MediaStream | null; remote: MediaStream | null;
  muted: boolean; cameraEnabled: boolean; channelReady: boolean; messages: Chat[];
  transport: string; receivedBytes: number;
};
export const initialView: LabView = { phase: "idle", error: "", local: null, remote: null, muted: false, cameraEnabled: true, channelReady: false, messages: [], transport: "Not connected", receivedBytes: 0 };

export class LabSession {
  private stopped = false;
  private capture?: Capture;
  private socket?: WebSocket;
  private peer?: RTCPeerConnection;
  private channel?: RTCDataChannel;
  private remote = new MediaStream();
  private pendingICE: RTCIceCandidateInit[] = [];
  private heartbeat?: number;
  private stats?: number;
  private room = "";
  private messages: Chat[] = [];

  constructor(private api: string, private update: (patch: Partial<LabView>) => void) {}
  async start(room: string, source: MediaSource, forceRelay: boolean) {
    this.room = room;
    this.update({ ...initialView, phase: "acquiring media" });
    try {
      const capture = await acquireMedia(source);
      if (this.stopped) { capture.stop(); return; }
      this.capture = capture;
      this.update({ local: capture.stream, phase: "connecting signaling" });
      const url = new URL("/v1/lab/ws", this.api);
      url.protocol = url.protocol === "https:" ? "wss:" : "ws:";
      const socket = new WebSocket(url); this.socket = socket;
      socket.onopen = () => { if (!this.stopped) this.send("room.join", { roomId: room }); };
      let pending = Promise.resolve();
      socket.onmessage = event => {
        pending = pending.then(async () => {
          if (this.stopped) return;
          if (typeof event.data !== "string") throw new Error("Invalid signaling frame.");
          await this.receive(parse(event.data), forceRelay);
        }).catch(error => this.fail(error));
      };
      socket.onerror = () => this.fail(new Error("Signaling connection failed."));
      socket.onclose = () => { if (!this.stopped) this.fail(new Error("Signaling disconnected. Create a new room to retry.")); };
      this.heartbeat = window.setInterval(() => this.send("presence.heartbeat", {}), 10000);
    } catch (error) { this.fail(error); }
  }
  private async receive(event: Envelope, forceRelay: boolean) {
    if (event.matchId && event.matchId !== this.room) throw new Error("Unexpected room.");
    if (event.type === "connection.ready") {
      const config = event.payload as { iceServers: RTCIceServer[] };
      const peer = new RTCPeerConnection({ iceServers: config.iceServers, iceTransportPolicy: forceRelay ? "relay" : "all" });
      this.peer = peer;
      this.capture?.stream.getTracks().forEach(track => peer.addTrack(track, this.capture!.stream));
      peer.onicecandidate = e => { if (e.candidate) this.send("webrtc.ice", e.candidate.toJSON()); };
      peer.ontrack = e => { this.remote.addTrack(e.track); this.update({ remote: this.remote }); };
      peer.ondatachannel = e => this.bindChannel(e.channel);
      peer.onconnectionstatechange = () => {
        if (this.stopped) return;
        this.update({ phase: peer.connectionState });
        if (peer.connectionState === "failed") this.fail(new Error("ICE connection failed. Check TURN address and relay ports."));
        if (peer.connectionState === "disconnected") this.update({ error: "Connection interrupted. Leave and create a new room if it does not recover." });
        if (peer.connectionState === "connected") this.update({ error: "" });
      };
      this.stats = window.setInterval(() => void this.inspect().catch(() => {}), 2000);
      this.update({ phase: "waiting for peer" });
    } else if (event.type === "match.found") {
      this.update({ phase: "negotiating" });
      if ((event.payload as { offerer: boolean }).offerer) {
        const peer = this.requiredPeer();
        this.bindChannel(peer.createDataChannel("chat", { ordered: true }));
        await peer.setLocalDescription(await peer.createOffer());
        if (!this.stopped) this.send("webrtc.offer", { type: "offer", sdp: peer.localDescription!.sdp });
      }
    } else if (event.type === "webrtc.offer" || event.type === "webrtc.answer") {
      const peer = this.requiredPeer();
      await peer.setRemoteDescription(event.payload as RTCSessionDescriptionInit);
      for (const candidate of this.pendingICE.splice(0)) await peer.addIceCandidate(candidate);
      if (event.type === "webrtc.offer") {
        await peer.setLocalDescription(await peer.createAnswer());
        if (!this.stopped) this.send("webrtc.answer", { type: "answer", sdp: peer.localDescription!.sdp });
      }
    } else if (event.type === "webrtc.ice") {
      const peer = this.requiredPeer();
      if (peer.remoteDescription) await peer.addIceCandidate(event.payload as RTCIceCandidateInit);
      else if (this.pendingICE.length < 128) this.pendingICE.push(event.payload as RTCIceCandidateInit);
      else throw new Error("Too many ICE candidates.");
    } else if (event.type === "match.ended") {
      this.close(); this.update({ phase: "ended", error: "Your peer left or the room expired. Create a new room to reconnect." });
    } else if (event.type === "error") {
      throw new Error("Room unavailable or signaling rejected. Create a new room and try again.");
    }
  }
  private requiredPeer() { if (!this.peer) throw new Error("Peer is not initialized."); return this.peer; }
  private bindChannel(channel: RTCDataChannel) {
    this.channel = channel;
    channel.onopen = () => this.update({ channelReady: true });
    channel.onclose = () => this.update({ channelReady: false });
    channel.onmessage = event => {
      if (typeof event.data === "string" && event.data.length <= 2000) this.addMessage({ from: "Peer", text: event.data });
    };
  }
  chat(text: string) {
    if (!text.trim() || text.length > 2000 || this.channel?.readyState !== "open" || this.channel.bufferedAmount > 65536) return;
    this.channel.send(text); this.addMessage({ from: "You", text });
  }
  private addMessage(chat: Chat) { this.messages = [...this.messages, chat].slice(-50); this.update({ messages: this.messages }); }
  mute() {
    const tracks = this.capture?.stream.getAudioTracks() ?? [];
    if (!tracks.length) return;
    const enabled = !tracks[0].enabled; tracks.forEach(track => { track.enabled = enabled; });
    this.update({ muted: !enabled });
  }
  camera() {
    const tracks = this.capture?.stream.getVideoTracks() ?? [];
    if (!tracks.length) return;
    const enabled = !tracks[0].enabled; tracks.forEach(track => { track.enabled = enabled; });
    this.update({ cameraEnabled: enabled });
  }
  private send(type: EventType, payload: unknown) {
    if (!this.stopped && this.socket?.readyState === WebSocket.OPEN) this.socket.send(JSON.stringify(message(type, payload, this.room)));
  }
  private async inspect() {
    if (!this.peer || this.stopped) return;
    const report = await this.peer.getStats();
    if (this.stopped) return;
    let receivedBytes = 0;
    report.forEach(stat => {
      if (stat.type === "inbound-rtp") receivedBytes += stat.bytesReceived ?? 0;
      if (stat.type === "transport" && stat.selectedCandidatePairId) {
        const pair = report.get(stat.selectedCandidatePairId);
        const local = report.get(pair?.localCandidateId);
        const remote = report.get(pair?.remoteCandidateId);
        if (local && remote) this.update({ transport: `${local.candidateType} ↔ ${remote.candidateType} (${local.protocol})` });
      }
    });
    this.update({ receivedBytes });
  }
  private fail(error: unknown) {
    if (this.stopped) return;
    this.close();
    this.update({ phase: "error", error: error instanceof Error ? error.message : "Connection failed." });
  }
  close() {
    if (this.stopped) return;
    this.send("match.leave", {});
    this.stopped = true;
    clearInterval(this.heartbeat); clearInterval(this.stats);
    if (this.socket) { this.socket.onopen = this.socket.onmessage = this.socket.onclose = this.socket.onerror = null; this.socket.close(); }
    if (this.channel) { this.channel.onmessage = this.channel.onopen = this.channel.onclose = null; this.channel.close(); }
    if (this.peer) { this.peer.ontrack = this.peer.onicecandidate = this.peer.ondatachannel = this.peer.onconnectionstatechange = null; this.peer.close(); }
    this.capture?.stop();
    this.remote.getTracks().forEach(track => track.stop());
    this.pendingICE = []; this.messages = [];
    this.update({ ...initialView, phase: "ended" });
  }
}
