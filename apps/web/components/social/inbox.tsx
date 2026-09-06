"use client";
import Image from "next/image";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { AppHeader, MobileNav } from "@/components/navigation/mobile-nav";
import { Icon } from "@/components/navigation/icon";

type Connection = { id: string; person: { displayName: string; avatarUrl: string } };
export function Inbox({ api, posts = false }: { api: string; posts?: boolean }) {
 const router = useRouter();
 const [connections, setConnections] = useState<Connection[]>([]);
 const [status, setStatus] = useState("Loading - ");
 useEffect(() => {
  let stopped = false;
  async function load() {
   try {
    const response = await fetch(new URL("/v1/connections", api), { credentials: "include" });
    if (response.status === 401) { router.replace("/"); return; }
    if (!response.ok) throw new Error("Unavailable");
    const data = await response.json();
    if (!stopped) { setConnections(data.connections); setStatus(""); }
   } catch { if (!stopped) setStatus("Could not load your account. Please refresh."); }
  }
  void load(); return () => { stopped = true; };
 }, [api, router]);
 return <main className="social-shell inbox-page"><AppHeader title={posts ? "Posts" : "Messages"} /><section className="inbox-heading"><p className="eyebrow">{posts ? "A space for later" : "Keep the conversation going"}</p><h1>{posts ? "A little more to share." : "Your inbox."}</h1><p>{posts ? "This is the future home of Posts. For now, the best stories start with a conversation." : "A familiar face. A new conversation. Your connections will be right here."}</p></section>
 <section className="coming-soon-card"><div className="empty-icon"><Icon name={posts ? "posts" : "messages"} width={40} height={40} /></div><span className="soon-badge">Coming soon</span><h2>{posts ? "Not quite ready to post." : "Messages are on the way."}</h2><p>{posts ? "Posting is not available yet. Head to Discover to meet someone new." : "Your connections are saved. Sending and receiving messages will arrive with the next milestone."}</p><Link className="primary-link" href="/app/discover">Explore Discover <Icon name="arrow" width={18} height={18} /></Link></section>
 {!posts && <section className="inbox-people"><div className="settings-heading"><h2>Your connections</h2><Link href="/app/connections">Manage</Link></div>{connections.map(item => <div className="inbox-person" key={item.id}>{item.person.avatarUrl ? <Image src={item.person.avatarUrl} alt="" width={48} height={48} unoptimized /> : <span className="avatar-fallback">{item.person.displayName[0]}</span>}<div><strong>{item.person.displayName}</strong><p>Connected  -  Messaging coming soon</p></div></div>)}{!connections.length && !status && <p>Meet someone and both choose Connect to find each other here.</p>}</section>}<p role="status">{status}</p><MobileNav current={posts ? "posts" : "messages"} /></main>;
}
