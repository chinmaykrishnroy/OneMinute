"use client";

import Link from "next/link";
import { useEffect, useRef, useState } from "react";
import { LabSession, initialView } from "@/lib/webrtc/lab-session";
import type { MediaSource } from "@/lib/webrtc/media";

function Video({ stream, muted, label }: { stream: MediaStream | null; muted?: boolean; label: string }) {
  const ref = useRef<HTMLVideoElement>(null);
  useEffect(() => {
    const video = ref.current;
    if (!video) return;
    video.srcObject = stream;
    return () => { video.srcObject = null; };
  }, [stream]);
  return <figure><video ref={ref} aria-label={label} autoPlay playsInline muted={muted} controls={!muted} /><figcaption>{label}</figcaption></figure>;
}

export function NetworkingLab({ api }: { api: string }) {
  const [room, setRoom] = useState("");
  const [source, setSource] = useState<MediaSource>("camera");
  const [relay, setRelay] = useState(false);
  const [view, setView] = useState(initialView);
  const [text, setText] = useState("");
  const [creating, setCreating] = useState(false);
  const session = useRef<LabSession | null>(null);
  useEffect(() => () => session.current?.close(), []);
  const active = !["idle", "ended", "error"].includes(view.phase);
  async function create() {
    setCreating(true);
    try {
      const response = await fetch(new URL("/v1/lab/rooms", api), { method: "POST", signal: AbortSignal.timeout(5000) });
      if (!response.ok) throw new Error("Could not create a room. Check the API or try again shortly.");
      const result = await response.json(); setRoom(result.roomId);
      setView(current => ({ ...current, error: "" }));
    } catch (error) { setView(current => ({ ...current, error: error instanceof Error ? error.message : "Could not create room." })); }
    finally { setCreating(false); }
  }
  function join() {
    session.current?.close();
    const current = new LabSession(api, patch => setView(previous => ({ ...previous, ...patch })));
    session.current = current;
    void current.start(room, source, relay);
  }
  return <main>
    <Link href="/">← Home</Link>
    <h1>Networking lab</h1>
    <p>Development only. Create a room, then enter its code in a second browser tab. Rooms expire after 10 minutes.</p>
    <div className="setup">
      <button onClick={create} disabled={active || creating}>{creating ? "Creating…" : "Create room"}</button>
      <label>Room code<input aria-label="Room code" value={room} onChange={e => setRoom(e.target.value.trim())} disabled={active} maxLength={32} /></label>
      <label>Media source<select value={source} onChange={e => setSource(e.target.value as MediaSource)} disabled={active}><option value="camera">Camera and microphone</option><option value="test">Test pattern (no camera)</option></select></label>
      <label className="check"><input type="checkbox" checked={relay} onChange={e => setRelay(e.target.checked)} disabled={active} />Force TURN relay</label>
      <button onClick={join} disabled={active || !/^[a-f0-9]{32}$/.test(room)}>Join room</button>
    </div>
    <p role="status">Status: <strong>{view.phase}</strong></p>
    {view.error && <p role="alert">{view.error}</p>}
    <div className="videos">
      <Video stream={view.remote} label="Remote video" />
      <Video stream={view.local} label="Local preview" muted />
    </div>
    <div className="controls">
      <button disabled={!active} onClick={() => session.current?.mute()}>{view.muted ? "Unmute" : "Mute"}</button>
      <button disabled={!active} onClick={() => session.current?.camera()}>{view.cameraEnabled ? "Camera off" : "Camera on"}</button>
      <button disabled={!active} onClick={() => session.current?.close()}>Leave</button>
    </div>
    <p>Selected ICE pair: <output aria-label="Selected ICE pair">{view.transport}</output> · Received media: <output aria-label="Received media">{view.receivedBytes}</output> bytes</p>
    <section aria-label="DataChannel chat">
      <h2>Chat</h2>
      <p>{view.channelReady ? "DataChannel open" : "Waiting for DataChannel"}. Messages stay in this call.</p>
      <ol aria-live="polite">{view.messages.map((chat, i) => <li key={i}><strong>{chat.from}:</strong> {chat.text}</li>)}</ol>
      <form onSubmit={e => { e.preventDefault(); session.current?.chat(text); setText(""); }}>
        <label>Message<input value={text} onChange={e => setText(e.target.value)} maxLength={2000} disabled={!view.channelReady} /></label>
        <button disabled={!view.channelReady || !text.trim()}>Send</button>
      </form>
    </section>
  </main>;
}
