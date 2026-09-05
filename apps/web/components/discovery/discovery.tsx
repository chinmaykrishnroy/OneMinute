"use client";

import Image from "next/image";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { FormEvent, useEffect, useRef, useState } from "react";

type User = { id: string; displayName: string; avatarUrl: string };
type Peer = { id: string; displayName: string; avatarUrl: string };
type Match = { id: string; peer: Peer; sharedInterests: string[]; intent?: string; recovered?: boolean };
type Phase = "connecting" | "ready" | "queued" | "matched" | "offline";
type Envelope = { version: 1; type: string; matchId?: string; payload: Record<string, unknown> };

const intents = [
  ["surprise_me", "Surprise me"], ["new_friends", "New friends"], ["dating", "Dating"],
  ["gaming", "Gaming"], ["language_exchange", "Language exchange"], ["tech_ideas", "Tech / ideas"],
  ["professional_networking", "Professional networking"],
] as const;
const languages = [["en", "English"], ["hi", "Hindi"], ["bn", "Bengali"], ["es", "Spanish"], ["fr", "French"], ["de", "German"], ["ja", "Japanese"]] as const;
const interests = ["ai", "art", "books", "films", "fitness", "gaming", "music", "nature", "photography", "science", "technology", "travel"];

export function Discovery({ api }: { api: string }) {
  const router = useRouter();
  const socket = useRef<WebSocket | null>(null);
  const [phase, setPhase] = useState<Phase>("connecting");
  const [user, setUser] = useState<User | null>(null);
  const [match, setMatch] = useState<Match | null>(null);
  const [intent, setIntent] = useState("surprise_me");
  const [language, setLanguage] = useState("en");
  const [selected, setSelected] = useState<string[]>(["music", "technology"]);
  const [message, setMessage] = useState("Checking your session…");

  useEffect(() => {
    let stopped = false;
    let heartbeat: ReturnType<typeof setInterval> | undefined;
    async function connect() {
      try {
        const me = await fetch(new URL("/v1/auth/me", api), { credentials: "include" });
        if (!me.ok) { router.replace("/"); return; }
        if (stopped) return;
        setUser(await me.json());
        const url = new URL("/v1/discovery/ws", api);
        url.protocol = url.protocol === "https:" ? "wss:" : "ws:";
        const ws = new WebSocket(url);
        socket.current = ws;
        ws.onmessage = event => {
          let incoming: Envelope;
          try { incoming = parse(event.data); }
          catch { ws.close(1002, "invalid server event"); return; }
          if (incoming.type === "connection.ready") { setPhase("ready"); setMessage("Choose what you feel like talking about today."); }
          else if (incoming.type === "queue.joined") { setPhase("queued"); setMessage("Looking for someone compatible…"); }
          else if (incoming.type === "queue.left") { setPhase("ready"); setMessage("You left the queue."); }
          else if (incoming.type === "match.found" && incoming.matchId) {
            const peer = incoming.payload.peer as Peer;
            setMatch({ id: incoming.matchId, peer, sharedInterests: (incoming.payload.sharedInterests as string[] | undefined) ?? [], intent: incoming.payload.intent as string | undefined, recovered: Boolean(incoming.payload.recovered) });
            setPhase("matched"); setMessage("");
          } else if (incoming.type === "peer.disconnected") setMessage("Your match lost connection. Their place is being kept briefly.");
          else if (incoming.type === "peer.reconnected") setMessage("Your match reconnected.");
          else if (incoming.type === "match.ended") { setMatch(null); setPhase("ready"); setMessage("That encounter ended. You can discover again."); }
          else if (incoming.type === "error") setMessage(errorMessage(incoming.payload.code));
        };
        ws.onopen = () => { heartbeat = setInterval(() => send(ws, "presence.heartbeat", {}), 20_000); };
        ws.onclose = () => { if (!stopped) { setPhase("offline"); setMessage("Discovery disconnected. Refresh to reconnect safely."); } };
        ws.onerror = () => setMessage("Could not reach discovery. Please try again.");
      } catch { if (!stopped) { setPhase("offline"); setMessage("Could not reach OneMinute. Please refresh."); } }
    }
    void connect();
    return () => { stopped = true; if (heartbeat) clearInterval(heartbeat); socket.current?.close(1000, "page closed"); };
  }, [api, router]);

  function join(event: FormEvent) {
    event.preventDefault();
    if (!socket.current || socket.current.readyState !== WebSocket.OPEN) return;
    send(socket.current, "queue.join", { intent, languages: [language], interests: selected });
    setMessage("Joining discovery…");
  }

  function leaveQueue() {
    if (socket.current) send(socket.current, "queue.leave", {});
  }

  function leaveMatch() {
    if (socket.current && match) send(socket.current, "match.leave", {}, match.id);
  }

  function toggleInterest(value: string) {
    setSelected(current => current.includes(value) ? current.filter(item => item !== value) : current.length < 8 ? [...current, value] : current);
  }

  return <main className="discover-shell">
    <header className="app-header"><Link className="wordmark" href="/">OneMinute</Link>{user && <span>Hi, {user.displayName}</span>}</header>
    {match ? <section className="match-card" aria-live="polite">
      <p className="eyebrow">You found each other</p>
      <div className="peer-heading">
        {match.peer.avatarUrl ? <Image src={match.peer.avatarUrl} alt="" width={72} height={72} unoptimized /> : <span className="avatar-fallback">{match.peer.displayName.slice(0, 1)}</span>}
        <div><h1>{match.peer.displayName}</h1><p>{match.recovered ? "Encounter recovered" : "Start with the person, not a profile."}</p></div>
      </div>
      {match.sharedInterests.length > 0 && <div className="shared-context"><strong>You both like</strong><div className="chips">{match.sharedInterests.map(item => <span key={item}>{label(item)}</span>)}</div></div>}
      <p className="intent-note">Current intent: {label(match.intent ?? intent)}</p>
      <p className="phase-note">Calling is not enabled in this discovery preview yet. You can leave and meet someone else.</p>
      <button className="danger-button" onClick={leaveMatch}>Leave match</button>
    </section> : <section className="discovery-card">
      <p className="eyebrow">Discover</p><h1>Who would you like to meet?</h1>
      <p>Your choice applies to this session. Dating only matches with another person who chose Dating.</p>
      <form className="preference-form" onSubmit={join}>
        <label>Current intent<select value={intent} onChange={event => setIntent(event.target.value)} disabled={phase === "queued"}>{intents.map(([value, text]) => <option value={value} key={value}>{text}</option>)}</select></label>
        <label>Conversation language<select value={language} onChange={event => setLanguage(event.target.value)} disabled={phase === "queued"}>{languages.map(([value, text]) => <option value={value} key={value}>{text}</option>)}</select></label>
        <fieldset disabled={phase === "queued"}><legend>A few things you enjoy</legend><div className="interest-grid">{interests.map(item => <label className="interest" key={item}><input type="checkbox" checked={selected.includes(item)} onChange={() => toggleInterest(item)} />{label(item)}</label>)}</div></fieldset>
        {phase === "queued" ? <button type="button" className="quiet-button" onClick={leaveQueue}>Leave queue</button> : <button type="submit" disabled={phase !== "ready"}>Start discovering</button>}
      </form>
      <p role="status" className="auth-status">{message}</p>
    </section>}
  </main>;
}

function send(socket: WebSocket, type: string, payload: object, matchId?: string) {
  socket.send(JSON.stringify({ version: 1, type, requestId: crypto.randomUUID(), matchId, payload }));
}
function parse(value: unknown): Envelope {
  const parsed = JSON.parse(String(value)) as Envelope;
  if (parsed.version !== 1 || typeof parsed.type !== "string" || !parsed.payload) throw new Error("Invalid event");
  return parsed;
}
function label(value: string) { return value.split("_").map(word => word.charAt(0).toUpperCase() + word.slice(1)).join(" "); }
function errorMessage(code: unknown) {
  if (code === "queue_unavailable") return "You are already matched or temporarily unavailable.";
  if (code === "matchmaking_unavailable") return "Matching is temporarily unavailable. You remain in the queue.";
  return "Discovery rejected an invalid action. Please reconnect.";
}
