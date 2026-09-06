"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { FormEvent, useEffect, useState } from "react";
import { MobileNav } from "@/components/navigation/mobile-nav";

type Profile = { id: string; displayName: string; avatarUrl: string; bio: string; countryCode: string; interests: string[]; languages: string[] };
const interestOptions = ["ai", "art", "books", "films", "fitness", "gaming", "music", "nature", "photography", "science", "technology", "travel"];
const languageOptions = [["en", "English"], ["hi", "Hindi"], ["bn", "Bengali"], ["es", "Spanish"], ["fr", "French"], ["de", "German"], ["ja", "Japanese"]];

export function ProfileEditor({ api }: { api: string }) {
  const router = useRouter();
  const [profile, setProfile] = useState<Profile | null>(null);
  const [blocked, setBlocked] = useState<Profile[]>([]);
  const [status, setStatus] = useState("Loading your profile…");

  useEffect(() => {
    let stopped = false;
    async function load() {
      try {
        const [profileResponse, blocksResponse] = await Promise.all([
          fetch(new URL("/v1/profile", api), { credentials: "include" }),
          fetch(new URL("/v1/blocks", api), { credentials: "include" }),
        ]);
        if (!profileResponse.ok) {
          router.replace("/");
          return;
        }
        const nextProfile = await profileResponse.json();
        const nextBlocks = blocksResponse.ok ? (await blocksResponse.json()).blocks : [];
        if (!stopped) {
          setProfile(nextProfile);
          setBlocked(nextBlocks);
          setStatus("");
        }
      } catch {
        if (!stopped) setStatus("Could not load your profile.");
      }
    }
    void load();
    return () => { stopped = true; };
  }, [api, router]);

  function toggle(kind: "interests" | "languages", value: string) {
    setProfile(current => current ? { ...current, [kind]: current[kind].includes(value) ? current[kind].filter(item => item !== value) : [...current[kind], value] } : current);
  }

  async function save(event: FormEvent) {
    event.preventDefault();
    if (!profile) return;
    setStatus("Saving…");
    const response = await fetch(new URL("/v1/profile", api), { method: "PATCH", credentials: "include", headers: { "Content-Type": "application/json" }, body: JSON.stringify(profile) });
    if (response.ok) { setProfile(await response.json()); setStatus("Profile saved."); } else setStatus("Could not save that profile.");
  }

  async function unblock(id: string) {
    const response = await fetch(new URL(`/v1/blocks/${id}`, api), { method: "DELETE", credentials: "include" });
    if (response.ok) setBlocked(items => items.filter(item => item.id !== id));
  }

  return <main className="social-shell">
    <header className="app-header"><Link className="wordmark" href="/">OneMinute</Link><nav className="desktop-nav"><Link href="/app/discover">Discover</Link><Link href="/app/connections">Connections</Link><Link href="/app/profile">Profile</Link></nav></header>
    <section className="social-card"><p className="eyebrow">Your profile</p><h1>What should connections know?</h1><p>Your richer profile becomes useful after you and another person both choose Connect.</p>{profile && <form className="profile-form" onSubmit={save}><label>Display name<input value={profile.displayName} maxLength={80} onChange={event => setProfile({ ...profile, displayName: event.target.value })} /></label><label>Bio<textarea value={profile.bio} maxLength={500} onChange={event => setProfile({ ...profile, bio: event.target.value })} placeholder="A little about you…" /></label><label>Country code<input value={profile.countryCode} maxLength={2} onChange={event => setProfile({ ...profile, countryCode: event.target.value.toUpperCase() })} placeholder="IN" /></label><fieldset><legend>Languages</legend><div className="interest-grid">{languageOptions.map(([value, text]) => <label className="interest" key={value}><input type="checkbox" checked={profile.languages.includes(value)} onChange={() => toggle("languages", value)} />{text}</label>)}</div></fieldset><fieldset><legend>Interests</legend><div className="interest-grid">{interestOptions.map(value => <label className="interest" key={value}><input type="checkbox" checked={profile.interests.includes(value)} onChange={() => toggle("interests", value)} />{label(value)}</label>)}</div></fieldset><button>Save profile</button></form>}<p role="status">{status}</p></section>
    <section className="social-card"><h2>Blocked people</h2>{blocked.length === 0 ? <p>No one is blocked.</p> : blocked.map(person => <div className="connection-row" key={person.id}><strong>{person.displayName}</strong><button className="quiet-button" onClick={() => void unblock(person.id)}>Unblock</button></div>)}</section>
    <MobileNav current="profile" />
  </main>;
}

function label(value: string) { return value.charAt(0).toUpperCase() + value.slice(1); }
