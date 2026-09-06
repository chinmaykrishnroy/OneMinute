"use client";
import Image from "next/image";
import Link from "next/link";
import { useEffect, useRef, useState, FormEvent } from "react";
import { useSearchParams, useRouter } from "next/navigation";
import { AppHeader, MobileNav } from "@/components/navigation/mobile-nav";
import { Icon } from "@/components/navigation/icon";
import { useCommunication } from "./runtime";

type Connection = {
  id: string;
  preview: string;
  unread: number;
  person: { id: string; displayName: string; avatarUrl: string };
};
type Message = {
  id: number;
  clientId: string;
  senderId: string;
  body: string;
  createdAt: string;
};
export function Messages() {
  const search = useSearchParams();
  const selected = search.get("connection") || "";
  return <ConversationInbox key={selected} selected={selected} />;
}
function ConversationInbox({ selected }: { selected: string }) {
  const {
    api,
    userId,
    online,
    settings,
    subscribe,
    startCall,
    refreshNotices,
  } = useCommunication();
  const router = useRouter();
  const [connections, setConnections] = useState<Connection[]>([]);
  const [items, setItems] = useState<Message[]>([]);
  const [draft, setDraft] = useState("");
  const [status, setStatus] = useState("Loading connections...");
  const [sending, setSending] = useState(false);
  const [typing, setTyping] = useState(false);
  const [readId, setReadId] = useState(0);
  const [deliveredId, setDeliveredId] = useState(0);
  const [hasOlder, setHasOlder] = useState(false);
  const olderExhausted = useRef(false);
  const skipScroll = useRef(false);
  const stickToBottom = useRef(true);
  const lastTyping = useRef(0);
  const retry = useRef<{ body: string; clientId: string } | null>(null);
  const typingTimer = useRef<ReturnType<typeof setTimeout> | undefined>(
    undefined,
  );
  const bottom = useRef<HTMLDivElement>(null);
  const person = connections.find((c) => c.id === selected)?.person;
  useEffect(() => {
    let stopped = false;
    async function load() {
      try {
        const response = await fetch(new URL("/v1/conversations", api), {
          credentials: "include",
        });
        if (response.status === 401) {
          router.replace("/");
          return;
        }
        if (!response.ok) throw new Error();
        const data = await response.json();
        if (!stopped) {
          setConnections(data.connections);
          setStatus("");
        }
      } catch {
        if (!stopped) setStatus("Could not load connections. Please refresh.");
      }
    }
    void load();
    const timer = setInterval(() => void load(), 15000);
    const unsubscribe = subscribe((event) => {
      if (event.type === "message.created") void load();
    });
    return () => {
      stopped = true;
      clearInterval(timer);
      unsubscribe();
    };
  }, [api, router, subscribe]);
  useEffect(() => {
    if (!selected) return;
    let stopped = false;
    let loading = false;
    async function load() {
      if (loading) return;
      loading = true;
      try {
        const response = await fetch(
          new URL(`/v1/connections/${selected}/messages`, api),
          { credentials: "include" },
        );
        if (!response.ok) {
          if (!stopped) {
            setStatus(
              response.status === 403
                ? "This connection is no longer available."
                : "Could not load messages.",
            );
            setItems([]);
          }
          return;
        }
        const data = await response.json();
        if (stopped) return;
        const next: Message[] = data.messages.reverse();
        setItems((current) =>
          [
            ...new Map([...current, ...next].map((m) => [m.id, m])).values(),
          ].sort((a, b) => a.id - b.id),
        );
        setHasOlder(!olderExhausted.current && data.messages.length === 60);
        setReadId(data.readId);
        setDeliveredId(data.deliveredId);
        setStatus("");
        const incoming = next.filter((m) => m.senderId !== userId).at(-1);
        if (incoming && userId) {
          await fetch(new URL(`/v1/connections/${selected}/receipt`, api), {
            method: "POST",
            credentials: "include",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({
              id: incoming.id,
              read: document.visibilityState === "visible",
            }),
          });
        }
      } catch {
        if (!stopped)
          setStatus("Connection interrupted. Your messages are saved.");
      } finally {
        loading = false;
      }
    }
    void load();
    const interval = setInterval(() => void load(), 5000);
    const visibility = () => {
      if (document.visibilityState === "visible") void load();
    };
    document.addEventListener("visibilitychange", visibility);
    const unsubscribe = subscribe((event) => {
      if (event.connectionId !== selected) return;
      if (event.type === "typing") {
        setTyping(true);
        clearTimeout(typingTimer.current);
        typingTimer.current = setTimeout(() => setTyping(false), 4000);
      } else if (event.type === "message.created" || event.type === "receipt")
        void load();
    });
    return () => {
      stopped = true;
      clearInterval(interval);
      unsubscribe();
      clearTimeout(typingTimer.current);
      document.removeEventListener("visibilitychange", visibility);
    };
  }, [api, selected, userId, subscribe]);
  useEffect(() => {
    if (skipScroll.current) {
      skipScroll.current = false;
      return;
    }
    if (stickToBottom.current)
      bottom.current?.scrollIntoView({ block: "nearest" });
  }, [items.length]);
  function choose(id: string) {
    router.replace(`/app/messages?connection=${id}`);
  }
  async function send(event: FormEvent) {
    event.preventDefault();
    if (!draft.trim() || sending || !selected) return;
    setSending(true);
    const pending =
      retry.current?.body === draft.trim()
        ? retry.current
        : { body: draft.trim(), clientId: crypto.randomUUID() };
    retry.current = pending;
    try {
      const response = await fetch(
        new URL(`/v1/connections/${selected}/messages`, api),
        {
          method: "POST",
          credentials: "include",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(pending),
        },
      );
      if (!response.ok) throw new Error();
      const message: Message = await response.json();
      stickToBottom.current = true;
      setItems((current) =>
        [...new Map([...current, message].map((m) => [m.id, m])).values()].sort(
          (a, b) => a.id - b.id,
        ),
      );
      setDraft("");
      retry.current = null;
      setStatus("");
      void refreshNotices().catch(() => {});
    } catch {
      setStatus("Not sent. Your draft is safe. Tap Send to retry.");
    } finally {
      setSending(false);
    }
  }
  async function older() {
    if (!items.length) return;
    const response = await fetch(
      new URL(
        `/v1/connections/${selected}/messages?before=${items[0].id}`,
        api,
      ),
      { credentials: "include" },
    );
    if (!response.ok) {
      setStatus("Could not load earlier messages.");
      return;
    }
    const data = await response.json();
    skipScroll.current = true;
    olderExhausted.current = data.messages.length < 60;
    setItems((current) => [...data.messages.reverse(), ...current]);
    setHasOlder(data.messages.length === 60);
  }
  async function call(video: boolean) {
    if (!person) return;
    try {
      await startCall(selected, person.displayName, video);
    } catch (error) {
      setStatus(error instanceof Error ? error.message : "Could not call.");
    }
  }
  return (
    <main className="social-shell messenger-shell">
      <AppHeader title="Messages" />
      <div className={`messenger ${selected ? "chat-selected" : ""}`}>
        <aside className="conversation-list">
          <h1>Messages</h1>
          <p className="field-hint">
            {online ? "Connected" : "Reconnecting..."}
          </p>
          {connections.map((c) => (
            <button
              className={`conversation-item ${c.id === selected ? "selected" : ""}`}
              key={c.id}
              onClick={() => choose(c.id)}
            >
              {c.person.avatarUrl ? (
                <Image
                  src={c.person.avatarUrl}
                  alt=""
                  width={44}
                  height={44}
                  unoptimized
                />
              ) : (
                <span className="avatar-fallback">
                  {c.person.displayName[0]}
                </span>
              )}
              <span>
                <strong>{c.person.displayName}</strong>
                <small>{c.preview || "Say hello"}</small>
                {c.unread > 0 && (
                  <small className="unread-count">{c.unread} unread</small>
                )}
              </span>
            </button>
          ))}
          {!connections.length && (
            <p>
              Your mutual connections will appear here.{" "}
              <Link href="/app/discover">Meet someone</Link>
            </p>
          )}
        </aside>
        <section className="chat-panel">
          {person ? (
            <>
              <header className="chat-header">
                <button
                  className="chat-back icon-button"
                  aria-label="Back to conversations"
                  onClick={() => {
                    router.replace("/app/messages");
                  }}
                >
                  <Icon name="back" />
                </button>
                <div>
                  <h2>{person.displayName}</h2>
                  <span className="field-hint">
                    {typing ? "Typing..." : "Your connection"}
                  </span>
                </div>
                <button
                  className="icon-button"
                  onClick={() => void call(false)}
                  disabled={!online}
                  aria-label="Start audio call"
                >
                  <Icon name="phone" />
                </button>
                <button
                  className="icon-button"
                  onClick={() => void call(true)}
                  disabled={!online}
                  aria-label="Start video call"
                >
                  <Icon name="video" />
                </button>
              </header>
              <div
                className="message-history"
                onScroll={(e) => {
                  const el = e.currentTarget;
                  stickToBottom.current =
                    el.scrollHeight - el.scrollTop - el.clientHeight < 80;
                }}
                role="log"
                aria-label="Conversation"
                aria-live="polite"
              >
                {hasOlder && (
                  <button className="quiet-button" onClick={() => void older()}>
                    Earlier messages
                  </button>
                )}
                {!items.length && (
                  <p className="chat-empty">
                    You both chose to connect. Keep your conversation going.
                  </p>
                )}
                {items.map((m) => (
                  <article
                    key={m.id}
                    className={`message-bubble ${m.senderId === userId ? "sent" : "received"}`}
                  >
                    <p>{m.body}</p>
                    <small>
                      {new Date(m.createdAt).toLocaleTimeString([], {
                        hour: "2-digit",
                        minute: "2-digit",
                      })}
                      {m.senderId === userId &&
                        ` - ${m.id <= readId ? "Read" : m.id <= deliveredId ? "Delivered" : "Sent"}`}
                    </small>
                  </article>
                ))}
                <div ref={bottom} />
              </div>
              <p role="status" className="message-status">
                {status}
              </p>
              <form className="message-composer" onSubmit={send}>
                <label className="sr-only" htmlFor="message-draft">
                  Message
                </label>
                <textarea
                  id="message-draft"
                  rows={2}
                  maxLength={4000}
                  value={draft}
                  placeholder="Write a message..."
                  onChange={(e) => {
                    setDraft(e.target.value);
                    if (
                      settings.typing &&
                      Date.now() - lastTyping.current > 2000
                    ) {
                      lastTyping.current = Date.now();
                      void fetch(
                        new URL(`/v1/connections/${selected}/typing`, api),
                        { method: "POST", credentials: "include" },
                      );
                    }
                  }}
                  onKeyDown={(e) => {
                    if (
                      e.key === "Enter" &&
                      !e.shiftKey &&
                      !e.nativeEvent.isComposing
                    ) {
                      e.preventDefault();
                      e.currentTarget.form?.requestSubmit();
                    }
                  }}
                />
                <button disabled={sending || !draft.trim()}>
                  {sending ? "Sending..." : "Send"}
                </button>
              </form>
            </>
          ) : (
            <div className="chat-empty">
              <h2>A conversation worth keeping.</h2>
              <p>Choose a connection to send a message or start a call.</p>
              <p role="status">{status}</p>
            </div>
          )}
        </section>
      </div>
      <MobileNav current="messages" />
    </main>
  );
}
