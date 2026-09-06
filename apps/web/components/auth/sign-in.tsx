"use client";

import Image from "next/image";
import { useRouter } from "next/navigation";
import { profileReady } from "@/lib/preferences";
import { useEffect, useRef, useState } from "react";

type User = { id: string; displayName: string; avatarUrl: string; newUser: boolean; googleEmailVerified: boolean };
type AuthConfig = { enabled: boolean; clientId: string; nonce: string };
type GoogleResponse = { credential: string };
type GoogleAccounts = {
  initialize(options: { client_id: string; nonce: string; callback: (response: GoogleResponse) => void }): void;
  renderButton(target: HTMLElement, options: { theme: string; size: string; shape: string; text: string; width: number }): void;
};

declare global { interface Window { google?: { accounts: { id: GoogleAccounts } } } }

export function SignIn({ api }: { api: string }) {
  const router = useRouter();
  const [user, setUser] = useState<User | null>(null);
  const [config, setConfig] = useState<AuthConfig | null>(null);
  const [message, setMessage] = useState("Checking your session…");
  const button = useRef<HTMLDivElement>(null);

  useEffect(() => {
    let cancelled = false;
    async function load() {
      try {
        const current = await fetch(new URL("/v1/auth/me", api), { credentials: "include" });
        if (!cancelled && current.ok) {
          setUser(await current.json()); const p = await fetch(new URL("/v1/profile", api), { credentials: "include" }); if (p.ok && !cancelled) router.replace(profileReady(await p.json()) ? "/app/discover" : "/onboarding"); setMessage(""); return;
        }
        const response = await fetch(new URL("/v1/auth/config", api), { credentials: "include" });
        if (!response.ok) { if (!cancelled) setMessage("Sign-in is temporarily unavailable."); return; }
        const next: AuthConfig = await response.json();
        if (!cancelled) { setConfig(next); setMessage(next.enabled ? "" : "Google sign-in needs a client ID."); }
      } catch {
        if (!cancelled) setMessage("Could not reach OneMinute. Please refresh and try again.");
      }
    }
    void load();
    return () => { cancelled = true; };
  }, [api, router]);

  useEffect(() => {
    if (!config?.enabled || !button.current) return;
    let disposed = false;
    const start = () => {
      if (disposed || !window.google || !button.current) return;
      window.google.accounts.id.initialize({ client_id: config.clientId, nonce: config.nonce, callback: response => void finish(response) });
      button.current.replaceChildren();
      window.google.accounts.id.renderButton(button.current, { theme: "outline", size: "large", shape: "pill", text: "continue_with", width: 280 });
    };
    async function finish(response: GoogleResponse) {
      setMessage("Signing you in…");
      try {
        const result = await fetch(new URL("/v1/auth/google", api), {
          method: "POST", credentials: "include", headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ credential: response.credential, nonce: config!.nonce }),
        });
        if (!result.ok) { setMessage("Google sign-in failed. Please try again."); return; }
        setUser(await result.json()); setMessage(""); const p = await fetch(new URL("/v1/profile", api), { credentials: "include" }); if (p.ok) router.replace(profileReady(await p.json()) ? "/app/discover" : "/onboarding");
      } catch {
        setMessage("Could not reach OneMinute. Please try again.");
      }
    }
    if (window.google) start();
    else {
      const script = document.createElement("script"); script.src = "https://accounts.google.com/gsi/client"; script.async = true; script.onload = start; script.onerror = () => setMessage("Could not load Google sign-in."); document.head.appendChild(script);
    }
    return () => { disposed = true; };
  }, [api, config, router]);

  async function logout() {
    try {
      const response = await fetch(new URL("/v1/auth/logout", api), { method: "POST", credentials: "include" });
      if (response.ok) { setUser(null); setConfig(null); window.location.reload(); }
      else setMessage("Could not sign out. Please try again.");
    } catch {
      setMessage("Could not reach OneMinute. Please try again.");
    }
  }

  if (user) return <section className="identity-card" aria-label="Signed-in account">
    {user.avatarUrl ? <Image src={user.avatarUrl} alt="" width={52} height={52} unoptimized /> : <span className="avatar-fallback" aria-hidden="true">{user.displayName.slice(0, 1).toUpperCase()}</span>}
    <div><p className="eyebrow">Signed in</p><h2>{user.displayName}</h2><p>{user.googleEmailVerified ? "Google-verified account" : "Google account"}</p></div>
    <a className="primary-link" href="/app/discover">Start discovering</a>
    <button className="quiet-button" onClick={() => void logout()}>Sign out</button>
  </section>;

  return <section className="signin-card" aria-label="Sign in">
    <p className="eyebrow">Your next conversation</p>
    <h2>Meet someone new.</h2>
    <p>Sign in once, then come back without repeating Google login on every visit.</p>
    <div ref={button} className="google-button" />
    {message && <p role="status" className="auth-status">{message}</p>}
  </section>;
}
