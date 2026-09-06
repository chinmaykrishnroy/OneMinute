"use client";
import { useState } from "react";
import Link from "next/link";
import { AppHeader, MobileNav } from "@/components/navigation/mobile-nav";
import { useCommunication } from "./runtime";

export function Activity() {
  const { notices, api, refreshNotices } = useCommunication();
  const [status, setStatus] = useState("");
  async function markRead() {
    try {
      const response = await fetch(new URL("/v1/notifications/read", api), {
        method: "POST",
        credentials: "include",
      });
      if (!response.ok) throw new Error();
      await refreshNotices();
      setStatus("All caught up.");
    } catch {
      setStatus("Could not update activity. Try again.");
    }
  }
  return (
    <main className="social-shell activity-page">
      <AppHeader
        title="Activity"
        action={
          <button
            className="quiet-button header-text-action"
            onClick={() => void markRead()}
          >
            Mark read
          </button>
        }
      />
      <p>Messages, calls, and the people you chose to keep.</p>
      <div className="activity-list">
        {notices.map((notice) => (
          <Link
            key={notice.id}
            className={notice.read ? "" : "unread"}
            href={`/app/messages?connection=${notice.connectionId}`}
          >
            <span className="avatar-fallback">{notice.name[0]}</span>
            <div>
              <strong>{notice.name}</strong>
              <p>
                {notice.kind === "message"
                  ? "Sent you a message"
                  : notice.kind === "call"
                    ? "Called you"
                    : "Connected with you"}
              </p>
              <time dateTime={notice.createdAt}>
                {new Date(notice.createdAt).toLocaleString()}
              </time>
            </div>
            {!notice.read && (
              <span className="status-dot" aria-label="Unread" />
            )}
          </Link>
        ))}
        {!notices.length && (
          <p>No activity yet. Your next hello could change that.</p>
        )}
      </div>
      <p role="status">{status}</p>
      <MobileNav current="profile" />
    </main>
  );
}
