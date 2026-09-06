"use client";
import {
  createContext,
  useContext,
  useEffect,
  useRef,
  useState,
  useCallback,
  ReactNode,
} from "react";
import { usePathname } from "next/navigation";
import { CallOverlay, CallSession } from "./video-call";

export type Settings = {
  theme: "system" | "light" | "dark";
  notifications: boolean;
  typing: boolean;
  readReceipts: boolean;
};
export type Notice = {
  id: number;
  connectionId: string;
  kind: string;
  name: string;
  read: boolean;
  createdAt: string;
};
export type LiveEvent = {
  type: string;
  connectionId?: string;
  call?: CallSession;
  payload?: unknown;
  message?: { id: number; senderId: string };
  user?: { id: string };
  ice?: RTCConfiguration;
  settings?: Settings;
};
type Runtime = {
  api: string;
  userId: string;
  online: boolean;
  settings: Settings;
  saveSettings: (s: Settings) => Promise<void>;
  notices: Notice[];
  refreshNotices: () => Promise<void>;
  subscribe: (fn: (event: LiveEvent) => void) => () => void;
  startCall: (id: string, name: string, video: boolean) => Promise<void>;
};
const defaults: Settings = {
  theme: "system",
  notifications: true,
  typing: true,
  readReceipts: true,
};
const Context = createContext<Runtime | null>(null);
export function useCommunication() {
  const value = useContext(Context);
  if (!value) throw new Error("Communication provider required");
  return value;
}

export function CommunicationProvider({
  api,
  children,
}: {
  api: string;
  children: ReactNode;
}) {
  const pathname = usePathname();
  const authenticatedArea =
    pathname.startsWith("/app/") || pathname === "/onboarding";
  return (
    <CommunicationRuntime
      key={String(authenticatedArea)}
      api={api}
      authenticatedArea={authenticatedArea}
    >
      {children}
    </CommunicationRuntime>
  );
}
function CommunicationRuntime({
  api,
  children,
  authenticatedArea,
}: {
  api: string;
  children: ReactNode;
  authenticatedArea: boolean;
}) {
  const [userId, setUserId] = useState("");
  const [online, setOnline] = useState(false);
  const [settings, setSettings] = useState<Settings>(defaults);
  const settingsRef = useRef(settings);
  const [notices, setNotices] = useState<Notice[]>([]);
  const [call, setCall] = useState<CallSession | null>(null);
  const [ice, setIce] = useState<RTCConfiguration>({ iceServers: [] });
  const listeners = useRef(new Set<(event: LiveEvent) => void>());
  const activeCall = useRef<CallSession | null>(null);
  useEffect(() => {
    activeCall.current = call;
  }, [call]);
  useEffect(() => {
    settingsRef.current = settings;
    const media = window.matchMedia("(prefers-color-scheme: dark)");
    const apply = () => {
      document.documentElement.dataset.theme =
        settings.theme === "system"
          ? media.matches
            ? "dark"
            : "light"
          : settings.theme;
    };
    apply();
    media.addEventListener("change", apply);
    return () => media.removeEventListener("change", apply);
  }, [settings]);
  async function refreshNotices() {
    const response = await fetch(new URL("/v1/notifications", api), {
      credentials: "include",
    });
    if (response.ok) setNotices(await response.json());
  }
  async function saveSettings(next: Settings) {
    const response = await fetch(new URL("/v1/settings", api), {
      method: "PUT",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(next),
    });
    if (!response.ok) throw new Error("Could not save settings");
    setSettings(await response.json());
  }
  const subscribe = useCallback((fn: (event: LiveEvent) => void) => {
    listeners.current.add(fn);
    return () => {
      listeners.current.delete(fn);
    };
  }, []);
  async function startCall(id: string, name: string, video: boolean) {
    if (activeCall.current) throw new Error("You already have a call open.");
    if (!online) throw new Error("Reconnecting. Please try again shortly.");
    const response = await fetch(new URL(`/v1/connections/${id}/calls`, api), {
      method: "POST",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ video }),
    });
    if (!response.ok)
      throw new Error("Could not call. They may already be in a call.");
    const value = await response.json();
    setCall({ ...value, name, outgoing: true });
  }
  useEffect(() => {
    if (!authenticatedArea) return;
    let identity = "";
    let disposed = false;
    let socket: WebSocket | undefined;
    let retry: ReturnType<typeof setTimeout> | undefined;
    let delay = 1000;
    const connect = () => {
      const url = new URL("/v1/events/ws", api);
      url.protocol = url.protocol === "https:" ? "wss:" : "ws:";
      socket = new WebSocket(url);
      socket.onmessage = (e) => {
        let event: LiveEvent;
        try {
          event = JSON.parse(e.data);
        } catch {
          return;
        }
        if (event.type === "ready") {
          setOnline(true);
          delay = 1000;
          if (event.user) {
            identity = event.user.id;
            setUserId(identity);
          }
          if (event.settings) setSettings(event.settings);
          if (event.ice) setIce(event.ice);
          void refresh().catch(() => {});
        }
        if (event.type === "settings.changed" && event.settings)
          setSettings(event.settings);
        if (event.type === "call.invited" && event.call && !activeCall.current)
          setCall({ ...event.call, outgoing: false });
        if (event.type === "message.created" || event.type === "call.invited") {
          void refresh().catch(() => {});
          const incoming =
            event.type === "call.invited" ||
            (event.message && event.message.senderId !== identity);
          if (
            event.type === "message.created" &&
            incoming &&
            event.message &&
            event.connectionId
          )
            void fetch(
              new URL(`/v1/connections/${event.connectionId}/receipt`, api),
              {
                method: "POST",
                credentials: "include",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ id: event.message.id, read: false }),
              },
            ).catch(() => {});
          try {
            if (
              incoming &&
              document.visibilityState === "hidden" &&
              settingsRef.current.notifications &&
              "Notification" in window &&
              Notification.permission === "granted"
            )
              new Notification("OneMinute", {
                body:
                  event.type === "call.invited"
                    ? "Incoming call from a connection"
                    : "You have a new message",
                tag: event.connectionId || event.call?.id,
              });
          } catch {
            /* Some mobile browsers require a service worker for native alerts. */
          }
        }
        listeners.current.forEach((fn) => fn(event));
      };
      socket.onclose = () => {
        if (!disposed) {
          setOnline(false);
          listeners.current.forEach((fn) => fn({ type: "transport.closed" }));
          retry = setTimeout(connect, delay);
          delay = Math.min(delay * 2, 15000);
        }
      };
    };
    async function refresh() {
      const response = await fetch(new URL("/v1/notifications", api), {
        credentials: "include",
      });
      if (response.ok && !disposed) setNotices(await response.json());
    }
    connect();
    const poll = setInterval(() => {
      void refresh().catch(() => {});
    }, 15000);
    return () => {
      disposed = true;
      clearInterval(poll);
      clearTimeout(retry);
      socket?.close();
    };
  }, [api, authenticatedArea]);
  return (
    <Context.Provider
      value={{
        api,
        userId,
        online,
        settings,
        saveSettings,
        notices,
        refreshNotices,
        subscribe,
        startCall,
      }}
    >
      {children}
      {call && (
        <CallOverlay
          key={call.id}
          call={call}
          ice={ice}
          onClose={() => setCall(null)}
        />
      )}
    </Context.Provider>
  );
}
