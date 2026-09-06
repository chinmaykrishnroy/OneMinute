"use client";
import { useState } from "react";
import { AppHeader, MobileNav } from "@/components/navigation/mobile-nav";
import { Settings, useCommunication } from "./runtime";
export function AppSettings() {
  const { settings, saveSettings } = useCommunication();
  const [status, setStatus] = useState("");
  const [busy, setBusy] = useState(false);
  async function change(next: Settings) {
    setBusy(true);
    try {
      await saveSettings(next);
      setStatus("Settings saved.");
    } catch {
      setStatus("Could not save settings. Please try again.");
    } finally {
      setBusy(false);
    }
  }
  async function permission() {
    if (!("Notification" in window)) {
      setStatus("This browser does not support desktop notifications.");
      return;
    }
    const result = await Notification.requestPermission();
    setStatus(
      result === "granted"
        ? "Browser notifications enabled while OneMinute is open."
        : "Notifications are disabled in your browser. You can change this in site permissions.",
    );
  }
  return (
    <main className="social-shell">
      <AppHeader title="Settings" />
      <section className="settings-page">
        <p className="eyebrow">Make it yours</p>
        <h1>App settings</h1>
        <fieldset disabled={busy}>
          <legend>Appearance</legend>
          <div className="theme-choices">
            {(["system", "light", "dark"] as const).map((theme) => (
              <button
                key={theme}
                aria-pressed={settings.theme === theme}
                onClick={() => void change({ ...settings, theme })}
              >
                {theme === "system"
                  ? "System default"
                  : theme === "light"
                    ? "Light"
                    : "Dark"}
              </button>
            ))}
          </div>
          <p>System default follows your device appearance automatically.</p>
        </fieldset>
        <label className="setting-row">
          <span>
            <strong>Notifications</strong>
            <small>Browser alerts for new messages and incoming calls</small>
          </span>
          <input
            type="checkbox"
            disabled={busy}
            checked={settings.notifications}
            onChange={(e) =>
              void change({ ...settings, notifications: e.target.checked })
            }
          />
        </label>
        <button className="quiet-button" onClick={() => void permission()}>
          Enable browser notifications
        </button>
        <label className="setting-row">
          <span>
            <strong>Typing indicators</strong>
            <small>Let your connections see when you are typing</small>
          </span>
          <input
            type="checkbox"
            disabled={busy}
            checked={settings.typing}
            onChange={(e) =>
              void change({ ...settings, typing: e.target.checked })
            }
          />
        </label>
        <label className="setting-row">
          <span>
            <strong>Read receipts</strong>
            <small>Let your connections know when you read new messages</small>
          </span>
          <input
            type="checkbox"
            disabled={busy}
            checked={settings.readReceipts}
            onChange={(e) =>
              void change({ ...settings, readReceipts: e.target.checked })
            }
          />
        </label>
        <p role="status">{status}</p>
      </section>
      <MobileNav current="profile" />
    </main>
  );
}
