"use client";

import Image from "next/image";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { FormEvent, useEffect, useRef, useState } from "react";
import { AppHeader, MobileNav } from "@/components/navigation/mobile-nav";
import { Icon } from "@/components/navigation/icon";
import { Profile, profileReady, interests, intents, languages, interestLabel } from "@/lib/preferences";

export function ProfileEditor({ api }: { api: string }) {
  const router = useRouter();
  const drawer = useRef<HTMLDialogElement>(null);
  const [profile, setProfile] = useState<Profile | null>(null);
  const [blocked, setBlocked] = useState<Profile[]>([]);
  const [connectionCount, setConnectionCount] = useState<number | null>(null);
  const [status, setStatus] = useState("Loading your profile�");
  const [saving, setSaving] = useState(false);
  const [savedReady, setSavedReady] = useState(false);
  const [showBlocked, setShowBlocked] = useState(false);
  const [accountStatus, setAccountStatus] = useState("");

  useEffect(() => {
    let stopped = false;
    async function load() {
      try {
        const [p, b, c] = await Promise.all(["/v1/profile", "/v1/blocks", "/v1/connections"].map(path => fetch(new URL(path, api), { credentials: "include" })));
        if (p.status === 401) { router.replace("/"); return; }
        if (!p.ok) throw new Error("Profile unavailable");
        const next: Profile = await p.json();
        const blocks = b.ok ? (await b.json()).blocks : [];
        const connections = c.ok ? (await c.json()).connections.length : null;
        if (!stopped) { setProfile(next); setSavedReady(profileReady(next)); setBlocked(blocks); setConnectionCount(connections); setStatus(b.ok && c.ok ? "" : "Some account details could not load. Refresh to try again."); }
      } catch { if (!stopped) setStatus("Could not load your profile. Please refresh."); }
    }
    void load(); return () => { stopped = true; };
  }, [api, router]);

  function toggle(kind: "interests" | "languages", value: string) {
    setProfile(current => {
      if (!current) return current;
      const values = current[kind];
      if (values.includes(value)) return { ...current, [kind]: values.filter(item => item !== value) };
      if (values.length >= (kind === "languages" ? 5 : 12)) return current;
      return { ...current, [kind]: [...values, value] };
    });
  }

  async function save(event: FormEvent) {
    event.preventDefault(); if (!profile || saving) return;
    setSaving(true); setStatus("Saving your changes�");
    try {
      const response = await fetch(new URL("/v1/profile", api), { method: "PATCH", credentials: "include", headers: { "Content-Type": "application/json" }, body: JSON.stringify(profile) });
      if (!response.ok) throw new Error("Save failed");
      const next: Profile = await response.json(); setProfile(next); setSavedReady(profileReady(next));
      setStatus("Saved. Discover will use these preferences.");
    } catch { setStatus("Could not save. Your changes are still here; please try again."); }
    finally { setSaving(false); }
  }

  async function unblock(id: string) {
    try {
      const response = await fetch(new URL(`/v1/blocks/${id}`, api), { method: "DELETE", credentials: "include" });
      if (!response.ok) throw new Error("Unblock failed");
      setBlocked(items => items.filter(item => item.id !== id)); setAccountStatus("Unblocked. This does not restore a removed connection.");
    } catch { setAccountStatus("Could not unblock. Please try again."); }
  }

  async function logout() {
    try {
      const response = await fetch(new URL("/v1/auth/logout", api), { method: "POST", credentials: "include" });
      if (!response.ok) throw new Error("Sign out failed");
      drawer.current?.close(); router.replace("/");
    } catch { setAccountStatus("Could not sign out. Please try again."); }
  }

  return <main className="social-shell account-page">
    <AppHeader title="You" />
    <section className="profile-overview">
      <div className="profile-cover"><span className="eyebrow">Your little corner</span><button className="settings-trigger" aria-label="Open account settings" onClick={() => { setShowBlocked(false); drawer.current?.showModal(); }}><Icon name="settings" /></button></div>
      <div className="profile-identity">
        {profile?.avatarUrl ? <Image className="profile-avatar" src={profile.avatarUrl} width={96} height={96} alt="" unoptimized /> : <span className="profile-avatar avatar-fallback">{profile?.displayName.slice(0, 1) || "?"}</span>}
        <div><h1>{profile?.displayName || "Your profile"}</h1><p>{profile?.bio || "A little about you. A lot of possibilities."}</p></div>
      </div>
      <div className="profile-shortcuts"><Link href="/app/connections"><Icon name="connections" /><strong>{connectionCount ?? "�"}</strong> connections<Icon name="arrow" width={18} height={18} /></Link><Link href="/app/messages"><Icon name="messages" />Your inbox<Icon name="arrow" width={18} height={18} /></Link></div>
    </section>
    {profile && !savedReady && <div className="onboarding-note"><Icon name="discover" /><div><strong>Make yourself at home.</strong><p>Choose who you want to meet and at least one language, then save to start discovering.</p></div></div>}
    {profile && <form className="account-form" onSubmit={save}>
      <section className="account-section" id="about"><div className="section-intro"><p className="eyebrow">01 / About you</p><h2>Put a little you into it.</h2><p>Your connections can see these details after you both choose Connect.</p></div><div className="profile-form"><label>Display name<input required value={profile.displayName} maxLength={80} onChange={event => setProfile({ ...profile, displayName: event.target.value })} /></label><label>A little about you<textarea value={profile.bio} maxLength={500} onChange={event => setProfile({ ...profile, bio: event.target.value })} placeholder="Weekend rituals, big curiosities, tiny obsessions�" /><span className="field-hint">{profile.bio.length}/500</span></label><label>Country code <span className="field-hint">Optional � two letters, like IN or US</span><input value={profile.countryCode} maxLength={2} pattern="[A-Z]{2}|^$" onChange={event => setProfile({ ...profile, countryCode: event.target.value.toUpperCase() })} placeholder="IN" /></label></div></section>
      <section className="account-section" id="preferences"><div className="section-intro"><p className="eyebrow">02 / Your kind of conversation</p><h2>Find your common ground.</h2><p>Discover uses these saved preferences every time. Change them whenever your mood changes.</p></div><div className="profile-form"><label>I�m here for<select value={profile.discoveryIntent} onChange={event => setProfile({ ...profile, discoveryIntent: event.target.value })}>{intents.map(([value, text]) => <option key={value} value={value}>{text}</option>)}</select></label><p className="field-hint">Dating only matches with someone who also chooses Dating.</p><fieldset><legend>Languages you�re comfortable speaking <span className="field-hint">Choose 1�5</span></legend><div className="interest-grid">{languages.map(([value, text]) => <label className="interest" key={value}><input type="checkbox" checked={profile.languages.includes(value)} disabled={!profile.languages.includes(value) && profile.languages.length >= 5} onChange={() => toggle("languages", value)} />{text}</label>)}{profile.languages.filter(value => !languages.some(([code]) => code === value)).map(value => <label className="interest" key={value}><input type="checkbox" checked onChange={() => toggle("languages", value)} />{value}</label>)}</div></fieldset><fieldset><legend>A few things you enjoy <span className="field-hint">Optional</span></legend><div className="interest-grid">{interests.map(value => <label className="interest" key={value}><input type="checkbox" checked={profile.interests.includes(value)} onChange={() => toggle("interests", value)} />{interestLabel(value)}</label>)}</div></fieldset></div></section>
      <div className="profile-save"><p role="status">{status}</p><button disabled={saving || !profileReady(profile)}>{saving ? "Saving�" : "Save changes"}</button>{savedReady && <Link className="quiet-link" href="/app/discover">Go to Discover <Icon name="arrow" width={18} height={18} /></Link>}</div>
    </form>}
    {!profile && <p role="status">{status}</p>}
    <dialog ref={drawer} className="account-drawer" aria-labelledby="account-settings-title" onClick={event => { if (event.target === event.currentTarget) drawer.current?.close(); }}>
      <div className="drawer-body"><div className="settings-heading"><div><p className="eyebrow">Your account</p><h2 id="account-settings-title">Settings & activity</h2></div><button className="icon-button" aria-label="Close settings" onClick={() => drawer.current?.close()}><Icon name="close" /></button></div>
      <p>Everything that makes this space yours.</p>
      <nav className="settings-links" aria-label="Account settings">
        <a href="#about" onClick={() => drawer.current?.close()}><Icon name="profile" />Edit profile<Icon name="arrow" /></a>
        <a href="#preferences" onClick={() => drawer.current?.close()}><Icon name="discover" />Discovery preferences<Icon name="arrow" /></a>
        <Link href="/app/connections"><Icon name="connections" />Connections<Icon name="arrow" /></Link>
        <button onClick={() => setShowBlocked(value => !value)} aria-expanded={showBlocked}><Icon name="shield" />Blocked people<span>{blocked.length}</span></button>
      </nav>
      {showBlocked && <div className="blocked-list"><h3>Blocked people</h3>{blocked.length ? blocked.map(person => <div className="connection-row" key={person.id}><strong>{person.displayName}</strong><button className="quiet-button" onClick={() => void unblock(person.id)}>Unblock</button></div>) : <p>No blocked people.</p>}</div>}
      <p role="status">{accountStatus}</p><button className="logout-button" onClick={() => void logout()}><Icon name="logout" />Log out</button><p className="drawer-footer">OneMinute � Meet the person first.</p></div>
    </dialog>
    <MobileNav current="profile" />
  </main>;
}
