"use client";
import { FormEvent, useEffect, useRef, useState } from "react";
import Link from "next/link";
import Image from "next/image";
import { useRouter } from "next/navigation";
import { AppHeader, MobileNav } from "@/components/navigation/mobile-nav";
import { Icon } from "@/components/navigation/icon";
import { useCommunication } from "@/components/communication/runtime";

type Moment = {
  id: string;
  userId: string;
  name: string;
  avatarUrl: string;
  body: string;
  tone: string;
  createdAt: string;
  expiresAt: string;
  connectionId: string;
};
export function Moments() {
  const { api, userId } = useCommunication();
  const router = useRouter();
  const [items, setItems] = useState<Moment[]>([]),
    [body, setBody] = useState(""),
    [tone, setTone] = useState("lilac"),
    [status, setStatus] = useState("Loading moments..."),
    [busy, setBusy] = useState(false),
    [now, setNow] = useState(0);
  const dialog = useRef<HTMLDialogElement>(null);
  const clockOffset = useRef(0);
  useEffect(() => {
    let disposed = false;
    async function load() {
      try {
        const response = await fetch(new URL("/v1/moments", api), {
          credentials: "include",
        });
        if (response.status === 401) {
          router.replace("/");
          return;
        }
        if (!response.ok) throw new Error();
        const values = await response.json();
        if (disposed) return;
        const date = response.headers.get("Date");
        if (date) clockOffset.current = Date.parse(date) - Date.now();
        setItems(values);
        setNow(Date.now() + clockOffset.current);
        setStatus("");
      } catch {
        if (!disposed) setStatus("Could not load moments. Please try again.");
      }
    }
    void load();
    const refresh = setInterval(() => void load(), 15000);
    const clock = setInterval(
      () => setNow(Date.now() + clockOffset.current),
      1000,
    );
    return () => {
      disposed = true;
      clearInterval(refresh);
      clearInterval(clock);
    };
  }, [api, router]);
  async function publish(event: FormEvent) {
    event.preventDefault();
    if (busy || !body.trim()) return;
    setBusy(true);
    try {
      const response = await fetch(new URL("/v1/moments", api), {
        method: "POST",
        credentials: "include",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ body, tone }),
      });
      if (!response.ok) {
        const result = await response.json();
        throw new Error(result?.error || "Could not share. Please try again.");
      }
      setBody("");
      dialog.current?.close();
      const list = await fetch(new URL("/v1/moments", api), {
        credentials: "include",
      });
      if (list.ok) setItems(await list.json());
      setNow(Date.now() + clockOffset.current);
      setStatus("Shared with your current connections for 24 hours.");
    } catch (error) {
      setStatus(error instanceof Error ? error.message : "Could not share.");
    } finally {
      setBusy(false);
    }
  }
  async function remove(id: string) {
    try {
      const response = await fetch(new URL(`/v1/moments/${id}`, api), {
        method: "DELETE",
        credentials: "include",
      });
      if (!response.ok) throw new Error();
      setItems((current) => current.filter((item) => item.id !== id));
      setStatus("Moment removed.");
    } catch {
      setStatus("Could not remove this moment.");
    }
  }
  const visible = items.filter((item) => Date.parse(item.expiresAt) > now);
  return (
    <main className="social-shell moments-page">
      <AppHeader
        title="Moments"
        action={
          <button
            className="header-action"
            aria-label="Create moment"
            onClick={() => dialog.current?.showModal()}
          >
            <Icon name="posts" />
          </button>
        }
      />
      <div className="page-intro">
        <p className="eyebrow">A little of your day</p>
        <h2>Something to talk about.</h2>
        <p>
          A thought, a recommendation, a small win. Only your current
          connections can see it, and it disappears after 24 hours.
        </p>
      </div>
      <p role="status">{status}</p>
      <div className="moments-list">
        {visible.map((item) => (
          <article className={`moment tone-${item.tone}`} key={item.id}>
            <header>
              {item.avatarUrl ? (
                <Image
                  src={item.avatarUrl}
                  width={44}
                  height={44}
                  alt=""
                  unoptimized
                />
              ) : (
                <span className="avatar-fallback">{item.name[0]}</span>
              )}
              <div>
                <strong>{item.userId === userId ? "You" : item.name}</strong>
                <small>
                  {Math.max(
                    1,
                    Math.ceil((Date.parse(item.expiresAt) - now) / 3600000),
                  )}
                  h left
                </small>
              </div>
              {item.userId === userId && (
                <button
                  className="header-action"
                  aria-label="Remove moment"
                  onClick={() => void remove(item.id)}
                >
                  <Icon name="close" />
                </button>
              )}
            </header>
            <p>{item.body}</p>
            {item.userId !== userId && item.connectionId && (
              <Link
                className="quiet-link"
                href={`/app/messages?connection=${item.connectionId}`}
              >
                <Icon name="messages" width={18} />
                Continue the conversation
              </Link>
            )}
          </article>
        ))}
        {!visible.length && (
          <div className="moment-empty">
            <Icon name="moments" width={48} height={48} />
            <h2>A small window into your day.</h2>
            <p>
              Share something that could spark a conversation with someone you
              have met.
            </p>
            <button onClick={() => dialog.current?.showModal()}>
              Share a moment
            </button>
          </div>
        )}
      </div>
      <dialog
        className="moment-composer"
        ref={dialog}
        aria-labelledby="moment-title"
      >
        <form onSubmit={publish}>
          <div className="settings-heading">
            <h2 id="moment-title">A moment from your day</h2>
            <button
              type="button"
              className="header-action"
              aria-label="Close composer"
              onClick={() => dialog.current?.close()}
            >
              <Icon name="close" />
            </button>
          </div>
          <label>
            What is on your mind?
            <textarea
              required
              maxLength={600}
              rows={5}
              placeholder="Lately, I have been into..."
              value={body}
              onChange={(event) => setBody(event.target.value)}
            />
          </label>
          <small>{body.length}/600</small>
          <fieldset>
            <legend>Choose a color</legend>
            <div className="theme-choices">
              {["lilac", "mint", "sand"].map((value) => (
                <button
                  type="button"
                  key={value}
                  className={`tone-${value}`}
                  aria-pressed={tone === value}
                  onClick={() => setTone(value)}
                >
                  {value}
                </button>
              ))}
            </div>
          </fieldset>
          <p>
            Visible to current connections for 24 hours. Up to three active
            moments.
          </p>
          <p role="status">{status}</p>
          <button disabled={busy || !body.trim()}>
            {busy ? "Sharing..." : "Share moment"}
          </button>
        </form>
      </dialog>
      <MobileNav current="moments" />
    </main>
  );
}
