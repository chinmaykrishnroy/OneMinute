"use client";

import Image from "next/image";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { FormEvent, useEffect, useState } from "react";
import { MobileNav } from "@/components/navigation/mobile-nav";

type Person = {
  id: string;
  displayName: string;
  avatarUrl: string;
  bio: string;
  countryCode: string;
  interests: string[];
  languages: string[];
};
type Connection = { id: string; createdAt: string; person: Person };

export function Connections({ api }: { api: string }) {
  const router = useRouter();
  const [items, setItems] = useState<Connection[]>([]);
  const [status, setStatus] = useState("Loading connections…");
  const [reporting, setReporting] = useState<Connection | null>(null);
  const [reportCategory, setReportCategory] = useState("spam");
  const [reportDetails, setReportDetails] = useState("");

  useEffect(() => {
    let stopped = false;
    async function load() {
      try {
        const response = await fetch(new URL("/v1/connections", api), { credentials: "include" });
        if (!response.ok) {
          router.replace("/");
          return;
        }
        const payload = await response.json();
        if (!stopped) {
          setItems(payload.connections);
          setStatus("");
        }
      } catch {
        if (!stopped) setStatus("Could not load connections.");
      }
    }
    void load();
    return () => { stopped = true; };
  }, [api, router]);

  async function remove(id: string) {
    const response = await fetch(new URL(`/v1/connections/${id}`, api), { method: "DELETE", credentials: "include" });
    if (response.ok) setItems(current => current.filter(item => item.id !== id));
  }

  async function block(item: Connection) {
    const response = await fetch(new URL("/v1/blocks", api), {
      method: "POST",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ targetUserId: item.person.id, connectionId: item.id }),
    });
    if (response.ok) setItems(current => current.filter(value => value.id !== item.id));
  }

  async function report(event: FormEvent) {
    event.preventDefault();
    if (!reporting) return;
    const response = await fetch(new URL("/v1/reports", api), {
      method: "POST",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ targetUserId: reporting.person.id, connectionId: reporting.id, category: reportCategory, details: reportDetails }),
    });
    if (response.ok) {
      setReporting(null);
      setReportDetails("");
      setStatus("Report received. Thank you for helping keep OneMinute safe.");
    } else setStatus("Could not submit the report.");
  }

  return <main className="social-shell">
    <header className="app-header"><Link className="wordmark" href="/">OneMinute</Link><nav className="desktop-nav"><Link href="/app/discover">Discover</Link><Link href="/app/connections">Connections</Link><Link href="/app/profile">Profile</Link></nav></header>
    <section className="social-card">
      <p className="eyebrow">Keep</p><h1>Your connections</h1><p>These are people you both chose after meeting.</p><p role="status">{status}</p>
      <div className="connection-list">
        {items.map(item => <article className="connection-card" key={item.id}>
          {item.person.avatarUrl ? <Image src={item.person.avatarUrl} alt="" width={64} height={64} unoptimized /> : <span className="avatar-fallback">{item.person.displayName[0]}</span>}
          <div><h2>{item.person.displayName}</h2><p>{item.person.bio || "You met on OneMinute."}</p>{item.person.interests.length > 0 && <div className="chips">{item.person.interests.map(value => <span key={value}>{value}</span>)}</div>}</div>
          <div className="connection-actions"><button className="quiet-button" onClick={() => void remove(item.id)}>Remove</button><button className="quiet-button" onClick={() => setReporting(item)}>Report</button><button className="danger-button" onClick={() => void block(item)}>Block</button></div>
        </article>)}
        {!status && items.length === 0 && <p>No connections yet. Meet someone before you judge the profile.</p>}
      </div>
    </section>
    {reporting && <div className="modal-backdrop" role="presentation" onMouseDown={() => setReporting(null)}><form className="settings-panel safety-panel" onSubmit={report} onMouseDown={event => event.stopPropagation()}><div className="settings-heading"><h2>Report {reporting.person.displayName}</h2><button type="button" className="icon-button" onClick={() => setReporting(null)} aria-label="Close">×</button></div><label>Reason<select value={reportCategory} onChange={event => setReportCategory(event.target.value)}><option value="spam">Spam</option><option value="harassment">Harassment</option><option value="sexual_content">Sexual content</option><option value="hate">Hate</option><option value="violence">Violence</option><option value="underage">Possible minor</option><option value="other">Other</option></select></label><label>Details<textarea value={reportDetails} onChange={event => setReportDetails(event.target.value)} maxLength={500} /></label><button>Submit report</button></form></div>}
    <MobileNav current="connections" />
  </main>;
}
