"use client";
import { useRef, useState } from "react";
import Link from "next/link";
import { useCommunication } from "./runtime";
export function NotificationCenter() {
  const { notices, api, refreshNotices } = useCommunication();
  const dialog = useRef<HTMLDialogElement>(null);
  const [status, setStatus] = useState("");
  const unread = notices.filter((n) => !n.read).length;
  async function readAll() {
    try {
      const response = await fetch(new URL("/v1/notifications/read", api), {
        method: "POST",
        credentials: "include",
      });
      if (!response.ok) throw new Error();
      await refreshNotices();
      setStatus("");
    } catch {
      setStatus("Could not update notifications.");
    }
  }
  return (
    <>
      <button
        className="notification-trigger"
        aria-label={`Notifications${unread ? `, ${unread} unread` : ""}`}
        onClick={() => {
          dialog.current?.showModal();
          void refreshNotices();
        }}
      >
        <svg
          width="22"
          height="22"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="1.8"
          aria-hidden="true"
        >
          <path d="M18 8a6 6 0 0 0-12 0c0 7-3 7-3 9h18c0-2-3-2-3-9M10 21h4" />
        </svg>
        {unread > 0 && <span>{unread > 9 ? "9+" : unread}</span>}
      </button>
      <dialog
        ref={dialog}
        className="account-drawer"
        aria-labelledby="notifications-title"
      >
        <div className="drawer-body">
          <div className="settings-heading">
            <h2 id="notifications-title">Notifications</h2>
            <button
              className="icon-button"
              onClick={() => dialog.current?.close()}
              aria-label="Close notifications"
            >
              Close
            </button>
          </div>
          <button className="quiet-button" onClick={() => void readAll()}>
            Mark all as read
          </button>
          <div className="notification-list">
            {notices.map((n) => (
              <Link
                key={n.id}
                className={n.read ? "" : "unread"}
                href={`/app/messages?connection=${n.connectionId}`}
                onClick={() => dialog.current?.close()}
              >
                <strong>{n.name}</strong>
                <span>
                  {n.kind === "message"
                    ? "Sent you a message"
                    : n.kind === "call"
                      ? "Called you"
                      : "Connected with you"}
                </span>
                <small>{new Date(n.createdAt).toLocaleString()}</small>
              </Link>
            ))}
            {!notices.length && <p>You are all caught up.</p>}
          </div>
          <p role="status">{status}</p>
        </div>
      </dialog>
    </>
  );
}
