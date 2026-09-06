"use client";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { ReactNode } from "react";
import { Icon, IconName } from "./icon";
import { useCommunication } from "@/components/communication/runtime";
export type Tab =
  | "discover"
  | "messages"
  | "moments"
  | "posts"
  | "profile"
  | "connections";
const tabs: { key: IconName; href: string; label: string }[] = [
  { key: "discover", href: "/app/discover", label: "Discover" },
  { key: "messages", href: "/app/messages", label: "Messages" },
  { key: "moments", href: "/app/moments", label: "Moments" },
  { key: "profile", href: "/app/profile", label: "You" },
];
export function MobileNav({ current }: { current: Tab }) {
  const { notices } = useCommunication();
  const unread = notices.some((n) => !n.read);
  return (
    <nav className="product-nav" aria-label="Main navigation">
      <Link
        href="/app/discover"
        className="rail-brand"
        aria-label="OneMinute home"
      >
        OneMinute<span>Meet the person first.</span>
      </Link>
      {tabs.map((tab) => (
        <Link
          key={tab.key}
          href={tab.href}
          aria-label={tab.label}
          title={tab.label}
          aria-current={
            (current === "posts"
              ? "moments"
              : current === "connections"
                ? "profile"
                : current) === tab.key
              ? "page"
              : undefined
          }
        >
          <Icon name={tab.key} />
          <span>{tab.label}</span>
          {tab.key === "profile" && unread && (
            <b className="nav-unread" aria-label="Unread activity" />
          )}
          <i aria-hidden="true" />
        </Link>
      ))}
    </nav>
  );
}
export function AppHeader({
  title,
  action,
  subtitle,
}: {
  title: string;
  action?: ReactNode;
  subtitle?: string;
}) {
  const router = useRouter();
  const defaultAction =
    title === "Discover" ? (
      <Link
        className="header-action"
        href="/app/profile#preferences"
        aria-label="Discovery preferences"
      >
        <Icon name="settings" />
      </Link>
    ) : title === "Messages" ? (
      <Link
        className="header-action"
        href="/app/connections"
        aria-label="Your connections"
      >
        <Icon name="connections" />
      </Link>
    ) : null;
  return (
    <header className="app-header product-header">
      <button
        className="header-action back-button"
        aria-label="Go back"
        onClick={() =>
          window.history.length > 1
            ? router.back()
            : router.replace("/app/discover")
        }
      >
        <Icon name="back" />
      </button>
      <div className="header-heading">
        <h1>{title}</h1>
        {subtitle && <span>{subtitle}</span>}
      </div>
      <div className="header-actions">{action ?? defaultAction}</div>
    </header>
  );
}
