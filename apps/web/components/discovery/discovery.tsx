"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { FormEvent, useEffect, useRef, useState } from "react";
import { MobileNav } from "@/components/navigation/mobile-nav";

type User = { id: string; displayName: string; avatarUrl: string };
type Peer = { id: string; displayName: string; avatarUrl: string };
type IceConfig = { iceServers: RTCIceServer[] };
type Match = { id: string; peer: Peer; sharedInterests: string[]; intent?: string; offerer: boolean; expiresAt: number; extended: boolean; connected: boolean };
type Phase = "connecting" | "ready" | "queued" | "matched" | "offline";
type Envelope = { version: 1; type: string; matchId?: string; payload: Record<string, unknown> };

const intents = [["surprise_me", "Surprise me"], ["new_friends", "New friends"], ["dating", "Dating"], ["gaming", "Gaming"], ["language_exchange", "Language exchange"], ["tech_ideas", "Tech / ideas"], ["professional_networking", "Professional networking"]] as const;
const languages = [["en", "English"], ["hi", "Hindi"], ["bn", "Bengali"], ["es", "Spanish"], ["fr", "French"], ["de", "German"], ["ja", "Japanese"]] as const;
const interests = ["ai", "art", "books", "films", "fitness", "gaming", "music", "nature", "photography", "science", "technology", "travel"];

export function Discovery({ api }: { api: string }) {
  const router = useRouter();
  const socket = useRef<WebSocket | null>(null), pc = useRef<RTCPeerConnection | null>(null);
  const localStream = useRef<MediaStream | null>(null), channel = useRef<RTCDataChannel | null>(null);
  const rawStream = useRef<MediaStream | null>(null), cameraSource = useRef<HTMLVideoElement | null>(null);
  const processorTimer = useRef<number | null>(null), mirrorRef = useRef(true);
  const localVideo = useRef<HTMLVideoElement | null>(null), remoteVideo = useRef<HTMLVideoElement | null>(null);
  const matchRef = useRef<Match | null>(null), requeue = useRef(false);
  const pendingIce = useRef<RTCIceCandidateInit[]>([]);
  const preferencesRef = useRef({ intent: "surprise_me", languages: ["en"], interests: ["music", "technology"] });
  const [phase, setPhase] = useState<Phase>("connecting"), [user, setUser] = useState<User | null>(null);
  const [match, setMatch] = useState<Match | null>(null), [message, setMessage] = useState("Checking your session…");
  const [intent, setIntent] = useState("surprise_me"), [language, setLanguage] = useState("en");
  const [selected, setSelected] = useState<string[]>(["music", "technology"]), [seconds, setSeconds] = useState(60);
  const [mirror, setMirror] = useState(true), [settingsOpen, setSettingsOpen] = useState(false);
  const [devices, setDevices] = useState<MediaDeviceInfo[]>([]), [camera, setCamera] = useState("");
  const [chat, setChat] = useState<string[]>([]), [draft, setDraft] = useState("");
  const [safetyOpen, setSafetyOpen] = useState(false), [reportCategory, setReportCategory] = useState("spam"), [reportDetails, setReportDetails] = useState("");

  useEffect(() => { preferencesRef.current = { intent, languages: [language], interests: selected }; }, [intent, language, selected]);
  useEffect(() => { matchRef.current = match; }, [match]);
  useEffect(() => { mirrorRef.current = mirror; }, [mirror]);
  useEffect(() => { if (match && localVideo.current && localStream.current) localVideo.current.srcObject = localStream.current; }, [match]);
  useEffect(() => {
    if (!match || match.extended) return;
    const update = () => setSeconds(Math.max(0, Math.ceil((match.expiresAt - Date.now()) / 1000)));
    update(); const timer = setInterval(update, 250); return () => clearInterval(timer);
  }, [match]);

  useEffect(() => {
    let stopped = false, heartbeat: ReturnType<typeof setInterval> | undefined;
    const cleanupPeer = () => {
      channel.current?.close(); channel.current = null; pc.current?.close(); pc.current = null; pendingIce.current = [];
      if (remoteVideo.current) remoteVideo.current.srcObject = null;
    };
    const cleanupMedia = () => {
      if (processorTimer.current !== null) window.clearInterval(processorTimer.current); processorTimer.current = null;
      localStream.current?.getTracks().forEach(track => track.stop()); localStream.current = null;
      rawStream.current?.getTracks().forEach(track => track.stop()); rawStream.current = null;
      cameraSource.current?.pause(); cameraSource.current = null;
      if (localVideo.current) localVideo.current.srcObject = null;
    };
    const cleanupCall = () => { cleanupPeer(); cleanupMedia(); };
    const attachChannel = (next: RTCDataChannel) => { channel.current = next; next.onmessage = event => setChat(items => [...items, `Them: ${String(event.data).slice(0, 500)}`]); };
    const waitForConnection = async () => { for (let attempt = 0; attempt < 600; attempt++) { if (pc.current) return pc.current; await new Promise(resolve => setTimeout(resolve, 50)); } return null; };
    const applyPendingIce = async (connection: RTCPeerConnection) => { for (const candidate of pendingIce.current.splice(0)) { try { await connection.addIceCandidate(candidate); } catch {} } };
    const startCall = async (found: Match, ice: IceConfig) => {
      cleanupPeer();
      try {
        if (!localStream.current || localStream.current.getTracks().some(track => track.readyState === "ended")) {
          cleanupMedia();
          const raw = await navigator.mediaDevices.getUserMedia({ audio: true, video: adaptiveVideoConstraints() });
          rawStream.current = raw;
          const source = document.createElement("video"); source.muted = true; source.playsInline = true; source.srcObject = raw; cameraSource.current = source; await source.play();
          const canvas = document.createElement("canvas"); canvas.width = 1920; canvas.height = 1080;
          const context = canvas.getContext("2d", { alpha: false });
          if (!context) throw new Error("video processing unavailable");
          const draw = () => {
            if (source.readyState < HTMLMediaElement.HAVE_CURRENT_DATA || !source.videoWidth) return;
            if (canvas.width !== source.videoWidth || canvas.height !== source.videoHeight) { canvas.width = source.videoWidth; canvas.height = source.videoHeight; }
            context.save();
            if (mirrorRef.current) { context.translate(canvas.width, 0); context.scale(-1, 1); }
            context.drawImage(source, 0, 0, canvas.width, canvas.height); context.restore();
          };
          draw(); processorTimer.current = window.setInterval(draw, 1000 / 30);
          const processedVideo = canvas.captureStream(30).getVideoTracks()[0]; processedVideo.contentHint = "motion";
          localStream.current = new MediaStream([...raw.getAudioTracks(), processedVideo]);
          setDevices((await navigator.mediaDevices.enumerateDevices()).filter(device => device.kind === "videoinput"));
        }
        const stream = localStream.current;
        if (!stream) throw new Error("media unavailable");
        if (localVideo.current) localVideo.current.srcObject = stream;
        const next = new RTCPeerConnection(ice); pc.current = next;
        for (const track of stream.getTracks()) {
          const sender = next.addTrack(track, stream);
          if (track.kind === "video") {
            const parameters = sender.getParameters(); parameters.degradationPreference = "balanced";
            await sender.setParameters(parameters);
          }
        }
        next.ontrack = event => { if (remoteVideo.current) remoteVideo.current.srcObject = event.streams[0]; };
        next.onicecandidate = event => { if (event.candidate && socket.current) send(socket.current, "webrtc.ice", event.candidate.toJSON(), found.id); };
        next.ondatachannel = event => attachChannel(event.channel);
        if (found.offerer) { attachChannel(next.createDataChannel("chat", { ordered: true })); const offer = await next.createOffer(); await next.setLocalDescription(offer); if (socket.current) send(socket.current, "webrtc.offer", offer, found.id); }
      } catch { setMessage("Camera and microphone access is needed for the encounter."); }
    };
    async function connect() {
      try {
        const me = await fetch(new URL("/v1/auth/me", api), { credentials: "include" });
        if (!me.ok) { router.replace("/"); return; } if (stopped) return; setUser(await me.json());
        const url = new URL("/v1/discovery/ws", api); url.protocol = url.protocol === "https:" ? "wss:" : "ws:";
        const ws = new WebSocket(url); socket.current = ws; let ice: IceConfig = { iceServers: [] };
        ws.onmessage = async event => {
          let incoming: Envelope; try { incoming = parse(event.data); } catch { ws.close(1002, "invalid server event"); return; }
          if (incoming.type === "connection.ready") { ice = incoming.payload.ice as IceConfig; setPhase("ready"); setMessage("Choose what you feel like talking about today."); }
          else if (incoming.type === "queue.joined") { setPhase("queued"); setMessage("Looking for someone compatible…"); }
          else if (incoming.type === "queue.left") { setPhase("ready"); setMessage("You left the queue."); }
          else if (incoming.type === "match.found" && incoming.matchId) {
            const found: Match = { id: incoming.matchId, peer: incoming.payload.peer as Peer, sharedInterests: (incoming.payload.sharedInterests as string[] | undefined) ?? [], intent: incoming.payload.intent as string | undefined, offerer: Boolean(incoming.payload.offerer), expiresAt: Number(incoming.payload.expiresAt), extended: incoming.payload.state === "extended", connected: Boolean(incoming.payload.connectionId) };
            matchRef.current = found; setMatch(found); setPhase("matched"); setMessage(""); setChat([]); await startCall(found, ice);
          } else if (incoming.type === "webrtc.offer" && incoming.matchId === matchRef.current?.id) {
            const connection = await waitForConnection(); if (!connection) return; await connection.setRemoteDescription(incoming.payload as unknown as RTCSessionDescriptionInit); await applyPendingIce(connection); const answer = await connection.createAnswer(); await connection.setLocalDescription(answer); send(ws, "webrtc.answer", answer, incoming.matchId);
          } else if (incoming.type === "webrtc.answer" && incoming.matchId === matchRef.current?.id) { const connection = await waitForConnection(); if (connection) { await connection.setRemoteDescription(incoming.payload as unknown as RTCSessionDescriptionInit); await applyPendingIce(connection); } }
          else if (incoming.type === "webrtc.ice" && incoming.matchId === matchRef.current?.id) { const candidate = incoming.payload as RTCIceCandidateInit, connection = pc.current; if (!connection?.remoteDescription) pendingIce.current.push(candidate); else try { await connection.addIceCandidate(candidate); } catch {} }
          else if (incoming.type === "match.extend_pending") setMessage("Your extension choice is private. Waiting for their choice.");
          else if (incoming.type === "match.extended") { setMatch(current => current ? { ...current, extended: true } : current); setMessage("You both chose to keep talking."); }
          else if (incoming.type === "match.connect_pending") setMessage("Your Connect choice is private. They will only know if they choose it too.");
          else if (incoming.type === "connection.created") { setMatch(current => current ? { ...current, connected: true } : current); setMessage("You both chose Connect. You can find each other in Connections."); }
          else if (incoming.type === "peer.disconnected") setMessage("Their connection dropped. Their place is held for 45 seconds.");
          else if (incoming.type === "peer.reconnected") { setMessage("They reconnected. Restoring video…"); const current = matchRef.current; if (current) await startCall(current, ice); }
          else if (incoming.type === "match.ended") {
            cleanupCall(); matchRef.current = null; setMatch(null); setPhase("ready");
            if (requeue.current) { requeue.current = false; send(ws, "queue.join", preferencesRef.current); setMessage("Finding your next conversation…"); }
            else setMessage(incoming.payload.reason === "expired" ? "That minute ended. Discover again when you are ready." : "That encounter ended. You can discover again.");
          } else if (incoming.type === "error") setMessage(errorMessage(incoming.payload.code));
        };
        ws.onopen = () => { heartbeat = setInterval(() => send(ws, "presence.heartbeat", {}), 20_000); };
        ws.onclose = () => { if (!stopped) { cleanupCall(); setPhase("offline"); setMessage("Discovery disconnected. Refresh to reconnect safely."); } };
        ws.onerror = () => setMessage("Could not reach discovery. Please try again.");
      } catch { if (!stopped) { setPhase("offline"); setMessage("Could not reach OneMinute. Please refresh."); } }
    }
    void connect(); return () => { stopped = true; if (heartbeat) clearInterval(heartbeat); cleanupCall(); socket.current?.close(1000, "page closed"); };
  }, [api, router]);

  function join(event?: FormEvent) { event?.preventDefault(); if (socket.current?.readyState === WebSocket.OPEN) { send(socket.current, "queue.join", preferencesRef.current); setMessage("Joining discovery…"); } }
  function matchAction(type: "match.leave" | "match.skip" | "match.extend" | "match.connect") { if (!socket.current || !match) return; if (type === "match.skip") requeue.current = true; send(socket.current, type, {}, match.id); }
  function toggleInterest(value: string) { setSelected(current => current.includes(value) ? current.filter(item => item !== value) : current.length < 8 ? [...current, value] : current); }
  function sendChat(event: FormEvent) { event.preventDefault(); const text = draft.trim().slice(0, 500); if (!text || channel.current?.readyState !== "open") return; channel.current.send(text); setChat(items => [...items, `You: ${text}`]); setDraft(""); }
  async function switchCamera(deviceId: string) { setCamera(deviceId); try { const nextStream = await navigator.mediaDevices.getUserMedia({ video: { ...adaptiveVideoConstraints(), ...(deviceId ? { deviceId: { exact: deviceId } } : {}) } }); const source = cameraSource.current, current = rawStream.current; if (!source || !current) { nextStream.getTracks().forEach(track => track.stop()); return; } source.srcObject = nextStream; await source.play(); current.getVideoTracks().forEach(track => { track.stop(); current.removeTrack(track); }); current.addTrack(nextStream.getVideoTracks()[0]); } catch { setMessage("Could not switch cameras."); } }
  async function blockPeer() { if (!match) return; const response = await fetch(new URL("/v1/blocks", api), { method: "POST", credentials: "include", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ targetUserId: match.peer.id, matchId: match.id }) }); if (response.ok) { setSafetyOpen(false); setMessage("Blocked. You will not be matched again."); } else setMessage("Could not block this person."); }
  async function reportPeer(event: FormEvent) { event.preventDefault(); if (!match) return; const response = await fetch(new URL("/v1/reports", api), { method: "POST", credentials: "include", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ targetUserId: match.peer.id, matchId: match.id, category: reportCategory, details: reportDetails }) }); if (response.ok) { setSafetyOpen(false); setMessage("Report received. Thank you for helping keep OneMinute safe."); } else setMessage("Could not submit the report."); }

  return <main className={match ? "encounter-shell" : "discover-shell"}>
    <header className="app-header"><Link className="wordmark" href="/">OneMinute</Link><nav className="desktop-nav"><Link href="/app/discover">Discover</Link><Link href="/app/connections">Connections</Link><Link href="/app/profile">Profile</Link></nav>{user && <span>Hi, {user.displayName}</span>}</header>
    {match ? <section className="encounter" aria-live="polite">
      <div className="encounter-top"><div><p className="eyebrow">Live encounter</p><strong>{match.peer.displayName}</strong></div><div className="encounter-meta"><button className="safety-button" onClick={() => setSafetyOpen(true)}>Safety</button><output className={match.extended ? "timer extended" : "timer"}>{match.extended ? "Extended" : `0:${String(seconds).padStart(2, "0")}`}</output></div></div>
      <div className="video-stage"><figure className="video-tile remote-tile"><video ref={remoteVideo} autoPlay playsInline /><figcaption>{match.peer.displayName}</figcaption></figure><figure className="video-tile local-tile"><video ref={localVideo} autoPlay muted playsInline /><figcaption>You</figcaption><button className="camera-settings" onClick={() => setSettingsOpen(true)} aria-label="Camera settings">⚙</button><div className="local-overlay-actions"><EncounterActions match={match} act={matchAction} /></div></figure></div>
      <div className="encounter-actions desktop-actions"><EncounterActions match={match} act={matchAction} /></div>
      <p role="status" className="encounter-status">{message || (match.sharedInterests.length ? `You both like ${match.sharedInterests.map(label).join(", ")}.` : "Start with hello.")}</p>
      <form className="temp-chat" onSubmit={sendChat}><div className="chat-log" aria-live="polite">{chat.length ? chat.map((item, index) => <p key={index}>{item}</p>) : <p>Messages stay between your browsers and disappear after this encounter.</p>}</div><label><span className="sr-only">Temporary message</span><input value={draft} onChange={event => setDraft(event.target.value)} maxLength={500} placeholder="Say something…" /></label><button>Send</button></form>
      {settingsOpen && <div className="modal-backdrop" role="presentation" onMouseDown={() => setSettingsOpen(false)}><div className="settings-panel" role="dialog" aria-modal="true" aria-labelledby="camera-title" onMouseDown={event => event.stopPropagation()}><div className="settings-heading"><h2 id="camera-title">Camera</h2><button className="icon-button" onClick={() => setSettingsOpen(false)} aria-label="Close">×</button></div><label className="check"><input type="checkbox" checked={mirror} onChange={event => setMirror(event.target.checked)} /> Mirror video for both people</label><label>Camera<select value={camera} onChange={event => void switchCamera(event.target.value)}><option value="">Default camera</option>{devices.map(device => <option key={device.deviceId} value={device.deviceId}>{device.label || "Camera"}</option>)}</select></label><p>Your preview is exactly what the other person receives. Selfie-style mirroring is on by default.</p></div></div>}
      {safetyOpen && <div className="modal-backdrop" role="presentation" onMouseDown={() => setSafetyOpen(false)}><form className="settings-panel safety-panel" onSubmit={reportPeer} onMouseDown={event => event.stopPropagation()}><div className="settings-heading"><h2>Safety</h2><button type="button" className="icon-button" onClick={() => setSafetyOpen(false)} aria-label="Close">×</button></div><label>Reason<select value={reportCategory} onChange={event => setReportCategory(event.target.value)}><option value="spam">Spam</option><option value="harassment">Harassment</option><option value="sexual_content">Sexual content</option><option value="hate">Hate</option><option value="violence">Violence</option><option value="underage">Possible minor</option><option value="other">Other</option></select></label><label>Details<textarea value={reportDetails} onChange={event => setReportDetails(event.target.value)} maxLength={500} /></label><button>Submit report</button><button type="button" className="danger-button" onClick={() => void blockPeer()}>Block and end encounter</button></form></div>}
    </section> : <section className="discovery-card"><p className="eyebrow">Discover</p><h1>Who would you like to meet?</h1><p>Your choice applies to this session. Dating only matches with another person who chose Dating.</p><form className="preference-form" onSubmit={join}><label>Current intent<select value={intent} onChange={event => setIntent(event.target.value)} disabled={phase === "queued"}>{intents.map(([value, text]) => <option value={value} key={value}>{text}</option>)}</select></label><label>Conversation language<select value={language} onChange={event => setLanguage(event.target.value)} disabled={phase === "queued"}>{languages.map(([value, text]) => <option value={value} key={value}>{text}</option>)}</select></label><fieldset disabled={phase === "queued"}><legend>A few things you enjoy</legend><div className="interest-grid">{interests.map(item => <label className="interest" key={item}><input type="checkbox" checked={selected.includes(item)} onChange={() => toggleInterest(item)} />{label(item)}</label>)}</div></fieldset>{phase === "queued" ? <button type="button" className="quiet-button" onClick={() => socket.current && send(socket.current, "queue.leave", {})}>Leave queue</button> : <button type="submit" disabled={phase !== "ready"}>Start discovering</button>}</form><p role="status" className="auth-status">{message}</p></section>}
    <MobileNav current="discover" />
  </main>;
}

function EncounterActions({ match, act }: { match: Match; act: (type: "match.leave" | "match.skip" | "match.extend" | "match.connect") => void }) { return <><button className="quiet-button" onClick={() => act("match.skip")}>Next</button><button onClick={() => act("match.extend")} disabled={match.extended}>Extend</button><button className="connect-button" onClick={() => act("match.connect")} disabled={match.connected}>{match.connected ? "Connected" : "Connect"}</button><button className="danger-button" onClick={() => act("match.leave")}>Leave</button></>; }

function adaptiveVideoConstraints(): MediaTrackConstraints { return { width: { ideal: 1920 }, height: { ideal: 1080 }, frameRate: { ideal: 30, max: 30 }, facingMode: "user" }; }

function send(socket: WebSocket, type: string, payload: object, matchId?: string) { socket.send(JSON.stringify({ version: 1, type, requestId: crypto.randomUUID(), matchId, payload })); }
function parse(value: unknown): Envelope { const parsed = JSON.parse(String(value)) as Envelope; if (parsed.version !== 1 || typeof parsed.type !== "string" || !parsed.payload) throw new Error("Invalid event"); return parsed; }
function label(value: string) { return value.split("_").map(word => word.charAt(0).toUpperCase() + word.slice(1)).join(" "); }
function errorMessage(code: unknown) { if (code === "queue_unavailable") return "You are already matched or temporarily unavailable."; if (code === "matchmaking_unavailable") return "Matching is temporarily unavailable. You remain in the queue."; if (code === "match_unavailable") return "That encounter has already ended."; return "Discovery rejected an invalid action. Please reconnect."; }
